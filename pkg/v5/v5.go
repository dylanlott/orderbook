package v5

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// The idea here is to use channels to guard reads and writes to the orderbook.

// OpRead gets an Order from the book.
type OpRead struct {
	side   string
	price  uint64
	result chan *Order
}

// OpWrite inserts an order into the Book
type OpWrite struct {
	side   string
	order  Order
	result chan Order
}

// Order is a struct for representing a simple order in the books.
type Order struct {
	ID       string
	Kind     string
	Side     string
	Price    uint64
	Open     uint64
	Filled   uint64
	Metadata map[string]string
	History  []Match
}

// Match holds a buy and a sell side order
type Match struct {
	Buy  *Order
	Sell *Order
}

// Book holds buy and sell side orders. OpRead and OpWrite are applied to
// to the book. Buy and sell side orders are kept as sorted lists orders.
type Book struct {
	buy  *PriceNode
	sell *PriceNode
}

// Listen takes a reads and a writes channel that it reads from an applies
// those updates to the Book.
func Listen(ctx context.Context, reads chan OpRead, writes chan OpWrite, output chan *Book, matches chan Match, errs chan error) {
	// book is protected by the Listen function.
	book := &Book{
		buy: &PriceNode{
			val:    0,
			orders: []*Order{},
			right:  &PriceNode{},
			left:   &PriceNode{},
		},
		sell: &PriceNode{
			val:    0,
			orders: []*Order{},
			right:  &PriceNode{},
			left:   &PriceNode{},
		},
	}

	for {
		select {
		case <-ctx.Done():
			// TODO: drain channels and cleanup
			return
		case r := <-reads:
			if r.side == "buy" {
				found, err := book.buy.Find(r.price)
				if err != nil {
					errs <- err
					continue
				}
				r.result <- found
				output <- book
			} else {
				found, err := book.sell.Find(r.price)
				if err != nil {
					errs <- err
					continue
				}
				r.result <- found
				output <- book
			}
			// try to match
		case w := <-writes:
			if w.side == "buy" {
				err := book.buy.Insert(w.order)
				if err != nil {
					errs <- err
					continue
				}
				// attempt to match
				w.result <- w.order
				output <- book
			} else {
				err := book.sell.Insert(w.order)
				if err != nil {
					errs <- err
					continue
				}
				w.result <- w.order
				output <- book
			}
		default:
			fmt.Println("\n===========================buy side\n=================================")
			book.buy.Print()
			fmt.Println("\n===========================sell side\n=================================")
			book.sell.Print()
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// Run listens sets up the reads and writes and listens for them on the book.
func Run() {
	ctx := context.Background()
	reads := make(chan OpRead)
	writes := make(chan OpWrite)
	out := make(chan *Book)
	matches := make(chan Match)
	errs := make(chan error)

	go Listen(ctx, reads, writes, out, matches, errs)

	for update := range out {
		log.Printf("[update] %+v", update)
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
	sync.Mutex

	val    uint64
	orders []*Order
	right  *PriceNode
	left   *PriceNode
}

// List grabs a lock on the whole tree and reads all of the orders from it.
func (t *PriceNode) List() ([]*Order, error) {
	t.Lock()
	defer t.Unlock()

	orders := []*Order{}
	stack := []*PriceNode{}
	var current *PriceNode
	for t != nil || len(stack) > 0 {
		if current != nil {
			stack = append(stack, t)
			current = current.left
		} else {
			current = stack[len(stack)-1]
			stack = stack[:len(stack)-1]
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
		t = &PriceNode{val: o.Price}
	}

	if t.val == o.Price {
		// when we find a price match for the Order's price,
		// insert the Order into this node's Order list.
		// lock because we're about to alter coure resource.

		t.Lock()
		if t.orders == nil {
			t.orders = make([]*Order, 0)
		}
		t.orders = append(t.orders, &o)
		t.Unlock()
		return nil
	}

	if o.Price < t.val {
		t.Lock()
		if t.left == nil {
			t.left = &PriceNode{val: o.Price}
			t.Unlock()
			return t.left.Insert(o)
		}
		t.Unlock()
		return t.left.Insert(o)
	}
	// and PREVIOUS WRITE HERE
	t.Lock()
	if t.right == nil {
		t.right = &PriceNode{val: o.Price}
		t.Unlock()
		return t.right.Insert(o)
	}

	t.Unlock()
	return t.right.Insert(o)
}

// Find returns the highest priority Order for a given price point.
// * If it can't find an order at that exact price, it will search for
// a cheaper order if one exists.
func (t *PriceNode) Find(price uint64) (*Order, error) {
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

		return nil, fmt.Errorf("no orders at this price")
	}

	if t.left != nil {
		return t.left.Find(price)
	}

	return nil, fmt.Errorf("no orders at this price")
}

// Match will iterate through the tree based on the price of the
// fillOrder and finds a bookOrder that matches its price.
func (t *PriceNode) Match(fillOrder *Order, cb func(bookOrder *Order)) {
	if t == nil {
		cb(nil)
		return
	}

	if fillOrder.Price == t.val {
		// callback with first order in the list
		bookOrder := t.orders[0]
		cb(bookOrder)
		return
	}

	if fillOrder.Price > t.val {
		if t.right != nil {
			t.right.Match(fillOrder, cb)
			return
		}
	}

	if fillOrder.Price < t.val {
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
func (t *PriceNode) Pull(price uint64) (*Order, error) {
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
func (t *PriceNode) Orders(price uint64) ([]*Order, error) {
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

// RemoveByID removes an order from the list of orders at a
// given price in our tree. It does not currently rebalance the tree.
// TODO: make this rebalance the tree at some threshold.
func (t *PriceNode) RemoveByID(order *Order) (*Order, error) {
	if t == nil {
		return nil, fmt.Errorf("order tree is nil")
	}

	t.Lock()
	if order.Price == t.val {
		for i, ord := range t.orders {
			if ord.ID == order.ID {
				t.orders = removeV2(t.orders, i)
				t.Unlock()
				return ord, nil
			}
		}
		return nil, fmt.Errorf("ErrNoExist")
	}

	if order.Price > t.val {
		if t.right != nil {
			t.Unlock()
			return t.right.RemoveByID(order)
		}

		t.Unlock()
		return nil, fmt.Errorf("ErrNoExist")
	}

	if t.left != nil {
		t.Unlock()
		return t.left.RemoveByID(order)
	}

	t.Unlock()
	return nil, fmt.Errorf("ErrNoExist")
}

// Print prints the elements in left-current-right order.
func (t *PriceNode) Print() {
	if t == nil {
		return
	}

	// WRITE HERE
	t.Lock()
	defer t.Unlock()

	t.left.Print()
	fmt.Printf("%+v\n", t.val)
	t.right.Print()
}

// remove removes the element in s at index i
func removeV2(s []*Order, i int) []*Order {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
