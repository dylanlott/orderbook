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

func Test_attemptFill(t *testing.T) {
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
			name: "should fill an order",
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
				acc:       &accounts.InMemoryManager{},
				fillorder: fillorder,
				matches:   make(chan Match),
				errs:      make(chan error),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			go attemptFill(tt.args.book, tt.args.acc, fillorder, tt.args.matches, tt.args.errs)
			got := tt.args.matches
			fmt.Printf("got: %v\n", got)
			fmt.Printf("tt.args.book: %v\n", tt.args.book)

			fmt.Printf("tt.args.book.buy.FindMax(): %v\n", tt.args.book.buy.FindMax())
			fmt.Printf("tt.args.book.sell.FindMin(): %v\n", tt.args.book.sell.FindMin())
		})
	}
}
