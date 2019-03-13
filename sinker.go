package metrics

// Sinker interface is used to transmit metrics information
// to an external system
type Sinker interface {
	// A Gauge should retain the last value it is set to
	SetGauge(key []string, val float32)
	SetGaugeWithLabels(key []string, val float32, labels []Label)

	// Should emit a Key/Value pair for each call
	EmitKey(key []string, val float32)

	// Counters should accumulate values
	IncrCounter(key []string, val float32)
	IncrCounterWithLabels(key []string, val float32, labels []Label)

	// Samples are for timing information, where quantiles are used
	AddSample(key []string, val float32)
	AddSampleWithLabels(key []string, val float32, labels []Label)
}

// BlackholeSink is used to just blackhole messages
type BlackholeSink struct{}

// SetGauge sets a value on a gauge
func (*BlackholeSink) SetGauge(key []string, val float32) {}

// SetGaugeWithLabels sets a value on a gauge with labels
func (*BlackholeSink) SetGaugeWithLabels(key []string, val float32, labels []Label) {}

// EmitKey emits a key value metric
func (*BlackholeSink) EmitKey(key []string, val float32) {}

// IncrCounter increases the value of a counter by a given value
func (*BlackholeSink) IncrCounter(key []string, val float32) {}

// IncrCounterWithLabels increases the value of a counter by a given value with labels
func (*BlackholeSink) IncrCounterWithLabels(key []string, val float32, labels []Label) {}

// AddSample adds a sample metrics
func (*BlackholeSink) AddSample(key []string, val float32) {}

// AddSampleWithLabels adds a sample metrics with labels
func (*BlackholeSink) AddSampleWithLabels(key []string, val float32, labels []Label) {}

// FanoutSink is used to sink to fanout values to multiple sinks
type FanoutSink []Sinker

// SetGauge sets a value on a gauge
func (fh FanoutSink) SetGauge(key []string, val float32) {
	fh.SetGaugeWithLabels(key, val, nil)
}

// SetGaugeWithLabels sets a value on a gauge with labels
func (fh FanoutSink) SetGaugeWithLabels(key []string, val float32, labels []Label) {
	for _, s := range fh {
		s.SetGaugeWithLabels(key, val, labels)
	}
}

// EmitKey emits a key value metric
func (fh FanoutSink) EmitKey(key []string, val float32) {
	for _, s := range fh {
		s.EmitKey(key, val)
	}
}

// IncrCounter increases the value of a counter by a given value
func (fh FanoutSink) IncrCounter(key []string, val float32) {
	fh.IncrCounterWithLabels(key, val, nil)
}

// IncrCounterWithLabels increases the value of a counter by a given value with labels
func (fh FanoutSink) IncrCounterWithLabels(key []string, val float32, labels []Label) {
	for _, s := range fh {
		s.IncrCounterWithLabels(key, val, labels)
	}
}

// AddSample adds a sample metrics
func (fh FanoutSink) AddSample(key []string, val float32) {
	fh.AddSampleWithLabels(key, val, nil)
}

// AddSampleWithLabels adds a sample metrics with labels
func (fh FanoutSink) AddSampleWithLabels(key []string, val float32, labels []Label) {
	for _, s := range fh {
		s.AddSampleWithLabels(key, val, labels)
	}
}
