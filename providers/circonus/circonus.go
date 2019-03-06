// Circonus Metrics Sink

package circonus

import (
	"strings"

	"github.com/hugoluchessi/go-metrics"
	cgm "github.com/circonus-labs/circonus-gometrics"
)

// Sink provides an interface to forward metrics to Circonus with
// automatic check creation and metric management
type Sink struct {
	metrics *cgm.CirconusMetrics
}

// Config options for Sink
// See https://github.com/circonus-labs/circonus-gometrics for configuration options
type Config cgm.Config

// NewSink - create new metric sink for circonus
//
// one of the following must be supplied:
//    - API Token - search for an existing check or create a new check
//    - API Token + Check Id - the check identified by check id will be used
//    - API Token + Check Submission URL - the check identified by the submission url will be used
//    - Check Submission URL - the check identified by the submission url will be used
//      metric management will be *disabled*
//
// Note: If submission url is supplied w/o an api token, the public circonus ca cert will be used
// to verify the broker for metrics submission.
func NewSink(cc *Config) (*Sink, error) {
	cfg := cgm.Config{}
	if cc != nil {
		cfg = cgm.Config(*cc)
	}

	metrics, err := cgm.NewCirconusMetrics(&cfg)
	if err != nil {
		return nil, err
	}

	return &Sink{
		metrics: metrics,
	}, nil
}

// Start submitting metrics to Circonus (flush every SubmitInterval)
func (s *Sink) Start() {
	s.metrics.Start()
}

// Flush manually triggers metric submission to Circonus
func (s *Sink) Flush() {
	s.metrics.Flush()
}

// SetGauge sets value for a gauge metric
func (s *Sink) SetGauge(key []string, val float32) {
	flatKey := s.flattenKey(key)
	s.metrics.SetGauge(flatKey, int64(val))
}

// SetGaugeWithLabels sets value for a gauge metric with the given labels
func (s *Sink) SetGaugeWithLabels(key []string, val float32, labels []metrics.Label) {
	flatKey := s.flattenKeyLabels(key, labels)
	s.metrics.SetGauge(flatKey, int64(val))
}

// EmitKey is not implemented in circonus
func (s *Sink) EmitKey(key []string, val float32) {
	// NOP
}

// IncrCounter increments a counter metric
func (s *Sink) IncrCounter(key []string, val float32) {
	flatKey := s.flattenKey(key)
	s.metrics.IncrementByValue(flatKey, uint64(val))
}

// IncrCounterWithLabels increments a counter metric with the given labels
func (s *Sink) IncrCounterWithLabels(key []string, val float32, labels []metrics.Label) {
	flatKey := s.flattenKeyLabels(key, labels)
	s.metrics.IncrementByValue(flatKey, uint64(val))
}

// AddSample adds a sample to a histogram metric
func (s *Sink) AddSample(key []string, val float32) {
	flatKey := s.flattenKey(key)
	s.metrics.RecordValue(flatKey, float64(val))
}

// AddSampleWithLabels adds a sample to a histogram metric with the given labels
func (s *Sink) AddSampleWithLabels(key []string, val float32, labels []metrics.Label) {
	flatKey := s.flattenKeyLabels(key, labels)
	s.metrics.RecordValue(flatKey, float64(val))
}

// Flattens key to Circonus metric name
func (s *Sink) flattenKey(parts []string) string {
	joined := strings.Join(parts, "`")
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ':
			return '_'
		default:
			return r
		}
	}, joined)
}

// Flattens the key along with labels for formatting, removes spaces
func (s *Sink) flattenKeyLabels(parts []string, labels []metrics.Label) string {
	for _, label := range labels {
		parts = append(parts, label.Value)
	}
	return s.flattenKey(parts)
}
