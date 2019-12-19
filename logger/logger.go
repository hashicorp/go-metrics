package logger

import (
	log "github.com/sirupsen/logrus"
)

// // Event stores messages to log later, from our standard interface
// type Event struct {
// 	id      int
// 	message string
// }

// StandardLogger enforces specific log message formats
type StandardLogger struct {
	*log.Logger
}

// NewLogger initializes the standard logger
func NewLogger() *StandardLogger {
	var baseLogger = log.New()

	var standardLogger = &StandardLogger{baseLogger}

	return standardLogger
}
