package orders

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

// TODO: add a leveled logger

// Market defines the most outer API for our books.
type Market interface {
	Name() string
	Orderbook() ([]Order, error)
	Place(order Order) (Order, error)
	Cancel(orderID string) error
}

// Filler is concurrently called passing the order and a context
// and is meant to handle the order from start to finish.
type Filler interface {
	Fill(ctx context.Context, o Order)
}

// market manages a set of Orders.
// It fulfills the Filler and Market interfaces.
type market struct {
	sync.Mutex
	// Accounts maintains a reference to account balances
	Accounts accounts.AccountManager
	// BuySide is for orders buying the Asset
	BuySide *TreeNode
	// SellSide is for order selling the Asset at the Quote
	SellSide *TreeNode
}

// Fill implements the filler function for *market. It repeatedly calls
// attemptFill on the Order until completion or cancellation.
func (fm *market) Fill(ctx context.Context, fillOrder Order) {
	stopper := 0
	for fillOrder.Quantity() != 0 && stopper < 100 {
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
// * TODO: Add rollback or atomic functionality.
// An order could currently transfer a balance and then
// fail to update the order totals, resulting in mishandled money.
func (fm *market) attemptFill(fillOrder Order) error {
	var fillErr error

	if fillOrder.Side() == "BUY" {

		fm.Lock()
		defer fm.Unlock()

		fm.SellSide.Match(fillOrder, func(bookOrder Order) {
			// should result in fillOrder being completely filled, bookOrder partial fill
			if fillOrder.Quantity() < bookOrder.Quantity() {
				fillErr = fm.handleWantLess(fillOrder, bookOrder)
			}

			// should result in total fill for both since we have equal quantities
			if fillOrder.Quantity() == bookOrder.Quantity() {
				fillErr = fm.handleEqualWant(fillOrder, bookOrder)
			}

			// should result in fillOrder being partially filled and bookOrder being totally filled.
			if fillOrder.Quantity() > bookOrder.Quantity() {
				fillErr = fm.handleWantMore(fillOrder, bookOrder)
			}
		})
	} else {
		// handle sell side
		fm.BuySide.Match(fillOrder, func(bookOrder Order) {
			log.Printf("matched sell order to buy order: %+v", bookOrder)
		})
		return fmt.Errorf("sell side not impl: %+v", fillOrder)
	}

	return fillErr
}

// handleWantLess ...
func (fm *market) handleWantLess(fillOrder, bookOrder Order) error {
	wanted := fillOrder.Quantity()
	available := bookOrder.Quantity()
	left := available - wanted

	// TODO: upgrade from float64 to integer-only handling
	total := float64(wanted) * bookOrder.Price()
	_, err := fm.Accounts.Tx(fillOrder.Owner().UserID(), bookOrder.Owner().UserID(), total)
	if err != nil {
		return fmt.Errorf("failed to transfer balances: %+v", err)
	}

	// complete fill of fillOrder
	updatedFill, err := fillOrder.Update(0, wanted)
	if err != nil {
		return fmt.Errorf("failed to update fill order: %+v", err)
	}

	// remove the fillOrder since it is now considered filled
	if err := fm.BuySide.RemoveFromPriceList(fillOrder); err != nil {
		return fmt.Errorf("failed to remove order %s from buy side: %+v", fillOrder.ID(), err)
	}

	// and bookOrder being partially filled.
	updatedBook, err := bookOrder.Update(left, wanted)
	if err != nil {
		return fmt.Errorf("failed to update fill order: %+v", err)
	}

	log.Printf("updated orders - fillOrder: %+v\nbookOrder: %+v", updatedFill, updatedBook)
	return nil
}

// handleEqualWant ...
func (fm *market) handleEqualWant(fillOrder, bookOrder Order) error {
	wanted := fillOrder.Quantity()
	available := bookOrder.Quantity()

	// TODO: upgrade form float64 to integer-only handling
	total := float64(wanted) * bookOrder.Price()
	_, err := fm.Accounts.Tx(fillOrder.Owner().UserID(), bookOrder.Owner().UserID(), total)
	if err != nil {
		return fmt.Errorf("failed to update fill order: %+v", err)
	}

	updatedFill, err := fillOrder.Update(0, available)
	if err != nil {
		return fmt.Errorf("failed to update fill order: %+v", err)
	}
	updatedBook, err := bookOrder.Update(0, available)
	if err != nil {
		return fmt.Errorf("failed to update fill order: %+v", err)
	}

	if err := fm.SellSide.RemoveFromPriceList(updatedBook); err != nil {
		return fmt.Errorf("failed to remove book order %s form sell side: %+v", bookOrder.ID(), err)
	}
	if err := fm.BuySide.RemoveFromPriceList(fillOrder); err != nil {
		return fmt.Errorf("failed to remove order %s from buy side: %+v", fillOrder.ID(), err)
	}

	log.Printf("updated orders - fillOrder: %+v\nbookOrder: %+v", updatedFill, updatedBook)
	return nil
}

// handleWantMore handles the case where the fill order wants more
// than is available in the bookOrder
// This function should be made atomic but is not currently atomic.
func (fm *market) handleWantMore(fill, book Order) error {
	left := fill.Quantity() - book.Quantity()
	taken := book.Quantity()
	wanted := float64(book.Quantity()) * book.Price()

	total := float64(wanted) * book.Price()
	_, err := fm.Accounts.Tx(fill.Owner().UserID(), book.Owner().UserID(), total)
	if err != nil {
		return fmt.Errorf("failed to update fill order: %+v", err)
	}

	updatedFill, err := fill.Update(left, taken)
	if err != nil {
		return fmt.Errorf("failed to update fill order: %+v", err)
	}

	updatedBook, err := book.Update(0, taken)
	if err != nil {
		return fmt.Errorf("failed to update book order: %+v", err)
	}

	if err := fm.SellSide.RemoveFromPriceList(updatedBook); err != nil {
		return fmt.Errorf("failed to remove book order %s form sell side: %+v", book.ID(), err)
	}

	log.Printf("updated orders - fillOrder: %+v\nbookOrder: %+v", updatedFill, updatedBook)
	return nil
}

// Place creates a new Order and adds it into the Order list.
func (fm *market) Place(order Order) (Order, error) {
	log.Printf("order placed %+v", order)
	if order.Owner() == nil || order.Owner().UserID() == "" {
		return nil, fmt.Errorf("each order must have an associated account")
	}

	fm.Mutex.Lock()
	defer fm.Mutex.Unlock()

	if order.Side() == "BUY" {
		// insert order buy side
		err := fm.BuySide.Insert(order)
		if err != nil {
			panic(err)
		}
	} else {
		// insert order sell side
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
