package inmem

import (
	"bytes"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/hugoluchessi/go-metrics"
)

// Sink provides a MetricSink that does in-memory aggregation
// without sending metrics over a network. It can be embedded within
// an application to provide profiling information.
type Sink struct {
	// How long is each aggregation interval
	interval time.Duration

	// Retain controls how many metrics interval we keep
	retain time.Duration

	// maxIntervals is the maximum length of intervals.
	// It is retain / interval.
	maxIntervals int

	// intervals is a slice of the retained intervals
	intervals    []*IntervalMetrics
	intervalLock sync.RWMutex

	rateDenom float64
}

// IntervalMetrics stores the aggregated metrics
// for a specific interval
type IntervalMetrics struct {
	sync.RWMutex

	// The start time of the interval
	Interval time.Time

	// Gauges maps the key to the last set value
	Gauges map[string]GaugeValue

	// Points maps the string to the list of emitted values
	// from EmitKey
	Points map[string][]float32

	// Counters maps the string key to a sum of the counter
	// values
	Counters map[string]SampledValue

	// Samples maps the key to an AggregateSample,
	// which has the rolled up view of a sample
	Samples map[string]SampledValue
}

// NewIntervalMetrics creates a new IntervalMetrics for a given interval
func NewIntervalMetrics(intv time.Time) *IntervalMetrics {
	return &IntervalMetrics{
		Interval: intv,
		Gauges:   make(map[string]GaugeValue),
		Points:   make(map[string][]float32),
		Counters: make(map[string]SampledValue),
		Samples:  make(map[string]SampledValue),
	}
}

// AggregateSample is used to hold aggregate metrics
// about a sample
type AggregateSample struct {
	Count       int       // The count of emitted pairs
	Rate        float64   // The values rate per time unit (usually 1 second)
	Sum         float64   // The sum of values
	SumSq       float64   `json:"-"` // The sum of squared values
	Min         float64   // Minimum value
	Max         float64   // Maximum value
	LastUpdated time.Time `json:"-"` // When value was last updated
}

// Stddev computes a Stddev of the values
func (a *AggregateSample) Stddev() float64 {
	num := (float64(a.Count) * a.SumSq) - math.Pow(a.Sum, 2)
	div := float64(a.Count * (a.Count - 1))
	if div == 0 {
		return 0
	}
	return math.Sqrt(num / div)
}

// Mean computes a mean of the values
func (a *AggregateSample) Mean() float64 {
	if a.Count == 0 {
		return 0
	}
	return a.Sum / float64(a.Count)
}

// Ingest is used to update a sample
func (a *AggregateSample) Ingest(v float64, rateDenom float64) {
	a.Count++
	a.Sum += v
	a.SumSq += (v * v)
	if v < a.Min || a.Count == 1 {
		a.Min = v
	}
	if v > a.Max || a.Count == 1 {
		a.Max = v
	}
	a.Rate = float64(a.Sum) / rateDenom
	a.LastUpdated = time.Now()
}

func (a *AggregateSample) String() string {
	if a.Count == 0 {
		return "Count: 0"
	} else if a.Stddev() == 0 {
		return fmt.Sprintf("Count: %d Sum: %0.3f LastUpdated: %s", a.Count, a.Sum, a.LastUpdated)
	} else {
		return fmt.Sprintf("Count: %d Min: %0.3f Mean: %0.3f Max: %0.3f Stddev: %0.3f Sum: %0.3f LastUpdated: %s",
			a.Count, a.Min, a.Mean(), a.Max, a.Stddev(), a.Sum, a.LastUpdated)
	}
}

// NewSink is used to construct a new in-memory sink.
// Uses an aggregation interval and maximum retention period.
func NewSink(interval, retain time.Duration) *Sink {
	rateTimeUnit := time.Second
	i := &Sink{
		interval:     interval,
		retain:       retain,
		maxIntervals: int(retain / interval),
		rateDenom:    float64(interval.Nanoseconds()) / float64(rateTimeUnit.Nanoseconds()),
	}
	i.intervals = make([]*IntervalMetrics, 0, i.maxIntervals)
	return i
}

// SetGauge sets a value on a gauge
func (i *Sink) SetGauge(key []string, val float32) {
	i.SetGaugeWithLabels(key, val, nil)
}

// SetGaugeWithLabels sets a value on a gauge with labels
func (i *Sink) SetGaugeWithLabels(key []string, val float32, labels []metrics.Label) {
	k, name := i.flattenKeyLabels(key, labels)
	intv := i.getInterval()

	intv.Lock()
	defer intv.Unlock()
	intv.Gauges[k] = GaugeValue{Name: name, Value: val, Labels: labels}
}

// EmitKey emits a key value metric
func (i *Sink) EmitKey(key []string, val float32) {
	k := i.flattenKey(key)
	intv := i.getInterval()

	intv.Lock()
	defer intv.Unlock()
	vals := intv.Points[k]
	intv.Points[k] = append(vals, val)
}

// IncrCounter increases the value of a counter by a given value
func (i *Sink) IncrCounter(key []string, val float32) {
	i.IncrCounterWithLabels(key, val, nil)
}

// IncrCounterWithLabels increases the value of a counter by a given value with labels
func (i *Sink) IncrCounterWithLabels(key []string, val float32, labels []metrics.Label) {
	k, name := i.flattenKeyLabels(key, labels)
	intv := i.getInterval()

	intv.Lock()
	defer intv.Unlock()

	agg, ok := intv.Counters[k]
	if !ok {
		agg = SampledValue{
			Name:            name,
			AggregateSample: &AggregateSample{},
			Labels:          labels,
		}
		intv.Counters[k] = agg
	}
	agg.Ingest(float64(val), i.rateDenom)
}

// AddSample adds a sample metrics
func (i *Sink) AddSample(key []string, val float32) {
	i.AddSampleWithLabels(key, val, nil)
}

// AddSampleWithLabels adds a sample metrics with labels
func (i *Sink) AddSampleWithLabels(key []string, val float32, labels []metrics.Label) {
	k, name := i.flattenKeyLabels(key, labels)
	intv := i.getInterval()

	intv.Lock()
	defer intv.Unlock()

	agg, ok := intv.Samples[k]
	if !ok {
		agg = SampledValue{
			Name:            name,
			AggregateSample: &AggregateSample{},
			Labels:          labels,
		}
		intv.Samples[k] = agg
	}
	agg.Ingest(float64(val), i.rateDenom)
}

// Data is used to retrieve all the aggregated metrics
// Intervals may be in use, and a read lock should be acquired
func (i *Sink) Data() []*IntervalMetrics {
	// Get the current interval, forces creation
	i.getInterval()

	i.intervalLock.RLock()
	defer i.intervalLock.RUnlock()

	n := len(i.intervals)
	intervals := make([]*IntervalMetrics, n)

	copy(intervals[:n-1], i.intervals[:n-1])
	current := i.intervals[n-1]

	// make its own copy for current interval
	intervals[n-1] = &IntervalMetrics{}
	copyCurrent := intervals[n-1]
	current.RLock()
	*copyCurrent = *current

	copyCurrent.Gauges = make(map[string]GaugeValue, len(current.Gauges))
	for k, v := range current.Gauges {
		copyCurrent.Gauges[k] = v
	}
	// saved values will be not change, just copy its link
	copyCurrent.Points = make(map[string][]float32, len(current.Points))
	for k, v := range current.Points {
		copyCurrent.Points[k] = v
	}
	copyCurrent.Counters = make(map[string]SampledValue, len(current.Counters))
	for k, v := range current.Counters {
		copyCurrent.Counters[k] = v
	}
	copyCurrent.Samples = make(map[string]SampledValue, len(current.Samples))
	for k, v := range current.Samples {
		copyCurrent.Samples[k] = v
	}
	current.RUnlock()

	return intervals
}

func (i *Sink) getExistingInterval(intv time.Time) *IntervalMetrics {
	i.intervalLock.RLock()
	defer i.intervalLock.RUnlock()

	n := len(i.intervals)
	if n > 0 && i.intervals[n-1].Interval == intv {
		return i.intervals[n-1]
	}
	return nil
}

func (i *Sink) createInterval(intv time.Time) *IntervalMetrics {
	i.intervalLock.Lock()
	defer i.intervalLock.Unlock()

	// Check for an existing interval
	n := len(i.intervals)
	if n > 0 && i.intervals[n-1].Interval == intv {
		return i.intervals[n-1]
	}

	// Add the current interval
	current := NewIntervalMetrics(intv)
	i.intervals = append(i.intervals, current)
	n++

	// Truncate the intervals if they are too long
	if n >= i.maxIntervals {
		copy(i.intervals[0:], i.intervals[n-i.maxIntervals:])
		i.intervals = i.intervals[:i.maxIntervals]
	}
	return current
}

// getInterval returns the current interval to write to
func (i *Sink) getInterval() *IntervalMetrics {
	intv := time.Now().Truncate(i.interval)
	if m := i.getExistingInterval(intv); m != nil {
		return m
	}
	return i.createInterval(intv)
}

// Flattens the key for formatting, removes spaces
func (i *Sink) flattenKey(parts []string) string {
	buf := &bytes.Buffer{}
	replacer := strings.NewReplacer(" ", "_")

	if len(parts) > 0 {
		replacer.WriteString(buf, parts[0])
	}
	for _, part := range parts[1:] {
		replacer.WriteString(buf, ".")
		replacer.WriteString(buf, part)
	}

	return buf.String()
}

// Flattens the key for formatting along with its labels, removes spaces
func (i *Sink) flattenKeyLabels(parts []string, labels []metrics.Label) (string, string) {
	buf := &bytes.Buffer{}
	replacer := strings.NewReplacer(" ", "_")

	if len(parts) > 0 {
		replacer.WriteString(buf, parts[0])
	}
	for _, part := range parts[1:] {
		replacer.WriteString(buf, ".")
		replacer.WriteString(buf, part)
	}

	key := buf.String()

	for _, label := range labels {
		replacer.WriteString(buf, fmt.Sprintf(";%s=%s", label.Name, label.Value))
	}

	return buf.String(), key
}
