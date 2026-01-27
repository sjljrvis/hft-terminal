package types

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	Name string
}

type Broker struct {
	Name string
}

// HFT groups user, broker, and timing metadata.
type HFT struct {
	User   User      // placeholder for user details
	Broker Broker    // placeholder for broker details
	Time   time.Time // timestamp context
	Status string    // status of the HFT
}

type Position struct {
	ID               string
	Kind             string // ( BUY, SELL )
	Type             string // ( ENTRY, EXIT )
	EntryPrice       float64
	Quantity         float64
	EntryTime        time.Time
	ExitPrice        float64
	ExitTime         time.Time
	Profit           float64
	ProfitPercentage float64
}

func (p *Position) Reset() {
	p.ID = ""
	p.Kind = ""
	p.Type = ""
	p.EntryPrice = 0
	p.Quantity = 0
	p.EntryTime = time.Now()
	p.ExitPrice = 0
	p.ExitTime = time.Now()
	p.Profit = 0
	p.ProfitPercentage = 0
}

func (p *Position) Buy(entryPrice float64, timestamp time.Time) {
	p.Reset()
	p.ID = uuid.New().String()
	p.Kind = "BUY"
	p.Type = "ENTRY"
	p.EntryPrice = entryPrice
	p.EntryTime = timestamp
}

func (p *Position) Sell(exitPrice float64, timestamp time.Time) {
	p.Reset()
	p.ID = uuid.New().String()
	p.Kind = "SELL"
	p.Type = "ENTRY"
	p.ExitPrice = exitPrice
	p.ExitTime = timestamp
	p.Profit = exitPrice - p.EntryPrice
	p.ProfitPercentage = (p.Profit / p.EntryPrice) * 100
}

func (p *Position) Exit(exitPrice float64, timestamp time.Time) {
	p.Type = "EXIT"
	p.ExitPrice = exitPrice
	p.ExitTime = timestamp

	// Calculate profit based on position direction.
	if p.Kind == "BUY" {
		p.Profit = exitPrice - p.EntryPrice
	} else {
		p.Profit = p.EntryPrice - exitPrice
	}

	if p.EntryPrice != 0 {
		p.ProfitPercentage = (p.Profit / p.EntryPrice) * 100
	}
}
