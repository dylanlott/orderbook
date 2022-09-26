package orders

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

// Market defines the most outer API for our books.
type Market interface {
	Name() string
	Orderbook() ([]Order, error)
	Place(order Order) (Order, error)
	Cancel(orderID string) error
}

// Filler defines an extensible function for filling orders.
// It is called as a goroutine.
type Filler interface {
	Fill(ctx context.Context, o Order)
}

// market manages a set of Orders.
type market struct {
	sync.Mutex

	Accounts accounts.AccountManager

	OrderTrie *TreeNode
}

// Fill returns the fill algorithm for this type of order.
func (fm *market) Fill(ctx context.Context, fillOrder Order) {
	// this function fulfills a fillOrder in a limit fill fashion
	log.Printf("attempting to fill order [%+v]", fillOrder)

	for {
		fm.OrderTrie.Match(fillOrder, func(bookOrder Order) {
			if err := fm.attemptFill(fillOrder, bookOrder); err != nil {
				log.Printf("attemptFill failed: %v", err)
			}

			// TODO: break and return if fill order is filled
			// TODO: send on channel to alert when filled
			return
		})
	}
}

// attemptFill attempts to fill an order as a Limit Fill order.
// * It removes the market order from the orderbook if it fully fills
// the order.
func (fm *market) attemptFill(fillOrder, bookOrder Order) error {
	// TODO: keep buy and sell side orders separately so as to avoid this check.
	if fillOrder.AssetInfo().Name == bookOrder.AssetInfo().Underlying {
		total := float64(fillOrder.Quantity()) * fillOrder.Price()

		if bookOrder.Quantity() < fillOrder.Quantity() {
			return fmt.Errorf("partial fills not implemented") // TODO
		}

		bookOrderOpen := bookOrder.Quantity() - fillOrder.Quantity()
		bookOrderFilled := fillOrder.Quantity() - bookOrder.Quantity()

		// attempt to transfer balances.
		accts, err := fm.Accounts.Tx(fillOrder.Owner().UserID(), bookOrder.Owner().UserID(), total)
		if err != nil {
			return fmt.Errorf("transaction failed: %s", err)
		}

		log.Printf("transferred balances: %v", accts)

		_, err = fillOrder.Update(bookOrderFilled, bookOrderOpen)
		_, err = bookOrder.Update(bookOrderOpen, bookOrderFilled)

		// TODO: remove mo from open orders
		// err = fm.OrderTrie.Remove(mo.ID())
		// if err != nil {
		// return fmt.Errorf("failed to remove order from orderbook: %+v", err)
		// }

		return nil
	}

	return nil
}

// Place creates a new Order and adds it into the Order list.
func (fm *market) Place(order Order) (Order, error) {
	if order.Owner() == nil {
		return nil, fmt.Errorf("each order must have an associated account")
	}

	fm.Mutex.Lock()
	fm.OrderTrie.Insert(order)
	fm.Mutex.Unlock()

	go fm.Fill(context.TODO(), order)

	return order, nil
}

// Cancel will remove the order from the books.
func (fm *market) Cancel(orderID string) error {
	return fmt.Errorf("failed to cancel order %s", orderID)
}

// remove removes the element in s at index i
func remove(s []Order, i int) []Order {
	s[i] = s[len(s)-1]
	return s[:len(s)-1]
}
