package orders

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

// Maybe it's premature to have the Order interface.

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

	Orders    []Order
	OrderTrie *TreeNode
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
