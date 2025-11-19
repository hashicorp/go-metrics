// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MIT

//go:build go1.9

package prometheus

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hashicorp/go-metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var (
	// DefaultPrometheusOpts is the default set of options used when creating a
	// PrometheusSink.
	DefaultPrometheusOpts = PrometheusOpts{
		Expiration: 60 * time.Second,
		Name:       "default_prometheus_sink",
	}
)

// PrometheusOpts is used to configure the Prometheus Sink
type PrometheusOpts struct {
	// Expiration is the duration a metric is valid for, after which it will be
	// untracked. If the value is zero, a metric is never expired.
	Expiration time.Duration
	Registerer prometheus.Registerer

	// Gauges, Summaries, and Counters allow us to pre-declare metrics by giving
	// their Name, Help, and ConstLabels to the PrometheusSink when it is created.
	// Metrics declared in this way will be initialized at zero and will not be
	// deleted or altered when their expiry is reached.
	//
	// Ex: PrometheusOpts{
	//     Expiration: 10 * time.Second,
	//     Gauges: []GaugeDefinition{
	//         {
	//           Name: []string{ "application", "component", "measurement"},
	//           Help: "application_component_measurement provides an example of how to declare static metrics",
	//           ConstLabels: []metrics.Label{ { Name: "my_label", Value: "does_not_change" }, },
	//         },
	//     },
	// }
	GaugeDefinitions   []GaugeDefinition
	SummaryDefinitions []SummaryDefinition
	CounterDefinitions []CounterDefinition
	Name               string
}

type PrometheusSink struct {
	// If these will ever be copied, they should be converted to *sync.Map values and initialized appropriately
	gauges     sync.Map
	summaries  sync.Map
	counters   sync.Map
	expiration time.Duration
	help       map[string]string
	name       string
}

// GaugeDefinition can be provided to PrometheusOpts to declare a constant gauge that is not deleted on expiry.
type GaugeDefinition struct {
	Name        []string
	ConstLabels []metrics.Label
	Help        string
}

type gauge struct {
	prometheus.Gauge
	updatedAtNS atomic.Int64
	canDelete   atomic.Bool
}

// SummaryDefinition can be provided to PrometheusOpts to declare a constant summary that is not deleted on expiry.
type SummaryDefinition struct {
	Name        []string
	ConstLabels []metrics.Label
	Help        string
}

type summary struct {
	prometheus.Summary
	updatedAtNS atomic.Int64
	canDelete   atomic.Bool
}

// CounterDefinition can be provided to PrometheusOpts to declare a constant counter that is not deleted on expiry.
type CounterDefinition struct {
	Name        []string
	ConstLabels []metrics.Label
	Help        string
}

type counter struct {
	prometheus.Counter
	updatedAtNS atomic.Int64
	canDelete   atomic.Bool
}

// NewPrometheusSink creates a new PrometheusSink using the default options.
func NewPrometheusSink() (*PrometheusSink, error) {
	return NewPrometheusSinkFrom(DefaultPrometheusOpts)
}

// NewPrometheusSinkFrom creates a new PrometheusSink using the passed options.
func NewPrometheusSinkFrom(opts PrometheusOpts) (*PrometheusSink, error) {
	name := opts.Name
	if name == "" {
		name = "default_prometheus_sink"
	}
	sink := &PrometheusSink{
		gauges:     sync.Map{},
		summaries:  sync.Map{},
		counters:   sync.Map{},
		expiration: opts.Expiration,
		help:       make(map[string]string),
		name:       name,
	}

	initGauges(&sink.gauges, opts.GaugeDefinitions, sink.help)
	initSummaries(&sink.summaries, opts.SummaryDefinitions, sink.help)
	initCounters(&sink.counters, opts.CounterDefinitions, sink.help)

	reg := opts.Registerer
	if reg == nil {
		reg = prometheus.DefaultRegisterer
	}

	return sink, reg.Register(sink)
}

// Describe sends a Collector.Describe value from the descriptor created around PrometheusSink.Name
// Note that we cannot describe all the metrics (gauges, counters, summaries) in the sink as
// metrics can be added at any point during the lifecycle of the sink, which does not respect
// the idempotency aspect of the Collector.Describe() interface
func (p *PrometheusSink) Describe(c chan<- *prometheus.Desc) {
	// dummy value to be able to register and unregister "empty" sinks
	// Note this is not actually retained in the PrometheusSink so this has no side effects
	// on the caller's sink. So it shouldn't show up to any of its consumers.
	prometheus.NewGauge(prometheus.GaugeOpts{Name: p.name, Help: p.name}).Describe(c)
}

// Collect meets the collection interface and allows us to enforce our expiration
// logic to clean up ephemeral metrics if their value haven't been set for a
// duration exceeding our allowed expiration time.
func (p *PrometheusSink) Collect(c chan<- prometheus.Metric) {
	p.collectAtTime(c, time.Now())
}

// collectAtTime allows internal testing of the expiry based logic here without
// mocking clocks or making tests timing sensitive.
func (p *PrometheusSink) collectAtTime(c chan<- prometheus.Metric, t time.Time) {
	// Counters
	p.counters.Range(func(k, v any) bool {
		cnt := v.(*counter)
		cnt.Collect(c)
		last := time.Unix(0, cnt.updatedAtNS.Load())
		stale := p.expiration > 0 && t.Sub(last) > p.expiration
		if stale && cnt.canDelete.Load() {
			p.counters.Delete(k)
		}
		return true
	})

	// Gauges
	p.gauges.Range(func(k, v any) bool {
		g := v.(*gauge)
		g.Collect(c)
		last := time.Unix(0, g.updatedAtNS.Load())
		stale := p.expiration > 0 && t.Sub(last) > p.expiration
		if stale && g.canDelete.Load() {
			p.gauges.Delete(k)
		}
		return true
	})

	// Summaries
	p.summaries.Range(func(k, v any) bool {
		s := v.(*summary)
		s.Collect(c)
		last := time.Unix(0, s.updatedAtNS.Load())
		stale := p.expiration > 0 && t.Sub(last) > p.expiration
		if stale && s.canDelete.Load() {
			p.summaries.Delete(k)
		}
		return true
	})
}

func initGauges(m *sync.Map, gauges []GaugeDefinition, help map[string]string) {
	for _, g := range gauges {
		key, hash := flattenKey(g.Name, g.ConstLabels)
		help[fmt.Sprintf("gauge.%s", key)] = g.Help
		pG := prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        key,
			Help:        g.Help,
			ConstLabels: prometheusLabels(g.ConstLabels),
		})
		m.Store(hash, &gauge{Gauge: pG})
	}
}

func initSummaries(m *sync.Map, summaries []SummaryDefinition, help map[string]string) {
	for _, s := range summaries {
		key, hash := flattenKey(s.Name, s.ConstLabels)
		help[fmt.Sprintf("summary.%s", key)] = s.Help
		pS := prometheus.NewSummary(prometheus.SummaryOpts{
			Name:        key,
			Help:        s.Help,
			MaxAge:      10 * time.Second,
			ConstLabels: prometheusLabels(s.ConstLabels),
			Objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		})
		m.Store(hash, &summary{Summary: pS})
	}
}

func initCounters(m *sync.Map, counters []CounterDefinition, help map[string]string) {
	for _, c := range counters {
		key, hash := flattenKey(c.Name, c.ConstLabels)
		help[fmt.Sprintf("counter.%s", key)] = c.Help
		pC := prometheus.NewCounter(prometheus.CounterOpts{
			Name:        key,
			Help:        c.Help,
			ConstLabels: prometheusLabels(c.ConstLabels),
		})
		m.Store(hash, &counter{Counter: pC})
	}
}

var forbiddenCharsReplacer = strings.NewReplacer(" ", "_", ".", "_", "=", "_", "-", "_", "/", "_")

func flattenKey(parts []string, labels []metrics.Label) (string, string) {
	key := strings.Join(parts, "_")
	key = forbiddenCharsReplacer.Replace(key)

	hash := key
	for _, label := range labels {
		hash += ";" + label.Name + "=" + label.Value
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

func (p *PrometheusSink) SetGauge(parts []string, val float32) {
	p.SetPrecisionGauge(parts, float64(val))
}

func (p *PrometheusSink) SetGaugeWithLabels(parts []string, val float32, labels []metrics.Label) {
	p.SetPrecisionGaugeWithLabels(parts, float64(val), labels)
}

func (p *PrometheusSink) SetPrecisionGauge(parts []string, val float64) {
	p.SetPrecisionGaugeWithLabels(parts, val, nil)
}

func (p *PrometheusSink) SetPrecisionGaugeWithLabels(parts []string, val float64, labels []metrics.Label) {
	key, hash := flattenKey(parts, labels)

	// Fast path: use existing
	if v, ok := p.gauges.Load(hash); ok {
		g := v.(*gauge)
		g.Set(val)
		g.updatedAtNS.Store(time.Now().UnixNano())
		return
	}

	// Create-or-get single instance
	help := p.help[fmt.Sprintf("gauge.%s", key)]
	if help == "" {
		help = key
	}
	w := &gauge{
		Gauge: prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        key,
			Help:        help,
			ConstLabels: prometheusLabels(labels),
		}),
	}
	w.canDelete.Store(true)
	actual, _ := p.gauges.LoadOrStore(hash, w)
	g := actual.(*gauge)
	g.Set(val)
	g.updatedAtNS.Store(time.Now().UnixNano())
}

func (p *PrometheusSink) AddSample(parts []string, val float32) {
	p.AddSampleWithLabels(parts, val, nil)
}

func (p *PrometheusSink) AddSampleWithLabels(parts []string, val float32, labels []metrics.Label) {
	key, hash := flattenKey(parts, labels)

	if v, ok := p.summaries.Load(hash); ok {
		s := v.(*summary)
		s.Observe(float64(val))
		s.updatedAtNS.Store(time.Now().UnixNano())
		return
	}

	help := p.help[fmt.Sprintf("summary.%s", key)]
	if help == "" {
		help = key
	}
	w := &summary{
		Summary: prometheus.NewSummary(prometheus.SummaryOpts{
			Name:        key,
			Help:        help,
			MaxAge:      10 * time.Second,
			ConstLabels: prometheusLabels(labels),
			Objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		}),
	}
	w.canDelete.Store(true)
	actual, _ := p.summaries.LoadOrStore(hash, w)
	s := actual.(*summary)
	s.Observe(float64(val))
	s.updatedAtNS.Store(time.Now().UnixNano())
}

// EmitKey is not implemented. Prometheus doesn’t offer a type for which an
// arbitrary number of values is retained, as Prometheus works with a pull
// model, rather than a push model.
func (p *PrometheusSink) EmitKey(key []string, val float32) {
}

func (p *PrometheusSink) IncrCounter(parts []string, val float32) {
	p.IncrCounterWithLabels(parts, val, nil)
}

func (p *PrometheusSink) IncrCounterWithLabels(parts []string, val float32, labels []metrics.Label) {
	key, hash := flattenKey(parts, labels)

	// Prometheus Counter.Add() panics if val < 0; log and return.
	if val < 0 {
		log.Printf("[ERR] 'IncrCounterWithLabels' called with a negative value: %v", val)
		return
	}

	if v, ok := p.counters.Load(hash); ok {
		c := v.(*counter)
		c.Add(float64(val))
		c.updatedAtNS.Store(time.Now().UnixNano())
		return
	}

	help := p.help[fmt.Sprintf("counter.%s", key)]
	if help == "" {
		help = key
	}
	w := &counter{
		Counter: prometheus.NewCounter(prometheus.CounterOpts{
			Name:        key,
			Help:        help,
			ConstLabels: prometheusLabels(labels),
		}),
	}
	w.canDelete.Store(true)
	actual, _ := p.counters.LoadOrStore(hash, w)
	c := actual.(*counter)
	c.Add(float64(val))
	c.updatedAtNS.Store(time.Now().UnixNano())
}

// PrometheusPushSink wraps a normal prometheus sink and provides an address and facilities to export it to an address
// on an interval.
type PrometheusPushSink struct {
	*PrometheusSink
	pusher       *push.Pusher
	address      string
	pushInterval time.Duration
	stopChan     chan struct{}
}

// NewPrometheusPushSink creates a PrometheusPushSink by taking an address, interval, and destination name.
func NewPrometheusPushSink(address string, pushInterval time.Duration, name string) (*PrometheusPushSink, error) {
	promSink := &PrometheusSink{
		gauges:     sync.Map{},
		summaries:  sync.Map{},
		counters:   sync.Map{},
		expiration: 60 * time.Second,
		name:       "default_prometheus_sink",
	}

	pusher := push.New(address, name).Collector(promSink)

	sink := &PrometheusPushSink{
		promSink,
		pusher,
		address,
		pushInterval,
		make(chan struct{}),
	}

	sink.flushMetrics()
	return sink, nil
}

func (s *PrometheusPushSink) flushMetrics() {
	ticker := time.NewTicker(s.pushInterval)

	go func() {
		for {
			select {
			case <-ticker.C:
				err := s.pusher.Push()
				if err != nil {
					log.Printf("[ERR] Error pushing to Prometheus! Err: %s", err)
				}
			case <-s.stopChan:
				ticker.Stop()
				return
			}
		}
	}()
}

// Shutdown tears down the PrometheusPushSink, and blocks while flushing metrics to the backend.
func (s *PrometheusPushSink) Shutdown() {
	close(s.stopChan)
	// Closing the channel only stops the running goroutine that pushes metrics.
	// To minimize the chance of data loss pusher.Push is called one last time.
	_ = s.pusher.Push()
}
