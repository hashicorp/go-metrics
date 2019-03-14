package metrics

import (
	"testing"
	"time"
)

func TestMetricsServiceConfig_DefaultConfig(t *testing.T) {
	conf := DefaultConfig("service")
	if conf.ServiceName != "service" {
		t.Fatalf("Bad name")
	}
	if conf.HostName == "" {
		t.Fatalf("missing hostname")
	}
	if !conf.EnableServiceName || !conf.EnableRuntimeMetrics {
		t.Fatalf("expect true")
	}
	if conf.EnableHostName || conf.EnableTypeSufix {
		t.Fatalf("expect false")
	}
	if conf.TimerGranularity != time.Millisecond {
		t.Fatalf("bad granularity")
	}
	if conf.ProfileInterval != 3*time.Second {
		t.Fatalf("bad interval")
	}
}
