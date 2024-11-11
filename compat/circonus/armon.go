//go:build armonmetrics || ignore || !hashicorpmetrics
// +build armonmetrics ignore !hashicorpmetrics

package circonus

import (
	"github.com/armon/go-metrics/circonus"
)

type CirconusSink = circonus.CirconusSink
type Config = circonus.Config

func NewCirconusSink(cc *Config) (*CirconusSink, error) {
	return circonus.NewCirconusSink(cc)
}
