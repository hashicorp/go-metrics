package inmem

import (
	"math"
	"testing"
	"time"

	"github.com/hugoluchessi/go-metrics"
)

func TestInmemSink(t *testing.T) {
	inm := NewSink(10*time.Millisecond, 50*time.Millisecond)

	data := inm.Data()
	if len(data) < 1 {
		t.Fatalf("bad: %v", data)
	}

	// Add data points
	inm.SetGauge([]string{"foo", "bar"}, 42)
	inm.SetGaugeWithLabels([]string{"foo", "bar"}, 23, []metrics.Label{{Name: "a", Value: "b"}})
	inm.EmitKey([]string{"foo", "bar"}, 42)
	inm.IncrCounter([]string{"foo", "bar"}, 20)
	inm.IncrCounter([]string{"foo", "bar"}, 22)
	inm.IncrCounterWithLabels([]string{"foo", "bar"}, 20, []metrics.Label{{Name: "a", Value: "b"}})
	inm.IncrCounterWithLabels([]string{"foo", "bar"}, 22, []metrics.Label{{Name: "a", Value: "b"}})
	inm.AddSample([]string{"foo", "bar"}, 20)
	inm.AddSample([]string{"foo", "bar"}, 22)
	inm.AddSampleWithLabels([]string{"foo", "bar"}, 23, []metrics.Label{{Name: "a", Value: "b"}})

	data = inm.Data()
	if len(data) != 1 {
		t.Fatalf("bad: %v", data)
	}

	intvM := data[0]
	intvM.RLock()

	if time.Now().Sub(intvM.Interval) > 10*time.Millisecond {
		t.Fatalf("interval too old")
	}
	if intvM.Gauges["foo.bar"].Value != 42 {
		t.Fatalf("bad val: %v", intvM.Gauges)
	}
	if intvM.Gauges["foo.bar;a=b"].Value != 23 {
		t.Fatalf("bad val: %v", intvM.Gauges)
	}
	if intvM.Points["foo.bar"][0] != 42 {
		t.Fatalf("bad val: %v", intvM.Points)
	}

	for _, agg := range []SampledValue{intvM.Counters["foo.bar"], intvM.Counters["foo.bar;a=b"]} {
		if agg.Count != 2 {
			t.Fatalf("bad val: %v", agg)
		}
		if agg.Rate != 4200 {
			t.Fatalf("bad val: %v", agg.Rate)
		}
		if agg.Sum != 42 {
			t.Fatalf("bad val: %v", agg)
		}
		if agg.SumSq != 884 {
			t.Fatalf("bad val: %v", agg)
		}
		if agg.Min != 20 {
			t.Fatalf("bad val: %v", agg)
		}
		if agg.Max != 22 {
			t.Fatalf("bad val: %v", agg)
		}
		if agg.AggregateSample.Mean() != 21 {
			t.Fatalf("bad val: %v", agg)
		}
		if agg.AggregateSample.Stddev() != math.Sqrt(2) {
			t.Fatalf("bad val: %v", agg)
		}

		if agg.LastUpdated.IsZero() {
			t.Fatalf("agg.LastUpdated is not set: %v", agg)
		}

		diff := time.Now().Sub(agg.LastUpdated).Seconds()
		if diff > 1 {
			t.Fatalf("time diff too great: %f", diff)
		}
	}

	if _, ok := intvM.Samples["foo.bar"]; !ok {
		t.Fatalf("missing sample")
	}

	if _, ok := intvM.Samples["foo.bar;a=b"]; !ok {
		t.Fatalf("missing sample")
	}

	intvM.RUnlock()

	for i := 1; i < 10; i++ {
		time.Sleep(10 * time.Millisecond)
		inm.SetGauge([]string{"foo", "bar"}, 42)
		data = inm.Data()
		if len(data) != min(i+1, 5) {
			t.Fatalf("bad: %v", data)
		}
	}

	// Should not exceed 5 intervals!
	time.Sleep(10 * time.Millisecond)
	inm.SetGauge([]string{"foo", "bar"}, 42)
	data = inm.Data()
	if len(data) != 5 {
		t.Fatalf("bad: %v", data)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func duration(t *testing.T, s string) time.Duration {
	dur, err := time.ParseDuration(s)
	if err != nil {
		t.Fatalf("error parsing duration: %s", err)
	}
	return dur
}
