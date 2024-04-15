module github.com/hashicorp/go-metrics

go 1.12

require (
	github.com/DataDog/datadog-go v4.8.3+incompatible
	github.com/circonus-labs/circonus-gometrics v2.3.1+incompatible
	github.com/golang/protobuf v1.5.4
	github.com/hashicorp/go-immutable-radix v1.3.1
	github.com/pascaldekloe/goe v0.1.1
	github.com/prometheus/client_golang v1.19.1
	github.com/prometheus/client_model v0.6.1
	github.com/prometheus/common v0.55.0
)

require (
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cespare/xxhash/v2 v2.2.0 // indirect
	github.com/circonus-labs/circonusllhist v0.1.3 // indirect
	github.com/hashicorp/go-cleanhttp v0.5.0 // indirect
	github.com/hashicorp/go-retryablehttp v0.5.3 // indirect
	github.com/hashicorp/golang-lru v0.5.1 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	github.com/tv42/httpunix v0.0.0-20150427012821-b75d8614f926 // indirect
	golang.org/x/sys v0.21.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

// Introduced undocumented breaking change to metrics sink interface
retract v0.3.11
