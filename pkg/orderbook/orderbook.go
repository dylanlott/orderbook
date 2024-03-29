// Package orderbook is an order-matching engine written in
// Go as part of an experiment of iteration on designs
// in a non-trivial domain.
package orderbook

import (
	"context"
	"log"
	"sort"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

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
	History  []*Match
}

// Orderbook is the core interface of the library.
// * It exposes the core filling algorithm of the engine.
// This algorithm should be able to be swapped out eventually
// so the module should be designed accordingly.
type Orderbook interface {
	Match(buy []Order, sell []Order) []Match
}

// Run starts looping the MatchOrders function. It is a blocking function
// and it is meant to completely own the buy and sell lists to prevent
// external modification.
func Run(
	ctx context.Context,
	accounts accounts.AccountManager,
	in chan *Order,
	out chan *Match,
	fills chan []*Order,
	status chan []*Order,
) {
	// NB: buy and sell are not accessible anywhere but here for safety.
	var buy, sell []*Order
	handleMatches(ctx, accounts, buy, sell, in, out, fills, status)
}

// handleMatches is a blocking function that handles the matches.
// It's meant to be called and held open while it matches orders.
func handleMatches(
	ctx context.Context,
	accts accounts.AccountManager,
	buy, sell []*Order,
	in chan *Order,
	out chan *Match,
	fillsCh chan []*Order,
	status chan []*Order,
) {
	// feed off the orders that accumulated since the last loop
	for o := range in {
		if o.Side == "buy" {
			buy = append(buy, o)
		} else {
			sell = append(sell, o)
		}
		// create the orderlist for state updates
		orderlist := []*Order{}
		orderlist = append(orderlist, buy...)
		orderlist = append(orderlist, sell...)
		status <- orderlist

		matches, fills := MatchOrders(accts, buy, sell)
		for _, match := range matches {
			log.Printf("[MATCH DETECTED]: %+v", match)
			out <- match
		}
		if len(fills) > 0 {
			fillsCh <- fills
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
func MatchOrders(accts accounts.AccountManager, buyOrders []*Order, sellOrders []*Order) ([]*Match, []*Order) {
	sort.Slice(buyOrders, func(i, j int) bool {
		return buyOrders[i].Price > buyOrders[j].Price
	})
	sort.Slice(sellOrders, func(i, j int) bool {
		return sellOrders[i].Price > sellOrders[j].Price
	})

	// Initialize the index variables
	buyIndex := 0
	sellIndex := 0
	var matches []*Match
	var fills []*Order

	// Loop until there are no more Sell orders left
	for sellIndex < len(sellOrders) {
		// Check if the current Buy order matches the current Sell order
		if buyOrders[buyIndex].Price >= sellOrders[sellIndex].Price {
			// Create a match and add it to the matches
			sell := sellOrders[sellIndex]
			buy := buyOrders[buyIndex]

			available := sell.Open - sell.Filled
			wanted := buy.Open - sell.Filled

			var taken uint64 = 0

			switch {
			case available > wanted:
				taken = wanted
				sell.Filled += taken
				buy.Filled += taken
			case available < wanted:
				taken = available
				sell.Filled += taken
				buy.Filled += taken
			default: // availabel == wanted
				taken = wanted
				sell.Filled += taken
				buy.Filled += taken
			}

			m := &Match{
				Buy:      buy,
				Sell:     sell,
				Price:    sell.Price,
				Quantity: taken,
				Total:    taken * sell.Price,
			}
			matches = append(matches, m)

			sell.History = append(sell.History, *m)
			buy.History = append(buy.History, *m)

			if sell.Filled == sell.Open {
				sellOrders = append(sellOrders[:sellIndex], sellOrders[sellIndex+1:]...)
				fills = append(fills, sell)
			}
			if buy.Filled == buy.Open {
				buyOrders = append(buyOrders[:buyIndex], buyOrders[buyIndex+1:]...)
				fills = append(fills, buy)
			}

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
	return matches, fills
}
