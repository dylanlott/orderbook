package orders

import (
	"time"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

// Order defines the interface for an order in our system.
type Order interface {
	ID() string
	Owner() accounts.Account
	AssetInfo() AssetInfo
	Price() float64  // returns the price of the amount filled.
	Quantity() int64 // returns the number of units ordered.
	CreatedAt() time.Time
	// Update updates the open and filled quantities to the given amounts
	Update(open, filled int64) (Order, error)
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

func (mo *MarketOrder) Update(open, filled int64) (Order, error) {
	mo.OpenQuantity = open
	mo.FilledQuantity = filled
	return mo, nil
}
