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
}
