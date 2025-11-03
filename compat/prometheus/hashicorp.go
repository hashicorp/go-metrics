// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MIT

//go:build hashicorpmetrics
// +build hashicorpmetrics

package prometheus

import (
	"time"

	"github.com/hashicorp/go-metrics/prometheus"
)

var DefaultPrometheusOpts = prometheus.DefaultPrometheusOpts

type CounterDefinition = prometheus.CounterDefinition
type GaugeDefinition = prometheus.GaugeDefinition
type PrometheusOpts = prometheus.PrometheusOpts
type PrometheusPushSink = prometheus.PrometheusPushSink
type PrometheusSink = prometheus.PrometheusSink
type SummaryDefinition = prometheus.SummaryDefinition

func NewPrometheusPushSink(address string, pushInterval time.Duration, name string) (*PrometheusPushSink, error) {
	return prometheus.NewPrometheusPushSink(address, pushInterval, name)
}

func NewPrometheusSink() (*PrometheusSink, error) {
	return prometheus.NewPrometheusSink()
}

func NewPrometheusSinkFrom(opts PrometheusOpts) (*PrometheusSink, error) {
	return prometheus.NewPrometheusSinkFrom(opts)
}
