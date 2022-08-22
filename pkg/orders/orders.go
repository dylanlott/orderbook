package orders

import (
	"context"
	"fmt"
	"log"
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
	Timestamp() time.Time
	Filled() int64   // returns the amount filled
	Price() float64  // returns the price of the amount filled.
	Quantity() int64 // returns the number of units ordered.
	OrderType() string
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

// Timestamp returns the time the order was placed.
func (mo *MarketOrder) Timestamp() time.Time {
	return mo.PlacedAt
}

// Owner returns the account for the order that should be charged.
func (mo *MarketOrder) Owner() accounts.Account {
	return mo.UserAccount
}

// AssetInfo returns the asset information for the market order.
func (mo *MarketOrder) AssetInfo() AssetInfo {
	return mo.Asset
}

// OrderType returns the type of order
func (mo *MarketOrder) OrderType() string {
	// TODO: this is a code smell, handle this with a type or interface.
	if mo.Asset.Name == "USD" {
		return "BUY"
	}
	return "SELL"
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

	Accounts accounts.AccountManager
	Orders   []Order
}

// Fill returns the fill algorithm for this type of order.
func (fm *market) Fill(ctx context.Context, order Order) {
	log.Printf("attempting to fill order [%+v]", order)
	// NB: naive implementation: loop until we find a match and then fill.
	// loop until we fill this order
	for {
		// loop over the orders repeatedly
		for _, ord := range fm.Orders {
			// detect an order that fits our criteria
			if order.AssetInfo().Name == ord.AssetInfo().Underlying {
				// ### Buy Side Order
				log.Printf("asset info: %v", order.AssetInfo())
				log.Printf("detected buy-side match: %v", order)

				// TODO: attach accounts to the market

				// we need to clear up the Asset field and method name collision
				// Then we need to distinguish between buy and sell side orders.
				// Then we need to compare buy vs sell side orders that match a price
				// and if all of those conditions are met, we need to determine the quantity.
				// this becomes a packing problem at this point.
				// if unfilled units, continue filling by recursing through this function.
				// if enough exists for it to fill itself up, then it should signal done.
				// it should then atomically charge the buyer and pay the seller.
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
	// TODO: upgrade to a trie structure for faster searching.
	fm.Orders = append(fm.Orders, order)
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
