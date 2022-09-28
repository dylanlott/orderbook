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

func (fm *market) Fill(ctx context.Context, fillOrder Order) {
	for {
		fm.OrderTrie.Match(fillOrder, func(bookOrder Order) {
			if err := fm.attemptFill(fillOrder, bookOrder); err != nil {
				log.Printf("attemptFill failed: %v", err)
			}
			// TODO: break and return if fill order is filled
			// TODO: send on channel to alert when filled
			if fillOrder.Quantity() == 0 {
				return
			}
		})
	}
}

// attemptFill attempts to fill an order as a Limit Fill order.
// * It removes the market order from the orderbook if it fully fills
// the order.
// * TODO: Add rollback functionality. An order could currently transfer a balance and then
// fail to update the order totals, resulting in mishandled money.
// * TODO: remove the order from the order trie node once we see it's filled.
func (fm *market) attemptFill(fillOrder, bookOrder Order) error {
	// TODO: keep buy and sell side orders separately so as to avoid this check.
	if fillOrder.AssetInfo().Name == bookOrder.AssetInfo().Underlying {
		total := float64(fillOrder.Quantity()) * fillOrder.Price()

		if bookOrder.Quantity() < fillOrder.Quantity() {
			return fmt.Errorf("partial fills not implemented") // TODO
		}

		// attempt to transfer balance from buyer to seller.
		// this can fail, so we want to do this before we update order information.
		_, err := fm.Accounts.Tx(fillOrder.Owner().UserID(), bookOrder.Owner().UserID(), total)
		if err != nil {
			return fmt.Errorf("transaction failed: %s", err)
		}

		// update the order quantities after we've successfully transferred balances.
		bookOrderOpen := bookOrder.Quantity() - fillOrder.Quantity()
		bookOrderFilled := fillOrder.Quantity() - bookOrder.Quantity()

		// update order fill quantity in order trie
		_, err = fillOrder.Update(bookOrderFilled, bookOrderOpen)
		if err != nil {
			log.Printf("error updating order %s: %s", fillOrder.ID(), err)
		}
		_, err = bookOrder.Update(bookOrderOpen, bookOrderFilled)
		if err != nil {
			log.Printf("error updating order %s: %s", fillOrder.ID(), err)
		}

		if fillOrder.Quantity() == 0 {
			if err := fm.OrderTrie.RemoveFromPriceList(fillOrder); err != nil {
				log.Printf("failed to remove order %s: %+v", fillOrder.ID(), err)
			}
		}

		if bookOrder.Quantity() == 0 {
			if err := fm.OrderTrie.RemoveFromPriceList(bookOrder); err != nil {
				log.Printf("failed to remove order %s: %+v", bookOrder.ID(), err)
			}
		}

		log.Printf("order filled %+v", fillOrder)

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
