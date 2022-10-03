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
	// Accounts maintains a reference to account balances
	Accounts accounts.AccountManager
	// BuySide is for orders buying the Asset
	BuySide *TreeNode
	// SellSide is for order selling the Asset at the Quote
	SellSide *TreeNode
	// Keeps a record of this market's asset information
	asset *AssetInfo
}

func (fm *market) Fill(ctx context.Context, fillOrder Order) {
	stopper := 0
	for fillOrder.Quantity() != 0 && stopper < 100 {
		log.Printf("fill attempt # %d", stopper)
		err := fm.attemptFill(fillOrder)
		if err != nil {
			log.Printf("FillErr: failed to fill order: %+v", err)
			return
		}
		stopper++
	}
	log.Printf("stopgap reached; stopped trying to fill order: %+v", fillOrder)
}

// attemptFill attempts to fill an order as a Limit Fill order.
// * It removes the market order from the orderbook if it fully fills
// the order.
// * TODO: Add rollback functionality. An order could currently transfer a balance and then
// fail to update the order totals, resulting in mishandled money.
// * TODO: remove the order from the order trie node once we see it's filled.
func (fm *market) attemptFill(fillOrder Order) error {
	var errorCollector []error

	if fillOrder.AssetInfo().Name == fm.asset.Name {
		log.Printf("detected buy side order: %+v", fillOrder)

		fm.SellSide.Match(fillOrder, func(bookOrder Order) {
			log.Printf("matched buy order to sell order: %+v", bookOrder)

			// should result in fillOrder being completely filled, bookOrder partial fill
			if fillOrder.Quantity() < bookOrder.Quantity() {
				wanted := fillOrder.Quantity()
				available := bookOrder.Quantity()
				left := available - wanted
				updatedFill, err := fillOrder.Update(0, wanted)
				if err != nil {
					errorCollector = append(errorCollector, fmt.Errorf("failed to update fill order: %+v", err))
				}
				// and bookOrder being partially filled.
				updatedBook, err := bookOrder.Update(left, wanted)
				if err != nil {
					errorCollector = append(errorCollector, fmt.Errorf("failed to update book order: %+v", err))
				}
				log.Printf("updated orders - fillOrder: %+v - bookOrder: %+v", updatedFill, updatedBook)
			}

			// should result in total fill for both since we have equal quantities
			if fillOrder.Quantity() == bookOrder.Quantity() {
				available := bookOrder.Quantity()
				updatedFill, err := fillOrder.Update(0, available)
				if err != nil {
					errorCollector = append(errorCollector, fmt.Errorf("failed to update fill order: %+v", err))
				}
				updatedBook, err := bookOrder.Update(0, available)
				if err != nil {
					errorCollector = append(errorCollector, fmt.Errorf("failed to update book order: %+v", err))
				}
				log.Printf("updated orders - fillOrder: %+v - bookOrder: %+v", updatedFill, updatedBook)
			}

			// should result in fillOrder being partially filled and bookOrder being totally filled.
			if fillOrder.Quantity() > bookOrder.Quantity() {
				left := fillOrder.Quantity() - bookOrder.Quantity()
				taken := bookOrder.Quantity()
				updatedFill, err := fillOrder.Update(left, taken)
				if err != nil {
					errorCollector = append(errorCollector, fmt.Errorf("failed to update fill order: %+v", err))
				}
				updatedBook, err := bookOrder.Update(0, taken)
				if err != nil {
					errorCollector = append(errorCollector, fmt.Errorf("failed to update book order: %+v", err))
				}
				log.Printf("updated orders - fillOrder: %+v - bookOrder: %+v", updatedFill, updatedBook)
			}

		})
	} else {
		// handle sell side
		log.Printf("detected sell side order: %+v", fillOrder)
		fm.BuySide.Match(fillOrder, func(bookOrder Order) {
			log.Printf("matched sell order to buy order: %+v", bookOrder)
		})
		return fmt.Errorf("sell side not impl: %+v", fillOrder)
	}

	log.Printf("Errors? %+v", errorCollector)

	return nil
}

// Place creates a new Order and adds it into the Order list.
func (fm *market) Place(order Order) (Order, error) {
	log.Printf("### Placing order: %+v", order)
	if order.Owner() == nil || order.Owner().UserID() == "" {
		return nil, fmt.Errorf("each order must have an associated account")
	}

	fm.Mutex.Lock()
	defer fm.Mutex.Unlock()

	// TODO: revisit this check; not sure this makes sense in a situation
	// where we already split the buy and sell orders
	if order.AssetInfo().Name == fm.asset.Name {
		// insert order buy side
		log.Printf("### Placing order BUY side: %+v", order)
		err := fm.BuySide.Insert(order)
		if err != nil {
			panic(err)
		}
	} else {
		// insert order sell side
		log.Printf("### Placing order SELL side: %+v", order)
		err := fm.SellSide.Insert(order)
		if err != nil {
			panic(err)
		}
	}

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
