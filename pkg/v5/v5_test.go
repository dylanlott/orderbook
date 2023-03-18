package v5

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"testing"
	"time"
)

var numWrites int = 1000
var bufferSize int = 1000

func TestListen(t *testing.T) {
	ctx := context.Background()
	reads := make(chan OpRead, bufferSize)
	writes := make(chan OpWrite, bufferSize)
	out := make(chan *Book, bufferSize)
	errs := make(chan error, bufferSize)
	matches := make(chan Match, bufferSize)
	wg := &sync.WaitGroup{}

	// Listen kicks off and processes reads and writes concurrently
	go Listen(ctx, reads, writes, out, matches, errs)

	go func() {
		for err := range errs {
			log.Printf("[error] %+v", err) // TODO: log.Fatalf here?
		}
	}()

	go func() {
		for update := range out {
			log.Printf("[update] %+v", update)
		}
	}()

	go func() {
		for match := range matches {
			fmt.Printf("match: %v\n", match)
		}
	}()

	// insert orders into the books
	for i := 0; i < numWrites; i++ {
		buy := OpWrite{
			side: "buy",
			order: Order{
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
			result: make(chan Order),
		}
		writes <- buy
		wg.Add(1)

		sell := OpWrite{
			side: "sell",
			order: Order{
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
			result: make(chan Order),
		}
		writes <- sell
		wg.Add(1)

		go func() {
			res := <-buy.result
			t.Logf("buy write result: %+v", res)
			wg.Done()
		}()

		go func() {
			res := <-sell.result
			t.Logf("sell write result: %+v", res)
			wg.Done()
		}()
	}

	wg.Wait()
}
