package orders

import (
	"context"
	"fmt"
	"log"
	"testing"
	"time"
)

var testOrders = []*LimitOrder{
	{
		ID: "foo",
		Strategy: func(ctx context.Context) error {
			log.Printf("hit strategy")
			return fmt.Errorf("not impl")
		},
	},
	{
		ID: "buzz",
		Strategy: func(ctx context.Context) error {
			log.Printf("hit strategy")
			return fmt.Errorf("not impl")
		},
	},
	{
		ID: "bar",
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

	// Launch some Poller goroutines.
	for i := 0; i < 2; i++ {
		go Worker(pending, complete, status)
	}

	// Send some Resources to the pending queue.
	go func() {
		for _, testOrder := range testOrders {
			pending <- testOrder
		}
	}()
	time.Sleep(3 * time.Second)
}
