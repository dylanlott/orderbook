package orders

import (
	"fmt"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

type Order interface {
	ID() string
	Owner() accounts.Account
	Filled() int64  // returns the amount filled
	Price() float64 // returns the price of the amount filled.
}

// MarketOrder fulfills Order and is a record of a single order
// in our exchange.
type MarketOrder struct {
	owner accounts.Account
	price float64
}

// OrderList maintains a list of orders.
type OrderList []Order

// Place creates a new Order and adds it into the Order list.
// Accept interfaces, return concrete types.
func (ol OrderList) Place(order Order) (MarketOrder, error) {
	o := MarketOrder{
		owner: order.Owner(),
		price: order.Price(), // TODO: should this be 0 for market orders?
	}
	// TODO: add into our list of orders
	return o, nil
}

func Cancel(order Order) error {
	return fmt.Errorf("not impl")
}
