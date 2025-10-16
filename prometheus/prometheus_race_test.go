// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package prometheus

// This test demonstrates a race condition when using PrometheusSink when run from multiple
// goroutines concurrently resulting in missed updates.

import (
	"sync"
	"testing"
	"time"

	prom "github.com/prometheus/client_golang/prometheus"
)

func TestPrometheusRaceCondition(t *testing.T) {
	// Create a new Prometheus sink with expiration
	promSink, err := NewPrometheusSinkFrom(PrometheusOpts{Expiration: 310 * time.Second})
	if err != nil {
		t.Fatal(err)
	}

	// Register it with a new Prometheus registry for isolation
	registry := prom.NewRegistry()
	registry.MustRegister(promSink)

	nrGoroutines := 2
	incrementsPerGoroutine := 100
	expectedTotal := int64(nrGoroutines * incrementsPerGoroutine)

	var wg sync.WaitGroup
	for range nrGoroutines {
		wg.Add(1)
		go func() {
			for range incrementsPerGoroutine {
				promSink.IncrCounter([]string{"race", "test", "counter"}, 1)
			}
			wg.Done()
		}()
	}
	wg.Wait()

	families, err := registry.Gather()
	if err != nil {
		t.Fatal(err)
	}
	var finalValue int64
	for _, family := range families {
		if family.GetName() == "race_test_counter" {
			for _, metric := range family.GetMetric() {
				if counter := metric.GetCounter(); counter != nil {
					finalValue = int64(counter.GetValue())
				}
			}
		}
	}
	if finalValue == 0 {
		t.Fatal("Counter metric not found")
	}

	if finalValue != expectedTotal {
		t.Errorf("Race condition detected: got %d want %d", finalValue, expectedTotal)
	}
}
