package orders

import (
	"context"
	"fmt"
	"time"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

type Order interface {
	Filler // allows different algorithms to be swapped on the Order.

	ID() string
	Owner() accounts.Account
	Timestamp() time.Time
	Filled() int64   // returns the amount filled
	Price() float64  // returns the price of the amount filled.
	Quantity() int64 // returns the number of units ordered.
}

type Filler interface {
	Fill(ctx context.Context) error
}

// MarketOrder fulfills Order and is a record of a single order
// in our exchange.
type MarketOrder struct{}

func (mo *MarketOrder) ID() string {
	return "not impl"
}

func (mo *MarketOrder) Owner() accounts.Account {
	return nil
}

// Fill returns the fill algorithm for this type of order.
func (mo *MarketOrder) Fill(ctx context.Context) error {
	return fmt.Errorf("not impl")
}

func (mo *MarketOrder) Filled() int64 {
	return 0
}

func (mo *MarketOrder) Price() float64 {
	return 0
}

func (mo *MarketOrder) Quantity() int64 {
	return 0
}

func (mo *MarketOrder) Timestamp() time.Time {
	return time.Now()
}

// Market defines the most outer API for our books.
type Market interface {
	Name() string
	Orderbook() ([]Order, error)
	Place(order Order) (Order, error)
	Cancel(orderID string) error
}

type market struct {
	Orders []Order
}

// Place creates a new Order and adds it into the Order list.
// Accept interfaces, return concrete types.
func (fm *market) Place(order Order) (Order, error) {
	if order.Owner() == nil {
		return nil, fmt.Errorf("each order must have an associated account")
	}
	// TODO: add into our list of orders
	return nil, fmt.Errorf("not impl")
}

func (fm *market) Cancel(orderID string) error {
	return fmt.Errorf("not impl")
}

func (fm *market) Fill(ctx context.Context) error {
	return fmt.Errorf("not impl")
}
