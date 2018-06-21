package prometheus

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus"
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
