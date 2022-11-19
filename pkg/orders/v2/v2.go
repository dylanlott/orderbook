// Package v2 uses a channel-based approach to order fulfillment to explore
// a concurrency-first design. It also defines a simpler Order interface
// to see how we can slim down that design from our previous approach.
package v2

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

// BUY or SELL side string constant definitions
const (
	// BUY marks an order as a buy side order
	BUY string = "BUY"
	// SELL marks an order as a sell side order
	SELL string = "SELL"
)

const StatusUnfilled = "unfilled"
const StatusFilled = "filled"

// StateMonitor returns a channel that outputs OrderStates as its receives them.
func StateMonitor() chan OrderState {
	updates := make(chan OrderState)
	orderStatus := make(map[string]Order)
	ticker := time.NewTicker(1 * time.Second)

	go func() {
		for {
			select {
			case s := <-updates:
				orderStatus[s.Order.ID()] = s.Order
			case <-ticker.C:
				logState(orderStatus)
			}
		}
	}()

	return updates
}

// a simple convenience function to display the current state of the engine.
func logState(orders map[string]Order) {
	fmt.Printf("orders: %v\n", orders)
}

////////////////////
// ORDERS INTERFACE
////////////////////

// Order holds a second approach at the Order interface
// that incorporates lessons learned from the first time around.
type Order interface {
	OwnerID() string
	ID() string
	Price() int64
	Side() string
	Open() uint64
	Fill(tx *Transaction) ([]*Transaction, error)
	History() []*Transaction
}

// Transaction
type Transaction struct {
	AccountID string // who filled the order
	Quantity  uint64 // The amount they filled it for
	Price     uint64 // The price they paid
	Total     uint64 // The total of the Transaction.
}

// OrderState holds the current state of an OrderV2 and
// binds it to a simple state object.
type OrderState struct {
	Order  Order
	Status string
	Err    error
}

// A Worker listens for orders from in, inserts them into the books, and then
// starts a filler for that Order.
func Worker(in <-chan Order, out chan<- Order, status chan<- OrderState, orderbook *Orderbook) {
	for o := range in {
		// attempt to fill the order
		go func(order Order) {
			if err := orderbook.Push(order); err != nil {
				status <- OrderState{
					Order:  order,
					Status: StatusUnfilled,
					Err:    fmt.Errorf("failed to push order: %+v", err),
				}
			}
			Filler(out, order, status, orderbook)
		}(o)
	}
}

// Filler will attempt to fill the Order until it's filled and then
// reports it to the output channel. It's a blocking function as it's
// meant to return only when the Order is filled.
func Filler(completed chan<- Order, order Order, status chan<- OrderState, book *Orderbook) {
	for order.Open() > 0 {
		_, err := book.attemptFill(order)
		if err != nil {
			// notify state that we failed to fill this order.
			status <- OrderState{
				Err:    fmt.Errorf("attmemptFill failed: %v", err),
				Order:  order,
				Status: StatusUnfilled,
			}
			continue
		}
	}
	status <- OrderState{
		Order:  order,
		Status: StatusFilled,
		Err:    nil,
	}
	completed <- order
}

///////////////
// Orderbook //
///////////////

// Orderbook binds a buy side and sell side order tree to an AccountManager.
// Orders should only ever be Push'd into the books and fill should happen
// entirely through attemptFill, a private method for Workers to call.
type Orderbook struct {
	Buy  *PriceNode
	Sell *PriceNode

	Accounts accounts.AccountManager
}

// Push inserts an order into the correct side of the books.
func (o *Orderbook) Push(order Order) error {
	if order.Side() == BUY {
		return o.Buy.Insert(order)
	} else {
		return o.Sell.Insert(order)
	}
}

// Fill is meant to be called concurrently and works on the Orderbook.
func (o *Orderbook) attemptFill(fillOrder Order) (Order, error) {
	if fillOrder.Side() == BUY {
		sellOrders, err := o.Sell.Orders(fillOrder.Price())
		if err != nil {
			return nil, err
		}
		if len(sellOrders) == 0 {
			return nil, fmt.Errorf("ErrNoOrders")
		}
		// fmt.Printf("sellOrders: %v\n", sellOrders)

		seller := sellOrders[0]
		availableForPurchase := seller.Open()
		needed := fillOrder.Open()

		if needed >= availableForPurchase {
			// take all available for purchase
			txlist, txerr := fillOrder.Fill(&Transaction{
				AccountID: seller.OwnerID(),
				Quantity:  availableForPurchase,
				Price:     uint64(seller.Price()),
				Total:     uint64(seller.Price()) * availableForPurchase,
			})
			if txerr != nil {
				log.Printf("FAILED TO TRANSACT: %+v", err)
			}
			fmt.Printf("### txlist: %v\n", txlist)
		} else {
			// partial fill - needed < availableForPurchase
			txlist, txerr := fillOrder.Fill(&Transaction{
				AccountID: seller.OwnerID(),
				Quantity:  needed,
				Price:     uint64(seller.Price()),
				Total:     uint64(seller.Price()) * needed,
			})
			if txerr != nil {
				log.Printf("FAILED TO TRANSACT: %+v", err)
			}
			fmt.Printf("successfully transacted: txlist: %v\n", txlist)
		}

		return fillOrder, nil
	} else {
		// handle sell order
		buyOrders, err := o.Buy.Orders(fillOrder.Price())
		if err != nil {
			return nil, err
		}
		if len(buyOrders) == 0 {
			return nil, fmt.Errorf("ErrNoOrders")
		}
		// fmt.Printf("buyOrders: %v\n", buyOrders)
		return fillOrder, nil
	}
}

//////////////////////////////
// LIMIT ORDER IMPLEMENTATION
//////////////////////////////

// LimitOrder fulfills the Order interface. FillStrategy implements a limit order
// fill algorithm.
type LimitOrder struct {
	sync.Mutex
	// id is a unique identifier
	id string
	// Holds a string identifier to the owner of the Order.
	owner string
	// price of the order in the asset's smallest atomic unit.
	// E.G. cents for USD, gwei for ETH, etc...
	price int64
	// Returns BUY if its a buy order, SELL if its a sell order.
	side string
	// open represents the number of items at the price being ordered
	open uint64
	// filled is a quantity of this order that has been filled,
	// aka purchased at a specific price
	filled uint64
	// txs is a list of actions on this order
	txs []*Transaction
}

// ID returns the private id of the LimitOrder
func (l *LimitOrder) ID() string {
	l.Lock()
	defer l.Unlock()
	return l.id
}

// Returns the owner ID of this order. It maps to the account ID.
func (l *LimitOrder) OwnerID() string {
	return l.owner
}

// Price returns the private price of the LimitOrder as int64
func (l *LimitOrder) Price() int64 {
	l.Lock()
	defer l.Unlock()
	return l.price
}

// Side returns the type of order, either BUY or SELL.
func (l *LimitOrder) Side() string {
	l.Lock()
	defer l.Unlock()
	return l.side
}

// Open returns the amount of this order still open for purchase
func (l *LimitOrder) Open() uint64 {
	l.Lock()
	defer l.Unlock()
	return l.open - l.filled
}

// History returns the transaction receipts for this order.
func (l *LimitOrder) History() []*Transaction {
	l.Lock()
	defer l.Unlock()
	return l.txs
}

// Fill will add a Transaction to the tx list of an order or report
// an error.
func (l *LimitOrder) Fill(tx *Transaction) ([]*Transaction, error) {
	l.Lock()
	defer l.Unlock()

	if l.txs == nil {
		l.txs = make([]*Transaction, 0)
	}
	// NB: ensure we call the method instead of accesssing the field directly.
	// This is probably a code smell 👃
	if tx.Quantity > l.Open() {
		return nil, fmt.Errorf("cannot purchase more than are available: %v", tx)
	}

	// ensure price is valid. We are okay with paying less if we're buying, and okay
	// taking more if we're selling.
	if l.side == BUY {
		if tx.Price > uint64(l.price) {
			return nil, fmt.Errorf("cannot pay less than limit order price")
		}
	} else {
		if tx.Price < uint64(l.price) {
			return nil, fmt.Errorf("cannot pay less than limit order price")
		}
	}

	l.filled += tx.Quantity
	l.txs = append(l.txs, tx)

	return l.txs, nil
}
