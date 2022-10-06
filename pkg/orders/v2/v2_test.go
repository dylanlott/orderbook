package v2

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

const (
	BUY  string = "BUY"
	SELL string = "SELL"
)

// LimitFill fills the given order with a limit strategy. A limit strategy fills orders
// at a hard max for buys and a hard minimum for sells with no time limit.
var LimitFill FillStrategy = func(ctx context.Context, self Order, b *Orderbook) error {
	return fmt.Errorf("not impl")
}

// MarketFill fills orders at the current market price until they're filled.
var MarketFill FillStrategy = func(ctx context.Context, self Order, books *Orderbook) error {
	return fmt.Errorf("not impl")
}

var testOrders = []*LimitOrder{
	{
		id:       "foo",
		side:     BUY,
		price:    100,
		Strategy: LimitFill,
	},
	{
		id:       "buzz",
		side:     SELL,
		price:    100,
		Strategy: LimitFill,
	},
	{
		id:       "bar",
		side:     BUY,
		price:    100,
		Strategy: LimitFill,
	},
}

func TestPoller(t *testing.T) {

	// Create our input and output channels.
	pending, complete := make(chan *LimitOrder), make(chan *LimitOrder)

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

	time.Sleep(3 * time.Second)

	orderbook.Buy.Print()
}
