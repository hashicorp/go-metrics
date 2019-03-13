package prometheus

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Collector is the Collector interface implementation
//  for a given sink
type Collector struct {
	*Sink
}

// Describe is needed to meet the Collector interface.
func (p *Collector) Describe(c chan<- *prometheus.Desc) {
	// We must emit some description otherwise an error is returned. This
	// description isn't shown to the user!
	prometheus.NewGauge(prometheus.GaugeOpts{Name: "Dummy", Help: "Dummy"}).Describe(c)
}

// Collect meets the collection interface and allows us to enforce our expiration
// logic to clean up ephemeral metrics if their value haven't been set for a
// duration exceeding our allowed expiration time.
func (p *Collector) Collect(c chan<- prometheus.Metric) {
	p.Sink.mu.Lock()
	defer p.Sink.mu.Unlock()

	expire := p.Sink.expiration != 0
	now := time.Now()
	for k, v := range p.Sink.gauges {
		last := p.Sink.updates[k]
		if expire && last.Add(p.Sink.expiration).Before(now) {
			delete(p.Sink.updates, k)
			delete(p.Sink.gauges, k)
		} else {
			v.Collect(c)
		}
	}
	for k, v := range p.Sink.summaries {
		last := p.Sink.updates[k]
		if expire && last.Add(p.Sink.expiration).Before(now) {
			delete(p.Sink.updates, k)
			delete(p.Sink.summaries, k)
		} else {
			v.Collect(c)
		}
	}
	for k, v := range p.Sink.counters {
		last := p.Sink.updates[k]
		if expire && last.Add(p.Sink.expiration).Before(now) {
			delete(p.Sink.updates, k)
			delete(p.Sink.counters, k)
		} else {
			v.Collect(c)
		}
	}
}
