// +build go1.9

package prometheus

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"regexp"

	"github.com/armon/go-metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/push"
)

var (
	// DefaultPrometheusOpts is the default set of options used when creating a
	// PrometheusSink.
	DefaultPrometheusOpts = PrometheusOpts{
		Expiration: 60 * time.Second,
	}
)

// PrometheusOpts is used to configure the Prometheus Sink
type PrometheusOpts struct {
	// Expiration is the duration a metric is valid for, after which it will be
	// untracked. If the value is zero, a metric is never expired.
	Expiration time.Duration
}

type PrometheusSink struct {
	// If these will ever be copied, they should be converted to *sync.Map values and initialized appropriately
	gauges     sync.Map
	summaries  sync.Map
	counters   sync.Map
	expiration time.Duration
}

type PrometheusGauge struct {
	prometheus.Gauge
	updatedAt time.Time
}

type PrometheusSummary struct {
	prometheus.Summary
	updatedAt time.Time
}

type PrometheusCounter struct {
	prometheus.Counter
	updatedAt time.Time
}

// NewPrometheusSink creates a new PrometheusSink using the default options.
func NewPrometheusSink() (*PrometheusSink, error) {
	return NewPrometheusSinkFrom(DefaultPrometheusOpts)
}

// NewPrometheusSinkFrom creates a new PrometheusSink using the passed options.
func NewPrometheusSinkFrom(opts PrometheusOpts) (*PrometheusSink, error) {
	sink := &PrometheusSink{
		gauges:     sync.Map{},
		summaries:  sync.Map{},
		counters:   sync.Map{},
		expiration: opts.Expiration,
	}

	return sink, prometheus.Register(sink)
}

// Describe is needed to meet the Collector interface.
func (p *PrometheusSink) Describe(c chan<- *prometheus.Desc) {
	// We must emit some description otherwise an error is returned. This
	// description isn't shown to the user!
	prometheus.NewGauge(prometheus.GaugeOpts{Name: "Dummy", Help: "Dummy"}).Describe(c)
}

// Collect meets the collection interface and allows us to enforce our expiration
// logic to clean up ephemeral metrics if their value haven't been set for a
// duration exceeding our allowed expiration time.
func (p *PrometheusSink) Collect(c chan<- prometheus.Metric) {
	expire := p.expiration != 0
	now := time.Now()
	p.gauges.Range(func(k, v interface{}) bool {
		lastUpdate := v.(*PrometheusGauge).updatedAt
		if expire && lastUpdate.Add(p.expiration).Before(now) {
			p.gauges.Delete(k)
		} else {
			v.(*PrometheusGauge).Collect(c)
		}
		return true
	})
	p.summaries.Range(func(k, v interface{}) bool {
		lastUpdate := v.(*PrometheusSummary).updatedAt
		if expire && lastUpdate.Add(p.expiration).Before(now) {
			p.summaries.Delete(k)
		} else {
			v.(*PrometheusSummary).Collect(c)
		}
		return true
	})
	p.counters.Range(func(k, v interface{}) bool {
		lastUpdate := v.(*PrometheusCounter).updatedAt
		if expire && lastUpdate.Add(p.expiration).Before(now) {
			p.counters.Delete(k)
		} else {
			v.(*PrometheusCounter).Collect(c)
		}
		return true
	})
}

var forbiddenChars = regexp.MustCompile("[ .=\\-/]")

func (p *PrometheusSink) flattenKey(parts []string, labels []metrics.Label) (string, string) {
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

func (p *PrometheusSink) SetGauge(parts []string, val float32) {
	p.SetGaugeWithLabels(parts, val, nil)
}

func (p *PrometheusSink) SetGaugeWithLabels(parts []string, val float32, labels []metrics.Label) {
	key, hash := p.flattenKey(parts, labels)
	pg, ok := p.gauges.Load(hash)
	if !ok {
		g := prometheus.NewGauge(prometheus.GaugeOpts{
			Name:        key,
			Help:        key,
			ConstLabels: prometheusLabels(labels),
		})
		pg = &PrometheusGauge{
			g, time.Now(),
		}
		p.gauges.Store(hash, pg)
	}
	pg.(*PrometheusGauge).Set(float64(val))
	pg.(*PrometheusGauge).updatedAt = time.Now()
}

func (p *PrometheusSink) AddSample(parts []string, val float32) {
	p.AddSampleWithLabels(parts, val, nil)
}

func (p *PrometheusSink) AddSampleWithLabels(parts []string, val float32, labels []metrics.Label) {
	key, hash := p.flattenKey(parts, labels)
	ps, ok := p.summaries.Load(hash)
	if !ok {
		s := prometheus.NewSummary(prometheus.SummaryOpts{
			Name:        key,
			Help:        key,
			MaxAge:      10 * time.Second,
			ConstLabels: prometheusLabels(labels),
			Objectives:  map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
		})
		ps = &PrometheusSummary{
			s, time.Now(),
		}
		p.summaries.Store(hash, ps)
	}
	ps.(*PrometheusSummary).Observe(float64(val))
	ps.(*PrometheusSummary).updatedAt = time.Now()
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
	key, hash := p.flattenKey(parts, labels)
	pc, ok := p.counters.Load(hash)
	if !ok {
		c := prometheus.NewCounter(prometheus.CounterOpts{
			Name:        key,
			Help:        key,
			ConstLabels: prometheusLabels(labels),
		})
		pc = &PrometheusCounter{
			c, time.Now(),
		}
		p.counters.Store(hash, pc)
	}
	pc.(*PrometheusCounter).Add(float64(val))
	pc.(*PrometheusCounter).updatedAt = time.Now()
}

type PrometheusPushSink struct {
	*PrometheusSink
	pusher       *push.Pusher
	address      string
	pushInterval time.Duration
	stopChan     chan struct{}
}

func NewPrometheusPushSink(address string, pushIterval time.Duration, name string) (*PrometheusPushSink, error) {
	promSink := &PrometheusSink{
		gauges:     sync.Map{},
		summaries:  sync.Map{},
		counters:   sync.Map{},
		expiration: 60 * time.Second,
	}

	pusher := push.New(address, name).Collector(promSink)

	sink := &PrometheusPushSink{
		promSink,
		pusher,
		address,
		pushIterval,
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

func (s *PrometheusPushSink) Shutdown() {
	close(s.stopChan)
}
