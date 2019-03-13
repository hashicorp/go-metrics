package prometheus

import "github.com/prometheus/client_golang/prometheus/promhttp"

// HttpHandlerFor creates a collector handler for http
func HttpHandlerFor(s *Sink) {
	promhttp.HandlerFor(s.registry, promhttp.HandlerOpts{})
}
