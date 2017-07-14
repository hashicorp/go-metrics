package metrics

import (
	"bufio"
	"bytes"
	"net"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestStatsd_Flatten(t *testing.T) {
	s := &StatsdSink{}
	flat := s.flattenKey([]string{"a", "b", "c", "d"})
	if flat != "a.b.c.d" {
		t.Fatalf("Bad flat")
	}
}

func TestStatsd_PushFullQueue(t *testing.T) {
	q := make(chan string, 1)
	q <- "full"

	s := &StatsdSink{metricQueue: q}
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

func TestStatsd_Conn(t *testing.T) {
	addr := "127.0.0.1:7524"
	done := make(chan bool)
	go func() {
		list, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 7524})
		if err != nil {
			panic(err)
		}
		defer list.Close()
		buf := make([]byte, 1500)
		n, err := list.Read(buf)
		if err != nil {
			panic(err)
		}
		buf = buf[:n]
		reader := bufio.NewReader(bytes.NewReader(buf))

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
		if line != "key.other:2.000000|kv\n" {
			t.Fatalf("bad line %s", line)
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}
		if line != "counter.me:3.000000|c\n" {
			t.Fatalf("bad line %s", line)
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}
		if line != "sample.slow_thingy:4.000000|ms\n" {
			t.Fatalf("bad line %s", line)
		}

		done <- true
	}()
	s, err := NewStatsdSink(addr)
	if err != nil {
		t.Fatalf("bad error")
	}

	s.SetGauge([]string{"gauge", "val"}, float32(1))
	s.EmitKey([]string{"key", "other"}, float32(2))
	s.IncrCounter([]string{"counter", "me"}, float32(3))
	s.AddSample([]string{"sample", "slow thingy"}, float32(4))

	select {
	case <-done:
		s.Shutdown()
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout")
	}
}

func TestNewStatsdSinkFromURL(t *testing.T) {
	for _, tc := range []struct {
		desc       string
		input      string
		expectErr  string
		expectAddr string
	}{
		{
			desc:       "address is populated",
			input:      "statsd://statsd.service.consul",
			expectAddr: "statsd.service.consul",
		},
		{
			desc:       "address includes port",
			input:      "statsd://statsd.service.consul:1234",
			expectAddr: "statsd.service.consul:1234",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			u, err := url.Parse(tc.input)
			if err != nil {
				t.Fatalf("error parsing URL: %s", err)
			}
			ms, err := NewStatsdSinkFromURL(u)
			if tc.expectErr != "" {
				if !strings.Contains(err.Error(), tc.expectErr) {
					t.Fatalf("expected err: %q, to contain: %q", err, tc.expectErr)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected err: %s", err)
				}
				is := ms.(*StatsdSink)
				if is.addr != tc.expectAddr {
					t.Fatalf("expected addr %s, got: %s", tc.expectAddr, is.addr)
				}
			}
		})
	}
}

func TestNewStatsdSinkWithPrefix(t *testing.T) {
	addr := "127.0.0.1:7524"
	done := make(chan bool)
	go func() {
		lis, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 7524})
		if err != nil {
			panic(err)
		}
		defer lis.Close()
		buf := make([]byte, 1500)
		n, err := lis.Read(buf)
		if err != nil {
			panic(err)
		}
		reader := bufio.NewReader(bytes.NewReader(buf[:n]))
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}
		if line != "some_prefix.gauge:1.000000|g\n" {
			t.Fatalf("bad line %s", line)
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}
		if line != "some_prefix.key:2.000000|kv\n" {
			t.Fatalf("bad line %s", line)
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}
		if line != "some_prefix.counter:3.000000|c\n" {
			t.Fatalf("bad line %s", line)
		}

		line, err = reader.ReadString('\n')
		if err != nil {
			t.Fatalf("unexpected err %s", err)
		}
		if line != "some_prefix.sample:4.000000|ms\n" {
			t.Fatalf("bad line %s", line)
		}

		done <- true
	}()
	s, err := NewStatsdSinkWithPrefix(addr, "some_prefix")
	if err != nil {
		t.Fatalf("bad error: %v", err)
	}

	s.SetGauge([]string{"gauge"}, float32(1))
	s.EmitKey([]string{"key"}, float32(2))
	s.IncrCounter([]string{"counter"}, float32(3))
	s.AddSample([]string{"sample"}, float32(4))

	select {
	case <-done:
		s.Shutdown()
	case <-time.After(3 * time.Second):
		t.Fatalf("timeout")
	}
}
