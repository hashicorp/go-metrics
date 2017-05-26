package metrics

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

type MockSink struct {
	keys [][]string
	vals []float32
}

func (m *MockSink) SetGauge(key []string, val float32) {
	m.keys = append(m.keys, key)
	m.vals = append(m.vals, val)
}
func (m *MockSink) EmitKey(key []string, val float32) {
	m.keys = append(m.keys, key)
	m.vals = append(m.vals, val)
}
func (m *MockSink) IncrCounter(key []string, val float32) {
	m.keys = append(m.keys, key)
	m.vals = append(m.vals, val)
}
func (m *MockSink) AddSample(key []string, val float32) {
	m.keys = append(m.keys, key)
	m.vals = append(m.vals, val)
}

func TestFanoutSink_Gauge(t *testing.T) {
	m1 := &MockSink{}
	m2 := &MockSink{}
	fh := &FanoutSink{m1, m2}

	k := []string{"test"}
	v := float32(42.0)
	fh.SetGauge(k, v)

	if !reflect.DeepEqual(m1.keys[0], k) {
		t.Fatalf("key not equal")
	}
	if !reflect.DeepEqual(m2.keys[0], k) {
		t.Fatalf("key not equal")
	}
	if !reflect.DeepEqual(m1.vals[0], v) {
		t.Fatalf("val not equal")
	}
	if !reflect.DeepEqual(m2.vals[0], v) {
		t.Fatalf("val not equal")
	}
}

func TestFanoutSink_Key(t *testing.T) {
	m1 := &MockSink{}
	m2 := &MockSink{}
	fh := &FanoutSink{m1, m2}

	k := []string{"test"}
	v := float32(42.0)
	fh.EmitKey(k, v)

	if !reflect.DeepEqual(m1.keys[0], k) {
		t.Fatalf("key not equal")
	}
	if !reflect.DeepEqual(m2.keys[0], k) {
		t.Fatalf("key not equal")
	}
	if !reflect.DeepEqual(m1.vals[0], v) {
		t.Fatalf("val not equal")
	}
	if !reflect.DeepEqual(m2.vals[0], v) {
		t.Fatalf("val not equal")
	}
}

func TestFanoutSink_Counter(t *testing.T) {
	m1 := &MockSink{}
	m2 := &MockSink{}
	fh := &FanoutSink{m1, m2}

	k := []string{"test"}
	v := float32(42.0)
	fh.IncrCounter(k, v)

	if !reflect.DeepEqual(m1.keys[0], k) {
		t.Fatalf("key not equal")
	}
	if !reflect.DeepEqual(m2.keys[0], k) {
		t.Fatalf("key not equal")
	}
	if !reflect.DeepEqual(m1.vals[0], v) {
		t.Fatalf("val not equal")
	}
	if !reflect.DeepEqual(m2.vals[0], v) {
		t.Fatalf("val not equal")
	}
}

func TestFanoutSink_Sample(t *testing.T) {
	m1 := &MockSink{}
	m2 := &MockSink{}
	fh := &FanoutSink{m1, m2}

	k := []string{"test"}
	v := float32(42.0)
	fh.AddSample(k, v)

	if !reflect.DeepEqual(m1.keys[0], k) {
		t.Fatalf("key not equal")
	}
	if !reflect.DeepEqual(m2.keys[0], k) {
		t.Fatalf("key not equal")
	}
	if !reflect.DeepEqual(m1.vals[0], v) {
		t.Fatalf("val not equal")
	}
	if !reflect.DeepEqual(m2.vals[0], v) {
		t.Fatalf("val not equal")
	}
}

func TestNewMetricSinkFromURL(t *testing.T) {
	cases := map[string]struct {
		Input string
		Check func(MetricSink, error) error
	}{
		"statsd": {
			Input: "statsd://someserver:123",
			Check: func(ms MetricSink, err error) error {
				if err != nil {
					return fmt.Errorf("unexpected err: %s", err)
				}
				ss, ok := ms.(*StatsdSink)
				if !ok {
					return fmt.Errorf("Response is not a *StatsdSink: %#v", ms)
				}
				expectAddr := "someserver:123"
				if ss.addr != expectAddr {
					return fmt.Errorf("Expected addr %q, got: %q", expectAddr, ss.addr)
				}
				return nil
			},
		},
		"statsite": {
			Input: "statsite://someserver:123",
			Check: func(ms MetricSink, err error) error {
				if err != nil {
					return fmt.Errorf("unexpected err: %s", err)
				}
				ss, ok := ms.(*StatsiteSink)
				if !ok {
					return fmt.Errorf("Response is not a *StatsiteSink: %#v", ms)
				}
				expectAddr := "someserver:123"
				if ss.addr != expectAddr {
					return fmt.Errorf("Expected addr %q, got: %q", expectAddr, ss.addr)
				}
				return nil
			},
		},
		"inmem": {
			Input: "inmem://?interval=30s&duration=30s",
			Check: func(ms MetricSink, err error) error {
				if err != nil {
					return fmt.Errorf("unexpected err: %s", err)
				}
				if _, ok := ms.(*InmemSink); !ok {
					return fmt.Errorf("Response is not a *InmemSink: %#v", ms)
				}
				return nil
			},
		},
		"unknown": {
			Input: "notasink://someserver:123",
			Check: func(ms MetricSink, err error) error {
				if err == nil {
					return fmt.Errorf("expected err, got none")
				}
				if !strings.Contains(err.Error(), "unrecognized sink name: \"notasink\"") {
					return fmt.Errorf("unexpected kind of err: %s", err)
				}
				return nil
			},
		},
	}

	for name, tc := range cases {
		output, err := NewMetricSinkFromURL(tc.Input)
		resultErr := tc.Check(output, err)
		if resultErr != nil {
			t.Errorf("%s: %s", name, resultErr)
		}
	}
}
