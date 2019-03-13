package metrics

import (
	"runtime"
	"time"
)

// MetricService is the metric emiter service, which emits metrics to
//  a given sink following the gicen config
type MetricService struct {
	MetricServiceConfig
	lastNumGC uint32
	Sink      Sinker
}

const (
	gaugeType   = "gauge"
	sampleType  = "sample"
	timerType   = "timer"
	counterType = "counter"
	keyType     = "kv"
)

// NewMetricService is used to create a new instance of MetricService
func NewMetricService(conf *MetricServiceConfig, sink Sinker) *MetricService {
	met := &MetricService{}
	met.MetricServiceConfig = *conf
	met.Sink = sink

	// Start the runtime collector
	if conf.EnableRuntimeMetrics {
		go met.collectStats()
	}
	return met
}

// SetGauge sets a value on a gauge
func (m *MetricService) SetGauge(key []string, val float32) {
	m.SetGaugeWithLabels(key, val, nil)
}

// SetGaugeWithLabels sets a value on a gauge with labels
func (m *MetricService) SetGaugeWithLabels(key []string, val float32, labels []Label) {
	k := m.getKey(key, gaugeType)
	m.Sink.SetGaugeWithLabels(k, val, labels)
}

// EmitKey emits a key value metric
func (m *MetricService) EmitKey(key []string, val float32) {
	k := m.getKey(key, keyType)
	m.Sink.EmitKey(k, val)
}

// IncrCounter increases the value of a counter by a given value
func (m *MetricService) IncrCounter(key []string, val float32) {
	m.IncrCounterWithLabels(key, val, nil)
}

// IncrCounterWithLabels increases the value of a counter by a given value with labels
func (m *MetricService) IncrCounterWithLabels(key []string, val float32, labels []Label) {
	k := m.getKey(key, counterType)
	m.Sink.IncrCounterWithLabels(k, val, labels)
}

// AddSample adds a sample metrics
func (m *MetricService) AddSample(key []string, val float32) {
	m.AddSampleWithLabels(key, val, nil)
}

// AddSampleWithLabels adds a sample metrics with labels
func (m *MetricService) AddSampleWithLabels(key []string, val float32, labels []Label) {
	k := m.getKey(key, sampleType)
	m.Sink.AddSampleWithLabels(k, val, labels)
}

// MeasureSince measure time since the start time until now
func (m *MetricService) MeasureSince(key []string, start time.Time) {
	m.MeasureSinceWithLabels(key, start, nil)
}

// MeasureSinceWithLabels measure time since the start time until now with labels
func (m *MetricService) MeasureSinceWithLabels(key []string, start time.Time, labels []Label) {
	k := m.getKey(key, timerType)
	now := time.Now()
	elapsed := now.Sub(start)
	msec := float32(elapsed.Nanoseconds()) / float32(m.TimerGranularity)
	m.Sink.AddSampleWithLabels(k, msec, labels)
}

func (m *MetricService) getKey(key []string, t string) []string {
	if m.EnableHostName && m.HostName != "" {
		key = append([]string{m.HostName}, key...)
	}

	if m.EnableServiceName && m.ServiceName != "" {
		key = append([]string{m.ServiceName}, key...)
	}

	if m.EnableTypeSufix && t != "" {
		key = append(key, t)
	}

	return key
}

// Periodically collects runtime stats to publish
func (m *MetricService) collectStats() {
	for {
		time.Sleep(m.ProfileInterval)
		m.emitRuntimeStats()
	}
}

// Emits various runtime statsitics
func (m *MetricService) emitRuntimeStats() {
	// Export number of Goroutines
	numRoutines := runtime.NumGoroutine()
	m.SetGauge([]string{"runtime", "num_goroutines"}, float32(numRoutines))

	// Export memory stats
	var stats runtime.MemStats
	runtime.ReadMemStats(&stats)
	m.SetGauge([]string{"runtime", "alloc_bytes"}, float32(stats.Alloc))
	m.SetGauge([]string{"runtime", "sys_bytes"}, float32(stats.Sys))
	m.SetGauge([]string{"runtime", "malloc_count"}, float32(stats.Mallocs))
	m.SetGauge([]string{"runtime", "free_count"}, float32(stats.Frees))
	m.SetGauge([]string{"runtime", "heap_objects"}, float32(stats.HeapObjects))
	m.SetGauge([]string{"runtime", "total_gc_pause_ns"}, float32(stats.PauseTotalNs))
	m.SetGauge([]string{"runtime", "total_gc_runs"}, float32(stats.NumGC))

	// Export info about the last few GC runs
	num := stats.NumGC

	// Handle wrap around
	if num < m.lastNumGC {
		m.lastNumGC = 0
	}

	// Ensure we don't scan more than 256
	if num-m.lastNumGC >= 256 {
		m.lastNumGC = num - 255
	}

	for i := m.lastNumGC; i < num; i++ {
		pause := stats.PauseNs[i%256]
		m.AddSample([]string{"runtime", "gc_pause_ns"}, float32(pause))
	}
	m.lastNumGC = num
}
