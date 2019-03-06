package factory

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/hugoluchessi/go-metrics/providers/inmem"
	"github.com/hugoluchessi/go-metrics/providers/statsd"
	"github.com/hugoluchessi/go-metrics/providers/statsite"
)

func TestNewMetricSinkFromURL(t *testing.T) {
	for _, tc := range []struct {
		desc      string
		input     string
		expect    reflect.Type
		expectErr string
	}{
		{
			desc:   "statsd scheme yields a StatsdSink",
			input:  "statsd://someserver:123",
			expect: reflect.TypeOf(&statsd.StatsdSink{}),
		},
		{
			desc:   "statsite scheme yields a StatsiteSink",
			input:  "statsite://someserver:123",
			expect: reflect.TypeOf(&statsite.StatsiteSink{}),
		},
		{
			desc:   "inmem scheme yields an InmemSink",
			input:  "inmem://?interval=30s&retain=30s",
			expect: reflect.TypeOf(&inmem.InmemSink{}),
		},
		{
			desc:      "unknown scheme yields an error",
			input:     "notasink://whatever",
			expectErr: "unrecognized sink name: \"notasink\"",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			ms, err := NewSinkFromURL(tc.input)
			if tc.expectErr != "" {
				if !strings.Contains(err.Error(), tc.expectErr) {
					t.Fatalf("expected err: %q to contain: %q", err, tc.expectErr)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected err: %s", err)
				}
				got := reflect.TypeOf(ms)
				if got != tc.expect {
					t.Fatalf("expected return type to be %v, got: %v", tc.expect, got)
				}
			}
		})
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
			ms, err := NewStatsiteSinkFromURL(u)
			if tc.expectErr != "" {
				if !strings.Contains(err.Error(), tc.expectErr) {
					t.Fatalf("expected err: %q, to contain: %q", err, tc.expectErr)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected err: %s", err)
				}
				is := ms.(*statsite.StatsiteSink)
				if is == nil {
					t.Fatal("expected sink not to be nil")
				}
			}
		})
	}
}

func TestNewInmemSinkFromURL(t *testing.T) {
	for _, tc := range []struct {
		desc           string
		input          string
		expectErr      string
		expectInterval time.Duration
		expectRetain   time.Duration
	}{
		{
			desc:  "interval and duration are set via query params",
			input: "inmem://?interval=11s&retain=22s",
		},
		{
			desc:      "interval is required",
			input:     "inmem://?retain=22s",
			expectErr: "Bad 'interval' param",
		},
		{
			desc:      "interval must be a duration",
			input:     "inmem://?retain=30s&interval=HIYA",
			expectErr: "Bad 'interval' param",
		},
		{
			desc:      "retain is required",
			input:     "inmem://?interval=30s",
			expectErr: "Bad 'retain' param",
		},
		{
			desc:      "retain must be a valid duration",
			input:     "inmem://?interval=30s&retain=HELLO",
			expectErr: "Bad 'retain' param",
		},
	} {
		t.Run(tc.desc, func(t *testing.T) {
			u, err := url.Parse(tc.input)
			if err != nil {
				t.Fatalf("error parsing URL: %s", err)
			}
			ms, err := NewInmemSinkFromURL(u)
			if tc.expectErr != "" {
				if !strings.Contains(err.Error(), tc.expectErr) {
					t.Fatalf("expected err: %q, to contain: %q", err, tc.expectErr)
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected err: %s", err)
				}
				is := ms.(*inmem.InmemSink)
				if is == nil {
					t.Fatal("expected sink not to be nil")
				}
			}
		})
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
				is := ms.(*statsd.StatsdSink)
				if is == nil {
					t.Fatal("expected sink not to be nil")
				}
			}
		})
	}
}
