// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

//go:build hashicorpmetrics
// +build hashicorpmetrics

package datadog

import (
	"github.com/hashicorp/go-metrics/datadog"
)

type DogStatsdSink = datadog.DogStatsdSink

func NewDogStatsdSink(addr string, hostName string) (*DogStatsdSink, error) {
	return datadog.NewDogStatsdSink(addr, hostName)
}
