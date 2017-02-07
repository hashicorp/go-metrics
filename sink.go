package metrics

// The MetricSink interface is used to transmit metrics information
// to an external system
type MetricSink interface {
	TagMetricSink

	// A Gauge should retain the last value it is set to
	SetGauge(key []string, val float32)

	// Should emit a Key/Value pair for each call
	EmitKey(key []string, val float32)

	// Counters should accumulate values
	IncrCounter(key []string, val float32)

	// Samples are for timing information, where quantiles are used
	AddSample(key []string, val float32)
}

// The TagMetricSink interface is used to transmit tagged metrics information
// to an external system.
type TagMetricSink interface {

	// A guage should retaun the last value it is set to.  Attempts  to tag if
	// able.
	SetGaugeWithTags(key []string, val float32, tags []string)

	// Counters should accumulate values.  Attempts to tag if able.
	IncrCounterWithTags(key []string, val float32, tags []string)

	// Samples are for timing informatio, where quantiles are used.  Attempts
	// to tag if able.
	AddSampleWithTags(key []string, val float32, tags []string)
}

// BlackholeSink is used to just blackhole messages
type BlackholeSink struct{}

func (*BlackholeSink) SetGauge(key []string, val float32)                           {}
func (*BlackholeSink) EmitKey(key []string, val float32)                            {}
func (*BlackholeSink) IncrCounter(key []string, val float32)                        {}
func (*BlackholeSink) AddSample(key []string, val float32)                          {}
func (*BlackholeSink) SetGaugeWithTags(key []string, val float32, tags []string)    {}
func (*BlackholeSink) IncrCounterWithTags(key []string, val float32, tags []string) {}
func (*BlackholeSink) AddSampleWithTags(key []string, val float32, tags []string)   {}

// FanoutSink is used to sink to fanout values to multiple sinks
type FanoutSink []MetricSink

func (fh FanoutSink) SetGauge(key []string, val float32) {
	for _, s := range fh {
		s.SetGauge(key, val)
	}
}

func (fh FanoutSink) EmitKey(key []string, val float32) {
	for _, s := range fh {
		s.EmitKey(key, val)
	}
}

func (fh FanoutSink) IncrCounter(key []string, val float32) {
	for _, s := range fh {
		s.IncrCounter(key, val)
	}
}

func (fh FanoutSink) AddSample(key []string, val float32) {
	for _, s := range fh {
		s.AddSample(key, val)
	}
}

func (fh FanoutSink) SetGaugeWithTags(key []string, val float32, tags []string) {
	for _, s := range fh {
		if inter, ok := s.(TagMetricSink); ok {
			inter.SetGaugeWithTags(key, val, tags)
		}

		s.SetGauge(key, val)
	}
}

func (fh FanoutSink) IncrCounterWithTags(key []string, val float32, tags []string) {
	for _, s := range fh {
		if inter, ok := s.(TagMetricSink); ok {
			inter.IncrCounterWithTags(key, val, tags)
		}

		s.IncrCounter(key, val)
	}
}

func (fh FanoutSink) AddSampleWithTags(key []string, val float32, tags []string) {
	for _, s := range fh {
		if inter, ok := s.(TagMetricSink); ok {
			inter.AddSampleWithTags(key, val, tags)
		}

		s.AddSample(key, val)
	}
}
