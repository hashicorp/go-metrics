//go:build armonmetrics || ignore || !hashicorpmetrics
// +build armonmetrics ignore !hashicorpmetrics

package datadog

import (
	"github.com/armon/go-metrics/datadog"
)

type DogStatsdSink = datadog.DogStatsdSink

func NewDogStatsdSink(addr string, hostName string) (*DogStatsdSink, error) {
	return datadog.NewDogStatsdSink(addr, hostName)
}
