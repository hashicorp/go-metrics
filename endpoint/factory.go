package factory

import (
	"fmt"
	"net/url"
	"time"

	"github.com/hugoluchessi/go-metrics"
	"github.com/hugoluchessi/go-metrics/providers/inmem"
	"github.com/hugoluchessi/go-metrics/providers/statsd"
	"github.com/hugoluchessi/go-metrics/providers/statsite"
)

// sinkURLFactoryFunc is an generic interface around the *SinkFromURL() function provided
// by each sink type
type sinkURLFactoryFunc func(*url.URL) (metrics.Sinker, error)

// sinkRegistry supports the generic NewSink function by mapping URL
// schemes to metric sink factory functions
var sinkRegistry = map[string]sinkURLFactoryFunc{
	"statsd":   newStatsdSinkFromURL,
	"statsite": newStatsiteSinkFromURL,
	"inmem":    newInmemSinkFromURL,
}

// NewSinkFromURL allows a generic URL input to configure any of the
// supported sinks. The scheme of the URL identifies the type of the sink, the
// and query parameters are used to set options.
//
// "statsd://" - Initializes a StatsdSink. The host and port are passed through
// as the "addr" of the sink
//
// "statsite://" - Initializes a StatsiteSink. The host and port become the
// "addr" of the sink
//
// "inmem://" - Initializes an InmemSink. The host and port are ignored. The
// "interval" and "duration" query parameters must be specified with valid
// durations, see NewInmemSink for details.
func NewSinkFromURL(urlStr string) (metrics.Sinker, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	sinkURLFactoryFunc := sinkRegistry[u.Scheme]
	if sinkURLFactoryFunc == nil {
		return nil, fmt.Errorf(
			"cannot create metric sink, unrecognized sink name: %q", u.Scheme)
	}

	return sinkURLFactoryFunc(u)
}

// NewInmemSinkFromURL creates an InmemSink from a URL. It is used
// (and tested) from NewSinkFromURL.
func newInmemSinkFromURL(u *url.URL) (metrics.Sinker, error) {
	params := u.Query()

	interval, err := time.ParseDuration(params.Get("interval"))
	if err != nil {
		return nil, fmt.Errorf("Bad 'interval' param: %s", err)
	}

	retain, err := time.ParseDuration(params.Get("retain"))
	if err != nil {
		return nil, fmt.Errorf("Bad 'retain' param: %s", err)
	}

	return inmem.NewSink(interval, retain), nil
}

// NewStatsiteSinkFromURL creates an StatsiteSink from a URL. It is used
// (and tested) from NewSinkFromURL.
func newStatsiteSinkFromURL(u *url.URL) (metrics.Sinker, error) {
	return statsite.NewSink(u.Host)
}

// NewStatsdSinkFromURL creates an StatsdSink from a URL. It is used
// (and tested) from NewSinkFromURL.
func newStatsdSinkFromURL(u *url.URL) (metrics.Sinker, error) {
	return statsd.NewSink(u.Host)
}
