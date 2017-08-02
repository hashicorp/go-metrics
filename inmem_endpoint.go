package metrics

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
)

// MetricsSummary holds a roll-up of metrics info for a given interval
type MetricsSummary struct {
	Timestamp string
	Gauges    []GaugeValue
	Points    []PointValue
	Counters  []SampledValue
	Samples   []SampledValue
}

type GaugeValue struct {
	Name   string
	Value  float32
	Labels map[string]string
}

type PointValue struct {
	Name   string
	Points []float32
}

type SampledValue struct {
	Name string
	*AggregateSample
	Mean   float64
	Stddev float64
	Labels map[string]string
}

// DisplayMetrics returns a summary of the metrics from the most recent finished interval.
func (i *InmemSink) DisplayMetrics(resp http.ResponseWriter, req *http.Request) (interface{}, error) {
	i.intervalLock.Lock()
	defer i.intervalLock.Unlock()

	var interval *IntervalMetrics
	n := len(i.intervals)
	switch {
	case n == 0:
		return nil, fmt.Errorf("no metric intervals have been initialized yet")
	case n == 1:
		interval = i.intervals[0]
	default:
		interval = i.intervals[n-2]
	}

	summary := MetricsSummary{
		Timestamp: interval.Interval.String(),
		Gauges:    []GaugeValue{},
		Points:    []PointValue{},
	}

	// Format and sort the output of each metric type, so it gets displayed in a
	// deterministic order.
	for name, points := range interval.Points {
		summary.Points = append(summary.Points, PointValue{name, points})
	}
	sort.Slice(summary.Points, func(i, j int) bool {
		return summary.Points[i].Name < summary.Points[j].Name
	})

	for name, value := range interval.Gauges {
		key, labels := extractLabels(name)
		summary.Gauges = append(summary.Gauges, GaugeValue{key, value, labels})
	}
	sort.Slice(summary.Gauges, func(i, j int) bool {
		a := combineNameLabels(summary.Gauges[i].Name, summary.Gauges[i].Labels)
		b := combineNameLabels(summary.Gauges[j].Name, summary.Gauges[j].Labels)
		return a < b
	})

	summary.Counters = formatSamples(interval.Counters)
	summary.Samples = formatSamples(interval.Samples)

	return summary, nil
}

func extractLabels(key string) (string, map[string]string) {
	labels := make(map[string]string)
	split := strings.Split(key, ";")
	if len(split) < 2 {
		return key, labels
	}

	for _, raw := range split[1:] {
		s := strings.SplitN(raw, "=", 2)
		labels[s[0]] = s[1]
	}

	return split[0], labels
}

func formatSamples(source map[string]*AggregateSample) []SampledValue {
	output := []SampledValue{}
	for name, aggregate := range source {
		key, labels := extractLabels(name)
		output = append(output, SampledValue{
			Name:            key,
			AggregateSample: aggregate,
			Mean:            aggregate.Mean(),
			Stddev:          aggregate.Stddev(),
			Labels:          labels,
		})
	}
	sort.Slice(output, func(i, j int) bool {
		a := combineNameLabels(output[i].Name, output[i].Labels)
		b := combineNameLabels(output[j].Name, output[j].Labels)
		return a < b
	})

	return output
}

func combineNameLabels(name string, labels map[string]string) string {
	var rawLabels []string
	for name, val := range labels {
		rawLabels = append(rawLabels, name+val)
	}
	sort.Strings(rawLabels)

	result := name
	for _, label := range rawLabels {
		result += label
	}
	return result
}
