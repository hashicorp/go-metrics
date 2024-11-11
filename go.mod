module github.com/hashicorp/go-metrics

go 1.12

require (
	github.com/DataDog/datadog-go v3.2.0+incompatible
	github.com/armon/go-metrics v0.4.1
	github.com/circonus-labs/circonus-gometrics v2.3.1+incompatible
	github.com/golang/protobuf v1.4.3
	github.com/hashicorp/go-immutable-radix v1.0.0
	github.com/pascaldekloe/goe v0.1.0
	github.com/prometheus/client_golang v1.11.1
	github.com/prometheus/client_model v0.2.0
	github.com/prometheus/common v0.26.0
)

// Introduced undocumented breaking change to metrics sink interface
retract v0.3.11
