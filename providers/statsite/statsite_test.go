package statsite

import (
	"bufio"
	"net"
	"testing"
	"time"

	"github.com/hugoluchessi/go-metrics"
)

func acceptConn(addr string) net.Conn {
	ln, _ := net.Listen("tcp", addr)
	conn, _ := ln.Accept()
	return conn
}

func TestStatsite_Flatten(t *testing.T) {
	s := &Sink{}
	flat := s.flattenKey([]string{"a", "b", "c", "d"})
	if flat != "a.b.c.d" {
		t.Fatalf("Bad flat")
	}
}

func TestStatsite_PushFullQueue(t *testing.T) {
	q := make(chan string, 1)
	q <- "full"

	s := &Sink{metricQueue: q}
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

func TestStatsite_Conn(t *testing.T) {
	addr := "localhost:7523"
	done := make(chan bool)
	go func() {
		conn := acceptConn(addr)
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

	time.Sleep(1 * time.Millisecond)

	s, err := NewSink(addr)
	if err != nil {
		t.Fatalf("bad error")
	}

	s.SetGauge([]string{"gauge", "val"}, float32(1))
	s.SetGaugeWithLabels([]string{"gauge_labels", "val"}, float32(2), []metrics.Label{{"a", "label"}})
	s.EmitKey([]string{"key", "other"}, float32(3))
	s.IncrCounter([]string{"counter", "me"}, float32(4))
	s.IncrCounterWithLabels([]string{"counter_labels", "me"}, float32(5), []metrics.Label{{"a", "label"}})
	s.AddSample([]string{"sample", "slow thingy"}, float32(6))
	s.AddSampleWithLabels([]string{"sample_labels", "slow thingy"}, float32(7), []metrics.Label{{"a", "label"}})

	select {
	case <-done:
		s.Shutdown()
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout")
	}
}
