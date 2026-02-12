package types

import (
	"time"
)

// Signal captures a strategy decision.
type Event struct {
	ID         string
	Kind       string // ( BUY, SELL )
	Type       string // ( ENTRY, EXIT )
	EntryPrice float64
	Timestamp  time.Time
	Reason     string // Exit reason: PROFIT_TARGET, STOP_LOSS, TRAILING_STOP, SIGNAL
	PeakProfit float64
	PeakLoss   float64
}

type LogEvent struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}
