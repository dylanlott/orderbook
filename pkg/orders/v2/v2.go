// Package v2 uses a channel-based approach to order fulfillment to explore
// a concurrency-first design. It also defines a simpler Order interface
// to see how we can slim down that design from our previous approach.
package v2

import (
	"context"
	"fmt"
	"log"
	"time"
)

// BUY or SELL side string constant definitions
const (
	// BUY marks an order as a buy side order
	BUY string = "BUY"
	// SELL marks an order as a sell side order
	SELL string = "SELL"
)

// LimitFill fills the given order with a limit strategy. A limit strategy fills orders
// at a hard max for buys and a hard minimum for sells with no time limit.
var LimitFill FillStrategy = func(ctx context.Context, self Order, b *Orderbook) error {
	return fmt.Errorf("not impl")
}

// MarketFill fills orders at the current market price until they're filled.
var MarketFill FillStrategy = func(ctx context.Context, self Order, books *Orderbook) error {
	return fmt.Errorf("not impl")
}

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

// FillStrategy is a synchronous function that is meant to be called in a
// goroutine that handles different fill strategies e.g. Limit Order,
// Market Order, etc... for the order's self.
type FillStrategy func(ctx context.Context, self Order, books *Orderbook) error

// LimitOrder fulfills the Order interface. FillStrategy implements a limit order
// fill algorithm.
type LimitOrder struct {
	// Holds a string identifier to the Owner of the Order.
	Owner string
	// Strategy is a blocking function that returns when the order is filled.
	Strategy FillStrategy
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

// Orderbook is worked on by Workers.
// TODO: Turn this into an interface to abstract away the underlying data structure
type Orderbook struct {
	Buy  *PriceNode
	Sell *PriceNode
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

///////////////////////////
// B-TREE IMPLEMENTATION //
///////////////////////////

// PriceNode represents a tree of nodes that maintain lists of Orders at that price.
// * Each PriceNode maintains an ordered list of Orders that share the same price.
// * This tree is a simple binary tree, where left nodes are lesser prices and right
// nodes are greater in price than the current node.
type PriceNode struct {
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

	panic("should not get here; this smells like a bug")
}

// RemoveFromPriceList removes an order from the list of orders at a
// given price in our tree. It does not currently rebalance the tree.
// TODO: make this rebalance the tree at some threshold.
func (t *PriceNode) RemoveFromPriceList(order Order) error {
	if t == nil {
		return fmt.Errorf("order tree is nil")
	}

	if order.Price() == t.val {
		for i, ord := range t.orders {
			if ord.ID() == order.ID() {
				t.orders = removeV2(t.orders, i)
				return nil
			}
		}
		return fmt.Errorf("ErrNoExist")
	}

	if order.Price() > t.val {
		if t.right != nil {
			return t.right.RemoveFromPriceList(order)
		}
		return fmt.Errorf("ErrNoExist")
	}

	if order.Price() < t.val {
		if t.left != nil {
			return t.left.RemoveFromPriceList(order)
		}
		return fmt.Errorf("ErrNoExist")
	}

	panic("should not get here; this smells like a bug")
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
