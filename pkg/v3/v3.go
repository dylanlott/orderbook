package v3

import (
	"fmt"
	"log"
	"time"
)

// v3 is going to be a map based implementation wrapped with an in and out channel.
// now that we deeply understand the problem domain, we can tune our application on the third revision.

// Price represents a price in our system which is an untyped 64bit integer
type Price uint64

// Side is the simplest binary way to represent buy and sell side in our system
type Side bool

// define buy and sell side values for consistent use.
var (
	buyside  = true
	sellside = false
)

// tickRate determines how often the state monitor publishes its view of the state
var tickRate = 300 * time.Millisecond

// OrderUpdate holds the current state of an OrderV2 and
// binds it to a simple state object.
type OrderUpdate struct {
	Order  *order
	Status string
	Err    error
}

// order holds the information that represents a single order in our system
type order struct {
	ownerID uint64 // relates to the user ID of the Account
	price   Price  // unit price of the order
	side    bool   // true if buy, sell if false
	open    uint64 // open, unfilled quantity
	filled  uint64 // filled quantity
}

// orderbook holds the set of orders that are active. if an order is
type orderbook struct {
	buy  map[Price][]*order // TODO: make buy and sell use FIFO queue generics
	sell map[Price][]*order
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
				log.Printf("book: %+v", orderStatus)
			}
		}
	}()

	return updates
}

// push inserts an order into the books and returns an error if order
// is invalid or if an error occurred during push.
func (o *orderbook) push(ord *order) error {
	// TODO: order validation here
	if ord.side {
		// handle buy side
		if v, ok := o.buy[ord.price]; !ok {
			o.buy[ord.price] = []*order{ord}
		} else {
			v = append(v, ord)
		}
	}

	// handle sell side
	if v, ok := o.sell[ord.price]; !ok {
		o.sell[ord.price] = []*order{ord}
	} else {
		v = append(v, ord)
	}

	return nil
}

// pull matches an order at the given price and pulls the appropriate order
func (o *orderbook) pull(price Price, side bool) (*order, error) {
	if side {
		// handle buy - pull lowest to find best price to buy.
		return o.pullLowest(price, buyside)
	}

	// handle sell - pull highest to find best sell price.
	return o.pullHighest(price, sellside)
}

// pullLowest pulls the cheapest price for the next order in the books
// at or below the given price.
func (o *orderbook) pullLowest(price Price, side bool) (*order, error) {
	return nil, fmt.Errorf("not impl")
}

// pullHighest pulls the highest priced order in the books at or above
// the given price.
func (o *orderbook) pullHighest(price Price, side bool) (*order, error) {
	return nil, fmt.Errorf("not impl")
}

// findLowest finds the lowest priced order on the given side but does not
// pull it from the list. It returns the order's index, the order, and an error value It returns the order's index, the order, and an error value.
func (o *orderbook) findLowest(price Price, side bool) (int64, *order, error) {
	return 0, nil, fmt.Errorf("not impl")
}

// findHighest finds the highest priced order on the given side but does not
// pull it from the list. It returns the order's index, the order, and an error value.
func (o *orderbook) findHighest(price Price, side bool) (int64, *order, error) {
	return 0, nil, fmt.Errorf("not impl")
}

// fill loops over the book and finds orders
func (o *orderbook) fill(fillorder *order, out chan<- *order) {
	// TODO: loop on the order until it's filled
	for {
		if fillorder.filled < fillorder.open {
			log.Printf("orderbook: attempting fill %+v", fillorder)
			if filled, err := o.attemptFill(fillorder); err != nil {
				log.Printf("error filling order: %v", err)
				// TODO: publish order status update with error
				time.Sleep(time.Second * 1)
			} else {
				log.Printf("filled: %+v", filled)
				log.Printf("orderbook: filled %d of %d", fillorder.filled, fillorder.open)
				if fillorder.filled == fillorder.open {
					break
				}
			}
		} else {
			break
		}
	}
	out <- fillorder
}

// attemptFill attempts to fill the given order and returns the amount it filled or an error.
func (o *orderbook) attemptFill(fillorder *order) (int64, error) {

	// TODO: calculate how much we want
	wanted := fillorder.open - fillorder.filled

	if wanted == 0 {
		return 0, fmt.Errorf("ErrAlreadyFilled")
	}

	if fillorder.side {
		// handle buy side
		pulled, err := o.pullLowest(fillorder.price, buyside)
		if err != nil {
			return 0, err
		}
		available := pulled.open - pulled.filled
		if available > wanted {
			// more are available than we want, fully fill the fillorder
		}
		if available < wanted {
			// less are available than we wanted, take all available and return
		}
	}

	_, err := o.pullHighest(fillorder.price, sellside)
	if err != nil {
		return 0, err
	}

	// handle sell side
	return 0, fmt.Errorf("not impl")
}
