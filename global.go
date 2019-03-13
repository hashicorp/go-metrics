package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Shared global metrics instance
var globalMetrics atomic.Value // *MetricService
var singleton sync.Once

// InitGlobal initializes a MetricService, but stores it on a
//  global accessible singleton store
func InitGlobal(conf *MetricServiceConfig, sink Sinker) *MetricService {
	metrics := NewMetricService(conf, sink)

	singleton.Do(func() {
		globalMetrics.Store(metrics)
	})

	return metrics
}

// SetGauge sets a value on a gauge
func SetGauge(key []string, val float32) {
	globalMetrics.Load().(*MetricService).SetGauge(key, val)
}

// SetGaugeWithLabels sets a value on a gauge with labels
func SetGaugeWithLabels(key []string, val float32, labels []Label) {
	globalMetrics.Load().(*MetricService).SetGaugeWithLabels(key, val, labels)
}

// EmitKey emits a key value metric
func EmitKey(key []string, val float32) {
	globalMetrics.Load().(*MetricService).EmitKey(key, val)
}

// IncrCounter increases the value of a counter by a given value
func IncrCounter(key []string, val float32) {
	globalMetrics.Load().(*MetricService).IncrCounter(key, val)
}

// IncrCounterWithLabels increases the value of a counter by a given value with labels
func IncrCounterWithLabels(key []string, val float32, labels []Label) {
	globalMetrics.Load().(*MetricService).IncrCounterWithLabels(key, val, labels)
}

// AddSample adds a sample metrics
func AddSample(key []string, val float32) {
	globalMetrics.Load().(*MetricService).AddSample(key, val)
}

// AddSampleWithLabels adds a sample metrics with labels
func AddSampleWithLabels(key []string, val float32, labels []Label) {
	globalMetrics.Load().(*MetricService).AddSampleWithLabels(key, val, labels)
}

// MeasureSince measure time since the start time until now
func MeasureSince(key []string, start time.Time) {
	globalMetrics.Load().(*MetricService).MeasureSince(key, start)
}

// MeasureSinceWithLabels measure time since the start time until now with labels
func MeasureSinceWithLabels(key []string, start time.Time, labels []Label) {
	globalMetrics.Load().(*MetricService).MeasureSinceWithLabels(key, start, labels)
}
