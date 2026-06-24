// Copyright IBM Corp. 2013, 2026
// SPDX-License-Identifier: MIT

package otel

import "time"

const (
	// ProtocolGRPC uses OTLP over gRPC (default).
	ProtocolGRPC = "grpc"
	// ProtocolHTTP uses OTLP over HTTP with protobuf encoding.
	ProtocolHTTP = "http"

	// DefaultPushInterval is the default interval for pushing metrics to the OTLP endpoint.
	DefaultPushInterval = 1 * time.Minute

	// DefaultShutdownTimeout is the default timeout for graceful shutdown.
	DefaultShutdownTimeout = 30 * time.Second

	// DefaultExpHistogramMaxSize is the maximum number of buckets in an
	// exponential histogram. Higher values increase resolution at the cost of memory.
	DefaultExpHistogramMaxSize = 160

	// DefaultExpHistogramMaxScale is the maximum scale factor for exponential
	// histograms. Higher values allow finer-grained bucket boundaries.
	DefaultExpHistogramMaxScale = 20
)

// OTELSinkOpts configures the OTEL metrics sink.
type OTELSinkOpts struct {
	// Endpoint is the OTLP endpoint to push metrics to (e.g., "localhost:4317"
	// for gRPC, "localhost:4318" for HTTP). TLS is used by default; set
	// Insecure to true for plain-text connections. Required.
	Endpoint string

	// Protocol selects the OTLP transport: "grpc" (default) or "http".
	Protocol string

	// URLPath overrides the default URL path for HTTP transport.
	// Only used when Protocol is "http". Defaults to "/v1/metrics".
	URLPath string

	// Insecure disables TLS when connecting to the OTLP endpoint.
	Insecure bool

	// Headers are optional headers sent with each export (e.g., for authentication).
	Headers map[string]string

	// PushInterval is how often to push metrics.
	PushInterval time.Duration

	// ResourceAttributes are resource attributes to include (e.g., "service.name", "host.name").
	ResourceAttributes map[string]string

	// UseExplicitHistograms switches from exponential to explicit bucket
	// histograms for AddSample metrics. These settings apply to all histograms.
	UseExplicitHistograms bool

	// HistogramBuckets are the explicit bucket boundaries when
	// UseExplicitHistograms is true. If empty, DefaultHistogramBuckets() is used.
	HistogramBuckets []float64

	// ExpHistogramMaxSize is the maximum number of buckets in exponential
	// histograms. Only used when UseExplicitHistograms is false.
	ExpHistogramMaxSize int32

	// ExpHistogramMaxScale is the maximum scale factor for exponential
	// histograms. Only used when UseExplicitHistograms is false.
	ExpHistogramMaxScale int32

	// TLSCertFile is the path to the client certificate for mTLS authentication.
	// Must be set together with TLSKeyFile. Ignored when Insecure is true.
	TLSCertFile string

	// TLSKeyFile is the path to the client private key for mTLS authentication.
	// Must be set together with TLSCertFile. Ignored when Insecure is true.
	TLSKeyFile string

	// TLSCAFile is the path to the CA certificate for verifying the server.
	// If empty, the system root CAs are used. Ignored when Insecure is true.
	TLSCAFile string

	// CardinalityLimit is the maximum number of unique attribute combinations
	// per instrument. When the limit is reached, new attribute sets are dropped
	// and aggregated into an overflow bucket. Zero (default) means no limit.
	// The OTEL spec recommends 2000 for most workloads.
	CardinalityLimit int

	// GzipCompression enables gzip compression for OTLP exports.
	GzipCompression bool

	// DeltaTemporality switches counters and histograms from cumulative
	// (default) to delta temporality. Cumulative reports the total since
	// process start; delta reports the change since the last export.
	DeltaTemporality bool

	// ShutdownTimeout is the maximum time to wait for graceful shutdown.
	ShutdownTimeout time.Duration
}

// defaultHistogramBuckets are the default explicit bucket boundaries,
// suitable for latency measurements in seconds.
var defaultHistogramBuckets = []float64{
	0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10,
}

// DefaultHistogramBuckets returns a copy of the default explicit bucket
// boundaries, suitable for latency measurements in seconds.
func DefaultHistogramBuckets() []float64 {
	return append([]float64{}, defaultHistogramBuckets...)
}
