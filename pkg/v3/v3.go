package v3

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// v3 is going to be a map based implementation wrapped with an in and out channel.
// now that we deeply understand the problem domain, we can tune our application on the third revision.

// nb: oh how the tables have turned and I already hate this version too.
// see we should be using iterators on lists of sorted orders, and the orderbook
// should just have two of them bound to itself as the core.
// the map structure has the weird property of orders possibly being really far apart in the
// search of even a lightly populated book. This is probably another example of premature optimization.

// Orderbook defines the interface that must be fulfilled by an exchange.
// This is the work in progress and interface may yet change.
type Orderbook interface {
	//  Push inserts the order into our books and immediately starts attempting to fill it.
	Push(o *order) error
	// Fill is called as a goroutine and sends on out when the order is completed.
	// It sends OrderUpdates on status whenever updates to the order occur.
	Fill(o *order, out chan<- *order, status chan<- OrderUpdate)
	// For example, Fill might instead should be a single argument function and out or status
	// channels should be implementation details that we ignore here.
}

// Price represents a price in our system which is an untyped 64bit integer
type Price uint64

// Side is the simplest binary way to represent buy and sell side in our system
type Side bool

// define buy and sell side values for consistent use.
var (
	buyside  = true
	sellside = false
)

// OrderUpdate holds the current state of an OrderV2 and
// binds it to a simple state object.
type OrderUpdate struct {
	Order  *order
	Status string
	Err    error
}

// order holds the information that represents a single order in our system
type order struct {
	sync.RWMutex

	ID      string
	ownerID uint64 // relates to the user ID of the Account
	price   Price  // unit price of the order
	side    bool   // true if buy, sell if false
	open    uint64 // open, unfilled quantity
	filled  uint64 // filled quantity
}

// isFilled returns true if the order is filled.
// it obtains a read-only lock.
func (o *order) isFilled() bool {
	o.RLock()
	defer o.RUnlock()

	return o.filled == o.open
}

// orderbook holds the set of orders that are active. if an order is
type orderbook struct {
	sync.Mutex

	buy  map[Price][]*order // TODO: make buy and sell use FIFO queue generics
	sell map[Price][]*order
}

var (
	statusErrored string = "canceled"
	statusPending string = "pending"
	statusFilled  string = "filled"
)

var (
	// ErrFilled is the error returned by attemptFill when the order is filled
	// and removed from the books.
	ErrFilled error = fmt.Errorf("ErrFilled")
	// ErrNotFound is returned when an order can't be found at a given price.
	ErrNotFound error = fmt.Errorf("ErrNotFound")
)

// Worker wraps an in and output channel around the orderbook.
// * Updates are published on the status channel.
// * Filled and completed orders are pushed on out.
// * Errored, cancelled, or otherwise incompleted but finished orders are only
// viewed through the status updates.
// * The Worker is responsible for Push and Fill - that's it.
func Worker(in <-chan *order, out chan<- *order, status chan<- OrderUpdate, book *orderbook) {
	for ord := range in {
		// push the order into our books
		err := book.push(ord)
		if err != nil {
			book.markError(ord, err, status)
		} else {
			book.markPending(ord, status)
			// go fill the order and push on out when it's' filled.
			go book.fill(ord, out, status)
		}
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
				for k, v := range orderStatus {
					log.Printf("%+v - %+v", k, v.isFilled())
				}
			}
		}
	}()

	return updates
}

// push inserts an order into the books and returns an error if order
// is invalid or if an error occurred during push.
func (o *orderbook) push(ord *order) error {
	o.Lock()
	defer o.Unlock()

	// TODO: order validation here

	if ord.side {
		// handle buy side
		if _, ok := o.buy[ord.price]; !ok {
			o.buy[ord.price] = []*order{ord}
		} else {
			o.buy[ord.price] = append(o.buy[ord.price], ord)
		}
		return nil
	}

	// handle sell side
	if _, ok := o.sell[ord.price]; !ok {
		o.sell[ord.price] = []*order{ord}
	} else {
		o.sell[ord.price] = append(o.sell[ord.price], ord)
	}
	return nil
}

// fill loops over the book and finds orders.
func (o *orderbook) fill(fillorder *order, out chan<- *order, status chan<- OrderUpdate) {
	for {
		if !fillorder.isFilled() {
			if err := o.attemptFill(fillorder, out, status); err != nil {
				// report error and attempt fill again
				o.markError(fillorder, err, status)
			}
			// TODO: time delay here?
		}
	}
}

// attemptFill attempts to fill the given order and returns the amount it filled or an error.
// * this level of abstraction is useful because it's where we have determined and vetted
// all of the preconditions and now know that we are for sure about to touch the books.
// thus we can safely grab a lock and know it was necessary and efficient.
// * attemptFill acquires locks on *order when necessary for as little time as possible.
// * attemptFill defers the unlock so be careful making this recursive and note when it cleans up
// to avoid deadlocks.
// ***TODO***: mark the deductions in appropriate accounts for all transactions.
// for now all this does is actually match the orders together.
func (o *orderbook) attemptFill(fillorder *order, out chan<- *order, status chan<- OrderUpdate) error {
	o.Lock()
	defer o.Unlock()

	fillorder.Lock()
	defer fillorder.Unlock()

	wanted := fillorder.open - fillorder.filled

	if fillorder.side == buyside {
		// match buy by finding lowest on sellside
		_, sellOrder, err := o.findLowest(fillorder.price, sellside)
		if err != nil {
			return err
		}

		sellOrder.Lock()
		defer sellOrder.Unlock()

		// find out how many units are available
		available := sellOrder.open - sellOrder.filled

		switch {
		case wanted > available:
			// more wanted than available, take what is available, remove and mark as filled
			sellOrder.filled += available
			fillorder.filled += available

			if fillorder.open == fillorder.filled {
				o.markFilled(fillorder, out, status)
			}
			if sellOrder.open == sellOrder.filled {
				o.markFilled(sellOrder, out, status)
			}
		case wanted <= available:
			sellOrder.filled = sellOrder.filled + wanted
			fillorder.filled = fillorder.filled + wanted

			if fillorder.open == fillorder.filled {
				o.markFilled(fillorder, out, status)
			}
			if sellOrder.open == sellOrder.filled {
				o.markFilled(sellOrder, out, status)
			}
		default:
			log.Panicln("should never get here; this smells like a bug.")
		}

		return nil
	}

	return nil
}

// markFilled updates an orders status, removes it from the books, and sends it on the output channel.
func (o *orderbook) markFilled(order *order, out chan<- *order, status chan<- OrderUpdate) {
	err := o.remove(order)
	if err != nil {
		status <- OrderUpdate{
			Order:  order,
			Status: statusFilled,
			Err:    err,
		}
	}
	out <- order
}

// markError logs the order's status as statusCanceled and sets the error
func (o *orderbook) markError(order *order, err error, status chan<- OrderUpdate) {
	// report on status if errored.
	status <- OrderUpdate{
		Order:  order,
		Status: statusErrored,
		Err:    err,
	}
}

// markPending logs the order's status as statusPending
func (o *orderbook) markPending(order *order, status chan<- OrderUpdate) {
	// update status to pending
	status <- OrderUpdate{
		Order:  order,
		Status: statusPending,
	}
}

// remove searches for the order by ID and removes it from the books.
func (o *orderbook) remove(order *order) error {
	if order.ID == "" {
		return fmt.Errorf("ErrInvalidID")
	}

	if order.side {
		list, ok := o.buy[order.price]
		if !ok {
			return ErrNotFound
		}

		for idx, v := range list {
			if v.ID == order.ID {
				list = append(list[:idx], list[idx+1:]...)
				o.buy[order.price] = list
				return nil
			}
		}
		return nil
	}

	list, ok := o.sell[order.price]
	if !ok {
		return fmt.Errorf("ErrNotFound")
	}

	for idx, v := range list {
		if v.ID == order.ID {
			list = append(list[:idx], list[idx+1:]...)
			o.sell[order.price] = list
			return nil
		}
	}

	return nil
}

// findLowest recursively searches for the lowest price in the map.
// It returns the index of the order, the order, and an error if anything went wrong.
func (o *orderbook) findLowest(price Price, side bool) (int64, *order, error) {
	if side {
		// handle buy list
		buyable, ok := o.buy[price]
		if !ok {
			if price == 0 {
				return -1, nil, fmt.Errorf("ErrNotFound")
			}
			// recursively search for next lowest price.
			nextLowest := price - 1
			return o.findLowest(nextLowest, side)
		}
		if len(buyable) == 0 {
			return -1, nil, fmt.Errorf("ErrNotFound")
		}

		return int64(0), buyable[0], nil
	}

	// handle the sell list
	sellable, ok := o.sell[price]
	if !ok {
		if price == 0 {
			return -1, nil, fmt.Errorf("ErrNotFound")
		}
		nextLowest := price - 1
		return o.findLowest(nextLowest, side)
	}
	if len(sellable) == 0 {
		return -1, nil, fmt.Errorf("ErrNotFound")
	}

	return int64(0), sellable[0], nil
}
