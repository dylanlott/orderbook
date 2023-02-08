package v2

import (
	"reflect"
	"sync"
	"testing"

	"github.com/dylanlott/orderbook/pkg/accounts"

	"github.com/matryer/is"
)

// // number of workers that will process orders
// var numWorkers = 2

// // numFillers is the number of fill workers that is started.
// var numFillers = 2

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

// func TestWorker(t *testing.T) {
// Create our input and output channels.
// pending, complete := make(chan Order), make(chan Order)
//
// Launch the StateMonitor.
// status := StateMonitor()
//
// Create a fresh orderbook and pass it to Worker
// orderbook := &Orderbook{
// Buy:  &PriceNode{val: 0.0},
// Sell: &PriceNode{val: 0.0},
// }
//
// for i := 0; i < numWorkers; i++ {
// go Worker(pending, complete, status, orderbook)
// }
//
// for i := 0; i < numFillers; i++ {
// go Filler(pending, complete, orderbook)
// }
//
// var wg = &sync.WaitGroup{}
// go func() {
// for _, testOrder := range testOrders {
// wg.Add(1)
// pending <- testOrder
// }
//
// for c := range complete {
// wg.Done()
// log.Printf("order %s completed", c.ID())
// }
// }()
//
// wg.Wait()
// }

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
				id:     "foo",
				side:   BUY,
				price:  100,
				open:   1,
				filled: 0,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.fields.book.Pull(tt.args.price, tt.args.side)
			if (err != nil) != tt.wantErr {
				t.Errorf("PriceNode.Pull() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PriceNode.Pull() got = %v, wantErr %v", got, tt.want)
			}
		})
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
					Owner:  "bar",
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
							id:           "foo",
							price:        100,
							side:         SELL,
							Owner:        "bar",
							open:         1,
							filled:       0,
							Transactions: []*Transaction{},
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

func TestPriceNode_List(t *testing.T) {
	type fields struct {
		val    int64
		orders []Order
		right  *PriceNode
		left   *PriceNode
	}
	tests := []struct {
		name    string
		fields  fields
		want    []Order
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &PriceNode{
				val:    tt.fields.val,
				orders: tt.fields.orders,
				right:  tt.fields.right,
				left:   tt.fields.left,
			}
			got, err := tr.List()
			if (err != nil) != tt.wantErr {
				t.Errorf("PriceNode.List() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("PriceNode.List() = %v, want %v", got, tt.want)
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

func TestMatch(t *testing.T) {
	is := is.New(t)
	// Create a fresh orderbook and pass it to Worker
	orderbook := &Orderbook{
		Buy:  &PriceNode{val: 0.0},
		Sell: &PriceNode{val: 0.0},
		Accounts: &accounts.InMemoryManager{
			Accounts: map[string]*accounts.UserAccount{
				"alice@test.com": {
					Email:          "alice@test.com",
					CurrentBalance: 1000,
				},
				"bob@test.com": {
					Email:          "bob@test.com",
					CurrentBalance: 1000,
				},
			},
		},
	}

	buy := &LimitOrder{
		Owner:  "alice@test.com",
		id:     "foo",
		price:  100,
		side:   BUY,
		open:   1,
		filled: 0,
	}
	is.NoErr(orderbook.Push(buy))

	sell := &LimitOrder{
		Owner:  "bob@test.com",
		id:     "bar",
		price:  100,
		side:   SELL,
		open:   1,
		filled: 0,
	}
	is.NoErr(orderbook.Push(sell))

	// assert that orderbook matches and fills an order
	filled, err := orderbook.Match(buy)
	is.NoErr(err)
	is.True(filled.ID() == buy.id)

	// assert that a transaction receipt is created
	buyReceipt := buy.History()[0]
	is.Equal(len(buy.History()), 1)
	is.True(buyReceipt.AccountID == sell.OwnerID())

	is.Equal(len(sell.History()), 1)
	sellReceipt := sell.History()[0]
	is.True(sellReceipt.AccountID == buy.OwnerID())

	// assert balances were adjusted
	updatedBuyer, err := orderbook.Accounts.Get(buy.OwnerID())
	is.NoErr(err)
	is.Equal(updatedBuyer.Balance(), float64(900))

	// assert orders are removed from books
	_, err = orderbook.Buy.Find(buy.price)
	is.True(err != nil)
	_, err = orderbook.Sell.Find(buy.price)
	is.True(err != nil)
}
