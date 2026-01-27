package execution

import (
	"log"
	"time"

	"hft/pkg/types"
)

// Executor submits and manages orders at the broker.
type Executor struct {
	mode string
}

// CurrentHFT holds the last connected HFT reference (global for quick access).
var CurrentHFT *types.HFT

// NewExecutor constructs an executor configured for the provided mode.
func NewExecutor(mode string) *Executor {
	return &Executor{mode: mode}
}

// Run starts the executor loop. Placeholder for real order logic.
func (e *Executor) Run() {
	log.Printf("executor routine started (mode=%s)", e.mode)
	// connect with broker
	hftRef := &types.HFT{
		Broker: types.Broker{Name: "DummyBroker"},
		User:   types.User{Name: "DummyUser"},
		Time:   time.Now(),
	}
	CurrentHFT = hftRef
	log.Printf("connected to dummy broker; hft time ref=%s", hftRef.Time.Format(time.RFC3339))
	// TODO: implement live/backtest execution behavior.
	select {} // block to simulate a long-lived executor loop
}
