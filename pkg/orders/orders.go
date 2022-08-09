package orders

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

// Order defines the interface for an order in our system.
type Order interface {
	Filler // allows different algorithms to be swapped on the Order.

	ID() string
	Owner() accounts.Account
	Timestamp() time.Time
	Filled() int64   // returns the amount filled
	Price() float64  // returns the price of the amount filled.
	Quantity() int64 // returns the number of units ordered.
}

// Filler defines an extensible function for filling orders.
type Filler interface {
	Fill(ctx context.Context) error
}

// Asset defines the underlying and name for an asset.
type Asset struct {
	Underlying string
	Name       string
}

// MarketOrder fulfills Order and is a record of a single order
// in our exchange.
type MarketOrder struct {
	Asset          Asset
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

// Fill returns the fill algorithm for this type of order.
func (mo *MarketOrder) Fill(ctx context.Context) error {
	return fmt.Errorf("not impl")
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

// Timestamp returns the time the order was placed.
func (mo *MarketOrder) Timestamp() time.Time {
	return mo.PlacedAt
}

// Owner returns the account for the order that should be charged.
func (mo *MarketOrder) Owner() accounts.Account {
	return mo.UserAccount
}

// Market defines the most outer API for our books.
type Market interface {
	Name() string
	Orderbook() ([]Order, error)
	Place(order Order) (Order, error)
	Cancel(orderID string) error
}

// market manages a set of Orders.
type market struct {
	sync.Mutex

	Orders []Order
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
	fm.Mutex.Unlock()

	log.Printf("placed order [%+v]", order)

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

// Fill is the strategy for filling Market orders.
func (fm *market) Fill(ctx context.Context) error {
	return fmt.Errorf("not impl")
}

// remove removes the element in s at index i
func remove(s []Order, i int) []Order {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
