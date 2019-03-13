package metrics

import (
	"io/ioutil"
	"log"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

func TestGlobalMetrics_Init(t *testing.T) {
	s := &MockSink{}
	cfg := &MetricServiceConfig{}

	m := InitGlobal(cfg, s)
	loaded := globalMetrics.Load().(*MetricService)

	if loaded != m {
		t.Fatal("invalid global instance of MetricService")
	}

	s = &MockSink{}
	cfg = &MetricServiceConfig{}

	_ = InitGlobal(cfg, s)
	loaded = globalMetrics.Load().(*MetricService)

	if loaded != m {
		t.Fatal("invalid global instance of MetricService")
	}
}

func TestGlobalMetrics_Metrics(t *testing.T) {
	var tests = []struct {
		desc string
		key  []string
		val  float32
		fn   func([]string, float32)
	}{
		{"SetGauge", []string{"test"}, 42, SetGauge},
		{"EmitKey", []string{"test"}, 42, EmitKey},
		{"IncrCounter", []string{"test"}, 42, IncrCounter},
		{"AddSample", []string{"test"}, 42, AddSample},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s := &MockSink{}
			globalMetrics.Store(&MetricService{MetricServiceConfig: MetricServiceConfig{}, sink: s})
			tt.fn(tt.key, tt.val)
			if got, want := s.keys[0], tt.key; !reflect.DeepEqual(got, want) {
				t.Fatalf("got key %s want %s", got, want)
			}
			if got, want := s.vals[0], tt.val; !reflect.DeepEqual(got, want) {
				t.Fatalf("got val %f want %f", got, want)
			}
		})
	}
}

func TestGlobalMetrics_WithLabels(t *testing.T) {
	labels := []Label{{"a", "b"}}
	var tests = []struct {
		desc   string
		key    []string
		val    float32
		fn     func([]string, float32, []Label)
		labels []Label
	}{
		{"SetGaugeWithLabels", []string{"test"}, 42, SetGaugeWithLabels, labels},
		{"IncrCounterWithLabels", []string{"test"}, 42, IncrCounterWithLabels, labels},
		{"AddSampleWithLabels", []string{"test"}, 42, AddSampleWithLabels, labels},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			s := &MockSink{}
			globalMetrics.Store(&MetricService{MetricServiceConfig: MetricServiceConfig{}, sink: s})
			tt.fn(tt.key, tt.val, tt.labels)
			if got, want := s.keys[0], tt.key; !reflect.DeepEqual(got, want) {
				t.Fatalf("got key %s want %s", got, want)
			}
			if got, want := s.vals[0], tt.val; !reflect.DeepEqual(got, want) {
				t.Fatalf("got val %f want %f", got, want)
			}
			if got, want := s.labels[0], tt.labels; !reflect.DeepEqual(got, want) {
				t.Fatalf("got val %s want %s", got, want)
			}
		})
	}
}

func TestGlobalMetrics_Timer(t *testing.T) {
	s := &MockSink{}
	m := &MetricService{sink: s, MetricServiceConfig: MetricServiceConfig{TimerGranularity: time.Millisecond}}
	globalMetrics.Store(m)

	k := []string{"test"}
	now := time.Now()
	MeasureSince(k, now)

	if !reflect.DeepEqual(s.keys[0], k) {
		t.Fatalf("key not equal")
	}
	if s.vals[0] > 0.1 {
		t.Fatalf("val too large %v", s.vals[0])
	}

	labels := []Label{{"a", "b"}}
	MeasureSinceWithLabels(k, now, labels)
	if got, want := s.keys[1], k; !reflect.DeepEqual(got, want) {
		t.Fatalf("got key %s want %s", got, want)
	}
	if s.vals[1] > 0.1 {
		t.Fatalf("val too large %v", s.vals[0])
	}
	if got, want := s.labels[1], labels; !reflect.DeepEqual(got, want) {
		t.Fatalf("got val %s want %s", got, want)
	}
}

// Benchmark_GlobalMetrics_Direct/direct-8         	 5000000	       278 ns/op
// Benchmark_GlobalMetrics_Direct/atomic.Value-8   	 5000000	       235 ns/op
func Benchmark_GlobalMetrics_Direct(b *testing.B) {
	log.SetOutput(ioutil.Discard)
	s := &MockSink{}
	m := &MetricService{sink: s}
	var v atomic.Value
	v.Store(m)
	k := []string{"test"}
	b.Run("direct", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			m.IncrCounter(k, 1)
		}
	})
	b.Run("atomic.Value", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v.Load().(*MetricService).IncrCounter(k, 1)
		}
	})
	// do something with m so that the compiler does not optimize this away
	b.Logf("%d", m.lastNumGC)
}
