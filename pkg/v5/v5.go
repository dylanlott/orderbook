package v5

import (
	"context"
	"log"
)

// The idea here is to use channels to guard reads and writes to the orderbook.

// OpRead gets an Order from the book.
type OpRead struct {
	side   bool
	price  uint64
	result chan Order
}

// OpWrite inserts an order into the Book
type OpWrite struct {
	side   bool
	order  Order
	result chan Order
}

// Order is a struct for representing a simple order in the books.
type Order struct {
	ID       string
	Kind     string
	Side     string
	Price    uint64
	Open     uint64
	Filled   uint64
	Metadata map[string]string
}

// PricePoint ties a list of orders to a common price.
type PricePoint struct {
	price  uint64
	orders []Order
}

// Book holds buy and sell side orders. OpRead and OpWrite are applied to
// to the book. Buy and sell side orders are kept as sorted lists orders.
type Book struct {
	// TODO: might be worth examining using the btree again here,
	// and focusing on the async channel handling
	buy  []PricePoint
	sell []PricePoint
}

// Listen takes a reads and a writes channel that it reads from an applies
// those updates to the Book.
func Listen(ctx context.Context, reads <-chan OpRead, writes <-chan OpWrite, output chan *Book) {
	book := &Book{
		buy:  [][]Order{},
		sell: [][]Order{},
	}

	// listen for updates and apply them
	for {
		select {
		case <-ctx.Done():
		case r := <-reads:
			if r.side == true {
				log.Printf("buy read: %+v", r)
				for _, p := range book {

				}
			} else {
				log.Printf("sell read: %+v", r)
			}
		case w := <-writes:
			if w.side == true {
				log.Printf("buy write: %+v", w)
			} else {
				log.Printf("sell write: %+v", w)
			}

		}
	}
}

// Find finds the best order
func (b *Book) Find(price uint64, side bool) Order {
	panic("not impl")
}

// Insert inserts the order at the right PricePoint
func (b *Book) Insert(price uint64, side bool) error {
	panic("not impl")
}

// Run listens sets up the reads and writes and listens for them on the book.
func Run() {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	reads := make(<-chan OpRead)
	writes := make(<-chan OpWrite)

	// Listen kicks off and processes reads and writes concurrently
	go Listen(ctx, reads, writes, out)

	// Listens for processed updates to the books
	for _, update := range out {
		log.Printf("[update] %+v", update)
	}
}
