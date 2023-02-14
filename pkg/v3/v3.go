package v3

import (
	"errors"
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
				log.Printf("statuses: %+v", orderStatus)
			}
		}
	}()

	return updates
}

// Worker wraps an in and output channel around the orderbook.
// * Updates are published on the status channel.
// * Filled orders are sent on the out channel.
// * Errored, cancelled, or otherwise incomplete but finished orders are only
// viewed through the status updates.
// * The Worker is responsible for Push and Fill - that's it.
func Worker(in <-chan *order, out chan<- *order, status chan<- OrderUpdate, book *orderbook) {
	for fillOrder := range in {
		err := book.push(fillOrder)
		if err != nil {
			book.markError(fillOrder, err, status)
			continue
		}

		book.markPending(fillOrder, status)
		go book.fill(fillOrder, out, status)
	}
}

// fill loops over the book and finds orders.
func (o *orderbook) fill(fillorder *order, out chan<- *order, status chan<- OrderUpdate) {
	for {
		if err := o.attemptFill(fillorder, out, status); err != nil {
			if errors.Is(err, ErrFilled) {
				break
			}
			o.markError(fillorder, err, status)
		}
	}
}

// attemptFill attempts to fill the given order and returns the amount it filled or an error.
// ***TODO***: mark the deductions in appropriate accounts for all transactions.
// for now all this does is actually match the orders together.
func (o *orderbook) attemptFill(fillOrder *order, out chan<- *order, status chan<- OrderUpdate) error {
	fillOrder.Lock()
	defer func() {
		fillOrder.Unlock()
	}()

	wanted := fillOrder.open - fillOrder.filled
	if wanted == 0 {
		if err := o.pull(fillOrder); err != nil {
			status <- OrderUpdate{
				Order:  fillOrder,
				Status: statusErrored,
				Err:    fmt.Errorf("failed to pull order %+w", err),
			}
		}
		return ErrFilled
	}

	if fillOrder.side == buyside {

		_, sellOrder, err := o.findLowest(fillOrder.price, sellside)
		if err != nil {
			return err
		}

		sellOrder.Lock()
		defer sellOrder.Unlock()

		available := sellOrder.open - sellOrder.filled

		switch {
		case wanted > available:
			sellOrder.filled += available // okay so the sell order is contended on
			fillOrder.filled += available
		case wanted <= available:
			sellOrder.filled += wanted
			fillOrder.filled += wanted
		default:
			log.Panicln("should never get here; this smells like a bug.")
		}

		if fillOrder.filled == fillOrder.open {
			if err := o.pull(sellOrder); err != nil {
				o.markError(fillOrder, err, status)
			} else {
				o.markFilled(fillOrder, out, status)
			}
			return ErrFilled
		}
		return nil
	}
	return nil
}

// push inserts an order into the books and returns an error if order
// is invalid or if an error occurred during push.
// TODO: order validation here
func (o *orderbook) push(ord *order) error {
	o.Lock()
	defer o.Unlock()

	if ord.side {
		if _, ok := o.buy[ord.price]; !ok {
			o.buy[ord.price] = []*order{ord}
		} else {
			o.buy[ord.price] = append(o.buy[ord.price], ord)
		}
		return nil
	}

	if _, ok := o.sell[ord.price]; !ok {
		o.sell[ord.price] = []*order{ord}
	} else {
		o.sell[ord.price] = append(o.sell[ord.price], ord)
	}
	return nil
}

// pull searches for the order by price and then ID and removes it from the books.
// Price and ID of the *order must be present on the *order being passed.
func (o *orderbook) pull(order *order) error {
	o.Lock()
	defer o.Unlock()

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

// markFilled updates an orders status, removes it from the books, and sends it on the output channel.
func (o *orderbook) markFilled(order *order, out chan<- *order, status chan<- OrderUpdate) {
	status <- OrderUpdate{
		Order:  order,
		Status: statusFilled,
	}
	out <- order
}

// markError logs the order's status as statusCanceled and sets the error.
func (o *orderbook) markError(order *order, err error, status chan<- OrderUpdate) {
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

// findLowest recursively searches for the lowest price in the map.
// It returns the index of the order, the order, and an error if anything went wrong.
func (o *orderbook) findLowest(price Price, side bool) (int64, *order, error) {
	o.Lock()
	defer o.Unlock()

	if side {
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
