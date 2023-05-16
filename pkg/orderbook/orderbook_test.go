package orderbook

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

var numOps = 10_000
var bufferSize = 1000

func TestRun(t *testing.T) {
	ctx := context.Background()
	wg := &sync.WaitGroup{}

	accts := &accounts.InMemoryManager{}
	writes := make(chan OpWrite, bufferSize)
	errs := make(chan error, bufferSize)
	fills := make(chan FillResult, bufferSize)

	go func() {
		for err := range errs {
			t.Logf("[error]: %+v", err)
		}
	}()

	go func() {
		for fill := range fills {
			t.Logf("[fill]: %+v", fill)
		}
	}()

	go Start(ctx, accts, writes, fills, errs)

	for i := 0; i < numOps; i++ {
		// BUY WRITE
		buyWrite := OpWrite{
			Order: Order{
				ID:     fmt.Sprintf("%v", i),
				Kind:   "limit",
				Side:   "buy",
				Price:  uint64(rand.Intn(100)),
				Open:   100,
				Filled: 0,
				Metadata: map[string]string{
					"createdAt": fmt.Sprintf("%v", time.Now()),
				},
			},
			Result: make(chan WriteResult),
		}
		go func() {
			<-buyWrite.Result
			wg.Done()
		}()
		wg.Add(1)
		writes <- buyWrite

		// SELL WRITE
		sellWrite := OpWrite{
			Order: Order{
				ID:     fmt.Sprintf("%v", i),
				Kind:   "limit",
				Side:   "sell",
				Price:  uint64(rand.Intn(100)),
				Open:   100,
				Filled: 0,
				Metadata: map[string]string{
					"createdAt": fmt.Sprintf("%v", time.Now()),
				},
			},
			Result: make(chan WriteResult),
		}
		go func() {
			<-sellWrite.Result
			wg.Done()
		}()
		wg.Add(1)
		writes <- sellWrite
	}

	wg.Wait()
}

func TestAttemptFill(t *testing.T) {
	acc := &accounts.InMemoryManager{
		Accounts: map[string]*accounts.UserAccount{
			"foo@test.com": {
				Email:          "foo@test.com",
				CurrentBalance: 1000.0,
			},
			"bar@test.com": {
				Email:          "bar@test.com",
				CurrentBalance: 1000.0,
			},
		},
	}
	var fillorder = &Order{
		Price:     11,
		ID:        "foo",
		Side:      "buy",
		Filled:    0,
		Open:      10,
		AccountID: "foo@test.com",
		Kind:      "market",
		History:   make([]Match, 0),
	}
	var sellorder = &Order{
		Price:     9,
		ID:        "bar",
		Side:      "sell",
		Filled:    0,
		Open:      10,
		AccountID: "bar@test.com",
		Kind:      "market",
		History:   make([]Match, 0),
	}
	type args struct {
		book      *Book
		acc       accounts.AccountManager
		fillorder *Order
		matches   chan Match
		errs      chan error
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "should fill exact",
			args: args{
				book: &Book{
					buy: &Node{
						Price:  10,
						Orders: []*Order{},
						Left:   &Node{},
						Right: &Node{
							Price: 11,
							Orders: []*Order{
								fillorder,
							},
							Left:  &Node{},
							Right: &Node{},
						},
					},
					sell: &Node{
						Price:  10,
						Orders: []*Order{},
						Left: &Node{
							Price: 9,
							Orders: []*Order{
								sellorder,
							},
						},
						Right: &Node{},
					},
				},
				acc:       acc,
				fillorder: fillorder,
				matches:   make(chan Match, 1000),
				errs:      make(chan error, 1000),
			},
		},
		{
			name: "should fill greedy",
			args: args{
				book: &Book{
					buy: &Node{
						Price:  10,
						Orders: []*Order{},
						Left:   &Node{},
						Right: &Node{
							Price: 11,
							Orders: []*Order{
								{
									Price:     11,
									ID:        "foo",
									Side:      "buy",
									Filled:    0,
									Open:      20,
									AccountID: "foo@test.com",
									Kind:      "market",
									History:   make([]Match, 0),
								},
							},
							Left:  &Node{},
							Right: &Node{},
						},
					},
					sell: &Node{
						Price:  10,
						Orders: []*Order{},
						Left: &Node{
							Price: 9,
							Orders: []*Order{
								{
									Price:     9,
									ID:        "bar",
									Side:      "sell",
									Filled:    0,
									Open:      10,
									AccountID: "bar@test.com",
									Kind:      "market",
									History:   make([]Match, 0),
								},
								{
									Price:     9,
									ID:        "baz",
									Side:      "sell",
									Filled:    0,
									Open:      10,
									AccountID: "baz@test.com",
									Kind:      "market",
									History:   make([]Match, 0),
								},
								{
									Price:     9,
									ID:        "baz",
									Side:      "sell",
									Filled:    0,
									Open:      10,
									AccountID: "baz@test.com",
									Kind:      "market",
									History:   make([]Match, 0),
								},
							},
						},
						Right: &Node{},
					},
				},
				acc:       acc,
				fillorder: fillorder,
				matches:   make(chan Match, 1000),
				errs:      make(chan error, 1000),
			},
		},
		{
			name: "should fill humble",
			args: args{
				book: &Book{
					buy: &Node{
						Price:  10,
						Orders: []*Order{},
						Left:   &Node{},
						Right: &Node{
							Price: 11,
							Orders: []*Order{
								{
									Price:     11,
									ID:        "foo",
									Side:      "buy",
									Filled:    0,
									Open:      20,
									AccountID: "foo@test.com",
									Kind:      "market",
									History:   make([]Match, 0),
								},
							},
							Left:  &Node{},
							Right: &Node{},
						},
					},
					sell: &Node{
						Price:  10,
						Orders: []*Order{},
						Left: &Node{
							Price: 9,
							Orders: []*Order{
								{
									Price:     9,
									ID:        "bar",
									Side:      "sell",
									Filled:    0,
									Open:      10,
									AccountID: "bar@test.com",
									Kind:      "market",
									History:   make([]Match, 0),
								},
								{
									Price:     9,
									ID:        "baz",
									Side:      "sell",
									Filled:    0,
									Open:      10,
									AccountID: "baz@test.com",
									Kind:      "market",
									History:   make([]Match, 0),
								},
								{
									Price:     9,
									ID:        "baz",
									Side:      "sell",
									Filled:    0,
									Open:      10,
									AccountID: "baz@test.com",
									Kind:      "market",
									History:   make([]Match, 0),
								},
							},
						},
						Right: &Node{},
					},
				},
				acc:       acc,
				fillorder: fillorder,
				matches:   make(chan Match, 1000),
				errs:      make(chan error, 1000),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go AttemptFill(tt.args.book, tt.args.acc, fillorder, tt.args.matches, tt.args.errs)
			got := <-tt.args.matches
			t.Logf("[got]: %+v", got)
		})
	}
}

func TestMatchOrders(t *testing.T) {
	buy, sell := newTestOrders(t, 1000)
	got := MatchOrders(buy, sell)
	for _, match := range got {
		t.Logf("\nmatch: [buy] %+v\n [sell] %+v\n", match.Buy, match.Sell)
	}
}

func newTestOrders(t *testing.T, count int) (buyOrders []Order, sellOrders []Order) {
	rand.Seed(time.Now().UnixNano())

	min := 100
	max := 10000

	for i := 0; i < count; i++ {
		o := Order{
			ID:        fmt.Sprintf("%d", i),
			AccountID: "", // TODO: add a random account owner
			Kind:      "market",
			Price:     uint64(rand.Intn(max-min) + min),
			Open:      uint64(rand.Intn(max-min) + min),
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
