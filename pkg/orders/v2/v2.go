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
				log.Printf("received state update: %+v", s)
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
	log.Printf("%+v\n", orders)
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
	History() []*Transaction
}

//////////////////////////////
// LIMIT ORDER IMPLEMENTATION
//////////////////////////////

// LimitOrder fulfills the Order interface. FillStrategy implements a limit order
// fill algorithm.
type LimitOrder struct {
	sync.Mutex

	// Holds a string identifier to the Owner of the Order.
	Owner string
	// Holds any errors that occurred during processing
	Err error
	// transactions is a list of actions on this order
	Transactions []*Transaction

	//// Private fields
	// id is a unique identifier
	id string
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
			log.Printf("received order %+v", order)

			if err := orderbook.Push(order); err != nil {
				// TODO: how can we define this away?
				log.Fatalf("failed to push order into books: %v", err)
			}

			// start attempting to fill the order
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
			log.Printf("attemptFill failed: %+v", err)
			// notify state that we failed to fill this order.
			status <- OrderState{
				Err:    err,
				Order:  order,
				Status: StatusUnfilled,
			}
			return
		}
	}
	log.Printf("FILLED ORDER %+v", order)
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

// Push inserst an order into the book and starts off the goroutine
// responsible for filling the Order.
func (o *Orderbook) Push(order Order) error {
	if order.Side() == BUY {
		err := o.Buy.Insert(order)
		if err != nil {
			return fmt.Errorf("failed to add order to the book: %w", err)
		}
		return nil
	} else {
		err := o.Sell.Insert(order)
		if err != nil {
			return fmt.Errorf("failed to add order to the book: %w", err)
		}

		return nil
	}
}

// Fill is meant to be called concurrently and works on the Orderbook.
func (o *Orderbook) attemptFill(fillOrder Order) (Order, error) {
	if fillOrder.Side() == BUY {
		// handle buy order
		sellOrders, err := o.Sell.Orders(fillOrder.Price())
		if err != nil {
			return nil, fmt.Errorf("failed to list orders: %+v", err)
		}

		first := sellOrders[0]
		log.Printf("first priority sellOrder: %+v", first)

		return fillOrder, fmt.Errorf("failed to fill: %v", fillOrder)
	} else {
		// handle sell order
		buyOrders, err := o.Buy.Orders(fillOrder.Price())
		if err != nil {
			return nil, fmt.Errorf("failed to list orders: %+v", err)
		}

		first := buyOrders[0]
		log.Printf("first priority buyOrder: %+v", first)

		return fillOrder, fmt.Errorf("failed to fill: %v", fillOrder)
	}
}

////////////////
// LimitOrder //
////////////////

// ID returns the private id of the LimitOrder
func (l *LimitOrder) ID() string {
	return l.id
}

// Price returns the private price of the LimitOrder as int64
func (l *LimitOrder) Price() int64 {
	return l.price
}

// Side returns the type of order, either BUY or SELL.
func (l *LimitOrder) Side() string {
	return l.side
}

// Open returns the amount of this order still open for purchase
func (l *LimitOrder) Open() uint64 {
	return l.open - l.filled
}

// History returns the transaction receipts for this order.
func (l *LimitOrder) History() []*Transaction {
	return l.Transactions
}

// Fill will add a Transaction to the tx list of an order or report
// an error.
func (l *LimitOrder) Fill(tx *Transaction) ([]*Transaction, error) {
	if l.Transactions == nil {
		l.Transactions = make([]*Transaction, 0)
	}
	// NB: ensure we call the method instead of accesssing the field directly.
	// This is probably a code smell 👃
	if tx.Quantity > l.Open() {
		return nil, fmt.Errorf("cannot purchase more than are available: %v", tx)
	}
	if tx.Price < uint64(l.price) {
		return nil, fmt.Errorf("cannot pay less than limit order price")
	}

	// NB: We probably want to only ever increase the filled quantity or maybe
	// move to a model where we entirely calculate open and filled by
	// analyzing the transaction list instead of maintaining them as fields
	// on the LimitOrder.
	l.filled = l.filled + tx.Quantity
	l.Transactions = append(l.Transactions, tx)

	return l.Transactions, nil
}

// Returns the owner ID of this order. It maps to the account ID.
func (l *LimitOrder) OwnerID() string {
	return l.Owner
}

///////////////////////////
// B-TREE IMPLEMENTATION //
///////////////////////////

// PriceNode represents a tree of nodes that maintain lists of Orders at that price.
// * Each PriceNode maintains an ordered list of Orders that share the same price.
// * This tree is a simple binary tree, where left nodes are lesser prices and right
// nodes are greater in price than the current node.
type PriceNode struct {
	sync.Mutex

	val    int64 // to represent price
	orders []Order
	right  *PriceNode
	left   *PriceNode
}

// List grabs a lock on the whole tree and reads all of the orders from it.
func (t *PriceNode) List() ([]Order, error) {
	t.Lock()
	defer t.Unlock()

	orders := []Order{}
	stack := []*PriceNode{}
	var current *PriceNode
	for t != nil || len(stack) > 0 {
		if current != nil {
			stack = append(stack, t)
			current = current.left
		} else {
			current = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			fmt.Printf("visited price: %d", t.val)
			orders = append(orders, t.orders...)
			current = current.right
		}
	}
	return orders, nil
}

// Insert will add an Order to the Tree. It traverses until it finds the right price
// or where the price should exist and creates a price node if it doesn't exist, then
// adds the Order to that price node.
func (t *PriceNode) Insert(o Order) error {
	if t == nil {
		t = &PriceNode{val: o.Price()}
	}

	if t.val == o.Price() {
		// when we find a price match for the Order's price,
		// insert the Order into this node's Order list.
		// lock because we're about to alter coure resource.
		t.Lock()
		defer t.Unlock()

		if t.orders == nil {
			t.orders = make([]Order, 0)
		}
		t.orders = append(t.orders, o)

		return nil
	}

	if o.Price() < t.val {
		if t.left == nil {
			t.left = &PriceNode{val: o.Price()}
			return t.left.Insert(o)
		}
		return t.left.Insert(o)
	} else {
		if t.right == nil {
			t.right = &PriceNode{val: o.Price()}
			return t.right.Insert(o)
		}
		return t.right.Insert(o)
	}
}

// Find returns the highest priority Order for a given price point.
// * If it can't find an order at that exact price, it will search for
// a cheaper order if one exists.
func (t *PriceNode) Find(price int64) (Order, error) {
	if t == nil {
		return nil, fmt.Errorf("err no exist")
	}

	if price == t.val {
		if len(t.orders) > 0 {
			return t.orders[0], nil
		}
		return nil, fmt.Errorf("no orders at this price")
	}

	if price > t.val {
		if t.right != nil {
			return t.right.Find(price)
		} else {
			return nil, fmt.Errorf("no orders at this price")
		}
	} else {
		if t.left != nil {
			return t.left.Find(price)
		} else {
			return nil, fmt.Errorf("no orders at this price")
		}
	}
}

// Match will iterate through the tree based on the price of the
// fillOrder and finds a bookOrder that matches its price.
func (t *PriceNode) Match(fillOrder Order, cb func(bookOrder Order)) {
	if t == nil {
		cb(nil)
		return
	}

	t.Lock()
	defer t.Unlock()

	if fillOrder.Price() == t.val {
		// callback with first order in the list
		bookOrder := t.orders[0]
		cb(bookOrder)
		return
	}

	if fillOrder.Price() > t.val {
		if t.right != nil {
			t.right.Match(fillOrder, cb)
			return
		}
	}

	if fillOrder.Price() < t.val {
		if t.left != nil {
			t.left.Match(fillOrder, cb)
			return
		}
	}

	log.Fatalf("error: fillOrder should not get here: %+v", fillOrder)
}

// Pull is used by the Orderbook to pull an order at a given price.
// * It is an atomic function and handles its own locking.
// * It will find and remove an Order at a given Price or return an error
func (t *PriceNode) Pull(price int64) (Order, error) {
	// NB: The disparity between having to find and remove suggests that
	// we could refactor this into a single function called Remove here.
	// NB: Locking here creates a deadlock. Another code smell.
	// We should refactor pull to atomically wrap the find and remove logic.
	pulled, err := t.Find(price)
	if err != nil {
		return nil, fmt.Errorf("failed to pull: %w", err)
	}
	_, err = t.RemoveByID(pulled)
	if err != nil {
		return nil, fmt.Errorf("failed to remove from books: %w", err)
	}
	log.Printf("pulling order: %+v", pulled)
	return pulled, nil
}

// Orders returns the list of Orders for a given price.
func (t *PriceNode) Orders(price int64) ([]Order, error) {
	if t == nil {
		return nil, fmt.Errorf("order tree is nil")
	}

	t.Lock()
	defer t.Unlock()

	if t.val == price {
		return t.orders, nil
	}

	if price > t.val {
		if t.right != nil {
			return t.right.Orders(price)
		}
	}

	if price < t.val {
		if t.left != nil {
			return t.left.Orders(price)
		}
	}

	return nil, fmt.Errorf("ErrNoOrders")
}

// Remove removes an order from the list of orders at a
// given price in our tree. It does not currently rebalance the tree.
// TODO: make this rebalance the tree at some threshold.
func (t *PriceNode) RemoveByID(order Order) (Order, error) {
	if t == nil {
		return nil, fmt.Errorf("order tree is nil")
	}
	if order.Price() == t.val {
		for i, ord := range t.orders {
			if ord.ID() == order.ID() {
				t.Lock()
				defer t.Unlock()

				t.orders = removeV2(t.orders, i)
				return ord, nil
			}
		}
		return nil, fmt.Errorf("ErrNoExist")
	}
	if order.Price() > t.val {
		if t.right != nil {
			return t.right.RemoveByID(order)
		}
		return nil, fmt.Errorf("ErrNoExist")
	} else {
		if t.left != nil {
			return t.left.RemoveByID(order)
		}
		return nil, fmt.Errorf("ErrNoExist")
	}
}

//Print prints the elements in left-current-right order.
func (t *PriceNode) Print() {
	if t == nil {
		return
	}
	t.left.Print()
	fmt.Printf("%+v\n", t.val)
	t.right.Print()
}

// remove removes the element in s at index i
func removeV2(s []Order, i int) []Order {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
