package orderbook

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

func TestMatchOrders(t *testing.T) {
	buy, sell := newTestOrders(1000)
	got := MatchOrders(&accounts.InMemoryManager{}, buy, sell)
	for _, match := range got {
		t.Logf("\nmatch: [buy] %+v\n [sell] %+v\n", match.Buy, match.Sell)
	}
}

func BenchmarkMatchOrders(b *testing.B) {
	buy, sell := newTestOrders(b.N)
	got := MatchOrders(&accounts.InMemoryManager{}, buy, sell)
	fmt.Printf("got #: %v\n", len(got))
}

func BenchmarkAttemptFill(b *testing.B) {
	ctx := context.Background()
	wg := &sync.WaitGroup{}

	accts := &accounts.InMemoryManager{}
	writes := make(chan OpWrite, bufferSize)
	errs := make(chan error, bufferSize)
	fills := make(chan FillResult, bufferSize)

	go Start(ctx, accts, writes, fills, errs)

	for i := 0; i < b.N; i++ {
		w := OpWrite{
			Order:  newRandOrder(fmt.Sprintf("%d", i), ""),
			Result: make(chan WriteResult),
		}
		go func() {
			<-w.Result
			wg.Done()
		}()
		wg.Add(1)
		writes <- w
	}

	wg.Wait()
}

func newTestOrders(count int) (buyOrders []Order, sellOrders []Order) {
	log.Printf("count %d", count)
	rand.Seed(time.Now().UnixNano())

	var minPrice, maxPrice = 100, 10_000
	var minOpen, maxOpen = 10, 1_000_000

	for i := 0; i < count; i++ {
		o := Order{
			ID:        fmt.Sprintf("%d", i),
			AccountID: "", // TODO: add a random account owner
			Kind:      "market",
			Price:     uint64(rand.Intn(maxPrice-minPrice) + minPrice),
			Open:      uint64(rand.Intn(maxOpen-minOpen) + minOpen),
			Filled:    0,
			History:   []Match{}, // history should be nil
		}

		// half buy, half sell orders
		if i%2 == 0 {
			o.Side = "buy"
			buyOrders = append(buyOrders, o)
		} else {
			o.Side = "sell"
			sellOrders = append(sellOrders, o)
		}
	}

	return
}

func newRandOrder(id, account string) Order {
	rand.Seed(time.Now().UnixNano())

	var minPrice, maxPrice = 100, 10_000
	var minOpen, maxOpen = 10, 1_000_000

	o := Order{
		ID:        id,
		AccountID: account, // TODO: add a random account owner
		Kind:      "market",
		Price:     uint64(rand.Intn(maxPrice-minPrice) + minPrice),
		Open:      uint64(rand.Intn(maxOpen-minOpen) + minOpen),
		Filled:    0,
		History:   []Match{}, // history should be nil
	}

	// half buy, half sell orders
	if o.Price%2 == 0 {
		o.Side = "buy"
	} else {
		o.Side = "sell"
	}

	return o
}
