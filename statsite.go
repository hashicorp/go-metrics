package metrics

import (
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// StatsiteSink provides a MetricSink that can be used with a
// statsite or statsd metrics server
type StatsiteSink struct {
	addr        string
	metricQueue chan string
}

// NewStatsiteSink is used to create a new StatsiteSink
func NewStatsiteSink(addr string) (*StatsiteSink, error) {
	s := &StatsiteSink{addr, make(chan string, 4096)}
	go s.flushMetrics()
	return s, nil
}

// Close is used to stop flushing to statsite
func (s *StatsiteSink) Shutdown() {
	close(s.metricQueue)
}

func (s *StatsiteSink) SetGauge(key []string, val float32) {
	flatKey := s.flattenKey(key)
	s.pushMetric(fmt.Sprintf("%s:%f|g\n", flatKey, val))
}

func (s *StatsiteSink) EmitKey(key []string, val float32) {
	flatKey := s.flattenKey(key)
	s.pushMetric(fmt.Sprintf("%s:%f|kv\n", flatKey, val))
}

func (s *StatsiteSink) IncrCounter(key []string, val float32) {
	flatKey := s.flattenKey(key)
	s.pushMetric(fmt.Sprintf("%s:%f|c\n", flatKey, val))
}

func (s *StatsiteSink) AddSample(key []string, val float32) {
	flatKey := s.flattenKey(key)
	s.pushMetric(fmt.Sprintf("%s:%f|ms\n", flatKey, val))
}

// Flattens the key for formatting, removes spaces
func (s *StatsiteSink) flattenKey(parts []string) string {
	joined := strings.Join(parts, ".")
	return strings.Replace(joined, " ", "_", -1)
}

// Does a non-blocking push to the metrics queue
func (s *StatsiteSink) pushMetric(m string) {
	select {
	case s.metricQueue <- m:
	default:
	}
}

// Flushes metrics
func (s *StatsiteSink) flushMetrics() {
	var sock net.Conn
	var err error
	var wait <-chan time.Time

CONNECT:
	// Attempt to connect
	sock, err = net.Dial("tcp", s.addr)
	if err != nil {
		log.Printf("[ERR] Error connecting to statsite! Err: %s", err)
		goto WAIT
	}

	for {
		// Get a metric from the queue
		metric, ok := <-s.metricQueue
		if !ok {
			goto QUIT
		}

		// Try to send to statsite
		_, err := sock.Write([]byte(metric))
		if err != nil {
			log.Printf("[ERR] Error writing to statsite! Err: %s", err)
			goto WAIT
		}
	}

WAIT:
	// Wait for a while
	wait = time.After(time.Duration(5) * time.Second)
	for {
		select {
		// Dequeue the messages to avoid backlog
		case _, ok := <-s.metricQueue:
			if !ok {
				goto QUIT
			}
		case <-wait:
			goto CONNECT
		}
	}
QUIT:
	s.metricQueue = nil
}
