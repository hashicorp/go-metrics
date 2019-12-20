package logger

import (
	log "github.com/hashicorp/go-hclog"
)

// OurLogger implements a logger interface
type OurLogger struct {
	ol Logger
}

//Logger interface is used to make stuff happen with things in places
type Logger interface {
	Error(string, ...interface{})
}

func (ol *OurLogger) Error(msg string, args ...interface{}) {

}

//New returns a new logger
func (ol *OurLogger) New() Logger {
	l := log.Default()
	return l
}
