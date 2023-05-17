package orderbook

import (
	"context"
	"fmt"
	"log"

	"github.com/dylanlott/orderbook/pkg/accounts"
	"github.com/sasha-s/go-deadlock"
)

// Book holds buy and sell side orders. OpRead and OpWrite are applied to
// to the book. Buy and sell side orders are binary trees of order lists.
type Book struct {
	// sync.RWMutex
	deadlock.Mutex

	buy  *Node
	sell *Node
}

// Start sets up the order book and wraps it in a read and write channel for
// receiving operations and output, match, and errs channels for
// handling outputs from the machine.
// The book itself is protected by this function and is intentionally never directly accessible.
func Start(
	ctx context.Context,
	accts accounts.AccountManager,
	writes chan OpWrite,
	fills chan FillResult,
	errs chan error,
) {
	matches := make(chan Match)

	// TODO: load the book in from a badger store.
	book := &Book{
		buy: &Node{
			Price:  0,
			Orders: []*Order{},
			Right:  &Node{},
			Left:   &Node{},
		},
		sell: &Node{
			Price:  0,
			Orders: []*Order{},
			Right:  &Node{},
			Left:   &Node{},
		},
	}

	go func() {
		for m := range matches {
			// execute on matches
			log.Printf("[match]: %+v\n", m)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			// TODO: drain channels and cleanup
			return
		case w := <-writes:
			if w.Order.Side == "buy" {
				o := &w.Order
				book.buy.Insert(o)
				go AttemptFill(book, accts, o, matches, errs)
				w.Result <- WriteResult{
					Order: *o,
					Err:   nil,
				}
			} else {
				o := &w.Order
				book.sell.Insert(o)
				go AttemptFill(book, accts, o, matches, errs)
				w.Result <- WriteResult{
					Order: *o,
					Err:   nil,
				}
			}
		}
	}
}

// AttemptFill attempts to fill an order until it's completed.
// * For simplicity, AttemptFill controls the book mutex.
// It loops until the order is filled.
func AttemptFill(
	book *Book,
	acc accounts.AccountManager,
	fillorder *Order,
	matches chan Match,
	errs chan error,
) {
	for {
		book.Lock()
		if fillorder.Side == "buy" {
			wanted := fillorder.Open - fillorder.Filled

			low := book.sell.FindMin()
			if len(low.Orders) == 0 {
				book.Unlock()
				continue
			}

			bookorder := low.Orders[0] // select highest time priority by first price-valid match
			available := bookorder.Open - bookorder.Filled

			match := &Match{
				Buy:  fillorder,
				Sell: bookorder,
			}

			switch {
			case wanted > available:
				greedy(book, acc, match, matches, errs)
				book.Unlock()
				continue
			case wanted < available:
				humble(book, acc, match, matches, errs)
				book.Unlock()
				return
			default:
				exact(book, acc, match, matches, errs)
				book.Unlock()
				return
			}
		}

		book.Unlock()
	}
}

// greedy, humble, and exact are the three order handlers for different scenarios
// of supply and demand between a match on price. These functions shouldn't handle
// locking or unlocking, that should all be handled in the AttemptFill function.

// exact is a buy order that wants the exact available amount from the sell order
func exact(book *Book, acc accounts.AccountManager, match *Match, matchCh chan Match, errs chan error) {
	available := match.Sell.Open - match.Sell.Filled
	wanted := match.Buy.Open - match.Buy.Filled

	if wanted == 0 {
		matchCh <- *match
		return
	}

	if available != wanted {
		log.Fatalf("should not happen, this is a bug - match: %+v", match)
	}

	amount := float64((available * match.Sell.Price) / 100)

	_, err := acc.Tx(match.Buy.AccountID, match.Sell.AccountID, amount)
	if err != nil {
		errs <- fmt.Errorf("failed to transfer: %v", err)
		return
	}

	match.Buy.Filled += available
	match.Sell.Filled += available

	match.Buy.History = append(match.Buy.History, *match)
	match.Sell.History = append(match.Sell.History, *match)

	if ok := book.buy.RemoveOrder(match.Buy); !ok {
		errs <- fmt.Errorf("failed to remove over from tree %+v", match.Buy)
		log.Fatalf("failed to remove order from tree %+v", match.Buy)
	}
	if ok := book.sell.RemoveOrder(match.Sell); !ok {
		errs <- fmt.Errorf("failed to remove over from tree %+v", match.Sell)
		log.Fatalf("failed to remove order from tree %+v", match.Sell)
	}

	matchCh <- *match
}

// humble fills a buy order that wants less than is available from the seller
func humble(
	book *Book,
	acc accounts.AccountManager,
	match *Match,
	matchCh chan Match,
	errs chan error,
) {
	// we know it's a humble fill, so we're taking less than the total available.
	wanted := match.Buy.Open - match.Buy.Filled
	amount := float64((wanted * match.Sell.Price) / 100)
	balances, err := acc.Tx(match.Buy.AccountID, match.Sell.AccountID, amount)
	if err != nil {
		errs <- fmt.Errorf("failed to transfer: %v", err)
		return
	}
	log.Printf("[TX] updated balances: %+v", balances)

	match.Buy.Filled += wanted
	match.Sell.Filled += wanted

	match.Buy.History = append(match.Buy.History, *match)
	match.Sell.History = append(match.Sell.History, *match)

	if ok := book.buy.RemoveOrder(match.Buy); !ok {
		errs <- fmt.Errorf("failed to remove order from buy side: %+v", match.Buy)
	}

	matchCh <- *match
}

// greedy is a buy order that wants more than is available from the sell order.
func greedy(
	book *Book,
	acc accounts.AccountManager,
	match *Match,
	matchCh chan Match,
	errs chan error,
) {
	// a greedy fill takes all that's available.
	available := match.Sell.Open - match.Sell.Filled

	amount := float64((available * match.Sell.Price) / 100)

	_, err := acc.Tx(match.Buy.AccountID, match.Sell.AccountID, amount)
	if err != nil {
		errs <- fmt.Errorf("failed to transfer: %v", err)
		return
	}

	match.Sell.Filled += available
	match.Buy.Filled += available

	match.Price = match.Sell.Price
	match.Quantity = available

	match.Buy.History = append(match.Buy.History, *match)
	match.Sell.History = append(match.Sell.History, *match)

	if ok := book.sell.RemoveOrder(match.Sell); !ok {
		errs <- fmt.Errorf("failed to remove sell order from the books %+v", match.Sell)
		return
	}

	matchCh <- *match
}
