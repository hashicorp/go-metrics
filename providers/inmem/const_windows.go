// +build windows

package inmem

import (
	"syscall"
)

const (
	// DefaultSignal is used with DefaultInmemSignal
	// Windows has no SIGUSR1, use SIGBREAK
	defaultSignal = syscall.Signal(21)
)
