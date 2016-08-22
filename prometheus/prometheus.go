// +build go1.3
package prometheus

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusSink struct {
	mu        sync.Mutex
	gauges    map[string]*prometheus.GaugeVec
	summaries map[string]*prometheus.SummaryVec
	counters  map[string]*prometheus.CounterVec
	mapper    PrometheusMapper
}

func NewPrometheusSink() (*PrometheusSink, *error) {
	return &PrometheusSink{
		gauges:    make(map[string]*prometheus.GaugeVec),
		summaries: make(map[string]*prometheus.SummaryVec),
		counters:  make(map[string]*prometheus.CounterVec),
		mapper:    &simpleMapper{},
	}, nil
}

func NewPrometheusSinkWithMapper(mapper PrometheusMapper) (*PrometheusSink, *error) {
	return &PrometheusSink{
		gauges:    make(map[string]*prometheus.GaugeVec),
		summaries: make(map[string]*prometheus.SummaryVec),
		counters:  make(map[string]*prometheus.CounterVec),
		mapper:    mapper,
	}, nil
}

func labelNames(labels prometheus.Labels) []string {
	names := make([]string, len(labels))
	i := 0
	for labelName, _ := range labels {
		names[i] = labelName
		i++
	}
	return names
}

func (p *PrometheusSink) SetGauge(parts []string, val float32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key, labels, present := p.mapper.MapMetric(parts)
	if !present {
		return
	}
	g, ok := p.gauges[key]
	if !ok {
		g = prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: key,
			Help: key,
		}, labelNames(labels))
		prometheus.MustRegister(g)
		p.gauges[key] = g
	}
	g.With(labels).Set(float64(val))
}

func (p *PrometheusSink) AddSample(parts []string, val float32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key, labels, present := p.mapper.MapMetric(parts)
	if !present {
		return
	}
	g, ok := p.summaries[key]
	if !ok {
		g = prometheus.NewSummaryVec(prometheus.SummaryOpts{
			Name:   key,
			Help:   key,
			MaxAge: 10 * time.Second,
		}, labelNames(labels))
		prometheus.MustRegister(g)
		p.summaries[key] = g
	}
	g.With(labels).Observe(float64(val))
}

// EmitKey is not implemented. Prometheus doesn’t offer a type for which an
// arbitrary number of values is retained, as Prometheus works with a pull
// model, rather than a push model.
func (p *PrometheusSink) EmitKey(key []string, val float32) {
}

func (p *PrometheusSink) IncrCounter(parts []string, val float32) {
	p.mu.Lock()
	defer p.mu.Unlock()
	key, labels, present := p.mapper.MapMetric(parts)
	if !present {
		return
	}
	g, ok := p.counters[key]
	if !ok {
		g = prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: key,
			Help: key,
		}, labelNames(labels))
		prometheus.MustRegister(g)
		p.counters[key] = g
	}
	g.With(labels).Add(float64(val))
}
