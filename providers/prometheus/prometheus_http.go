package prometheus

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HTTPHandlerFor creates a collector handler for http
func HTTPHandlerFor(s *Sink) http.Handler {
	return promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{})
}
