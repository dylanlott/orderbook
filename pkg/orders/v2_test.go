package orders

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

var testOrders = []*LimitOrder{
	{
		ID:   "foo",
		Side: BUY,
		Strategy: func(ctx context.Context) error {
			log.Printf("hit strategy")
			return fmt.Errorf("not impl")
		},
	},
	{
		ID:   "buzz",
		Side: SELL,
		Strategy: func(ctx context.Context) error {
			log.Printf("hit strategy")
			return fmt.Errorf("not impl")
		},
	},
	{
		ID:   "bar",
		Side: BUY,
		Strategy: func(ctx context.Context) error {
			log.Printf("hit strategy")
			return fmt.Errorf("not impl")
		},
	},
}

func TestPoller(t *testing.T) {
	// Create our input and output channels.
	pending, complete := make(chan *LimitOrder), make(chan *LimitOrder)

	// Launch the StateMonitor.
	status := StateMonitor()

	// Create a fresh orderbook and pass it to Worker
	orderbook := &Orderbook{
		Buy:  &TreeNodeV2{},
		Sell: &TreeNodeV2{},
	}

	// Launch some Poller goroutines.
	for i := 0; i < 2; i++ {
		go Worker(pending, complete, status, orderbook)
	}

	// Send some Resources to the pending queue.
	go func() {
		for _, testOrder := range testOrders {
			pending <- testOrder
		}

		// TODO: Assert against received completed orders
		for c := range complete {
			log.Printf("order %s completed", c.ID)
		}
	}()

	time.Sleep(3 * time.Second)
}
