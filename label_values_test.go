// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MIT

package metrics

import (
	"sync"
	"testing"
	"time"
)

func TestBlockedLabelValues(t *testing.T) {
	for name, emit := range map[string]func(*Metrics, []Label){
		"gauge":           func(m *Metrics, l []Label) { m.SetGaugeWithLabels([]string{"key"}, 1, l) },
		"precision-gauge": func(m *Metrics, l []Label) { m.SetPrecisionGaugeWithLabels([]string{"key"}, 1, l) },
		"counter":         func(m *Metrics, l []Label) { m.IncrCounterWithLabels([]string{"key"}, 1, l) },
		"sample":          func(m *Metrics, l []Label) { m.AddSampleWithLabels([]string{"key"}, 1, l) },
		"timer":           func(m *Metrics, l []Label) { m.MeasureSinceWithLabels([]string{"key"}, time.Now(), l) },
	} {
		t.Run(name, func(t *testing.T) {
			blocked := []Label{{Name: "status", Value: "bad"}}
			conf := DefaultConfig("")
			conf.EnableRuntimeMetrics = false
			conf.EnableHostname = false
			conf.BlockedLabelValues = blocked
			conf.AllowedLabels = []string{"keep"}
			sink := &MockSink{}
			m, err := New(conf, sink)
			if err != nil {
				t.Fatal(err)
			}
			blocked[0].Value = "changed"
			emit(m, []Label{{Name: "status", Value: "bad"}, {Name: "keep", Value: "ok"}})
			if len(sink.getKeys()) != 0 {
				t.Fatal("blocked sample reached sink")
			}
			emit(m, []Label{{Name: "status", Value: "good"}, {Name: "keep", Value: "ok"}})
			if len(sink.getKeys()) != 1 {
				t.Fatal("allowed sample was dropped")
			}
			if len(sink.labels[0]) != 1 || sink.labels[0][0].Name != "keep" {
				t.Fatal("name filtering changed")
			}
			m.UpdateBlockedLabelValues(nil)
			emit(m, []Label{{Name: "status", Value: "bad"}})
			if len(sink.getKeys()) != 2 {
				t.Fatal("cleared blocklist remained active")
			}
		})
	}
}

func TestBlockedLabelValuesGeneratedLabelsAndGlobal(t *testing.T) {
	previous := globalMetrics.Load()
	t.Cleanup(func() { globalMetrics.Store(previous) })
	conf := DefaultConfig("blocked-service")
	conf.EnableRuntimeMetrics = false
	conf.EnableServiceLabel = true
	sink := &MockSink{}
	if _, err := NewGlobal(conf, sink); err != nil {
		t.Fatal(err)
	}
	UpdateBlockedLabelValues([]Label{{Name: "service", Value: "blocked-service"}})
	SetGaugeWithLabels([]string{"key"}, 1, nil)
	if len(sink.getKeys()) != 0 {
		t.Fatal("generated service label ignored")
	}
	EmitKey([]string{"key"}, 1)
	if len(sink.getKeys()) != 1 {
		t.Fatal("unlabelled key was blocked")
	}
}

func TestBlockedLabelValuesConcurrentUpdates(t *testing.T) {
	conf := DefaultConfig("")
	conf.EnableRuntimeMetrics = false
	m, err := New(conf, &BlackholeSink{})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	wg.Go(func() {
		for range 100 {
			m.UpdateBlockedLabelValues([]Label{{Name: "status", Value: "bad"}})
			m.UpdateBlockedLabelValues(nil)
		}
	})
	wg.Go(func() {
		for range 100 {
			m.SetGaugeWithLabels([]string{"key"}, 1, []Label{{Name: "status", Value: "bad"}})
		}
	})
	wg.Wait()
}
