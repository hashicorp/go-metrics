// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package prometheus

// This test demonstrates a race condition when using PrometheusSink when run from multiple
// goroutines concurrently resulting in missed updates.

import (
	"sync"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	dto "github.com/prometheus/client_model/go"
)

func TestPrometheusRaceCondition(t *testing.T) {
	reg := prometheus.NewRegistry()

	promSink, err := NewPrometheusSinkFrom(PrometheusOpts{
		Registerer: reg,
	})
	if err != nil {
		t.Fatal(err)
	}

	nrGoroutines := 20
	incrementsPerGoroutine := 1000
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

	// Collect metrics after all updates
	timeAfterUpdates := time.Now()
	ch := make(chan prometheus.Metric, 10)
	promSink.collectAtTime(ch, timeAfterUpdates)

	// Read and verify the counter
	select {
	case m := <-ch:
		var pb dto.Metric
		if err := m.Write(&pb); err != nil {
			t.Fatalf("unexpected error reading metric: %s", err)
		}
		if pb.Counter == nil {
			t.Fatalf("expected counter metric, got %v", pb)
		}
		if *pb.Counter.Value != float64(expectedTotal) {
			t.Fatalf("expected counter value %d, got %f", expectedTotal, *pb.Counter.Value)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("timed out waiting to collect counter metric")
	}
}
