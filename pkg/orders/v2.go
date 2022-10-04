package orders

import (
	"context"
	"fmt"
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
	Side  string
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

// Orderbook is worked on by Workers.
// TODO: Turn this into an interface to abstract away the underlying data structure
type Orderbook struct {
	Buy  *TreeNodeV2
	Sell *TreeNodeV2
}

func Worker(in <-chan *LimitOrder, out chan<- *LimitOrder, status chan<- OrderState, orderbook *Orderbook) {
	for o := range in {

		// attempt to fill the order
		go func(order *LimitOrder) {
			log.Printf("received order %+v", order)

			// insert the order into the correct side of our books
			switch order.Side {
			case "BUY":
				log.Printf("Buy order: %+v", order)
			case "SELL":
				log.Printf("Sell order: %+v", order)
			default:
				panic("must specify an order side")
			}

			// start attempting to fill the order
			err := order.Strategy(context.Background())
			status <- OrderState{
				Order: order,
				Err:   err,
			}
			out <- order
		}(o)
	}
}

// TreeNodeV2 represents a tree of nodes that maintain lists of Orders at that price.
// * Each TreeNodeV2 maintains an ordered list of Orders that share the same price.
// * This tree is a simple binary tree, where left nodes are lesser prices and right
// nodes are greater in price than the current node.
type TreeNodeV2 struct {
	val    int64 // to represent price
	orders []OrderV2
	right  *TreeNodeV2
	left   *TreeNodeV2
}

// Insert will add an Order to the Tree. It traverses until it finds the right price
// or where the price should exist and creates a price node if it doesn't exist, then
// adds the Order to that price node.
func (t *TreeNodeV2) Insert(o OrderV2) error {
	if t == nil {
		t = &TreeNodeV2{val: o.Price()}
	}

	if t.val == o.Price() {
		// when we find a price match for the Order's price,
		// insert the Order into this node's Order list.
		if t.orders == nil {
			t.orders = make([]OrderV2, 0)
		}
		t.orders = append(t.orders, o)
		return nil
	}

	if o.Price() < t.val {
		if t.left == nil {
			t.left = &TreeNodeV2{val: o.Price()}
			return t.left.Insert(o)
		}
		return t.left.Insert(o)
	}

	if o.Price() > t.val {
		if t.right == nil {
			t.right = &TreeNodeV2{val: o.Price()}
			return t.right.Insert(o)
		}
		return t.right.Insert(o)
	}

	panic("should not get here; this smells like a bug")
}

type OrderV2 interface {
	ID() string
	Price() int64
}

// Find returns the highest priority Order for a given price point.
// * If it can't find an order at that exact price, it will search for
// a cheaper order if one exists.
func (t *TreeNodeV2) Find(price int64) (OrderV2, error) {
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
func (t *TreeNodeV2) Match(fillOrder OrderV2, cb func(bookOrder OrderV2)) {
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
func (t *TreeNodeV2) Orders(price int64) ([]OrderV2, error) {
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
func (t *TreeNodeV2) RemoveFromPriceList(order OrderV2) error {
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

//PrintInorder prints the elements in left-current-right order.
func (t *TreeNodeV2) PrintInorder() {
	if t == nil {
		return
	}
	t.left.PrintInorder()
	fmt.Printf("%+v\n", t.val)
	t.right.PrintInorder()
}

// remove removes the element in s at index i
func removeV2(s []OrderV2, i int) []OrderV2 {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
