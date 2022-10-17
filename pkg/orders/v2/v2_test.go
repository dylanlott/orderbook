package v2

import (
	"log"
	"reflect"
	"sync"
	"testing"
)

// number of workers that will process orders
var numWorkers = 2

var testOrders = []Order{
	&LimitOrder{
		id:    "foo",
		side:  BUY,
		price: 100,
	},
	&LimitOrder{
		id:    "buzz",
		side:  SELL,
		price: 100,
	},
	&LimitOrder{
		id:    "bar",
		side:  BUY,
		price: 100,
	},
}

func TestWorker(t *testing.T) {
	// Create our input and output channels.
	pending, complete := make(chan Order), make(chan Order)

	// Launch the StateMonitor.
	status := StateMonitor()

	// Create a fresh orderbook and pass it to Worker
	orderbook := &Orderbook{
		Buy:  &PriceNode{val: 0.0},
		Sell: &PriceNode{val: 0.0},
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
		}
	}()

	wg.Wait()
}

func TestOrderbookPull(t *testing.T) {
	type fields struct {
		book *Orderbook
	}
	type args struct {
		price int64
		side  string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    Order
		wantErr bool
	}{
		{
			name: "should pull an order from the tree",
			fields: fields{
				book: &Orderbook{
					Buy: &PriceNode{
						val:    100,
						orders: testOrders,
					},
					Sell: &PriceNode{
						val:    0,
						orders: []Order{},
					},
				},
			},
			args: args{
				price: 100,
				side:  BUY,
			},
			want: &LimitOrder{
				id:    "foo",
				price: 100,
				side:  BUY,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fields.book.Pull(tt.args.price, tt.args.side)
			if (err != nil) != tt.wantErr {
				t.Errorf("PriceNode.Find() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PriceNode.Find() = %v, want %v", got, tt.want)
			}
		})
	}
}
