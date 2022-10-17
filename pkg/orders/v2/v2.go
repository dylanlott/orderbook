// Package v2 uses a channel-based approach to order fulfillment to explore
// a concurrency-first design. It also defines a simpler Order interface
// to see how we can slim down that design from our previous approach.
package v2

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// BUY or SELL side string constant definitions
const (
	// BUY marks an order as a buy side order
	BUY string = "BUY"
	// SELL marks an order as a sell side order
	SELL string = "SELL"
)

// StateMonitor returns a channel that outputts OrderStates as its receives them.
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

// Order holds a second approach at the Order interface
// that incorporates lessons learned from the first time around.
type Order interface {
	ID() string
	Price() int64
	Side() string
	Fill(ctx context.Context, o *Orderbook) (Order, error)
}

// LimitOrder fulfills the Order interface. FillStrategy implements a limit order
// fill algorithm.
type LimitOrder struct {
	// Holds a string identifier to the Owner of the Order.
	Owner string
	// Holds any errors that occurred during processing
	Err error

	//// Private fields
	// id is a unique identifier
	id string
	// price of the order in the asset's smallest atomic unit.
	// E.G. cents for USD, gwei for ETH, etc...
	price int64
	// Returns BUY if its a buy order, SELL if its a sell order.
	side string
}

// OrderState holds the current state of an OrderV2 and
// binds it to a simple state object.
type OrderState struct {
	Order  Order
	Status string
	Err    error
}

// Worker defines a function meant to be spawned concurrently that listens to the in
// channel and fills orders as they are received and processes them in their own
// gouroutine.
func Worker(in <-chan Order, out chan<- Order, status chan<- OrderState, orderbook *Orderbook) {
	for o := range in {

		// attempt to fill the order
		go func(order Order) {
			log.Printf("received order %+v", order)

			// insert the order into the correct side of our books
			switch order.Side() {
			case "BUY":
				log.Printf("Buy order: %+v", order)
				if err := orderbook.Buy.Insert(order); err != nil {
					status <- OrderState{
						Order: order,
						Err:   fmt.Errorf("failed to insert into buy tree: %w", err),
					}
					return
				}
			case "SELL":
				log.Printf("Sell order: %+v", order)
				if err := orderbook.Sell.Insert(order); err != nil {
					status <- OrderState{
						Order: order,
						Err:   fmt.Errorf("failed to insert into sell tree: %w", err),
					}
					return
				}
			default:
				panic("must specify an order side")
			}

			// start attempting to fill the order
			filled, err := order.Fill(context.Background(), orderbook)
			status <- OrderState{
				Order: filled,
				Err:   err,
			}
			out <- order
		}(o)
	}
}

///////////////
// Orderbook //
///////////////

// Orderbook is worked on by Workers.
// TODO: Turn this into an interface to abstract away the underlying data structure
type Orderbook struct {
	Buy  *PriceNode
	Sell *PriceNode
}

// Pull routes a pull to a price and a side of the Orderbook.
// It returns that Order or an order.
func (o *Orderbook) Pull(price int64, side string) (Order, error) {
	if side == BUY {
		return o.Buy.Pull(price)
	} else {
		return o.Sell.Pull(price)
	}
}

// Push inserst an order into the book
func (o *Orderbook) Push(order Order) error {
	if order.Side() == BUY {
		return o.Buy.Insert(order)
	} else {
		return o.Sell.Insert(order)
	}
}

// Fill fills the given Order on the Orderbook.
// * it is meant to be called as a go function.
func (o *Orderbook) Fill(ctx context.Context, order Order) error {
	return fmt.Errorf("not impl")
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

// Fill is a synchronous function that implements a LimitOrder fill and returns
// an error if there were any issues.
func (l *LimitOrder) Fill(ctx context.Context, o *Orderbook) (Order, error) {
	return l, fmt.Errorf("Fill not impl")
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
	}

	if o.Price() > t.val {
		if t.right == nil {
			t.right = &PriceNode{val: o.Price()}
			return t.right.Insert(o)
		}
		return t.right.Insert(o)
	}

	panic("should not get here; this smells like a bug")
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
		}
	}

	if price < t.val {
		if t.left != nil {
			return t.left.Find(price)
		}
	}

	panic("should not get here; this smells like a bug")
}

// Match will iterate through the tree based on the price of the
// fillOrder and finds a bookOrder that matches its price.
func (t *PriceNode) Match(fillOrder Order, cb func(bookOrder Order)) {
	if t == nil {
		cb(nil)
		return
	}

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

	panic("should not get here; this smells like a bug")
}

// Pull is used by the Orderbook to pull an order at a given price.
// * It is an atomic function and handles its own locking.
// * It will find and remove an Order at a given Price or return an error
func (t *PriceNode) Pull(price int64) (Order, error) {
	// NB: The disparity between having to find and remove suggests that
	// we could refactor this into a single function called Remove here.
	t.Lock()
	defer t.Unlock()
	pulled, err := t.Find(price)
	if err != nil {
		return nil, fmt.Errorf("failed to pull: %w", err)
	}
	_, err = t.RemoveByID(pulled)
	if err != nil {
		return nil, fmt.Errorf("failed to remove from books: %w", err)
	}
	return pulled, nil
}

// Orders returns the list of Orders for a given price.
func (t *PriceNode) Orders(price int64) ([]Order, error) {
	if t == nil {
		return nil, fmt.Errorf("order tree is nil")
	}

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
	}

	if order.Price() < t.val {
		if t.left != nil {
			return t.left.RemoveByID(order)
		}
		return nil, fmt.Errorf("ErrNoExist")
	}

	return nil, fmt.Errorf("failed to remove: %+V", order)
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
