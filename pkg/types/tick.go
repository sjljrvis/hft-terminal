package types

import "time"

// Tick models a market tick (OHLCV).
type Tick struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Symbol    string    `json:"symbol"`
	TF        string    `json:"tf"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Time      int64     `json:"time"`   // epoch seconds
	Volume    float64   `json:"volume"` // could be int64; keep float64 for flexibility
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
