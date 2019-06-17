package prometheus

import (
	"testing"

	"github.com/armon/go-metrics"
)

func TestImplementsMetricSink(t *testing.T) {
	var _ metrics.MetricSink = &PrometheusSink{}
}
