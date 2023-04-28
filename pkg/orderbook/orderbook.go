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
				removed := book.buy.Remove(low.Price)
				fmt.Printf("removed from the binary tree ### removed.Price: %v\n", removed.Price)
				continue
			}

			// set some initial parameters
			match := low.Orders[0]
			available := match.Open - match.Filled
			wanted := fillorder.Open - fillorder.Filled

			// lookup buyer and seller accounts
			buyer, err := acc.Get(fillorder.AccountID)
			if err != nil {
				errs <- fmt.Errorf("failed to look up buyer: %s", fillorder.AccountID)
				return
			}
			seller, err := acc.Get(match.AccountID)
			if err != nil {
				errs <- fmt.Errorf("failed to look up seller: %s", match.AccountID)
				return
			}

			if wanted > available {
				m := Match{
					Buy:  fillorder,
					Sell: match,
				}

				// amount is calculated from price and available quantity
				amount := float64((available * fillorder.Price) / 100)
				balances, err := acc.Tx(buyer.UserID(), seller.UserID(), amount)
				if err != nil {
					errs <- fmt.Errorf("failed to transfer: %v", err)
					return
				}
				log.Printf("[TX] updated balances: %+v", balances)

				// remove it from books,
				remaining := low.Orders[1:]
				low.Orders = remaining
				log.Printf("[BOOK] updated orders: %+v", low.Orders)

				// fill both sides
				fillorder.Filled += available
				match.Filled += available

				// mark as filled
				log.Printf("[MATCH] %+v", m)
				matches <- m
			}

			if wanted < available {
				m := Match{
					Buy:  fillorder,
					Sell: match,
				}

				// amount is calculated from price and available quantity
				amount := float64((available * fillorder.Price) / 100)
				balances, err := acc.Tx(buyer.UserID(), seller.UserID(), amount)
				if err != nil {
					errs <- fmt.Errorf("failed to transfer: %v", err)
					return
				}
				log.Printf("[TX] updated balances: %+v", balances)

				// update order quantities
				fillorder.Filled += wanted
				match.Filled += wanted

				// remove the order from the buyside books
				if ok := book.buy.RemoveOrder(fillorder.ID); !ok {
					errs <- fmt.Errorf("failed to remove order from buy side: %+v", fillorder)
				}

				// mark as filled
				matches <- m
			}

			if wanted == available {
				_ = Match{
					Buy:  fillorder,
					Sell: match,
				}

				// amount is calculated from price and available quantity
				amount := float64((available * fillorder.Price) / 100)
				balances, err := acc.Tx(buyer.UserID(), seller.UserID(), amount)
				if err != nil {
					errs <- fmt.Errorf("failed to transfer: %v", err)
					return
				}
				log.Printf("[TX] updated balances: %+v", balances)

				// mark both as filled
				match.Filled += available
				fillorder.Filled += available

				// remove it from books,
				remaining := low.Orders[1:]
				low.Orders = remaining

				// remvoe the buy order from the tree
				if ok := book.buy.RemoveOrder(fillorder.ID); !ok {
					// NB: technically this means the order wasn't found, because it can't fail.
					errs <- fmt.Errorf("failed to remove order from books: %+v", fillorder)
				}
			}
		}
	}
}

// An alternative approach to order matching that relies on sorting
// two opposing slices of Orders.
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
