package execution

import (
	"time"

	"hft/pkg/types"
)

// Broker abstracts a broker adapter.
type Broker interface {
	Connect() *types.HFT
}

// DummyBroker is a placeholder broker that returns a dummy user reference.
type DummyBroker struct {
	Name string
}

// Connect returns a dummy HFT reference with the current timestamp.
func (DummyBroker) Connect() *types.HFT {
	return &types.HFT{
		Time: time.Now(),
	}
}
