// Package orderbook is an order-matching engine written in
// Go as part of an experiment of iteration on designs
// in a non-trivial domain.
package orderbook

import (
	"context"
	"log"
	"sort"
	"time"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

var delay time.Duration = time.Second * 1

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

// Match holds a buy and a sell side order at a quantity per price.
// Matches can be made for any type of order, including limit or market orders.
type Match struct {
	Buy      *Order
	Sell     *Order
	Price    uint64 // at what price was each unit purchased by the buyer from the seller
	Quantity uint64 // how many units were transferred from seller to buyer
	Total    uint64 // total = price * quantity
}

// Orderbook is the core interface of the library.
// * It exposes the core filling algorithm of the engine.
// This algorithm should be able to be swapped out eventually
// so the module should be designed accordingly.
type Orderbook interface {
	Match(buy []Order, sell []Order) []Match
}

// Run starts looping the MatchOrders function.
func Run(
	ctx context.Context,
	accounts accounts.AccountManager,
	in chan Order,
	out chan *Match,
	status chan []Order,
) {
	// NB: buy and sell are not accessible anywhere but here for safety.
	var buy, sell []Order
	handleMatches(ctx, accounts, buy, sell, in, out, status)
}

// handleMatches is a blocking function that handles the matches.
// It's meant to be called and held open while it matches orders.
func handleMatches(
	ctx context.Context,
	accts accounts.AccountManager,
	buy, sell []Order,
	in chan Order,
	out chan *Match,
	status chan []Order,
) {
	for {
		orderlist := []Order{}
		orderlist = append(orderlist, buy...)
		orderlist = append(orderlist, sell...)
		status <- orderlist

		matches := MatchOrders(accts, buy, sell)
		for _, match := range matches {
			log.Printf("%+v", match)
			time.Sleep(delay)
			// feed to ouptut
		}
	}
}

// MatchOrders is an alternative approach to order matching that
// works by aligning two opposing sorted slices of Orders then
// iterating through them to generate matches.
// * It generates multiple matches for a buy order until all
// matching sell options are exhausted,
// * When it exhausts all f it ratchets up the buy index again and finds all matching
// orders.
func MatchOrders(accts accounts.AccountManager, buyOrders []Order, sellOrders []Order) []Match {
	sort.Slice(buyOrders, func(i, j int) bool {
		return buyOrders[i].Price > buyOrders[j].Price
	})
	sort.Slice(sellOrders, func(i, j int) bool {
		return sellOrders[i].Price > sellOrders[j].Price
	})

	// Initialize the index variables
	buyIndex := 0
	sellIndex := 0
	var matches []Match

	// Loop until there are no more Sell orders left
	for sellIndex < len(sellOrders) {
		// Check if the current Buy order matches the current Sell order
		if buyOrders[buyIndex].Price >= sellOrders[sellIndex].Price {
			// Create a Match of the Buy and Sell side
			m := Match{
				Buy:  &buyOrders[buyIndex],
				Sell: &sellOrders[sellIndex],
			}
			matches = append(matches, m)
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
	return matches
}
