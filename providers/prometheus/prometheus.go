package prometheus

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"regexp"

	"github.com/hugoluchessi/go-metrics"
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// DefaultSinkOptions is the default set of options used when creating a
	// Sink.
	DefaultSinkOptions = SinkOptions{
		Expiration: 60 * time.Second,
	}
)

// SinkOptions is used to configure the Prometheus Sink
type SinkOptions struct {
	// Expiration is the duration a metric is valid for, after which it will be
	// untracked. If the value is zero, a metric is never expired.
	Expiration time.Duration
}

// Sink is the Prometheus implementation of the Sinker interface
type Sink struct {
	mu         sync.Mutex
	gauges     map[string]prometheus.Gauge
	summaries  map[string]prometheus.Summary
	counters   map[string]prometheus.Counter
	updates    map[string]time.Time
	expiration time.Duration
	registry   *prometheus.Registry
}

// NewSink creates a new Sink using the default options.
func NewSink() (*Sink, error) {
	return NewSinkFrom(DefaultSinkOptions)
}

// NewSinkFrom creates a new Sink using the passed options.
func NewSinkFrom(opts SinkOptions) (*Sink, error) {
	sink := &Sink{
		gauges:     make(map[string]prometheus.Gauge),
		summaries:  make(map[string]prometheus.Summary),
		counters:   make(map[string]prometheus.Counter),
		updates:    make(map[string]time.Time),
		expiration: opts.Expiration,
		registry:   prometheus.NewRegistry(),
	}

	c := &Collector{sink}

	return sink, sink.registry.Register(c)
}

var forbiddenChars = regexp.MustCompile("[ .=\\-/]")

func (p *Sink) flattenKey(parts []string, labels []metrics.Label) (string, string) {
	key := strings.Join(parts, "_")
	key = forbiddenChars.ReplaceAllString(key, "_")

	hash := key
	for _, label := range labels {
		hash += fmt.Sprintf(";%s=%s", label.Name, label.Value)
	}

	return key, hash
}

func prometheusLabels(labels []metrics.Label) prometheus.Labels {
	l := make(prometheus.Labels)
	for _, label := range labels {
		l[label.Name] = label.Value
	}
	return l
}

// SetGauge sets a value on a gauge
func (p *Sink) SetGauge(parts []string, val float32) {
	p.SetGaugeWithLabels(parts, val, nil)
}

// SetGaugeWithLabels sets a value on a gauge with labels
func (p *Sink) SetGaugeWithLabels(parts []string, val float32, labels []metrics.Label) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key, hash := p.flattenKey(parts, labels)
	g, ok := p.gauges[hash]
	if !ok {
		g = prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        key,
			Help:        key,
			ConstLabels: prometheusLabels(labels),
		})
		p.gauges[hash] = g
	}
	g.Set(float64(val))
	p.updates[hash] = time.Now()
}

// EmitKey is not implemented. Prometheus doesn’t offer a type for which an
// arbitrary number of values is retained, as Prometheus works with a pull
// model, rather than a push model.
func (p *Sink) EmitKey(key []string, val float32) {
}

// IncrCounter increases the value of a counter by a given value
func (p *Sink) IncrCounter(parts []string, val float32) {
	p.IncrCounterWithLabels(parts, val, nil)
}

// IncrCounterWithLabels increases the value of a counter by a given value with labels
func (p *Sink) IncrCounterWithLabels(parts []string, val float32, labels []metrics.Label) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key, hash := p.flattenKey(parts, labels)
	g, ok := p.counters[hash]
	if !ok {
		g = prometheus.NewCounter(prometheus.CounterOpts{
			Name:        key,
			Help:        key,
			ConstLabels: prometheusLabels(labels),
		})
		p.counters[hash] = g
	}
	g.Add(float64(val))
	p.updates[hash] = time.Now()
}

// AddSample adds a sample metrics
func (p *Sink) AddSample(parts []string, val float32) {
	p.AddSampleWithLabels(parts, val, nil)
}

// AddSampleWithLabels adds a sample metrics with labels
func (p *Sink) AddSampleWithLabels(parts []string, val float32, labels []metrics.Label) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key, hash := p.flattenKey(parts, labels)
	g, ok := p.summaries[hash]
	if !ok {
		g = prometheus.NewSummary(prometheus.SummaryOpts{
			Name:        key,
			Help:        key,
			MaxAge:      10 * time.Second,
			ConstLabels: prometheusLabels(labels),
		})
		p.summaries[hash] = g
	}
	g.Observe(float64(val))
	p.updates[hash] = time.Now()
}
