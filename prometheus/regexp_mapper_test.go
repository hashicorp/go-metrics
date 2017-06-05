package prometheus

import "testing"

func TestRegexpMapper(t *testing.T) {
	m := NewRegexpMapper([]RegexpMappingRule{
		RegexpMappingRule{
			Pattern:         `metric_(.*?)_on_(.*)`,
			NameReplacement: "$1",
			LabelReplacements: map[string]string{
				"host":      "$2.local",
				"hostdummy": "$2",
				"multi":     "$2.$2.local",
			},
		},
		RegexpMappingRule{
			Pattern:         `metric_(.*?)_on_(.*)`,
			NameReplacement: "wrongmatch",
		},
		RegexpMappingRule{
			Pattern:         `metric2_(.*?)_on_(.*)`,
			NameReplacement: "secondmatch",
		},
	})
	name, labels, present := m.MapMetric([]string{"metric", "memory", "on", "localhost"})

	if name != "memory" {
		t.Fatalf("regexp mapper returns wrong metric name: %v", name)
	}

	if labels == nil {
		t.Fatalf("regexp mapper returns nil labels")
	}

	if value, ok := labels["host"]; !ok || value != "localhost.local" {
		t.Fatalf("regexp mapper returns wrong label value: %v", value)
	}

	if value, ok := labels["hostdummy"]; !ok || value != "localhost" {
		t.Fatalf("regexp mapper returns wrong label value: %v", value)
	}

	if value, ok := labels["multi"]; !ok || value != "localhost.localhost.local" {
		t.Fatalf("regexp mapper return wrong label value: %v", labels["multi"])
	}
	if !present {
		t.Fatalf("regexp mapper returns present = false")
	}

	name, labels, present = m.MapMetric([]string{"nomatch"})

	if name != "" || labels != nil || present {
		t.Fatalf("regexp mapper matches where it shouldn't")
	}

	name, _, _ = m.MapMetric([]string{"metric2_test_on_localhost"})

	if name != "secondmatch" {
		t.Fatalf("wrong match: %v", name)
	}

}
