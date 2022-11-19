package v2

import (
	"log"
	"sync"
	"testing"

	"github.com/dylanlott/orderbook/pkg/accounts"
	"github.com/matryer/is"
)

// number of workers that will process orders
var numWorkers = 2

var testOrders = []Order{
	&LimitOrder{
		id:     "foo",
		side:   BUY,
		price:  100,
		open:   1,
		filled: 0,
	},
	&LimitOrder{
		id:     "buzz",
		side:   SELL,
		price:  100,
		open:   1,
		filled: 0,
	},
	&LimitOrder{
		id:     "bar",
		side:   BUY,
		price:  100,
		open:   1,
		filled: 0,
	},
}

func TestWorker(t *testing.T) {
	is := is.New(t)
	// Create our input and output channels.
	pending, complete := make(chan Order), make(chan Order)

	// Launch the StateMonitor.
	status := StateMonitor()

	// Create a fresh orderbook and pass it to Worker
	orderbook := &Orderbook{
		Buy:  &PriceNode{val: 0.0},
		Sell: &PriceNode{val: 0.0},
		Accounts: &accounts.InMemoryManager{
			Accounts: map[string]*accounts.UserAccount{
				"seller@test.com": {
					Email:          "seller@test.com",
					CurrentBalance: 1000,
				},
				"buyer@test.com": {
					Email:          "buyer@test.com",
					CurrentBalance: 500,
				},
			},
		},
	}

	for i := 0; i < numWorkers; i++ {
		go Worker(pending, complete, status, orderbook)
	}

	var wg = &sync.WaitGroup{}
	go func() {
		for _, testOrder := range testOrders {
			wg.Add(1)
			pending <- testOrder
		}

		for c := range complete {
			wg.Done()
			log.Printf("order %s completed", c.ID())
			is.Equal(c.Open(), uint64(0))
		}
	}()
	wg.Wait()
	for os := range status {
		t.Logf("received order status update: %+v", os)
	}
}

func TestOrderbookPush(t *testing.T) {
	type fields struct {
		Buy  *PriceNode
		Sell *PriceNode
	}
	type args struct {
		order Order
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *Orderbook
		wantErr bool
	}{
		{
			name: "should push an order into the orderbook",
			fields: fields{
				Buy: &PriceNode{
					Mutex:  sync.Mutex{},
					val:    50,
					orders: []Order{},
					right:  &PriceNode{},
					left:   &PriceNode{},
				},
				Sell: &PriceNode{
					Mutex:  sync.Mutex{},
					val:    100,
					orders: []Order{},
					right:  &PriceNode{},
					left:   &PriceNode{},
				},
			},
			args: args{
				order: &LimitOrder{
					id:     "foo",
					price:  100,
					side:   SELL,
					owner:  "bar",
					open:   1,
					filled: 0,
				},
			},
			want: &Orderbook{
				Buy: &PriceNode{
					val:    50,
					orders: []Order{},
					right:  &PriceNode{},
					left:   &PriceNode{},
				},
				Sell: &PriceNode{
					val: 100,
					orders: []Order{
						&LimitOrder{
							id:     "foo",
							price:  100,
							side:   SELL,
							owner:  "bar",
							open:   1,
							filled: 0,
							txs:    []*Transaction{},
						},
					},
					right: &PriceNode{},
					left:  &PriceNode{},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &Orderbook{
				Buy:  tt.fields.Buy,
				Sell: tt.fields.Sell,
			}
			if err := o.Push(tt.args.order); (err != nil) != tt.wantErr {
				t.Errorf("Orderbook.Push() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				compare(t, tt.want, o)
			}
		})
	}
}

func compare(t *testing.T, want *Orderbook, got *Orderbook) {
	if want.Buy != nil {
		for i, order := range want.Buy.orders {
			g := got.Buy.orders[i]
			if g.ID() != order.ID() {
				t.Fail()
			}
		}
	}
	if want.Sell != nil {
		for i, order := range want.Sell.orders {
			g := got.Sell.orders[i]
			if g.ID() != order.ID() {
				t.Fail()
			}
		}
	}
}
