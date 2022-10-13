package v2

import (
	"log"
	"testing"
	"time"
)

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

func TestPoller(t *testing.T) {

	// Create our input and output channels.
	pending, complete := make(chan Order), make(chan Order)

	// Launch the StateMonitor.
	status := StateMonitor()

	// Create a fresh orderbook and pass it to Worker
	orderbook := &Orderbook{
		Buy:  &PriceNode{val: 0.0},
		Sell: &PriceNode{val: 0.0},
	}

	for i := 0; i < 2; i++ {
		go Worker(pending, complete, status, orderbook)
	}

	go func() {
		for _, testOrder := range testOrders {
			pending <- testOrder
		}

		// TODO: Assert against received completed orders
		for c := range complete {
			log.Printf("order %s completed", c.ID())
		}
	}()

	time.Sleep(1 * time.Second)

	orderbook.Buy.Print()
}
