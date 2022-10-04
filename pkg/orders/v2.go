package orders

import (
	"context"
	"log"
	"time"
)

func StateMonitor() chan OrderState {
	updates := make(chan OrderState)
	orderStatus := make(map[string]LimitOrder)
	ticker := time.NewTicker(1 * time.Second)

	go func() {
		for {
			select {
			case s := <-updates:
				log.Printf("received state update: %+v", s)
			case <-ticker.C:
				logState(orderStatus)
			}
		}
	}()

	return updates
}

func logState(orders map[string]LimitOrder) {
	log.Printf("%+v\n", orders)
}

// LimitOrder represents an Order in our orderbook
type LimitOrder struct {
	ID       string
	Strategy func(ctx context.Context) error
}

// OrderState holds the current state of the orderbook.
type OrderState struct {
	Order *LimitOrder
	Err   error
}

func Worker(in <-chan *LimitOrder, out chan<- *LimitOrder, status chan<- OrderState) {
	for o := range in {
		log.Printf("received order %+v", o)
		err := o.Strategy(context.Background())
		status <- OrderState{
			Order: o,
			Err:   err,
		}
	}
}
