package datadog

import (
	"fmt"
	"strings"

	"github.com/DataDog/datadog-go/statsd"
	"github.com/hugoluchessi/go-metrics"
)

// Sink provides a MetricSink that can be used
// with a dogstatsd server. It utilizes the Dogstatsd client at github.com/DataDog/datadog-go/statsd
type Sink struct {
	client            *statsd.Client
	hostName          string
	propagateHostname bool
}

// NewSink is used to create a new Sink with sane defaults
func NewSink(addr string, hostName string) (*Sink, error) {
	client, err := statsd.New(addr)
	if err != nil {
		return nil, err
	}
	sink := &Sink{
		client:            client,
		hostName:          hostName,
		propagateHostname: false,
	}
	return sink, nil
}

// SetTags sets common tags on the Dogstatsd Client that will be sent
// along with all dogstatsd packets.
// Ref: http://docs.datadoghq.com/guides/dogstatsd/#tags
func (s *Sink) SetTags(tags []string) {
	s.client.Tags = tags
}

// EnableHostnamePropagation forces a Dogstatsd `host` tag with the value specified by `s.HostName`
// Since the go-metrics package has its own mechanism for attaching a hostname to metrics,
// setting the `propagateHostname` flag ensures that `s.HostName` overrides the host tag naively set by the DogStatsd server
func (s *Sink) EnableHostNamePropagation() {
	s.propagateHostname = true
}

func (s *Sink) flattenKey(parts []string) string {
	joined := strings.Join(parts, ".")
	return strings.Map(sanitize, joined)
}

func sanitize(r rune) rune {
	switch r {
	case ':':
		fallthrough
	case ' ':
		return '_'
	default:
		return r
	}
}

func (s *Sink) parseKey(key []string) ([]string, []metrics.Label) {
	// Since DogStatsd supports dimensionality via tags on metric keys, this sink's approach is to splice the hostname out of the key in favor of a `host` tag
	// The `host` tag is either forced here, or set downstream by the DogStatsd server

	var labels []metrics.Label
	hostName := s.hostName

	// Splice the hostname out of the key
	for i, el := range key {
		if el == hostName {
			key = append(key[:i], key[i+1:]...)
			break
		}
	}

	if s.propagateHostname {
		labels = append(labels, metrics.Label{"host", hostName})
	}
	return key, labels
}

// Implementation of methods in the MetricSink interface

func (s *Sink) SetGauge(key []string, val float32) {
	s.SetGaugeWithLabels(key, val, nil)
}

func (s *Sink) IncrCounter(key []string, val float32) {
	s.IncrCounterWithLabels(key, val, nil)
}

// EmitKey is not implemented since DogStatsd does not provide a metric type that holds an
// arbitrary number of values
func (s *Sink) EmitKey(key []string, val float32) {
}

func (s *Sink) AddSample(key []string, val float32) {
	s.AddSampleWithLabels(key, val, nil)
}

// The following ...WithLabels methods correspond to Datadog's Tag extension to Statsd.
// http://docs.datadoghq.com/guides/dogstatsd/#tags
func (s *Sink) SetGaugeWithLabels(key []string, val float32, labels []metrics.Label) {
	flatKey, tags := s.getFlatkeyAndCombinedLabels(key, labels)
	rate := 1.0
	s.client.Gauge(flatKey, float64(val), tags, rate)
}

func (s *Sink) IncrCounterWithLabels(key []string, val float32, labels []metrics.Label) {
	flatKey, tags := s.getFlatkeyAndCombinedLabels(key, labels)
	rate := 1.0
	s.client.Count(flatKey, int64(val), tags, rate)
}

func (s *Sink) AddSampleWithLabels(key []string, val float32, labels []metrics.Label) {
	flatKey, tags := s.getFlatkeyAndCombinedLabels(key, labels)
	rate := 1.0
	s.client.TimeInMilliseconds(flatKey, float64(val), tags, rate)
}

func (s *Sink) getFlatkeyAndCombinedLabels(key []string, labels []metrics.Label) (string, []string) {
	key, parsedLabels := s.parseKey(key)
	flatKey := s.flattenKey(key)
	labels = append(labels, parsedLabels...)

	var tags []string
	for _, label := range labels {
		label.Name = strings.Map(sanitize, label.Name)
		label.Value = strings.Map(sanitize, label.Value)
		if label.Value != "" {
			tags = append(tags, fmt.Sprintf("%s:%s", label.Name, label.Value))
		} else {
			tags = append(tags, label.Name)
		}
	}

	return flatKey, tags
}
