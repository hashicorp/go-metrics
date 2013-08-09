package metrics

import (
	"fmt"
	"os"
	"time"
)

// Config is used to configure metrics settings
type Config struct {
	ServiceName          string        // Prefixed with keys to seperate services
	EnableHostname       bool          // Enable prefixing gauge values with hostname
	EnableRuntimeMetrics bool          // Enables profiling of runtime metrics (GC, Goroutines, Memory)
	EnableTypePrefix     bool          // Prefixes key with a type ("counter", "gauge", "timer")
	TimerGranularity     time.Duration // Granularity of timers.
	ProfileInterval      time.Duration // Interval to profile runtime metrics
}

// Metrics represents an instance of a metrics sink that can
// be used to emit
type Metrics struct {
	Config
	hostName  string
	lastNumGC uint32
	sink      MetricSink
}

// Shared global metrics instance
var globalMetrics *Metrics

func init() {
	// Initialize to a blackhole sink to avoid errors
	globalMetrics = &Metrics{sink: &BlackholeSink{}}
}

// DefaultConfig provides a sane default configuration
func DefaultConfig(serviceName string) *Config {
	return &Config{
		serviceName,      // Use client provided service
		true,             // Enable hostname prefix
		true,             // Enable runtime profiling
		true,             // Enable type prefix
		time.Millisecond, // Timers are in milliseconds
		time.Second,      // Poll runtime every second
	}
}

// New is used to create a new instance of Metrics. It takes a
// service name which is prefixed to all keys (unless blank), a
// bool to enableHostname when emiting gauges, and a sink implementation.
func New(conf *Config, sink MetricSink) (*Metrics, error) {
	met := &Metrics{}
	met.Config = *conf

	// Get the hostname
	if conf.EnableHostname {
		hostName, err := os.Hostname()
		if err != nil {
			return nil, fmt.Errorf("Failed to get hostname! Got: %s", err)
		}
		met.hostName = hostName
	}

	// Start the runtime collector
	if conf.EnableRuntimeMetrics {
		go met.collectStats()
	}
	return met, nil
}

// NewGlobal is the same as New, but it assigns the metrics object to be
// used globally as well as returning it.
func NewGlobal(conf *Config, sink MetricSink) (*Metrics, error) {
	metrics, err := New(conf, sink)
	if err != nil {
		globalMetrics = metrics
	}
	return metrics, err
}

// Proxy all the methods to the globalMetrics instance
func SetGauge(key []string, val float32) {
	globalMetrics.SetGauge(key, val)
}

func EmitKey(key []string, val float32) {
	globalMetrics.EmitKey(key, val)
}

func IncrCounter(key []string, val float32) {
	globalMetrics.IncrCounter(key, val)
}

func AddSample(key []string, val float32) {
	globalMetrics.AddSample(key, val)
}

func MeasureSince(key []string, start time.Time) {
	globalMetrics.MeasureSince(key, start)
}
