package prometheus

import "testing"

func TestSimpleMapper(t *testing.T) {
	m := &simpleMapper{}
	name, labels, present := m.MapMetric([]string{"this", "is", "a", "metric"})

	if name != "this_is_a_metric" || labels != nil || !present {
		t.Fatalf("simple mapper builds wrong metric name")
	}
}
