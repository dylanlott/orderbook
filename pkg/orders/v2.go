package orders

import (
	"context"
	"log"
	"time"
)

func StateMonitor() chan OrderState {
	updates := make(chan OrderState)
	orderStatus := make(map[string]*LimitOrder)
	ticker := time.NewTicker(1 * time.Second)

	go func() {
		for {
			select {
			case s := <-updates:
				log.Printf("received state update: %+v", s)
				orderStatus[s.Order.ID] = s.Order
			case <-ticker.C:
				logState(orderStatus)
			}
		}
	}()

	return updates
}

func logState(orders map[string]*LimitOrder) {
	log.Printf("%+v\n", orders)
}

// LimitOrder represents an Order in our orderbook
type LimitOrder struct {
	ID string
	// Holds a string identifier to the Owner of the Order.
	Owner string
	// Strategy is a blocking function that returns when the order is completed.
	Strategy func(ctx context.Context) error
	// Holds any errors that occurred during processing
	Err error
}

// OrderState holds the current state of the orderbook.
type OrderState struct {
	Order *LimitOrder
	Err   error
}

func Worker(in <-chan *LimitOrder, out chan<- *LimitOrder, status chan<- OrderState) {
	for o := range in {
		log.Printf("received order %+v", o)

		go func(order *LimitOrder) {
			err := order.Strategy(context.Background())
			status <- OrderState{
				Order: order,
				Err:   err,
			}
			out <- order
		}(o)
	}
}
