package orders

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

// Maybe it's premature to have the Order interface.

// Order defines the interface for an order in our system.
type Order interface {
	ID() string
	Owner() accounts.Account
	AssetInfo() AssetInfo
	Price() float64  // returns the price of the amount filled.
	Quantity() int64 // returns the number of units ordered.
	CreatedAt() time.Time
}

// Filler defines an extensible function for filling orders.
// It is called as a goroutine.
type Filler interface {
	Fill(ctx context.Context)
}

// AssetInfo defines the underlying and name for an asset.
type AssetInfo struct {
	Underlying string
	Name       string
}

// MarketOrder fulfills Order and is a record of a single order
// in our exchange.
type MarketOrder struct {
	Asset          AssetInfo
	UserAccount    *accounts.UserAccount
	UUID           string
	OpenQuantity   int64
	FilledQuantity int64
	PlacedAt       time.Time
	MarketPrice    float64
}

// ID returns the MarketOrder's UUID
func (mo *MarketOrder) ID() string {
	return mo.UUID
}

// Filled returns the number of units for the order that have been filled.
func (mo *MarketOrder) Filled() int64 {
	return mo.FilledQuantity
}

// Price returns the market price of the market order.
func (mo *MarketOrder) Price() float64 {
	return mo.MarketPrice
}

// Quantity returns the quantity of the asset being purchased.
func (mo *MarketOrder) Quantity() int64 {
	return mo.OpenQuantity
}

// Owner returns the account for the order that should be charged.
func (mo *MarketOrder) Owner() accounts.Account {
	return mo.UserAccount
}

// CreatedAt returns the time the order was created for time priority organization
func (mo *MarketOrder) CreatedAt() time.Time {
	return mo.PlacedAt
}

// AssetInfo returns the asset information for the market order.
func (mo *MarketOrder) AssetInfo() AssetInfo {
	return mo.Asset
}

// Market defines the most outer API for our books.
type Market interface {
	Name() string
	Orderbook() ([]Order, error)
	Place(order Order) (Order, error)
	Cancel(orderID string) error
}

// TreeNode represents a tree of nodes that maintain lists of Orders at that price.
type TreeNode struct {
	val    float64 // to represent price
	orders []Order
	right  *TreeNode
	left   *TreeNode
}

// market manages a set of Orders.
type market struct {
	sync.Mutex

	Accounts accounts.AccountManager

	Orders    []Order
	OrderTrie *TreeNode
}

// Insert will add an Order to the Tree.
func (t *TreeNode) Insert(o Order) error {
	if t == nil {
		t = &TreeNode{val: o.Price()}
	}

	if t.val == o.Price() {
		// when we find a price match for the order,
		// insert the order into this node's order list.
		if t.orders == nil {
			t.orders = make([]Order, 0)
		}
		t.orders = append(t.orders, o)
		return nil
	}

	if t.val > o.Price() {
		if t.left == nil {
			t.left = &TreeNode{val: o.Price()}
			return t.left.Insert(o)
		}
		return t.left.Insert(o)
	}

	if t.val < o.Price() {
		if t.right == nil {
			t.right = &TreeNode{val: o.Price()}
			return t.right.Insert(o)
		}
		return t.right.Insert(o)
	}

	panic("should not get here; this smells like a bug")
}

// Find returns the highest priority order for a given price point.
// It returns the Order or an error.
// * If it can't find an order at that exact price, it will search for
// a cheaper order if one exists.
func (t *TreeNode) Find(price float64) (Order, error) {
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

	return nil, fmt.Errorf("ErrFind")
}

//PrintInorder prints the elements in left-current-right order.
func (t *TreeNode) PrintInorder() {
	if t == nil {
		return
	}
	t.left.PrintInorder()
	fmt.Printf("%+v\n", t.val)
	t.right.PrintInorder()
}

// sortByTimePriority sorts orders by oldest to newest
func sortByTimePriority(orders []Order) []Order {
	sort.SliceStable(orders, func(i, j int) bool {
		return orders[i].CreatedAt().After(orders[j].CreatedAt())
	})
	return orders
}

// Fill returns the fill algorithm for this type of order.
func (fm *market) Fill(ctx context.Context, fillOrder Order) {
	// this function fulfills a fillOrder in a limit fill fashion
	log.Printf("attempting to fill order [%+v]", fillOrder)
	// NB: naive implementation: loop until we find a match and then fill.
	// loop until we fill this order
	for {
		// loop over the orders repeatedly until filled
		for _, bookOrder := range fm.Orders {
			// detect an order that fits our criteria
			if fillOrder.AssetInfo().Name == bookOrder.AssetInfo().Underlying {
				// ### Buy Side Order

				fillerBalance := fillOrder.Owner().Balance()
				total := float64(fillOrder.Quantity()) * fillOrder.Price()

				if total > fillerBalance {
					log.Printf("insufficient balance, unable to fill")
					return
				}

				// TODO: Order's should have some functionality to mark them as filled
				// so that we avoid having to hard-cast them.
				// This hard-cast is an abstraction leakage because it relies on the concrete type.
				mo, ok := fillOrder.(*MarketOrder)
				if !ok {
					panic("failed to cast fillOrder as MarketOrder")
				}

				// attempt to transfer balances.
				_, err := fm.Accounts.Tx(fillOrder.Owner().UserID(), bookOrder.Owner().UserID(), total)
				if err != nil {
					log.Printf("transaction failed: %s", err)
					return
				}

				mo.OpenQuantity = 0
				mo.FilledQuantity = fillOrder.Quantity()

				// TODO: remove mo from open orders

				return
			}
		}
	}
	// TODO: send on channel when filled?
}

// Place creates a new Order and adds it into the Order list.
// Accept interfaces, return concrete types.
func (fm *market) Place(order Order) (Order, error) {
	if order.Owner() == nil {
		return nil, fmt.Errorf("each order must have an associated account")
	}

	log.Printf("order owner [%+v]", order.Owner().UserID())

	fm.Mutex.Lock()

	fm.Orders = append(fm.Orders, order)
	fm.OrderTrie.Insert(order)

	fm.Mutex.Unlock()

	log.Printf("placed order [%+v]", order)

	go fm.Fill(context.TODO(), order)

	return order, nil
}

// Cancel will remove the order from the books.
func (fm *market) Cancel(orderID string) error {
	for i, ord := range fm.Orders {
		if ord.ID() == orderID {
			fm.Lock()
			fm.Orders = remove(fm.Orders, i)
			fm.Unlock()
			return nil
		}
	}

	return fmt.Errorf("failed to find order %s to cancel", orderID)
}

// remove removes the element in s at index i
func remove(s []Order, i int) []Order {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
