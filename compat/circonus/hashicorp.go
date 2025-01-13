// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MIT

//go:build hashicorpmetrics
// +build hashicorpmetrics

package circonus

import (
	"github.com/hashicorp/go-metrics/circonus"
)

type CirconusSink = circonus.CirconusSink
type Config = circonus.Config

func NewCirconusSink(cc *Config) (*CirconusSink, error) {
	return circonus.NewCirconusSink(cc)
}
