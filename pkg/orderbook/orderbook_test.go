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
	"github.com/stretchr/testify/require"

	"github.com/brianvoe/gofakeit/v6"
)

var numTestOrders = 1000
var numTestAccounts = 10

func TestRunLoad(t *testing.T) {
	in := make(chan *Order, 1)
	out := make(chan *Match, 1)
	status := make(chan []*Order, 1)
	fills := make(chan []*Order)

	// Generate default random accounts for testing
	accts, ids := newTestAccountManager(t, numTestAccounts)

	// Start the server
	go Run(context.Background(), accts, in, out, fills, status)

	// Consume the status updates
	go func() {
		for state := range status {
			_ = state
		}
	}()

	// Consume fills
	go func() {
		for fill := range fills {
			t.Logf("[FILL]: %+v", fill)
		}
	}()

	// Consume matches
	go func() {
		for match := range out {
			var _ = match
		}
	}()

	// Generate test orders
	buy, sell := newTestOrders(numTestOrders)

	for _, o := range buy {
		o.AccountID = gofakeit.RandomString(ids)
		in <- o
	}
	for _, o := range sell {
		o.AccountID = gofakeit.RandomString(ids) // assign to a random account last of all
		in <- o
	}
}

func TestMatchOrders(t *testing.T) {
	buy, sell := newTestOrders(1000)
	matches, fills := MatchOrders(&accounts.InMemoryManager{}, buy, sell)
	require.NotEmpty(t, matches)
	require.NotEmpty(t, fills)
}

func BenchmarkMatchOrders(b *testing.B) {
	buy, sell := newTestOrders(b.N)
	_, _ = MatchOrders(&accounts.InMemoryManager{}, buy, sell)
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

// newTestAccountManager returns a new account manager and the set of
// ids that it randomly generated.
func newTestAccountManager(t *testing.T, num int) (accounts.AccountManager, []string) {
	acct := accounts.NewAccountManager("")
	ids := []string{}

	for i := 0; i < num; i++ {
		email := gofakeit.Email()
		balance := gofakeit.Float64()
		_, err := acct.Create(email, balance)
		ids = append(ids, email)
		if err != nil {
			t.Error(err)
		}
	}
	return acct, ids
}

// newTestOrders creates a set of buy and sell orders with a random
// price between minPrice and maxPrice, an open quantity between minOpen
// and maxOpen, an equal chance to be owned by foo or bar,
// and with an even chance of being a buy or sell order.
func newTestOrders(count int) (buyOrders, sellOrders []*Order) {
	log.Printf("generating %d new test orders...", count)
	rand.Seed(time.Now().UnixNano())

	var minPrice, maxPrice = 100, 10_000
	var minOpen, maxOpen = 10, 1_000_000

	for i := 0; i < count; i++ {
		o := &Order{
			ID:      fmt.Sprintf("%d", i),
			Kind:    "market",
			Price:   uint64(rand.Intn(maxPrice-minPrice) + minPrice),
			Open:    uint64(rand.Intn(maxOpen-minOpen) + minOpen),
			Filled:  0,
			History: []Match{},
		}

		randBuy := uint64(rand.Intn(maxPrice-minPrice) + minPrice)
		if randBuy%2 == 0 {
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
