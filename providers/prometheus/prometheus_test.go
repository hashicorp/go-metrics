package prometheus

import (
	"strings"
	"testing"

	"github.com/hugoluchessi/go-metrics"
	dto "github.com/prometheus/client_model/go"
)

func TestPrometheusSink(t *testing.T) {
	p, _ := NewSink()

	// Add data points
	p.SetGauge([]string{"gauge", "one"}, 42)
	p.SetGaugeWithLabels([]string{"gauge", "two"}, 23, []metrics.Label{{Name: "a", Value: "b"}})
	p.IncrCounter([]string{"counter", "one"}, 22)
	p.IncrCounterWithLabels([]string{"counter", "two"}, 23, []metrics.Label{{Name: "a", Value: "b"}})
	p.AddSample([]string{"sum", "one"}, 22)
	p.AddSampleWithLabels([]string{"sum", "two"}, 23, []metrics.Label{{Name: "a", Value: "b"}})

	metrics, _ := p.registry.Gather()

	for _, m := range metrics {
		descString := m.String()
		dtoM := m.Metric[0]

		switch *m.Type {
		case dto.MetricType_GAUGE:
			AssertGaugeDTOMetric(t, descString, dtoM)
			break
		case dto.MetricType_COUNTER:
			AssertCounterDTOMetric(t, descString, dtoM)
			break
		case dto.MetricType_SUMMARY:
			AssertSummaryDTOMetric(t, descString, dtoM)
			break
		}
	}
}

func AssertGaugeDTOMetric(t *testing.T, desc string, dtoM *dto.Metric) {
	lbs := dtoM.GetLabel()
	g := dtoM.GetGauge()

	if strings.Contains(desc, "gauge_one") {
		expectedValue := float64(42)

		if *g.Value != expectedValue {
			t.Fatalf("expected gauge_one to have value %f got %f", expectedValue, *g.Value)
		}
	} else if strings.Contains(desc, "gauge_two") {
		expectedValue := float64(23)
		labelName := "a"
		labelValue := "b"

		if *g.Value != expectedValue {
			t.Fatalf("expected gauge_two to have value %f got %f", expectedValue, *g.Value)
		}

		lb := lbs[0]
		if *lb.Name != labelName {
			t.Fatalf("expected gauge_two label name 'a' got %s", *lb.Name)
		}

		if *lb.Value != labelValue {
			t.Fatalf("expected gauge_two label value 'b' got %s", *lb.Name)
		}
	} else {
		t.Fatal("unexpected gauge desc")
	}
}

func AssertCounterDTOMetric(t *testing.T, desc string, m *dto.Metric) {
	lbs := m.GetLabel()
	c := m.GetCounter()

	if strings.Contains(desc, "counter_one") {
		expectedValue := float64(22)

		if *c.Value != expectedValue {
			t.Fatalf("expected counter_one to have value %f got %f", expectedValue, *c.Value)
		}
	} else if strings.Contains(desc, "counter_two") {
		expectedValue := float64(23)
		labelName := "a"
		labelValue := "b"

		if *c.Value != expectedValue {
			t.Fatalf("expected counter_two to have value %f got %f", expectedValue, *c.Value)
		}

		lb := lbs[0]
		if *lb.Name != labelName {
			t.Fatalf("expected counter_two label name 'a' got %s", *lb.Name)
		}

		if *lb.Value != labelValue {
			t.Fatalf("expected counter_two label value 'b' got %s", *lb.Name)
		}
	} else {
		t.Fatal("unexpected gauge desc")
	}
}

func AssertSummaryDTOMetric(t *testing.T, desc string, m *dto.Metric) {
	lbs := m.GetLabel()
	s := m.GetSummary()

	if strings.Contains(desc, "sum_one") {
		expectedValue := float64(22)

		if *s.SampleSum != expectedValue {
			t.Fatalf("expected sum_one to have value %f got %f", expectedValue, *s.SampleSum)
		}
	} else if strings.Contains(desc, "sum_two") {
		expectedValue := float64(23)
		labelName := "a"
		labelValue := "b"

		if *s.SampleSum != expectedValue {
			t.Fatalf("expected sum_two to have value %f got %f", expectedValue, *s.SampleSum)
		}

		lb := lbs[0]
		if *lb.Name != labelName {
			t.Fatalf("expected sum_two label name 'a' got %s", *lb.Name)
		}

		if *lb.Value != labelValue {
			t.Fatalf("expected sum_two label value 'b' got %s", *lb.Name)
		}
	} else {
		t.Fatal("unexpected gauge desc")
	}
}
