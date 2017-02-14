// +build !windows
// +build !appengine

package metrics

import (
	"syscall"
)

const (
	// DefaultSignal is used with DefaultInmemSignal
	DefaultSignal = syscall.SIGUSR1
)
