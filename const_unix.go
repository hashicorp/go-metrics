// Copyright IBM Corp. 2013, 2025
// SPDX-License-Identifier: MIT

//go:build !windows && !js
// +build !windows,!js

package metrics

import (
	"syscall"
)

const (
	// DefaultSignal is used with DefaultInmemSignal
	DefaultSignal = syscall.SIGUSR1
)
