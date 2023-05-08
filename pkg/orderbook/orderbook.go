// Package orderbook is an order-matching engine written in
// Go as part of an experiment of iteration on designs
// in a non-trivial domain.
package orderbook

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

// OpWrite inserts an order into the Book
type OpWrite struct {
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
	sync.RWMutex

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

	// TODO: factor out into a handleFills function
	for {
		select {
		case <-ctx.Done():
			// TODO: drain channels and cleanup
			return
		case w := <-writes:
			if w.Order.Side == "buy" {
				o := &w.Order
				book.buy.Insert(o)
				go attemptFill(book, accts, o, matches, errs)
				w.Result <- WriteResult{
					Order: *o,
					Err:   nil,
				}
			} else {
				o := &w.Order
				book.sell.Insert(o)
				go attemptFill(book, accts, o, matches, errs)
				w.Result <- WriteResult{
					Order: *o,
					Err:   nil,
				}
			}
		default:
			book.buy.Print()
			book.sell.Print()
		}
	}
}

func attemptFill(
	book *Book,
	acc accounts.AccountManager,
	fillorder *Order,
	matches chan Match,
	errs chan error,
) {
	// Loop as long as the order is not filled
	for fillorder.Filled < fillorder.Open {
		book.RWMutex.Lock()

		if fillorder.Side == "buy" {
			// match to sell
			low := book.sell.FindMin()
			if len(low.Orders) == 0 {
				continue
			}

			bookorder := low.Orders[0]
			available := bookorder.Open - bookorder.Filled
			wanted := fillorder.Open - fillorder.Filled

			match := &Match{
				Buy:  fillorder,
				Sell: bookorder,
			}

			if wanted > available {
				greedy(book, acc, match, matches, errs)
			}

			if wanted < available {
				humble(book, acc, match, matches, errs)
			}

			if wanted == available {
				exact(book, acc, match, matches, errs)
			}
		}
	}
}

// exact is a buy order that wants the exact available amount from the sell order
func exact(book *Book, acc accounts.AccountManager, match *Match, matchCh chan Match, errs chan error) {
	available := match.Sell.Open - match.Sell.Filled
	wanted := match.Buy.Open - match.Buy.Filled
	if available != wanted {
		log.Fatalf("should not happen, this is a bug - match: %+v", match)
	}

	// amount is calculated from price and available quantity
	amount := float64((available * match.Sell.Price) / 100)

	_, err := acc.Tx(match.Buy.AccountID, match.Sell.AccountID, amount)
	if err != nil {
		errs <- fmt.Errorf("failed to transfer: %v", err)
		return
	}

	// mark both as filled
	match.Buy.Filled += available
	match.Sell.Filled += available

	// remove it from books,
	if ok := book.buy.RemoveOrder(match.Buy); !ok {
		errs <- fmt.Errorf("failed to remove over from tree %+v", match.Buy)
	}
	if ok := book.sell.RemoveOrder(match.Sell); !ok {
		errs <- fmt.Errorf("failed to remove over from tree %+v", match.Sell)
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
	// amount is calculated from price and available quantity
	amount := float64((wanted * match.Sell.Price) / 100)
	balances, err := acc.Tx(match.Buy.AccountID, match.Sell.AccountID, amount)
	if err != nil {
		errs <- fmt.Errorf("failed to transfer: %v", err)
		return
	}
	log.Printf("[TX] updated balances: %+v", balances)

	// update order quantities
	match.Buy.Filled += wanted
	match.Sell.Filled += wanted

	// remove the order from the buyside books
	if ok := book.buy.RemoveOrder(match.Buy); !ok {
		errs <- fmt.Errorf("failed to remove order from buy side: %+v", match.Buy)
	}

	// mark as filled
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

	// amount is calculated from price and available quantity
	amount := float64((available * match.Sell.Price) / 100)

	// transfer from buyer to seller the agreed upon amount
	balances, err := acc.Tx(match.Buy.AccountID, match.Sell.AccountID, amount)
	if err != nil {
		errs <- fmt.Errorf("failed to transfer: %v", err)
		return
	}
	log.Printf("[TX] updated balances: %+v", balances)

	// fill both sides
	match.Sell.Filled += available
	match.Buy.Filled += available

	if ok := book.sell.RemoveOrder(match.Sell); !ok {
		errs <- fmt.Errorf("failed to remove sell order from the books %+v", match.Sell)
	}

	matchCh <- *match
}

// MatchOrders is an alternative approach to order matching that
// works by aligning two opposing sorted slices of Orders.
func MatchOrders(buyOrders []Order, sellOrders []Order) []Order {
	// Sort the orders by price
	sort.Slice(buyOrders, func(i, j int) bool {
		return buyOrders[i].Price > buyOrders[j].Price
	})
	sort.Slice(sellOrders, func(i, j int) bool {
		return sellOrders[i].Price > sellOrders[j].Price
	})

	// Initialize the index variables
	buyIndex := 0
	sellIndex := 0
	var filledOrders []Order

	// Loop until there are no more Sell orders left
	for sellIndex < len(sellOrders) {
		// Check if the current Buy order matches the current Sell order
		if buyOrders[buyIndex].Price >= sellOrders[sellIndex].Price {
			// Fill the Buy order with the Sell order
			filledOrder := Order{
				// Buyer: buyOrders[buyIndex].Buyer,
				// Seller: sellOrders[sellIndex].Seller,
				Price: sellOrders[sellIndex].Price,
			}
			filledOrders = append(filledOrders, filledOrder)
			// Increment the Sell order index
			sellIndex++
		} else {
			// Move on to the next Buy order
			buyIndex++
		}
		// Check if there are no more Buy orders left
		if buyIndex == len(buyOrders) {
			break
		}
	}

	// Return the list of filled orders
	return filledOrders
}
