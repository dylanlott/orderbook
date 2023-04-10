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
	OwnerID() string
	ID() string
	Price() int64
	Side() string
	Open() uint64

	// Fill adds a transaction that ties a Transaction event to an Account.
	Fill(tx *Transaction) ([]*Transaction, error)
	History() []*Transaction
}

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

// Transaction the data for a trade.
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

// Worker defines a function meant to be spawned concurrently that listens to the in
// channel and fills orders as they are received and processes them in their own
// gouroutine.
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
			// listenForCompletion(out, order)
		}(o)

	}
}

// Filler handles txs from the input channel and passes them to the orderbook with a
// reference which returns their output on the out channel upon completion or cancellation.
func Filler(in <-chan Order, out chan<- Order, book *Orderbook) {
	// make this constantly walk the tree
	for {
		book.Buy.Print()
		time.Sleep(time.Second * 2)
	}
}

// func listenForCompletion(completed chan<- Order, order Order) {
// 	// TODO: This should block until we're filled
// 	completed <- order
// 	// log.Printf("TODO: Wait for Order to Fill")
// }

///////////////
// Orderbook //
///////////////

// Orderbook is worked on by Workers.
// TODO: Turn this into an interface to abstract away the underlying data structure
type Orderbook struct {
	Buy  *PriceNode
	Sell *PriceNode

	Accounts accounts.AccountManager
}

// Pull routes a pull to a price and a side of the Orderbook.
// It returns that Order or an order.
func (o *Orderbook) Pull(price int64, side string) (Order, error) {
	if side == BUY {
		return o.Buy.Pull(price)
	}
	return o.Sell.Pull(price)
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
	}

	err := o.Sell.Insert(order)
	if err != nil {
		return fmt.Errorf("failed to add order to the book: %w", err)
	}

	// TODO: kick off order for filling

	return nil
}

// Match will match a Buy to a Sell and attempts to charge the buyer.
// TODO: ensure this is atomic.
func (o *Orderbook) Match(buyOrder Order) (Order, error) {
	if buyOrder.Side() == BUY {
		sellOrder, err := o.Sell.Find(buyOrder.Price())
		if err != nil {
			return nil, err
		}

		buyer, err := o.Accounts.Get(buyOrder.OwnerID())
		if err != nil {
			return nil, err
		}
		seller, err := o.Accounts.Get(sellOrder.OwnerID())
		if err != nil {
			return nil, err
		}

		if buyOrder.Open() < sellOrder.Open() {
			log.Printf("order wants %v - found order has %v", buyOrder.Open(), sellOrder.Open())
			return nil, fmt.Errorf("partial fills not impl")
		}

		available := sellOrder.Open()
		if buyOrder.Open() >= available {
			// NB: be careful, this is lossy precision.
			total := available * uint64(sellOrder.Price())

			// Attempt to transfer balances
			_, err := o.Accounts.Tx(buyer.UserID(), seller.UserID(), float64(total))
			if err != nil {
				return buyOrder, err
			}

			//NB: These two Fill calls could potentially fail and leave us in a
			// weird state. We need to figure out how to make this atomic.
			// Maybe orders should be kept in a simpler data store?

			// Add a record to the Sell side transaction pointing to the Buyer.
			_, err = sellOrder.Fill(&Transaction{
				AccountID: buyOrder.OwnerID(),
				Quantity:  available,
				Price:     uint64(sellOrder.Price()),
				Total:     uint64(sellOrder.Price()) * available,
			})
			if err != nil {
				return buyOrder, fmt.Errorf("failed to fill sell side order: %+v", err)
			}

			// Add record of the Sell side asset to the Buy side order.
			_, err = buyOrder.Fill(&Transaction{
				AccountID: sellOrder.OwnerID(),
				Quantity:  buyOrder.Open(),
				Price:     uint64(sellOrder.Price()),
				Total:     total,
			})
			if err != nil {
				log.Printf("failed to update account: %v", err)
				return buyOrder, fmt.Errorf("failed to add order fill transaction receipt: %+v", err)
			}

			// cleanup orders from books if successfully transferred funds.
			if buyOrder.Open() == 0 {
				_, err := o.Buy.RemoveByID(buyOrder)
				if err != nil {
					log.Printf("failed to remove order from books: %v", err)
				}
			}
			if sellOrder.Open() == 0 {
				_, err := o.Sell.RemoveByID(sellOrder)
				if err != nil {
					log.Printf("failed to remove order from books: %v", err)
				}
			}

			return buyOrder, nil
		}
	} else {
		return nil, fmt.Errorf("ErrSellSide")
	}

	return nil, fmt.Errorf("not impl")
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

// OwnerID Returns the owner ID of this order. It maps to the account ID.
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

// orderTree defines the function for interacting with a tree that holds Order types.
// TODO: Design a generics interface implementation for this.
type orderTree interface {
	// List grabs and sorts every order in the books and returns the processed list
	List() ([]Order, error)
	// Insert adds an order to the books and returns an error if that failed.
	Insert(o Order) error
	// Find returns the order or an error but it does not remove the order from the book.
	// This is for seeing the current price orders for example and other queries on the books.
	Find(price int64) (Order, error)
	// Pull finds an order at a given price and returns it or an error. If it errors,
	// it does alter the books.
	Pull(price int64) (Order, error)
	// Match is a callback function that gives you access to the order in the tree
	// It gives you access to both, but removal of the order should happen by calling Pull.
	Match(fillOrder Order, cb func(bookOrder Order))
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
		if t.orders == nil {
			t.orders = make([]Order, 0)
		}
		t.orders = append(t.orders, o)
		t.Unlock()
		return nil
	}

	if o.Price() < t.val {
		t.Lock()
		if t.left == nil {
			t.left = &PriceNode{val: o.Price()}
			t.Unlock()
			return t.left.Insert(o)
		}
		t.Unlock()
		return t.left.Insert(o)
	}
	// and PREVIOUS WRITE HERE
	t.Lock()
	if t.right == nil {
		t.right = &PriceNode{val: o.Price()}
		t.Unlock()
		return t.right.Insert(o)
	}

	t.Unlock()
	return t.right.Insert(o)
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

		return nil, fmt.Errorf("no orders at this price")
	}

	if t.left != nil {
		return t.left.Find(price)
	}

	return nil, fmt.Errorf("no orders at this price")
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
func (t *PriceNode) RemoveByID(order Order) (Order, error) {
	if t == nil {
		return nil, fmt.Errorf("order tree is nil")
	}

	t.Lock()
	if order.Price() == t.val {
		for i, ord := range t.orders {
			if ord.ID() == order.ID() {
				t.orders = removeV2(t.orders, i)
				t.Unlock()
				return ord, nil
			}
		}
		return nil, fmt.Errorf("ErrNoExist")
	}

	if order.Price() > t.val {
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
func removeV2(s []Order, i int) []Order {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
