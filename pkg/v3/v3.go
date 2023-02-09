package v3

import (
	"fmt"
	"log"
	"sync"
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
	ID      string
	ownerID uint64 // relates to the user ID of the Account
	price   Price  // unit price of the order
	side    bool   // true if buy, sell if false
	open    uint64 // open, unfilled quantity
	filled  uint64 // filled quantity
}

// orderbook holds the set of orders that are active. if an order is
type orderbook struct {
	sync.Mutex

	buy  map[Price][]*order // TODO: make buy and sell use FIFO queue generics
	sell map[Price][]*order
}

var (
	statusCanceled string = "canceled"
	statusPending  string = "pending"
	statusFilled   string = "filled"
)

// Worker wraps an in and output channel around the orderbook.
// * Updates are published on the status channel.
// * Filled and completed orders are pushed on out.
// * Errored, cancelled, or otherwise incompleted but finished orders are only
// viewed through the status updates.
func Worker(in <-chan *order, out chan<- *order, status chan<- OrderUpdate, book *orderbook) {
	for ord := range in {
		// push the order into our books
		err := book.push(ord)
		if err != nil {
			// report on status if errored.
			status <- OrderUpdate{
				Order:  ord,
				Status: statusCanceled,
				Err:    err,
			}
		}

		// go fill the order and push on out when it's' filled.
		go book.fill(ord, out, status)
	}
}

// StateMonitor returns a channel that emits order status updates at a given interval.
func StateMonitor(interval time.Duration) chan OrderUpdate {
	updates := make(chan OrderUpdate)
	orderStatus := make(map[string]*order)
	ticker := time.NewTicker(interval)

	go func() {
		for {
			select {
			case orderUpdated := <-updates:
				orderStatus[orderUpdated.Order.ID] = orderUpdated.Order
			case <-ticker.C:
				log.Printf("status updates: %+v", orderStatus)
			}
		}
	}()

	return updates
}

// push inserts an order into the books and returns an error if order
// is invalid or if an error occurred during push.
func (o *orderbook) push(ord *order) error {
	// push alters the state of orderbook so we lock it.
	o.Lock()
	defer o.Unlock()

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

// pull takes a given [price] and pulls a [quantity] of next-in-line orders from
// the correct [side] of the books.
// * it mutates the order book so it acquires a lock
func (o *orderbook) pull(price Price, side bool) (*order, error) {
	o.Lock()
	defer o.Unlock()

	if side {
		// handle buy - pull lowest to find best price to buy.
		return o.pullLowest(price, buyside)
	}

	// handle sell - pull highest to find best sell price.
	return o.pullHighest(price, sellside)
}

// fill loops over the book and finds orders.
func (o *orderbook) fill(fillorder *order, out chan<- *order, status chan<- OrderUpdate) {
	// TODO: loop on the order until it's filled
	for {
		if filled(fillorder) {
			// attempt to fill the order
			if _, err := o.attemptFill(fillorder); err != nil {

				// if order is already filled, then log and break
				if err == fmt.Errorf("ErrAlreadyFilled") {
					status <- OrderUpdate{
						Order:  fillorder,
						Status: statusFilled,
					}
					break
				}

				status <- OrderUpdate{
					Order:  fillorder,
					Status: statusPending,
					Err:    err,
				}
			} else {
				if fillorder.filled == fillorder.open {
					break
				}
			}
		} else {
			break
		}
	}

	// push the filled order into output channel
	out <- fillorder
}

// attemptFill attempts to fill the given order and returns the amount it filled or an error.
// this is the only function that actually mutates the book.
// * this level of abstraction is useful because it's where we have determined and vetted
// all of the preconditions and now know that we are for sure about to touch the books.
// thus we can safely grab a lock and know it was necessary and efficient.
func (o *orderbook) attemptFill(fillorder *order) (int64, error) {
	o.Lock()
	defer o.Unlock()

	// TODO: calculate how much we want
	wanted := fillorder.open - fillorder.filled

	if fillorder.side {
		// handle buy side
		pulled, err := o.pullLowest(fillorder.price, fillorder.side)
		if err != nil {

		}

		available := pulled.open - pulled.filled
		if available > wanted {
			// take first, always

			// more are available than we want, fully fill the fillorder
			fillorder.filled += wanted
		}
		if available < wanted {
			// less are available than we wanted, take all available and return
		}
	} else {
		_, err := o.pullHighest(fillorder.price, sellside)
		if err != nil {
			return 0, err
		}

		// handle sell side
		return 0, fmt.Errorf("not impl")
	}

}

func (o *orderbook) pullLowest(price Price, side bool) (*order, error) {
	if side {
		// handle buy list
		list, ok := o.buy[price]
		if !ok {
			if price == 0 {
				return nil, fmt.Errorf("ErrNotFound")
			}
			// recursively search for next lowest price.
			nextLowest := price - 1
			return o.pullLowest(nextLowest, side)
		}
		if len(list) == 0 {
			return nil, fmt.Errorf("ErrNotFound")
		}

		return list[0], nil
	}

	// handle the sell list
	list, ok := o.sell[price]
	if !ok {
		if price == 0 {
			return nil, fmt.Errorf("ErrNotFound")
		}
		nextLowest := price - 1
		return o.pullLowest(nextLowest, side)
	}
	if len(list) == 0 {
		return nil, fmt.Errorf("ErrNotFound")
	}

	return list[0], nil
}

func (o *orderbook) pullHighest(price Price, side bool) (*order, error) {
	return nil, fmt.Errorf("not impl")
}

func filled(fillorder *order) bool {
	return fillorder.filled < fillorder.open
}
