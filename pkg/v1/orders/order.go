package orders

import (
	"sync"
	"time"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

// Order defines the interface for an order in our system.
type Order interface {
	ID() string
	Side() string
	Owner() accounts.Account
	AssetInfo() AssetInfo
	Price() float64  // returns the price of the amount filled.
	Quantity() int64 // returns the number of units ordered.
	CreatedAt() time.Time
	Update(open, filled int64) (Order, error)
	Done() <-chan Order
}

// AssetInfo defines the underlying and name for an asset.
type AssetInfo struct {
	Name string
}

// MarketOrder fulfills Order and is a record of a single order
// in our exchange.
type MarketOrder struct {
	sync.Mutex

	Asset          AssetInfo
	UserAccount    *accounts.UserAccount
	UUID           string
	OpenQuantity   int64
	FilledQuantity int64
	PlacedAt       time.Time
	MarketPrice    float64

	side string
	done chan Order
}

// ID returns the MarketOrder's UUID
func (mo *MarketOrder) ID() string {
	mo.Lock()
	defer mo.Unlock()
	return mo.UUID
}

// Side returns buy or sell side
func (mo *MarketOrder) Side() string {
	mo.Lock()
	defer mo.Unlock()
	return mo.side
}

// Filled returns the number of units for the order that have been filled.
func (mo *MarketOrder) Filled() int64 {
	mo.Lock()
	defer mo.Unlock()
	return mo.FilledQuantity
}

// Price returns the market price of the market order.
func (mo *MarketOrder) Price() float64 {
	mo.Lock()
	defer mo.Unlock()
	return mo.MarketPrice
}

// Quantity returns the quantity of the asset being purchased.
func (mo *MarketOrder) Quantity() int64 {
	mo.Lock()
	defer mo.Unlock()
	return mo.OpenQuantity
}

// Owner returns the account for the order that should be charged.
func (mo *MarketOrder) Owner() accounts.Account {
	mo.Lock()
	defer mo.Unlock()
	return mo.UserAccount
}

// CreatedAt returns the time the order was created for time priority organization
func (mo *MarketOrder) CreatedAt() time.Time {
	mo.Lock()
	defer mo.Unlock()
	return mo.PlacedAt
}

// AssetInfo returns the asset information for the market order.
func (mo *MarketOrder) AssetInfo() AssetInfo {
	mo.Lock()
	defer mo.Unlock()
	return mo.Asset
}

// Update updates the open and filled quantities to the given amounts
func (mo *MarketOrder) Update(open, filled int64) (Order, error) {
	mo.Lock()
	defer mo.Unlock()

	mo.OpenQuantity = open
	mo.FilledQuantity = filled

	// Notify order completed when Open is 0
	if mo.OpenQuantity == 0 {
		mo.done <- mo
	}

	return mo, nil
}

// Done returns a channel that sends on completion of the Order.
// * Completion can mean complete fulfillment, partial fulfillment, expiration, or
// cancellation. Status and fulfillment details must be checked, this only signals
// it's not in the books anymore.
func (mo *MarketOrder) Done() <-chan Order {
	return mo.done
}
