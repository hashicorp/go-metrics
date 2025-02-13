// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

package metrics

import (
	"bufio"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestStatsite_Flatten(t *testing.T) {
	s := &StatsiteSink{}
	flat := s.flattenKey([]string{"a", "b", "c", "d"})
	if flat != "a.b.c.d" {
		t.Fatalf("Bad flat")
	}
}

func TestStatsite_PushFullQueue(t *testing.T) {
	q := make(chan string, 1)
	q <- "full"

	s := &StatsiteSink{metricQueue: q}
	s.pushMetric("omit")

	out := <-q
	if out != "full" {
		t.Fatalf("bad val %v", out)
	}

	select {
	case v := <-q:
		t.Fatalf("bad val %v", v)
	default:
	}
}

// initially there was some error connecting to the server.
// to correct it used the 8126 port and also added container creation in workflow
// but on above changes showed invalid address or nil pointer reference
func TestStatsite_Conn(t *testing.T) {
	addr := "127.0.0.1:7523"

	ln, _ := net.Listen("tcp", addr)

	done := make(chan bool)
	go func() {
		conn, err := ln.Accept()
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}

		reader := bufio.NewReader(conn)

		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}
		if line != "gauge.val:1.000000|g\n" {
			t.Fatalf("bad line %s", line)
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}
		if line != "gauge_labels.val.label:2.000000|g\n" {
			t.Fatalf("bad line %s", line)
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}
		if line != "key.other:3.000000|kv\n" {
			t.Fatalf("bad line %s", line)
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}
		if line != "counter.me:4.000000|c\n" {
			t.Fatalf("bad line %s", line)
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}
		if line != "counter_labels.me.label:5.000000|c\n" {
			t.Fatalf("bad line %s", line)
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}
		if line != "sample.slow_thingy:6.000000|ms\n" {
			t.Fatalf("bad line %s", line)
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}
		if line != "sample_labels.slow_thingy.label:7.000000|ms\n" {
			t.Fatalf("bad line %s", line)
		}

		conn.Close()
		done <- true
	}()
	s, err := NewStatsiteSink(addr)
	if err != nil {
		t.Fatalf("bad error")
	}

	s.SetGauge([]string{"gauge", "val"}, float32(1))
	s.SetGaugeWithLabels([]string{"gauge_labels", "val"}, float32(2), []Label{{"a", "label"}})
	s.SetPrecisionGauge([]string{"gauge", "val"}, float64(1))
	s.SetPrecisionGaugeWithLabels([]string{"gauge_labels", "val"}, float64(2), []Label{{"a", "label"}})
	s.EmitKey([]string{"key", "other"}, float32(3))
	s.IncrCounter([]string{"counter", "me"}, float32(4))
	s.IncrCounterWithLabels([]string{"counter_labels", "me"}, float32(5), []Label{{"a", "label"}})
	s.AddSample([]string{"sample", "slow thingy"}, float32(6))
	s.AddSampleWithLabels([]string{"sample_labels", "slow thingy"}, float32(7), []Label{{"a", "label"}})

	select {
	case <-done:
		s.Shutdown()
	case <-time.After(5 * time.Second):
		t.Fatalf("timeout")
	}
}

func TestNewStatsiteSinkFromURL(t *testing.T) {
	for _, tc := range []struct {
		desc       string
		input      string
		expectErr  string
		expectAddr string
	}{
		{
			desc:       "address is populated",
			input:      "statsd://127.0.0.1:7523",
			expectAddr: "127.0.0.1:7523",
		},
		{
			desc:       "address includes port",
			input:      "statsd://127.0.0.1:7523",
			expectAddr: "127.0.0.1:7523",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			u, err := url.Parse(tc.input)
			if err != nil {
				t.Fatalf("error parsing URL: %s", err)
			}
			ms, err := NewStatsiteSinkFromURL(u)
			if tc.expectErr != "" {
				if !strings.Contains(err.Error(), tc.expectErr) {
					t.Fatalf("expected err: %q, to contain: %q", err, tc.expectErr)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected err: %s", err)
				}
				is := ms.(*StatsiteSink)
				if is.addr != tc.expectAddr {
					t.Fatalf("expected addr %s, got: %s", tc.expectAddr, is.addr)
				}
			}
		})
	}
}
