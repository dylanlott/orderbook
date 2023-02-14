package v3

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/matryer/is"
)

var numWorkers = 2

var testOrders []*order = []*order{
	{
		ID:      "foo",
		ownerID: 111,
		price:   10,
		side:    buyside,
		open:    10,
		filled:  0,
	},
	{
		ID:      "bar",
		ownerID: 222,
		price:   10,
		side:    sellside,
		open:    20,
		filled:  0,
	},
	{

		ID:      "buz",
		ownerID: 333,
		price:   10,
		side:    buyside,
		open:    20,
		filled:  0,
	},
	{
		ID:      "baz",
		ownerID: 444,
		price:   10,
		side:    sellside,
		open:    20,
		filled:  0,
	},
	{
		ID:      "bex",
		ownerID: 444,
		price:   10,
		side:    sellside,
		open:    10,
		filled:  0,
	},
	{
		ID:      "bez",
		ownerID: 444,
		price:   10,
		side:    buyside,
		open:    5,
		filled:  0,
	},
	{
		ID:      "bem",
		ownerID: 444,
		price:   10,
		side:    sellside,
		open:    10,
		filled:  0,
	},
	{
		ID:      "beu",
		ownerID: 444,
		price:   10,
		side:    sellside,
		open:    5,
		filled:  0,
	},
}

func TestWorker(t *testing.T) {
	is := is.New(t)
	pending, complete := make(chan *order), make(chan *order)
	status := StateMonitor(time.Second * 1)

	o := &orderbook{
		buy:  map[Price][]*order{0: {}},
		sell: map[Price][]*order{0: {}},
	}

	for i := 0; i < numWorkers; i++ {
		go Worker(pending, complete, status, o)
	}

	wg := &sync.WaitGroup{}
	for _, v := range testOrders {
		pending <- v
	}
	wg.Add(3)

	go func(wg *sync.WaitGroup) {
		for v := range complete {
			fmt.Printf("v: %v\n", v)
			is.Equal(v.filled, v.open)
			wg.Done()
		}
	}(wg)

	wg.Wait()
}
