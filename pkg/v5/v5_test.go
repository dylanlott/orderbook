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

var numOps int = 1000
var bufferSize int = 1000

func TestListen(t *testing.T) {
	ctx := context.Background()

	reads := make(chan OpRead, bufferSize)
	writes := make(chan OpWrite, bufferSize)

	out := make(chan *Book, bufferSize)
	errs := make(chan error, bufferSize)
	matches := make(chan Match, bufferSize)

	wg := &sync.WaitGroup{}

	go func() {
		for err := range errs {
			log.Printf("[error]: %+v", err) // TODO: log.Fatalf here?
		}
	}()

	go func() {
		for update := range out {
			log.Printf("[update]: %+v", update)
		}
	}()

	go func() {
		for match := range matches {
			fmt.Printf("[match]: %v\n", match)
		}
	}()

	go Listen(ctx, reads, writes, out, matches, errs)

	for i := 0; i < numOps; i++ {

		// BUY WRITE
		buyWrite := OpWrite{
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
		go func() {
			<-buyWrite.result
			wg.Done()
		}()
		wg.Add(1)
		writes <- buyWrite

		// SELL WRITE
		sellWrite := OpWrite{
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
		go func() {
			<-sellWrite.result
			wg.Done()
		}()
		wg.Add(1)
		writes <- sellWrite
	}

	wg.Wait()
}
