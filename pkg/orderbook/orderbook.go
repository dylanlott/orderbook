package orderbook

import (
	"context"
	"fmt"
	"log"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

// The idea here is to use channels to guard reads and writes to the orderbook.

// OpRead gets an Order from the book.
type OpRead struct {
	Side    string
	Price   uint64
	OrderID string
	Result  chan ReadResult
}

// ReadResult is returned for an OpRead
type ReadResult struct {
	Order Order
	Err   error
}

// OpWrite inserts an order into the Book
type OpWrite struct {
	Side   string
	Order  Order
	Result chan WriteResult
}

// WriteResult is returned as the result of an OpWrite.
type WriteResult struct {
	Order Order
	Err   error
}

// FillResult contains the buy and sell order that were
// matched and filled. FillResult is only created after
// everything has been committed to state.
type FillResult struct {
	Buy    *Order
	Sell   *Order
	Filled uint64
}

// Order is a struct for representing a simple order in the books.
type Order struct {
	ID        string
	AccountID string
	Kind      string
	Side      string
	Price     uint64
	Open      uint64
	Filled    uint64
	History   []Match
	Metadata  map[string]string
}

// Match holds a buy and a sell side order
type Match struct {
	Buy  *Order
	Sell *Order
	Tx   accounts.Transaction
}

// Book holds buy and sell side orders. OpRead and OpWrite are applied to
// to the book. Buy and sell side orders are binary trees of order lists.
type Book struct {
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
	reads chan OpRead,
	writes chan OpWrite,
	fills chan FillResult,
	errs chan error,
) {
	matches := make(chan Match)

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

	go handleMatches(ctx, accts, book, matches, fills, errs)

	// TODO: factor out into a handleFills function
	for {
		select {
		case <-ctx.Done():
			// TODO: drain channels and cleanup
			return
		case r := <-reads:
			if r.Side == "buy" {
				panic("not impl")
			}
		case w := <-writes:
			if w.Side == "buy" {
				book.buy.Insert(&w.Order)
				go attemptFill(book, w.Order, matches, errs)
				w.Result <- WriteResult{
					Order: w.Order,
					Err:   nil,
				}
			} else {
				book.sell.Insert(&w.Order)
				go attemptFill(book, w.Order, matches, errs)
				w.Result <- WriteResult{
					Order: w.Order,
					Err:   nil,
				}
			}
		}
	}
}

// handleMatches listens for incoming Matches and executes their
// balance transfers and updates the order balances.
func handleMatches(
	ctx context.Context,
	accts accounts.AccountManager,
	book *Book,
	matchCh chan Match,
	fills chan FillResult,
	errs chan error,
) {
	for {
		select {
		case <-ctx.Done():
			return
		case match := <-matchCh:
			// lookup buyer and seller accounts
			buyer, err := accts.Get(match.Buy.AccountID)
			if err != nil {
				errs <- fmt.Errorf("failed to look up buyer: %s", match.Buy.AccountID)
				return
			}
			seller, err := accts.Get(match.Sell.AccountID)
			if err != nil {
				errs <- fmt.Errorf("failed to look up seller: %s", match.Sell.AccountID)
				return
			}

			// determine quantities for price calculations
			wanted := match.Buy.Open
			available := match.Sell.Open

			// handle differences in offerings
			switch {
			case wanted == available:
				// want exactly as much as is available, take it all
				total := wanted * match.Sell.Price

				// try to transfer; accts.Tx errors if buyer has insufficient funds
				balances, err := accts.Tx(buyer.UserID(), seller.UserID(), float64(total))
				if err != nil {
					log.Printf("accounts error: %+v", err)
					errs <- err
					return
				}

				log.Printf("balances updated: %+v", balances)

				// TODO: remove orders from the books

				fr := FillResult{
					Filled: wanted,
					Buy:    match.Buy,
					Sell:   match.Sell,
				}

				fills <- fr
			case wanted > available:
				// want more than available, take it all
				total := available * match.Sell.Price

				// try to transfer; accts.Tx errors if buyer has insufficient funds
				balances, err := accts.Tx(buyer.UserID(), seller.UserID(), float64(total))
				if err != nil {
					log.Printf("accounts error: %+v", err)
					errs <- err
					return
				}

				log.Printf("balances updated: %+v", balances)

				// _, err = book.sell.RemoveByID(match.Sell)
				// if err != nil {
				// 	errs <- fmt.Errorf("failed to remove order: %+v", err)
				// 	return
				// }

				fr := FillResult{
					Filled: available,
					Buy:    match.Buy,
					Sell:   match.Sell,
				}

				fills <- fr
			case wanted < available:
				// want less than available, only take wanted
				total := wanted * match.Sell.Price

				// try to transfer; accts.Tx errors if buyer has insufficient funds
				balances, err := accts.Tx(buyer.UserID(), seller.UserID(), float64(total))
				if err != nil {
					log.Printf("accounts error: %+v", err)
					errs <- err
					return
				}

				log.Printf("balances updated: %+v", balances)

				match.Buy.Filled += wanted
				match.Sell.Filled += wanted

				// _, err = book.buy.RemoveByID(match.Buy)
				// if err != nil {
				// 	errs <- fmt.Errorf("failed to remove order: %+v", err)
				// 	return
				// }

				fr := FillResult{
					Filled: available,
					Buy:    match.Buy,
					Sell:   match.Buy,
				}

				fills <- fr
			}
		}
	}

}

// attemptFill is meant to be called in a goroutine and
// loops the books until it finds a Match then it sends
// then on matchse channel for execution.
func attemptFill(
	book *Book,
	fillorder Order,
	matches chan Match,
	errs chan error,
) {
	for fillorder.Filled < fillorder.Open {
		// Loop as long as the order is not filled
		if fillorder.Side == "buy" {
			//
			// BUY SIDE order; match to sell side
			//
			panic("not impl")
		} else if fillorder.Side == "sell" {
			//
			// SELL SIDE order; match to buy side
			//
			panic("not impl")
		}
	}
}
