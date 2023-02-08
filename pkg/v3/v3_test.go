package v3

import (
	"testing"
)

var numWorkers = 2

var testOrders []*order = []*order{
	&order{
		ownerID: 111,
		price:   10,
		side:    buyside,
		open:    10,
		filled:  0,
	},
	&order{
		ownerID: 222,
		price:   10,
		side:    sellside,
		open:    10,
		filled:  0,
	},
	&order{
		ownerID: 333,
		price:   10,
		side:    sellside,
		open:    10,
		filled:  0,
	},
}

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

	// push test orders into queue
	for _, v := range testOrders {
		pending <- v
	}

	for c := range complete {
		t.Logf("order completed %+v", c)
	}
}
