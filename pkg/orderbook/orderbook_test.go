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

var numOps int = 10_000
var bufferSize int = 1000

var statsTotal int = 0

func TestRun(t *testing.T) {
	ctx := context.Background()
	wg := &sync.WaitGroup{}

	accts := &accounts.InMemoryManager{}
	reads := make(chan OpRead, bufferSize)
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

	go Start(ctx, accts, reads, writes, fills, errs)

	for i := 0; i < numOps; i++ {
		// BUY WRITE
		buyWrite := OpWrite{
			Side: "buy",
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
			Side: "sell",
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

func TestFindLowest(t *testing.T) {

}
