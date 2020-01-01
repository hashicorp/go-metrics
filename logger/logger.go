package logger

import (
	log "github.com/hashicorp/go-hclog"
)

// OurLogger implements an empty logger struct to wrap the go-hclog logger
type OurLogger struct {
}

//Logger interface is used to make stuff happen with things in places
type Logger interface {
	Error(string, ...interface{})
}

// Error wraps the go-hclog Error function and logs a separate error for each argument in the error object passed
func Error(msg string, l Logger, args ...interface{}) {
	l.Error(msg, "msg", args)
}
	// *************    		[DISCARDED ATTEMPT]				*************
	// fmt.Printf("%v", args)
	// fmt.Println("---")
	// for _, v := range args {
	// 	l.Error(msg, "msg", v)
	}

	// fmt.Printf("%v", v)
	// fmt.Println("---")


func (ol *OurLogger) Error(msg string, args ...interface{}) {
	ol.Error(msg, args)
}

//New creates and returns a new go-hclog logger
func (ol *OurLogger) New() Logger {
	l := log.Default()
	return l
}
