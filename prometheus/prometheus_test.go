package prometheus

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	dto "github.com/prometheus/client_model/go"

	"github.com/armon/go-metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/expfmt"
)

const (
	TestHostname = "test_hostname"
)

func TestNewPrometheusSinkFrom(t *testing.T) {
	reg := prometheus.NewRegistry()

	sink, err := NewPrometheusSinkFrom(PrometheusOpts{
		Registerer: reg,
	})

	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}

	//check if register has a sink by unregistering it.
	ok := reg.Unregister(sink)
	if !ok {
		t.Fatalf("Unregister(sink) = false, want true")
	}
}

func TestNewPrometheusSink(t *testing.T) {
	sink, err := NewPrometheusSink()
	if err != nil {
		t.Fatalf("err = %v, want nil", err)
	}

	//check if register has a sink by unregistering it.
	ok := prometheus.Unregister(sink)
	if !ok {
		t.Fatalf("Unregister(sink) = false, want true")
	}
}

func TestDefinitions(t *testing.T) {
	gaugeDef := GaugeDefinition{
		Name:        []string{"my", "test", "gauge"},
		Help:        "A gauge for testing? How helpful!",
	}
	summaryDef := SummaryDefinition{
		Name:        []string{"my", "test", "summary"},
		Help:        "A summary for testing? How helpful!",
	}
	counterDef := CounterDefinition{
		Name:        []string{"my", "test", "summary"},
		Help:        "A counter for testing? How helpful!",
	}

	// PrometheusSink config w/ definitions for each metric type
	cfg := PrometheusOpts{
		Expiration:         5 * time.Second,
		GaugeDefinitions:   append([]GaugeDefinition{}, gaugeDef),
		SummaryDefinitions: append([]SummaryDefinition{}, summaryDef),
		CounterDefinitions: append([]CounterDefinition{}, counterDef),
	}
	sink, err := NewPrometheusSinkFrom(cfg)
	if err != nil {
		t.Fatalf("err = #{err}, want nil")
	}

	// We can't just len(x) where x is a sync.Map, so we range over the single item and assert the name in our metric
	// definition matches the key we have for the map entry. Should fail if any metrics exist that aren't defined, or if
	// the defined metrics don't exist.
	sink.gauges.Range(func(key, value interface{}) bool {
		name, _ := flattenKey(gaugeDef.Name, gaugeDef.ConstLabels)
		if name != key {
			t.Fatalf("expected my_test_gauge, got #{name}")
		}
		return true
	})
	sink.summaries.Range(func(key, value interface{}) bool {
		name, _ := flattenKey(summaryDef.Name, summaryDef.ConstLabels)
		fmt.Printf("k: %+v, v: %+v", key, value)
		if name != key {
			t.Fatalf("expected my_test_summary, got #{name}")
		}
		return true
	})
	sink.counters.Range(func(key, value interface{}) bool {
		name, _ := flattenKey(counterDef.Name, counterDef.ConstLabels)
		if name != key {
			t.Fatalf("expected my_test_counter, got #{name}")
		}
		return true
	})
}

func MockGetHostname() string {
	return TestHostname
}

func fakeServer(q chan string) *httptest.Server {
	handler := func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(202)
		w.Header().Set("Content-Type", "application/json")
		defer r.Body.Close()
		dec := expfmt.NewDecoder(r.Body, expfmt.FmtProtoDelim)
		m := &dto.MetricFamily{}
		dec.Decode(m)
		expectedm := &dto.MetricFamily{
			Name: proto.String("default_one_two"),
			Help: proto.String("default_one_two"),
			Type: dto.MetricType_GAUGE.Enum(),
			Metric: []*dto.Metric{
				&dto.Metric{
					Label: []*dto.LabelPair{
						&dto.LabelPair{
							Name:  proto.String("host"),
							Value: proto.String(MockGetHostname()),
						},
					},
					Gauge: &dto.Gauge{
						Value: proto.Float64(42),
					},
				},
			},
		}
		if !reflect.DeepEqual(m, expectedm) {
			msg := fmt.Sprintf("Unexpected samples extracted, got: %+v, want: %+v", m, expectedm)
			q <- errors.New(msg).Error()
		} else {
			q <- "ok"
		}
	}

	return httptest.NewServer(http.HandlerFunc(handler))
}

func TestSetGauge(t *testing.T) {
	q := make(chan string)
	server := fakeServer(q)
	defer server.Close()
	u, err := url.Parse(server.URL)
	if err != nil {
		log.Fatal(err)
	}
	host := u.Hostname() + ":" + u.Port()
	sink, err := NewPrometheusPushSink(host, time.Second, "pushtest")
	metricsConf := metrics.DefaultConfig("default")
	metricsConf.HostName = MockGetHostname()
	metricsConf.EnableHostnameLabel = true
	metrics.NewGlobal(metricsConf, sink)
	metrics.SetGauge([]string{"one", "two"}, 42)
	response := <-q
	if response != "ok" {
		t.Fatal(response)
	}
}
