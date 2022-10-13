package v2

import (
	"log"
	"sync"
	"testing"
)

// number of workers that will process orders
var numWorkers = 2

var testOrders = []Order{
	&LimitOrder{
		id:       "foo",
		side:     BUY,
		price:    100,
		Strategy: LimitFill,
	},
	&LimitOrder{
		id:       "buzz",
		side:     SELL,
		price:    100,
		Strategy: LimitFill,
	},
	&LimitOrder{
		id:       "bar",
		side:     BUY,
		price:    100,
		Strategy: LimitFill,
	},
}

func TestWorker(t *testing.T) {
	// Create our input and output channels.
	pending, complete := make(chan Order), make(chan Order)

	// Launch the StateMonitor.
	status := StateMonitor()

	// Create a fresh orderbook and pass it to Worker
	orderbook := &Orderbook{
		Buy:  &PriceNode{val: 0.0},
		Sell: &PriceNode{val: 0.0},
	}

	for i := 0; i < numWorkers; i++ {
		go Worker(pending, complete, status, orderbook)
	}

	var wg = &sync.WaitGroup{}
	go func() {
		for _, testOrder := range testOrders {
			wg.Add(1)
			pending <- testOrder
		}

		for c := range complete {
			wg.Done()
			log.Printf("order %s completed", c.ID())
		}
	}()

	wg.Wait()
}
