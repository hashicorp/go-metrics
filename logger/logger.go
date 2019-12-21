package logger

import (
	log "github.com/hashicorp/go-hclog"
)

// OurLogger implements a logger interface
type OurLogger struct {
}

//Logger interface is used to make stuff happen with things in places
type Logger interface {
	Error(string, ...interface{})
}

func Error(msg string, l Logger, args ...interface{}) {
	for a := range args {
		l.Error(msg, a)
	}
}
func (ol *OurLogger) Error(msg string, args ...interface{}) {
	ol.Error(msg, args)
}

//New returns a new logger
func (ol *OurLogger) New() Logger {
	l := Logger(log.Default())
	return l
}
