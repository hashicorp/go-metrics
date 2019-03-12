package inmem

import (
	"testing"
	"time"

	"github.com/hugoluchessi/go-metrics"
)

func TestDisplayMetrics(t *testing.T) {
	interval := 10 * time.Millisecond
	inm := NewSink(interval, 50*time.Millisecond)

	// Add data points
	inm.SetGauge([]string{"foo", "bar"}, 42)
	inm.SetGaugeWithLabels([]string{"foo", "bar"}, 23, []metrics.Label{{Name: "a", Value: "b"}})
	inm.EmitKey([]string{"foo", "bar"}, 42)
	inm.IncrCounter([]string{"foo", "bar"}, 20)
	inm.IncrCounter([]string{"foo", "bar"}, 22)
	inm.IncrCounterWithLabels([]string{"foo", "bar"}, 20, []metrics.Label{{Name: "a", Value: "b"}})
	inm.IncrCounterWithLabels([]string{"foo", "bar"}, 40, []metrics.Label{{Name: "a", Value: "b"}})
	inm.AddSample([]string{"foo", "bar"}, 20)
	inm.AddSample([]string{"foo", "bar"}, 24)
	inm.AddSampleWithLabels([]string{"foo", "bar"}, 23, []metrics.Label{{Name: "a", Value: "b"}})
	inm.AddSampleWithLabels([]string{"foo", "bar"}, 33, []metrics.Label{{Name: "a", Value: "b"}})

	data := inm.Data()
	if len(data) != 1 {
		t.Fatalf("bad: %v", data)
	}

	expected := MetricsSummary{
		Timestamp: data[0].Interval.Round(time.Second).UTC().String(),
		Gauges: []GaugeValue{
			{
				Name:          "foo.bar",
				Hash:          "foo.bar",
				Value:         float32(42),
				DisplayLabels: map[string]string{},
			},
			{
				Name:          "foo.bar",
				Hash:          "foo.bar;a=b",
				Value:         float32(23),
				DisplayLabels: map[string]string{"a": "b"},
			},
		},
		Points: []PointValue{
			{
				Name:   "foo.bar",
				Points: []float32{42},
			},
		},
		Counters: []SampledValue{
			{
				Name: "foo.bar",
				Hash: "foo.bar",
				AggregateSample: &AggregateSample{
					Count: 2,
					Min:   20,
					Max:   22,
					Sum:   42,
					SumSq: 884,
					Rate:  4200,
				},
				Mean:   21,
				Stddev: 1.4142135623730951,
			},
			{
				Name: "foo.bar",
				Hash: "foo.bar;a=b",
				AggregateSample: &AggregateSample{
					Count: 2,
					Min:   20,
					Max:   40,
					Sum:   60,
					SumSq: 2000,
					Rate:  6000,
				},
				Mean:          30,
				Stddev:        14.142135623730951,
				DisplayLabels: map[string]string{"a": "b"},
			},
		},
		Samples: []SampledValue{
			{
				Name: "foo.bar",
				Hash: "foo.bar",
				AggregateSample: &AggregateSample{
					Count: 2,
					Min:   20,
					Max:   24,
					Sum:   44,
					SumSq: 976,
					Rate:  4400,
				},
				Mean:   22,
				Stddev: 2.8284271247461903,
			},
			{
				Name: "foo.bar",
				Hash: "foo.bar;a=b",
				AggregateSample: &AggregateSample{
					Count: 2,
					Min:   23,
					Max:   33,
					Sum:   56,
					SumSq: 1618,
					Rate:  5600,
				},
				Mean:          28,
				Stddev:        7.0710678118654755,
				DisplayLabels: map[string]string{"a": "b"},
			},
		},
	}

	raw, err := inm.DisplayMetrics(nil, nil)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	result := raw.(MetricsSummary)

	// Ignore the LastUpdated field, we don't export that anyway
	for i, got := range result.Counters {
		expected.Counters[i].LastUpdated = got.LastUpdated
	}
	for i, got := range result.Samples {
		expected.Samples[i].LastUpdated = got.LastUpdated
	}
}
