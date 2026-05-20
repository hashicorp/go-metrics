// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MIT

package otel

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/hashicorp/go-metrics"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/metric/metricdata"
	"go.opentelemetry.io/otel/sdk/resource"
)

// Compile-time interface checks.
var (
	_ metrics.MetricSink               = (*OTELSink)(nil)
	_ metrics.PrecisionGaugeMetricSink = (*OTELSink)(nil)
	_ metrics.ShutdownSink             = (*OTELSink)(nil)
)

// newTestSink creates a sink backed by a ManualReader-based MeterProvider.
// It constructs the sink directly (bypassing NewOTELSink) so tests don't
// need a real OTLP endpoint.
func newTestSink(t *testing.T) (*OTELSink, *metric.ManualReader) {
	t.Helper()
	return newTestSinkWithHistogram(t, false, nil)
}

func newTestSinkWithHistogram(t *testing.T, useExplicit bool, buckets []float64) (*OTELSink, *metric.ManualReader) {
	t.Helper()
	reader := metric.NewManualReader()
	provider := buildMeterProvider(
		mustBuildResource(t, nil),
		reader,
		OTELSinkOpts{
			UseExplicitHistograms: useExplicit,
			HistogramBuckets:      buckets,
		},
	)
	ctx, cancel := context.WithCancel(context.Background())
	sink := &OTELSink{
		ctx:             ctx,
		cancel:          cancel,
		shutdownTimeout: DefaultShutdownTimeout,
	}
	sink.provider = provider
	sink.meter = provider.Meter("go-metrics/otel")
	t.Cleanup(func() { provider.Shutdown(context.Background()) }) //nolint:errcheck
	return sink, reader
}

// mustBuildResource calls buildResource and fatals on error.
func mustBuildResource(t *testing.T, attrs map[string]string) *resource.Resource {
	t.Helper()
	res, err := buildResource(attrs)
	if err != nil {
		t.Fatalf("buildResource failed: %v", err)
	}
	return res
}

// collectMetrics collects from a ManualReader.
func collectMetrics(t *testing.T, reader *metric.ManualReader) metricdata.ResourceMetrics {
	t.Helper()
	var rm metricdata.ResourceMetrics
	if err := reader.Collect(context.Background(), &rm); err != nil {
		t.Fatalf("failed to collect metrics: %v", err)
	}
	return rm
}

// findMetric finds a metric by name in collected resource metrics.
func findMetric(rm metricdata.ResourceMetrics, name string) *metricdata.Metrics {
	for _, sm := range rm.ScopeMetrics {
		for i := range sm.Metrics {
			if sm.Metrics[i].Name == name {
				return &sm.Metrics[i]
			}
		}
	}
	return nil
}

// --- NewOTELSink (endpoint constructor) tests ---

func TestNewOTELSink_EmptyEndpoint(t *testing.T) {
	_, err := NewOTELSink(OTELSinkOpts{})
	if err == nil {
		t.Fatal("expected error for empty endpoint")
	}
	if !strings.Contains(err.Error(), "Endpoint must not be empty") {
		t.Errorf("expected 'Endpoint must not be empty', got: %v", err)
	}
}

func TestNewOTELSink_ValidEndpoint(t *testing.T) {
	sink, err := NewOTELSink(OTELSinkOpts{
		Endpoint:        "localhost:4317",
		Insecure:        true,
		ShutdownTimeout: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sink == nil {
		t.Fatal("sink should not be nil")
	}
	sink.Shutdown()
}

func TestNewOTELSink_NegativeExpHistogramMaxSize(t *testing.T) {
	_, err := NewOTELSink(OTELSinkOpts{
		Endpoint:            "localhost:4317",
		Insecure:            true,
		ExpHistogramMaxSize: -1,
	})
	if err == nil {
		t.Fatal("expected error for negative ExpHistogramMaxSize")
	}
	if !strings.Contains(err.Error(), "ExpHistogramMaxSize must not be negative") {
		t.Errorf("expected error containing 'ExpHistogramMaxSize must not be negative', got: %v", err)
	}
}

func TestNewOTELSink_NegativeExpHistogramMaxScale(t *testing.T) {
	_, err := NewOTELSink(OTELSinkOpts{
		Endpoint:             "localhost:4317",
		Insecure:             true,
		ExpHistogramMaxScale: -1,
	})
	if err == nil {
		t.Fatal("expected error for negative ExpHistogramMaxScale")
	}
	if !strings.Contains(err.Error(), "ExpHistogramMaxScale must not be negative") {
		t.Errorf("expected error containing 'ExpHistogramMaxScale must not be negative', got: %v", err)
	}
}

// --- Protocol tests ---

func TestNewOTELSink_HTTPProtocol(t *testing.T) {
	sink, err := NewOTELSink(OTELSinkOpts{
		Endpoint:        "localhost:4318",
		Protocol:        ProtocolHTTP,
		Insecure:        true,
		ShutdownTimeout: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sink == nil {
		t.Fatal("sink should not be nil")
	}
	sink.Shutdown()
}

func TestNewOTELSink_InvalidProtocol(t *testing.T) {
	_, err := NewOTELSink(OTELSinkOpts{
		Endpoint: "localhost:4317",
		Protocol: "websocket",
		Insecure: true,
	})
	if err == nil {
		t.Fatal("expected error for unsupported protocol")
	}
	if !strings.Contains(err.Error(), "unsupported Protocol") {
		t.Errorf("expected error containing 'unsupported Protocol', got: %v", err)
	}
}

// --- TLS configuration tests ---

func TestNewOTELSink_TLSCertWithoutKey(t *testing.T) {
	_, err := NewOTELSink(OTELSinkOpts{
		Endpoint:    "localhost:4317",
		TLSCertFile: "/some/cert.pem",
	})
	if err == nil {
		t.Fatal("expected error when TLSCertFile is set without TLSKeyFile")
	}
	if !strings.Contains(err.Error(), "TLSCertFile and TLSKeyFile must both be set") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewOTELSink_TLSKeyWithoutCert(t *testing.T) {
	_, err := NewOTELSink(OTELSinkOpts{
		Endpoint:   "localhost:4317",
		TLSKeyFile: "/some/key.pem",
	})
	if err == nil {
		t.Fatal("expected error when TLSKeyFile is set without TLSCertFile")
	}
	if !strings.Contains(err.Error(), "TLSCertFile and TLSKeyFile must both be set") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewOTELSink_TLSCertFileNotFound(t *testing.T) {
	_, err := NewOTELSink(OTELSinkOpts{
		Endpoint:    "localhost:4317",
		TLSCertFile: "/nonexistent/cert.pem",
		TLSKeyFile:  "/nonexistent/key.pem",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent cert files")
	}
	if !strings.Contains(err.Error(), "failed to load client certificate") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewOTELSink_TLSCAFileNotFound(t *testing.T) {
	_, err := NewOTELSink(OTELSinkOpts{
		Endpoint:  "localhost:4317",
		TLSCAFile: "/nonexistent/ca.pem",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent CA file")
	}
	if !strings.Contains(err.Error(), "failed to read CA certificate") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestNewOTELSink_TLSCAFileInvalidPEM(t *testing.T) {
	// Write a file with invalid PEM content.
	tmpFile := t.TempDir() + "/bad-ca.pem"
	if err := os.WriteFile(tmpFile, []byte("not a valid PEM"), 0600); err != nil {
		t.Fatalf("failed to write temp file: %v", err)
	}

	_, err := NewOTELSink(OTELSinkOpts{
		Endpoint:  "localhost:4317",
		TLSCAFile: tmpFile,
	})
	if err == nil {
		t.Fatal("expected error for invalid PEM in CA file")
	}
	if !strings.Contains(err.Error(), "failed to parse CA certificate") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestBuildTLSConfig_ValidCerts(t *testing.T) {
	// Generate a self-signed CA and client cert for testing.
	caDir := t.TempDir()
	certFile, keyFile, caFile := generateTestCerts(t, caDir)

	cfg, err := buildTLSConfig(certFile, keyFile, caFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Certificates) != 1 {
		t.Errorf("expected 1 client certificate, got %d", len(cfg.Certificates))
	}
	if cfg.RootCAs == nil {
		t.Error("expected RootCAs to be set")
	}
}

func TestBuildTLSConfig_CAOnly(t *testing.T) {
	caDir := t.TempDir()
	_, _, caFile := generateTestCerts(t, caDir)

	cfg, err := buildTLSConfig("", "", caFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Certificates) != 0 {
		t.Errorf("expected no client certificates, got %d", len(cfg.Certificates))
	}
	if cfg.RootCAs == nil {
		t.Error("expected RootCAs to be set")
	}
}

// generateTestCerts creates a self-signed CA and client certificate in dir.
// Returns paths to the cert, key, and CA files.
func generateTestCerts(t *testing.T, dir string) (certFile, keyFile, caFile string) {
	t.Helper()

	// Generate CA key and self-signed cert.
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate CA key: %v", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "Test CA"},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
	}

	caCertDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create CA certificate: %v", err)
	}

	caFile = dir + "/ca.pem"
	if err := os.WriteFile(caFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caCertDER}), 0600); err != nil {
		t.Fatalf("failed to write CA cert: %v", err)
	}

	// Generate client key and cert signed by the CA.
	clientKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("failed to generate client key: %v", err)
	}

	caCert, err := x509.ParseCertificate(caCertDER)
	if err != nil {
		t.Fatalf("failed to parse CA certificate: %v", err)
	}

	clientTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "Test Client"},
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	}

	clientCertDER, err := x509.CreateCertificate(rand.Reader, clientTemplate, caCert, &clientKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("failed to create client certificate: %v", err)
	}

	certFile = dir + "/client-cert.pem"
	if err := os.WriteFile(certFile, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: clientCertDER}), 0600); err != nil {
		t.Fatalf("failed to write client cert: %v", err)
	}

	keyDER, err := x509.MarshalECPrivateKey(clientKey)
	if err != nil {
		t.Fatalf("failed to marshal client key: %v", err)
	}

	keyFile = dir + "/client-key.pem"
	if err := os.WriteFile(keyFile, pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}), 0600); err != nil {
		t.Fatalf("failed to write client key: %v", err)
	}

	return certFile, keyFile, caFile
}

// --- buildResource tests ---

func TestBuildResource_Attributes(t *testing.T) {
	res, err := buildResource(map[string]string{
		"service.name": "my-service",
		"host.name":    "my-host",
		"env":          "test",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("resource should not be nil")
	}

	serviceName, found := res.Set().Value("service.name")
	if !found || serviceName.AsString() != "my-service" {
		t.Errorf("expected service.name 'my-service', got %v (found=%v)", serviceName, found)
	}

	hostName, found := res.Set().Value("host.name")
	if !found || hostName.AsString() != "my-host" {
		t.Errorf("expected host.name 'my-host', got %v (found=%v)", hostName, found)
	}

	env, found := res.Set().Value("env")
	if !found || env.AsString() != "test" {
		t.Errorf("expected env='test', got %v (found=%v)", env, found)
	}
}

func TestBuildResource_Empty(t *testing.T) {
	res, err := buildResource(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res == nil {
		t.Fatal("resource should not be nil (should return default)")
	}
}

// --- Metric recording tests ---

func TestSetGaugeWithLabels(t *testing.T) {
	sink, reader := newTestSink(t)
	t.Cleanup(sink.Shutdown)

	sink.SetGaugeWithLabels([]string{"test", "gauge"}, 42.5, []metrics.Label{
		{Name: "env", Value: "prod"},
		{Name: "region", Value: "us-west"},
	})

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "test.gauge")
	if m == nil {
		t.Fatal("metric not found")
	}

	gauge, ok := m.Data.(metricdata.Gauge[float64])
	if !ok {
		t.Fatalf("expected Gauge data, got %T", m.Data)
	}
	if len(gauge.DataPoints) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(gauge.DataPoints))
	}

	dp := gauge.DataPoints[0]
	attrs := dp.Attributes

	env, found := attrs.Value("env")
	if !found || env.AsString() != "prod" {
		t.Errorf("expected env=prod, got %v", env)
	}
	region, found := attrs.Value("region")
	if !found || region.AsString() != "us-west" {
		t.Errorf("expected region=us-west, got %v", region)
	}
}

func TestIncrCounterWithLabels(t *testing.T) {
	sink, reader := newTestSink(t)
	t.Cleanup(sink.Shutdown)

	sink.IncrCounterWithLabels([]string{"requests"}, 1, []metrics.Label{
		{Name: "method", Value: "GET"},
	})
	sink.IncrCounterWithLabels([]string{"requests"}, 1, []metrics.Label{
		{Name: "method", Value: "POST"},
	})
	sink.IncrCounterWithLabels([]string{"requests"}, 2, []metrics.Label{
		{Name: "method", Value: "GET"},
	})

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "requests")
	if m == nil {
		t.Fatal("metric not found")
	}

	sum, ok := m.Data.(metricdata.Sum[float64])
	if !ok {
		t.Fatalf("expected Sum data, got %T", m.Data)
	}
	if len(sum.DataPoints) != 2 {
		t.Fatalf("expected 2 data points (one per label set), got %d", len(sum.DataPoints))
	}

	var getVal, postVal float64
	for _, dp := range sum.DataPoints {
		method, _ := dp.Attributes.Value("method")
		switch method.AsString() {
		case "GET":
			getVal = dp.Value
		case "POST":
			postVal = dp.Value
		}
	}

	if getVal != 3 {
		t.Errorf("expected GET value 3, got %f", getVal)
	}
	if postVal != 1 {
		t.Errorf("expected POST value 1, got %f", postVal)
	}
}

func TestAddSampleWithLabels(t *testing.T) {
	sink, reader := newTestSinkWithHistogram(t, true, nil)
	t.Cleanup(sink.Shutdown)

	sink.AddSampleWithLabels([]string{"latency"}, 10, []metrics.Label{
		{Name: "endpoint", Value: "/api/users"},
	})

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "latency")
	if m == nil {
		t.Fatal("metric not found")
	}

	hist, ok := m.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("expected Histogram data, got %T", m.Data)
	}

	dp := hist.DataPoints[0]
	endpoint, found := dp.Attributes.Value("endpoint")
	if !found || endpoint.AsString() != "/api/users" {
		t.Errorf("expected endpoint=/api/users, got %v", endpoint)
	}
}

func TestEmitKey_Dropped(t *testing.T) {
	sink, reader := newTestSink(t)
	t.Cleanup(sink.Shutdown)

	// EmitKey should be a no-op.
	sink.EmitKey([]string{"some", "key"}, 42)

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "some.key")
	if m != nil {
		t.Error("EmitKey should not create a metric")
	}
}

// --- Histogram configuration tests ---

func TestHistogram_ExponentialDefault(t *testing.T) {
	sink, reader := newTestSink(t)
	t.Cleanup(sink.Shutdown)

	sink.AddSample([]string{"latency"}, 1)
	sink.AddSample([]string{"latency"}, 10)
	sink.AddSample([]string{"latency"}, 100)

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "latency")
	if m == nil {
		t.Fatal("metric not found")
	}

	hist, ok := m.Data.(metricdata.ExponentialHistogram[float64])
	if !ok {
		t.Fatalf("expected ExponentialHistogram data, got %T", m.Data)
	}
	if len(hist.DataPoints) != 1 {
		t.Fatalf("expected 1 data point, got %d", len(hist.DataPoints))
	}

	dp := hist.DataPoints[0]
	if dp.Count != 3 {
		t.Errorf("expected count 3, got %d", dp.Count)
	}
	if dp.Sum != 111 {
		t.Errorf("expected sum 111, got %f", dp.Sum)
	}
}

func TestHistogram_ExplicitBuckets(t *testing.T) {
	sink, reader := newTestSinkWithHistogram(t, true, []float64{5, 10, 25, 50, 100})
	t.Cleanup(sink.Shutdown)

	sink.AddSample([]string{"latency"}, 3)   // <= 5
	sink.AddSample([]string{"latency"}, 7)   // <= 10
	sink.AddSample([]string{"latency"}, 15)  // <= 25
	sink.AddSample([]string{"latency"}, 200) // > 100

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "latency")
	if m == nil {
		t.Fatal("metric not found")
	}

	hist, ok := m.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("expected Histogram data, got %T", m.Data)
	}

	dp := hist.DataPoints[0]
	if dp.Count != 4 {
		t.Errorf("expected count 4, got %d", dp.Count)
	}

	expectedBounds := []float64{5, 10, 25, 50, 100}
	if len(dp.Bounds) != len(expectedBounds) {
		t.Errorf("expected %d bucket boundaries, got %d", len(expectedBounds), len(dp.Bounds))
	}
	for i, b := range expectedBounds {
		if dp.Bounds[i] != b {
			t.Errorf("expected boundary[%d]=%f, got %f", i, b, dp.Bounds[i])
		}
	}
}

func TestHistogram_DefaultExplicitBuckets(t *testing.T) {
	sink, reader := newTestSinkWithHistogram(t, true, nil)
	t.Cleanup(sink.Shutdown)

	sink.AddSample([]string{"latency"}, 0.001)

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "latency")
	if m == nil {
		t.Fatal("metric not found")
	}

	hist, ok := m.Data.(metricdata.Histogram[float64])
	if !ok {
		t.Fatalf("expected Histogram data, got %T", m.Data)
	}

	dp := hist.DataPoints[0]
	if len(dp.Bounds) != len(DefaultHistogramBuckets()) {
		t.Errorf("expected %d bucket boundaries (default), got %d", len(DefaultHistogramBuckets()), len(dp.Bounds))
	}
}

// --- Shutdown tests ---

func TestShutdown_OwnedProvider(t *testing.T) {
	sink, err := NewOTELSink(OTELSinkOpts{
		Endpoint:        "localhost:4317",
		Insecure:        true,
		ShutdownTimeout: 1 * time.Second,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	sink.IncrCounter([]string{"test"}, 1)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	err = sink.ShutdownContext(ctx)
	if err != nil {
		t.Logf("shutdown error (expected without a live server): %v", err)
	}

	// Second shutdown should return the cached result, not panic.
	err2 := sink.ShutdownContext(ctx)
	if err2 != err {
		t.Errorf("second shutdown returned different error: got %v, want %v", err2, err)
	}
}

func TestShutdownContext(t *testing.T) {
	sink, _ := newTestSink(t)

	sink.IncrCounter([]string{"pre", "shutdown"}, 1)

	ctx := context.Background()
	err := sink.ShutdownContext(ctx)
	if err != nil {
		t.Errorf("expected nil error from ShutdownContext, got: %v", err)
	}

	// Second call should return the same result.
	err = sink.ShutdownContext(ctx)
	if err != nil {
		t.Errorf("expected nil error from second ShutdownContext call, got: %v", err)
	}
}

func TestShutdown_MultipleCalls(t *testing.T) {
	sink, _ := newTestSink(t)

	// Calling Shutdown() multiple times should not panic.
	sink.Shutdown()
	sink.Shutdown()
	sink.Shutdown()
}

func TestRecordAfterShutdown(t *testing.T) {
	sink, _ := newTestSink(t)

	sink.Shutdown()

	// Recording metrics after Shutdown() should not panic.
	sink.SetGauge([]string{"post", "shutdown", "gauge"}, 1.0)
	sink.IncrCounter([]string{"post", "shutdown", "counter"}, 1.0)
	sink.AddSample([]string{"post", "shutdown", "histogram"}, 1.0)
}

// --- Concurrency tests ---

func TestConcurrentMetricRecording(t *testing.T) {
	sink, reader := newTestSink(t)
	t.Cleanup(sink.Shutdown)

	const numGoroutines = 10
	const numIterations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numIterations; j++ {
				sink.IncrCounter([]string{"concurrent", "counter"}, 1)
				sink.SetGauge([]string{"concurrent", "gauge"}, float32(j))
				sink.AddSample([]string{"concurrent", "histogram"}, float32(j))
			}
		}()
	}

	wg.Wait()

	rm := collectMetrics(t, reader)

	counter := findMetric(rm, "concurrent.counter")
	if counter == nil {
		t.Fatal("counter metric not found")
	}
	sum, ok := counter.Data.(metricdata.Sum[float64])
	if !ok {
		t.Fatalf("expected Sum data, got %T", counter.Data)
	}
	expectedCount := float64(numGoroutines * numIterations)
	if sum.DataPoints[0].Value != expectedCount {
		t.Errorf("expected counter value %f, got %f", expectedCount, sum.DataPoints[0].Value)
	}

	gauge := findMetric(rm, "concurrent.gauge")
	if gauge == nil {
		t.Fatal("gauge metric not found")
	}

	hist := findMetric(rm, "concurrent.histogram")
	if hist == nil {
		t.Fatal("histogram metric not found")
	}
}

func TestConcurrentInstrumentCreation(t *testing.T) {
	sink, reader := newTestSink(t)
	t.Cleanup(sink.Shutdown)

	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// All goroutines try to create the same metric simultaneously.
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			sink.IncrCounter([]string{"race", "counter"}, 1)
		}()
	}

	wg.Wait()

	rm := collectMetrics(t, reader)
	counter := findMetric(rm, "race.counter")
	if counter == nil {
		t.Fatal("counter metric not found")
	}

	sum, ok := counter.Data.(metricdata.Sum[float64])
	if !ok {
		t.Fatalf("expected Sum data, got %T", counter.Data)
	}
	if sum.DataPoints[0].Value != float64(numGoroutines) {
		t.Errorf("expected counter value %d, got %f", numGoroutines, sum.DataPoints[0].Value)
	}
}

func TestConcurrentShutdownAndRecording(t *testing.T) {
	sink, _ := newTestSink(t)

	const numRecorders = 5
	readyCh := make(chan struct{}, numRecorders)
	var wg sync.WaitGroup

	for i := 0; i < numRecorders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			readyCh <- struct{}{}
			for j := 0; j < 100; j++ {
				sink.SetGauge([]string{"concurrent", "gauge"}, float32(j))
				sink.IncrCounter([]string{"concurrent", "counter"}, 1)
				sink.AddSample([]string{"concurrent", "histogram"}, float32(j))
			}
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numRecorders; i++ {
			<-readyCh
		}
		sink.Shutdown()
	}()

	wg.Wait()
}

// --- Edge case tests ---

func TestIncrCounter_NegativeValue(t *testing.T) {
	sink, reader := newTestSink(t)
	t.Cleanup(sink.Shutdown)

	// Negative counter increments should be silently dropped with a warning log.
	sink.IncrCounter([]string{"negative", "counter"}, -1.0)

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "negative.counter")
	if m != nil {
		t.Fatal("expected metric 'negative.counter' to NOT exist after negative increment")
	}

	// Verify positive values still work after a negative attempt.
	sink.IncrCounter([]string{"negative", "counter"}, 5.0)
	rm = collectMetrics(t, reader)
	m = findMetric(rm, "negative.counter")
	if m == nil {
		t.Fatal("expected metric to exist after positive increment")
	}
	sum, ok := m.Data.(metricdata.Sum[float64])
	if !ok {
		t.Fatalf("expected Sum data, got %T", m.Data)
	}
	if len(sum.DataPoints) < 1 {
		t.Fatal("expected at least one data point")
	}
	if sum.DataPoints[0].Value != 5.0 {
		t.Errorf("expected counter value 5.0, got %f", sum.DataPoints[0].Value)
	}
}

func TestSetGauge_EmptyKey(t *testing.T) {
	sink, reader := newTestSink(t)
	t.Cleanup(sink.Shutdown)

	// Passing an empty key slice produces an empty string metric name.
	sink.SetGauge([]string{}, 42.5)

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "")
	if m == nil {
		t.Fatal("expected metric with empty name to exist")
	}
	gauge, ok := m.Data.(metricdata.Gauge[float64])
	if !ok {
		t.Fatalf("expected Gauge data, got %T", m.Data)
	}
	if len(gauge.DataPoints) == 0 {
		t.Fatal("expected at least 1 data point")
	}
	if gauge.DataPoints[0].Value != 42.5 {
		t.Errorf("expected value 42.5, got %f", gauge.DataPoints[0].Value)
	}
}

func TestIncrCounter_NilKey(t *testing.T) {
	sink, reader := newTestSink(t)
	t.Cleanup(sink.Shutdown)

	// Passing nil key should not panic.
	sink.IncrCounter(nil, 1.0)

	rm := collectMetrics(t, reader)
	m := findMetric(rm, "")
	if m == nil {
		t.Fatal("expected metric with empty name to exist")
	}
	sum, ok := m.Data.(metricdata.Sum[float64])
	if !ok {
		t.Fatalf("expected Sum data, got %T", m.Data)
	}
	if len(sum.DataPoints) == 0 {
		t.Fatal("expected at least 1 data point")
	}
	if sum.DataPoints[0].Value != 1 {
		t.Errorf("expected counter value 1, got %f", sum.DataPoints[0].Value)
	}
}
