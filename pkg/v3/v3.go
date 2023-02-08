package v3

import (
	"fmt"
	"log"
	"time"
)

// v3 is going to be a map based implementation wrapped with an in and out channel.
// now that we deeply understand the problem domain, we can tune our application on the third revision.

// OrderUpdate holds the current state of an OrderV2 and
// binds it to a simple state object.
type OrderUpdate struct {
	Order  *order
	Status string
	Err    error
}

// Worker wraps an in and output channel around the orderbook.
// * Updates are published on the status channel.
// * Filled and completed orders are pushed on out.
// * Errored, cancelled, or otherwise incompleted but finished orders are only
// viewed through the status updates.
func Worker(in <-chan *order, out chan<- *order, status chan<- OrderUpdate, book *orderbook) {
	for ord := range in {
		go func(o *order) {
			log.Printf("creating order %+v", o)
			// push into our books
			if err := book.push(ord); err != nil {
				status <- OrderUpdate{
					Order:  o,
					Status: "errored",
					Err:    err,
				}
			}

			// go fill the order
			go book.fill(o, out)
		}(ord)
	}
}

// tickRate determines how often the state monitor publishes its view of the state
var tickRate = 300 * time.Millisecond

// StateMonitor returns a channel that emits order status updates.
func StateMonitor() chan OrderUpdate {
	updates := make(chan OrderUpdate)
	orderStatus := make(map[string]*order)
	ticker := time.NewTicker(tickRate)

	go func() {
		for {
			select {
			case s := <-updates:
				log.Printf("order state updated: %+v", s)
			case <-ticker.C:
				log.Printf("%+v", orderStatus)
			}
		}
	}()

	return updates
}

// Price represents a price in our system which is an untyped 64bit integer
type Price uint64

// Side is the simplest binary way to represent buy and sell side in our system
type Side bool

var (
	buyside  = true
	sellside = false
)

// order holds the information that represents a single order in our system
type order struct {
	ownerID uint64 // relates to the user ID of the Account
	amount  Price  // unit price of the order
	side    bool   // true if buy, sell if false
	open    uint64 // open, unfilled quantity
	filled  uint64 // filled quantity
}

// orderbook holds the set of orders that are active. if an order is
type orderbook struct {
	buy  map[Price][]*order
	sell map[Price][]*order
}

// push pushes an order into the books
func (o *orderbook) push(ord *order) error {
	if ord.side {
		// handle buy side

	}

	return fmt.Errorf("not impl")
}

// pull matches an order at the given price and pulls the appropriate order
func (o *orderbook) pull(price Price, side string) (*order, error) {
	return nil, fmt.Errorf("not impl")
}

// fill loops over the book and finds orders
func (o *orderbook) fill(fillorder *order, out chan<- *order) {
	// TODO: loop on the order until it's filled
	for fillorder.filled < fillorder.open {

	}
	// TODO: send on out when it's filled successfully
	panic("not impl")
}
