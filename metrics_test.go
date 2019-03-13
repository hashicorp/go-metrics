package metrics

import (
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func mockMetric() (*MockSink, *MetricService) {
	m := &MockSink{}
	met := &MetricService{MetricServiceConfig: MetricServiceConfig{}, sink: m}
	return m, met
}

func TestMetricService_New(t *testing.T) {
	m := &MockSink{}
	met := NewMetricService(&MetricServiceConfig{}, m)

	if met == nil {
		t.Fatalf("met must noe be nil")
	}
}

func TestMetricService_SetGauge(t *testing.T) {
	m, met := mockMetric()
	met.SetGauge([]string{"key"}, float32(1))
	if m.keys[0][0] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}

	m, met = mockMetric()
	labels := []Label{{"a", "b"}}
	met.SetGaugeWithLabels([]string{"key"}, float32(1), labels)
	if m.keys[0][0] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}
	if !reflect.DeepEqual(m.labels[0], labels) {
		t.Fatalf("")
	}

	m, met = mockMetric()
	met.HostName = "test"
	met.EnableHostName = true
	met.SetGauge([]string{"key"}, float32(1))
	if m.keys[0][0] != "test" || m.keys[0][1] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}

	m, met = mockMetric()
	met.EnableTypeSufix = true
	met.SetGauge([]string{"key"}, float32(1))
	if m.keys[0][0] != "key" || m.keys[0][1] != "gauge" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}

	m, met = mockMetric()
	met.ServiceName = "service"
	met.EnableServiceName = true
	met.SetGauge([]string{"key"}, float32(1))
	if m.keys[0][0] != "service" || m.keys[0][1] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}
}

func TestMetricService_EmitKey(t *testing.T) {
	m, met := mockMetric()
	met.EmitKey([]string{"key"}, float32(1))
	if m.keys[0][0] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}

	m, met = mockMetric()
	met.EnableTypeSufix = true
	met.EmitKey([]string{"key"}, float32(1))
	if m.keys[0][0] != "key" || m.keys[0][1] != "kv" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}

	m, met = mockMetric()
	met.ServiceName = "service"
	met.EnableServiceName = true
	met.EmitKey([]string{"key"}, float32(1))
	if m.keys[0][0] != "service" || m.keys[0][1] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}
}

func TestMetricService_IncrCounter(t *testing.T) {
	m, met := mockMetric()
	met.IncrCounter([]string{"key"}, float32(1))
	if m.keys[0][0] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}

	m, met = mockMetric()
	labels := []Label{{"a", "b"}}
	met.IncrCounterWithLabels([]string{"key"}, float32(1), labels)
	if m.keys[0][0] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}
	if !reflect.DeepEqual(m.labels[0], labels) {
		t.Fatalf("")
	}

	m, met = mockMetric()
	met.EnableTypeSufix = true
	met.IncrCounter([]string{"key"}, float32(1))
	if m.keys[0][0] != "key" || m.keys[0][1] != "counter" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}

	m, met = mockMetric()
	met.ServiceName = "service"
	met.EnableServiceName = true
	met.IncrCounter([]string{"key"}, float32(1))
	if m.keys[0][0] != "service" || m.keys[0][1] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}
}

func TestMetricService_AddSample(t *testing.T) {
	m, met := mockMetric()
	met.AddSample([]string{"key"}, float32(1))
	if m.keys[0][0] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}

	m, met = mockMetric()
	labels := []Label{{"a", "b"}}
	met.AddSampleWithLabels([]string{"key"}, float32(1), labels)
	if m.keys[0][0] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}
	if !reflect.DeepEqual(m.labels[0], labels) {
		t.Fatalf("")
	}

	m, met = mockMetric()
	met.EnableTypeSufix = true
	met.AddSample([]string{"key"}, float32(1))
	if m.keys[0][0] != "key" || m.keys[0][1] != "sample" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}

	m, met = mockMetric()
	met.ServiceName = "service"
	met.EnableServiceName = true
	met.AddSample([]string{"key"}, float32(1))
	if m.keys[0][0] != "service" || m.keys[0][1] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] != 1 {
		t.Fatalf("")
	}
}

func TestMetricService_MeasureSince(t *testing.T) {
	m, met := mockMetric()
	met.TimerGranularity = time.Millisecond
	n := time.Now()
	met.MeasureSince([]string{"key"}, n)
	if m.keys[0][0] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] > 0.1 {
		t.Fatalf("")
	}

	m, met = mockMetric()
	met.TimerGranularity = time.Millisecond
	labels := []Label{{"a", "b"}}
	met.MeasureSinceWithLabels([]string{"key"}, n, labels)
	if m.keys[0][0] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] > 0.1 {
		t.Fatalf("")
	}
	if !reflect.DeepEqual(m.labels[0], labels) {
		t.Fatalf("")
	}

	m, met = mockMetric()
	met.TimerGranularity = time.Millisecond
	met.EnableTypeSufix = true
	met.MeasureSince([]string{"key"}, n)
	if m.keys[0][0] != "key" || m.keys[0][1] != "timer" {
		t.Fatalf("")
	}
	if m.vals[0] > 0.1 {
		t.Fatalf("")
	}

	m, met = mockMetric()
	met.TimerGranularity = time.Millisecond
	met.ServiceName = "service"
	met.EnableServiceName = true
	met.MeasureSince([]string{"key"}, n)
	if m.keys[0][0] != "service" || m.keys[0][1] != "key" {
		t.Fatalf("")
	}
	if m.vals[0] > 0.1 {
		t.Fatalf("")
	}
}

func TestMetricService_EmitRuntimeStats(t *testing.T) {
	runtime.GC()
	m, met := mockMetric()
	met.emitRuntimeStats()

	if m.keys[0][0] != "runtime" || m.keys[0][1] != "num_goroutines" {
		t.Fatalf("bad key %v", m.keys)
	}
	if m.vals[0] <= 1 {
		t.Fatalf("bad val: %v", m.vals)
	}

	if m.keys[1][0] != "runtime" || m.keys[1][1] != "alloc_bytes" {
		t.Fatalf("bad key %v", m.keys)
	}
	if m.vals[1] <= 40000 {
		t.Fatalf("bad val: %v", m.vals)
	}

	if m.keys[2][0] != "runtime" || m.keys[2][1] != "sys_bytes" {
		t.Fatalf("bad key %v", m.keys)
	}
	if m.vals[2] <= 100000 {
		t.Fatalf("bad val: %v", m.vals)
	}

	if m.keys[3][0] != "runtime" || m.keys[3][1] != "malloc_count" {
		t.Fatalf("bad key %v", m.keys)
	}
	if m.vals[3] <= 100 {
		t.Fatalf("bad val: %v", m.vals)
	}

	if m.keys[4][0] != "runtime" || m.keys[4][1] != "free_count" {
		t.Fatalf("bad key %v", m.keys)
	}
	if m.vals[4] <= 100 {
		t.Fatalf("bad val: %v", m.vals)
	}

	if m.keys[5][0] != "runtime" || m.keys[5][1] != "heap_objects" {
		t.Fatalf("bad key %v", m.keys)
	}
	if m.vals[5] <= 100 {
		t.Fatalf("bad val: %v", m.vals)
	}

	if m.keys[6][0] != "runtime" || m.keys[6][1] != "total_gc_pause_ns" {
		t.Fatalf("bad key %v", m.keys)
	}
	if m.vals[6] <= 5000 {
		t.Fatalf("bad val: %v", m.vals)
	}

	if m.keys[7][0] != "runtime" || m.keys[7][1] != "total_gc_runs" {
		t.Fatalf("bad key %v", m.keys)
	}
	if m.vals[7] < 1 {
		t.Fatalf("bad val: %v", m.vals)
	}

	if m.keys[8][0] != "runtime" || m.keys[8][1] != "gc_pause_ns" {
		t.Fatalf("bad key %v", m.keys)
	}
	if m.vals[8] <= 1000 {
		t.Fatalf("bad val: %v", m.vals)
	}
}

func TestMetricService_getKey(t *testing.T) {
	m := &MockSink{}

	var hostName = "someServer"
	var serviceName = "testMetrics"
	var metric *MetricService
	var expectedMetricKey []string
	var gotMetricKey []string

	testKey := []string{"some", "good", "key"}

	t.Run("get key with hostname enabled", func(*testing.T) {
		expectedMetricKey = []string{"some", "good", "key"}
		metric = &MetricService{MetricServiceConfig: MetricServiceConfig{EnableHostName: true}, sink: m}
		gotMetricKey = metric.getKey(testKey, "gauge")

		if !reflect.DeepEqual(expectedMetricKey, gotMetricKey) {
			t.Fatalf("expected key to be '%s' got '%s'", strings.Join(expectedMetricKey, ","), strings.Join(gotMetricKey, ","))
		}

		expectedMetricKey = []string{hostName, "some", "good", "key"}
		metric = &MetricService{MetricServiceConfig: MetricServiceConfig{HostName: hostName, EnableHostName: true}, sink: m}
		gotMetricKey = metric.getKey(testKey, "gauge")

		if !reflect.DeepEqual(expectedMetricKey, gotMetricKey) {
			t.Fatalf("expected key to be '%s' got '%s'", strings.Join(expectedMetricKey, ","), strings.Join(gotMetricKey, ","))
		}
	})

	t.Run("get key with service name enabled", func(*testing.T) {
		expectedMetricKey = []string{"some", "good", "key"}
		metric = &MetricService{MetricServiceConfig: MetricServiceConfig{EnableServiceName: true}, sink: m}
		gotMetricKey = metric.getKey(testKey, "gauge")

		if !reflect.DeepEqual(expectedMetricKey, gotMetricKey) {
			t.Fatalf("expected key to be '%s' got '%s'", strings.Join(expectedMetricKey, ","), strings.Join(gotMetricKey, ","))
		}

		expectedMetricKey = []string{serviceName, "some", "good", "key"}
		metric = &MetricService{MetricServiceConfig: MetricServiceConfig{ServiceName: serviceName, EnableServiceName: true}, sink: m}
		gotMetricKey = metric.getKey(testKey, "gauge")

		if !reflect.DeepEqual(expectedMetricKey, gotMetricKey) {
			t.Fatalf("expected key to be '%s' got '%s'", strings.Join(expectedMetricKey, ","), strings.Join(gotMetricKey, ","))
		}
	})

	t.Run("get key with service name enabled", func(*testing.T) {
		expectedMetricKey = []string{"some", "good", "key", "gauge"}
		metric = &MetricService{MetricServiceConfig: MetricServiceConfig{EnableTypeSufix: true}, sink: m}
		gotMetricKey = metric.getKey(testKey, "gauge")

		if !reflect.DeepEqual(expectedMetricKey, gotMetricKey) {
			t.Fatalf("expected key to be '%s' got '%s'", strings.Join(expectedMetricKey, ","), strings.Join(gotMetricKey, ","))
		}
	})
}
