package v3

import (
	"testing"
)

var numWorkers = 2

func TestWorker(t *testing.T) {
	// Create our input and output channels.
	pending, complete := make(chan *order), make(chan *order)

	// Launch the StateMonitor.
	status := StateMonitor()

	// create an orderbook
	o := &orderbook{
		buy: map[Price][]*order{
			0: {},
		},
		sell: map[Price][]*order{
			0: {},
		},
	}

	for i := 0; i < numWorkers; i++ {
		go Worker(pending, complete, status, o)
	}

	for c := range complete {
		t.Logf("order completed %+v", c)
	}
}
