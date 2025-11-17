// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MIT

//go:build hashicorpmetrics

package circonus

import (
	"github.com/hashicorp/go-metrics/circonus"
)

type CirconusSink = circonus.CirconusSink
type Config = circonus.Config

func NewCirconusSink(cc *Config) (*CirconusSink, error) {
	return circonus.NewCirconusSink(cc)
}
