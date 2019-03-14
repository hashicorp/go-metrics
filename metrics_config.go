package metrics

import (
	"os"
	"time"
)

// MetricServiceConfig is used to configure metrics settings
type MetricServiceConfig struct {
	HostName         string        // Hostname to use. If not provided and EnableHostname, it will be os.Hostname
	ServiceName      string        // Prefixed with keys to separate services
	TimerGranularity time.Duration // Granularity of timers.
	ProfileInterval  time.Duration // Interval to profile runtime metrics

	EnableHostName       bool // Enable prefixing metrics keys with hostname
	EnableServiceName    bool // Enable prefixing metrics keys with service name
	EnableTypeSufix      bool // Sufixes key with a type ("counter", "gauge", "timer")
	EnableRuntimeMetrics bool // Enables profiling of runtime metrics (GC, Goroutines, Memory)
}

// DefaultConfig provides a sane default configuration
func DefaultConfig(serviceName string) *MetricServiceConfig {
	c := &MetricServiceConfig{
		ServiceName:          serviceName, // Use client provided service
		HostName:             "",
		EnableHostName:       false,            // Enable hostname prefix
		EnableServiceName:    true,             // Enable Service name prefix
		EnableRuntimeMetrics: true,             // Enable runtime profiling
		EnableTypeSufix:      false,            // Disable type prefix
		TimerGranularity:     time.Millisecond, // Timers are in milliseconds
		ProfileInterval:      3 * time.Second,  // Poll runtime every 3 seconds
	}

	// Try to get the hostname
	name, _ := os.Hostname()
	c.HostName = name
	return c
}
