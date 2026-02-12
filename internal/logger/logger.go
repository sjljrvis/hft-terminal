package logger

import (
	"hft/pkg/types"
	"time"
)

// Logger is a placeholder for structured logging implementation.
func Log(logEvents chan *types.LogEvent, message string) {
	logEvents <- &types.LogEvent{
		Message:   message,
		Timestamp: time.Now(),
	}
}
