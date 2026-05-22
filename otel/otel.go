// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MIT

// Package otel provides a MetricSink that pushes metrics to an
// OpenTelemetry-compatible backend via OTLP (gRPC or HTTP).
package otel

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/go-metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	otelmetric "go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
	"google.golang.org/grpc/credentials"
)

// OTELSink is a MetricSink that exports metrics to OpenTelemetry-compatible backends.
type OTELSink struct {
	provider *metric.MeterProvider
	meter    otelmetric.Meter
	ctx      context.Context
	cancel   context.CancelFunc

	// Instrument caches
	gauges      sync.Map // map[string]otelmetric.Float64Gauge
	counters    sync.Map // map[string]otelmetric.Float64Counter
	histograms  sync.Map // map[string]otelmetric.Float64Histogram
	warnedNames sync.Map // tracks names that already logged a creation error

	// Configuration
	shutdownTimeout time.Duration
	shutdownOnce    sync.Once
	shutdownErr     error
	emitKeyWarned   sync.Once
}

// NewOTELSink creates a new OpenTelemetry metrics sink that pushes metrics to an
// OTLP endpoint using either gRPC (default) or HTTP transport. The sink creates
// the exporter, reader, and MeterProvider internally and owns their lifecycle;
// it shuts them down when Shutdown is called.
//
// opts.Endpoint must not be empty; an error is returned if it is.
// opts.Protocol must be "", "grpc", or "http"; an error is returned otherwise.
func NewOTELSink(opts OTELSinkOpts) (*OTELSink, error) {
	if opts.Endpoint == "" {
		return nil, fmt.Errorf("otel: Endpoint must not be empty")
	}
	if opts.ExpHistogramMaxSize < 0 {
		return nil, fmt.Errorf("otel: ExpHistogramMaxSize must not be negative")
	}
	if opts.ExpHistogramMaxScale < 0 {
		return nil, fmt.Errorf("otel: ExpHistogramMaxScale must not be negative")
	}
	if (opts.TLSCertFile == "") != (opts.TLSKeyFile == "") {
		return nil, fmt.Errorf("otel: TLSCertFile and TLSKeyFile must both be set or both be empty")
	}

	exporter, err := buildExporter(opts)
	if err != nil {
		return nil, err
	}

	pushInterval := defaultDuration(opts.PushInterval, DefaultPushInterval)
	reader := metric.NewPeriodicReader(exporter, metric.WithInterval(pushInterval))

	res, err := buildResource(opts.ResourceAttributes)
	if err != nil {
		_ = reader.Shutdown(context.Background())
		return nil, fmt.Errorf("otel: failed to build resource: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	sink := &OTELSink{
		ctx:             ctx,
		cancel:          cancel,
		shutdownTimeout: defaultDuration(opts.ShutdownTimeout, DefaultShutdownTimeout),
	}
	sink.provider = buildMeterProvider(res, reader, opts)
	sink.meter = sink.provider.Meter("go-metrics/otel")
	return sink, nil
}

// buildExporter creates the OTLP metric exporter based on the configured protocol.
func buildExporter(opts OTELSinkOpts) (metric.Exporter, error) {
	protocol := opts.Protocol
	if protocol == "" {
		protocol = ProtocolGRPC
	}

	switch protocol {
	case ProtocolGRPC:
		return buildGRPCExporter(opts)
	case ProtocolHTTP:
		return buildHTTPExporter(opts)
	default:
		return nil, fmt.Errorf("otel: unsupported Protocol %q (must be %q or %q)", opts.Protocol, ProtocolGRPC, ProtocolHTTP)
	}
}

// buildGRPCExporter creates an OTLP gRPC metric exporter.
func buildGRPCExporter(opts OTELSinkOpts) (metric.Exporter, error) {
	exporterOpts := []otlpmetricgrpc.Option{
		otlpmetricgrpc.WithEndpoint(opts.Endpoint),
	}
	if opts.Insecure {
		exporterOpts = append(exporterOpts, otlpmetricgrpc.WithInsecure())
	} else if opts.TLSCertFile != "" || opts.TLSCAFile != "" {
		tlsCfg, err := buildTLSConfig(opts.TLSCertFile, opts.TLSKeyFile, opts.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("otel: %w", err)
		}
		exporterOpts = append(exporterOpts, otlpmetricgrpc.WithTLSCredentials(credentials.NewTLS(tlsCfg)))
	}
	if len(opts.Headers) > 0 {
		exporterOpts = append(exporterOpts, otlpmetricgrpc.WithHeaders(opts.Headers))
	}
	if opts.GzipCompression {
		exporterOpts = append(exporterOpts, otlpmetricgrpc.WithCompressor("gzip"))
	}
	if opts.DeltaTemporality {
		exporterOpts = append(exporterOpts, otlpmetricgrpc.WithTemporalitySelector(
			func(metric.InstrumentKind) metricdata.Temporality {
				return metricdata.DeltaTemporality
			},
		))
	}

	exp, err := otlpmetricgrpc.New(context.Background(), exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("otel: failed to create OTLP gRPC exporter: %w", err)
	}
	return exp, nil
}

// buildHTTPExporter creates an OTLP HTTP metric exporter.
func buildHTTPExporter(opts OTELSinkOpts) (metric.Exporter, error) {
	exporterOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(opts.Endpoint),
	}
	if opts.Insecure {
		exporterOpts = append(exporterOpts, otlpmetrichttp.WithInsecure())
	} else if opts.TLSCertFile != "" || opts.TLSCAFile != "" {
		tlsCfg, err := buildTLSConfig(opts.TLSCertFile, opts.TLSKeyFile, opts.TLSCAFile)
		if err != nil {
			return nil, fmt.Errorf("otel: %w", err)
		}
		exporterOpts = append(exporterOpts, otlpmetrichttp.WithTLSClientConfig(tlsCfg))
	}
	if len(opts.Headers) > 0 {
		exporterOpts = append(exporterOpts, otlpmetrichttp.WithHeaders(opts.Headers))
	}
	if opts.GzipCompression {
		exporterOpts = append(exporterOpts, otlpmetrichttp.WithCompression(otlpmetrichttp.GzipCompression))
	}
	if opts.DeltaTemporality {
		exporterOpts = append(exporterOpts, otlpmetrichttp.WithTemporalitySelector(
			func(metric.InstrumentKind) metricdata.Temporality {
				return metricdata.DeltaTemporality
			},
		))
	}
	if opts.URLPath != "" {
		exporterOpts = append(exporterOpts, otlpmetrichttp.WithURLPath(opts.URLPath))
	}

	exp, err := otlpmetrichttp.New(context.Background(), exporterOpts...)
	if err != nil {
		return nil, fmt.Errorf("otel: failed to create OTLP HTTP exporter: %w", err)
	}
	return exp, nil
}

// buildResource creates the OTEL resource from the given attributes.
func buildResource(resourceAttributes map[string]string) (*resource.Resource, error) {
	var attrs []attribute.KeyValue
	for k, v := range resourceAttributes {
		attrs = append(attrs, attribute.String(k, v))
	}

	if len(attrs) == 0 {
		return resource.Default(), nil
	}

	// Create a resource without schema URL to avoid conflicts with resource.Default().
	customRes := resource.NewSchemaless(attrs...)
	merged, err := resource.Merge(resource.Default(), customRes)
	if err != nil {
		return nil, fmt.Errorf("failed to merge resources: %w", err)
	}
	return merged, nil
}

// buildTLSConfig creates a tls.Config from optional cert, key, and CA file paths.
func buildTLSConfig(certFile, keyFile, caFile string) (*tls.Config, error) {
	tlsCfg := &tls.Config{}

	if certFile != "" && keyFile != "" {
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}
		tlsCfg.Certificates = []tls.Certificate{cert}
	}

	if caFile != "" {
		caPEM, err := os.ReadFile(caFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}
		caPool := x509.NewCertPool()
		if !caPool.AppendCertsFromPEM(caPEM) {
			return nil, fmt.Errorf("failed to parse CA certificate from %s", caFile)
		}
		tlsCfg.RootCAs = caPool
	}

	return tlsCfg, nil
}

// buildMeterProvider constructs a MeterProvider with the given resource, reader,
// and histogram configuration.
func buildMeterProvider(res *resource.Resource, reader metric.Reader, opts OTELSinkOpts) *metric.MeterProvider {
	providerOpts := []metric.Option{
		metric.WithResource(res),
		metric.WithReader(reader),
	}
	if opts.CardinalityLimit > 0 {
		providerOpts = append(providerOpts, metric.WithCardinalityLimit(opts.CardinalityLimit))
	}

	if !opts.UseExplicitHistograms {
		maxSize := opts.ExpHistogramMaxSize
		if maxSize == 0 {
			maxSize = DefaultExpHistogramMaxSize
		}
		maxScale := opts.ExpHistogramMaxScale
		if maxScale == 0 {
			maxScale = DefaultExpHistogramMaxScale
		}
		providerOpts = append(providerOpts,
			metric.WithView(metric.NewView(
				metric.Instrument{Kind: metric.InstrumentKindHistogram},
				metric.Stream{
					Aggregation: metric.AggregationBase2ExponentialHistogram{
						MaxSize:  maxSize,
						MaxScale: maxScale,
					},
				},
			)))
	} else if len(opts.HistogramBuckets) > 0 {
		providerOpts = append(providerOpts,
			metric.WithView(metric.NewView(
				metric.Instrument{Kind: metric.InstrumentKindHistogram},
				metric.Stream{
					Aggregation: metric.AggregationExplicitBucketHistogram{
						Boundaries: opts.HistogramBuckets,
					},
				},
			)))
	} else {
		providerOpts = append(providerOpts,
			metric.WithView(metric.NewView(
				metric.Instrument{Kind: metric.InstrumentKindHistogram},
				metric.Stream{
					Aggregation: metric.AggregationExplicitBucketHistogram{
						Boundaries: DefaultHistogramBuckets(),
					},
				},
			)))
	}

	return metric.NewMeterProvider(providerOpts...)
}

// SetGauge sets a gauge value with 32-bit precision.
func (s *OTELSink) SetGauge(key []string, val float32) {
	s.SetGaugeWithLabels(key, val, nil)
}

// SetGaugeWithLabels sets a gauge value with 32-bit precision and labels.
func (s *OTELSink) SetGaugeWithLabels(key []string, val float32, labels []metrics.Label) {
	s.SetPrecisionGaugeWithLabels(key, float64(val), labels)
}

// SetPrecisionGauge sets a gauge value with 64-bit precision.
func (s *OTELSink) SetPrecisionGauge(key []string, val float64) {
	s.SetPrecisionGaugeWithLabels(key, val, nil)
}

// SetPrecisionGaugeWithLabels sets a gauge value with 64-bit precision and labels.
func (s *OTELSink) SetPrecisionGaugeWithLabels(key []string, val float64, labels []metrics.Label) {
	name := s.flattenKey(key)
	gauge := getOrCreate(&s.gauges, &s.warnedNames, name, func(n string) (otelmetric.Float64Gauge, error) {
		return s.meter.Float64Gauge(n)
	})
	gauge.Record(s.ctx, val, otelmetric.WithAttributeSet(labelsToAttributes(labels)))
}

// EmitKey is a no-op for the OTEL sink. OpenTelemetry does not have a direct
// equivalent for arbitrary key/value emissions.
func (s *OTELSink) EmitKey(key []string, val float32) {
	s.emitKeyWarned.Do(func() {
		log.Printf("[WARN] go-metrics/otel: EmitKey is not supported by the OTEL sink; calls will be dropped")
	})
}

// IncrCounter increments a counter.
func (s *OTELSink) IncrCounter(key []string, val float32) {
	s.IncrCounterWithLabels(key, val, nil)
}

// IncrCounterWithLabels increments a counter with labels.
func (s *OTELSink) IncrCounterWithLabels(key []string, val float32, labels []metrics.Label) {
	name := s.flattenKey(key)
	if val < 0 {
		if _, alreadyWarned := s.warnedNames.LoadOrStore("neg:"+name, struct{}{}); !alreadyWarned {
			log.Printf("[WARN] go-metrics/otel: ignoring negative counter increment for %q: %v", name, val)
		}
		return
	}
	counter := getOrCreate(&s.counters, &s.warnedNames, name, func(n string) (otelmetric.Float64Counter, error) {
		return s.meter.Float64Counter(n)
	})
	counter.Add(s.ctx, float64(val), otelmetric.WithAttributeSet(labelsToAttributes(labels)))
}

// AddSample adds a sample to a histogram.
func (s *OTELSink) AddSample(key []string, val float32) {
	s.AddSampleWithLabels(key, val, nil)
}

// AddSampleWithLabels adds a sample to a histogram with labels.
func (s *OTELSink) AddSampleWithLabels(key []string, val float32, labels []metrics.Label) {
	name := s.flattenKey(key)
	histogram := getOrCreate(&s.histograms, &s.warnedNames, name, func(n string) (otelmetric.Float64Histogram, error) {
		return s.meter.Float64Histogram(n)
	})
	histogram.Record(s.ctx, float64(val), otelmetric.WithAttributeSet(labelsToAttributes(labels)))
}

// Shutdown stops the sink and flushes any remaining metrics.
// It blocks until metrics are flushed or the default shutdown timeout is reached.
func (s *OTELSink) Shutdown() {
	ctx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
	defer cancel()
	if err := s.ShutdownContext(ctx); err != nil {
		log.Printf("[WARN] go-metrics/otel: shutdown error (metrics may not have been flushed): %v", err)
	}
}

// ShutdownContext stops the sink and flushes any remaining metrics.
// It blocks until metrics are flushed or the context is cancelled.
// Returns an error if shutdown fails or times out.
// Only the first call performs shutdown; subsequent calls return the result
// of the first call regardless of the provided context.
func (s *OTELSink) ShutdownContext(ctx context.Context) error {
	s.shutdownOnce.Do(func() {
		s.cancel()
		s.shutdownErr = s.provider.Shutdown(ctx)
	})
	return s.shutdownErr
}

// flattenKey joins key parts with dots.
func (s *OTELSink) flattenKey(parts []string) string {
	return strings.Join(parts, ".")
}

// getOrCreate returns a cached instrument or creates one via the create function.
// It uses LoadOrStore to handle concurrent creation safely.
func getOrCreate[I any](cache *sync.Map, warned *sync.Map, name string, create func(string) (I, error)) I {
	if v, ok := cache.Load(name); ok {
		return v.(I)
	}
	inst, err := create(name)
	if err != nil {
		if _, alreadyWarned := warned.LoadOrStore(name, struct{}{}); !alreadyWarned {
			log.Printf("[WARN] go-metrics/otel: failed to create instrument %q: %v", name, err)
		}
	}
	actual, _ := cache.LoadOrStore(name, inst)
	return actual.(I)
}

// labelsToAttributes converts go-metrics labels to an OTEL attribute.Set,
// which is the deduplicated, sorted form the SDK expects for recording.
func labelsToAttributes(labels []metrics.Label) attribute.Set {
	if len(labels) == 0 {
		return *attribute.EmptySet()
	}
	attrs := make([]attribute.KeyValue, len(labels))
	for i, l := range labels {
		attrs[i] = attribute.String(l.Name, l.Value)
	}
	return attribute.NewSet(attrs...)
}

// defaultDuration returns the value if positive, otherwise the default.
func defaultDuration(value, defaultValue time.Duration) time.Duration {
	if value <= 0 {
		return defaultValue
	}
	return value
}
