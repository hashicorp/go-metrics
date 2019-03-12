// +build !windows

package inmem

import (
	"syscall"
)

const (
	// DefaultSignal is used with DefaultInmemSignal
	defaultSignal = syscall.SIGUSR1
)
