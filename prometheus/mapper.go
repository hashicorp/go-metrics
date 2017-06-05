package prometheus

import (
	"strings"

	"github.com/prometheus/client_golang/prometheus"
)

type PrometheusMapper interface {
	MapMetric(parts []string) (name string, labels prometheus.Labels, present bool)
}

type simpleMapper struct {
}

func (m *simpleMapper) MapMetric(parts []string) (string, prometheus.Labels, bool) {
	joined := strings.Join(parts, "_")
	joined = strings.Replace(joined, " ", "_", -1)
	joined = strings.Replace(joined, ".", "_", -1)
	joined = strings.Replace(joined, "-", "_", -1)
	joined = strings.Replace(joined, "=", "_", -1)

	return joined, nil, true
}
