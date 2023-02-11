package v3

import (
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
		side:    buyside,
		open:    20,
		filled:  0,
	},
}

func TestWorker(t *testing.T) {
	is := is.New(t)

	// Create our input and output channels.
	pending, complete := make(chan *order), make(chan *order)

	// Launch the StateMonitor.
	status := StateMonitor(time.Second * 1)

	// create an orderbook
	o := &orderbook{
		buy: map[Price][]*order{
			0: {},
		},
		sell: map[Price][]*order{
			0: {},
		},
	}

	for i := 0; i < numWorkers; i++ {
		go Worker(pending, complete, status, o)
	}

	// push test orders into queue and
	wg := &sync.WaitGroup{}
	for _, v := range testOrders {
		pending <- v
	}

	wg.Add(3)

	// gather completed orders
	go func(wg *sync.WaitGroup) {
		for v := range complete {
			is.Equal(v.filled, v.open)
			is.True(v.ID != "")
			wg.Done()
		}
	}(wg)

	// wait for work to finish
	wg.Wait()
}
