package orders

import (
	"context"
	"testing"
	"time"

	"github.com/matryer/is"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

// Starting balances
const BUYER_STARTING_BALANCE = 500.00
const SELLER_STARTING_BALANCE = 1000.00

func Test_market_Fill(t *testing.T) {

	is := is.New(t)
	type fields struct {
		asset    *AssetInfo
		Accounts accounts.AccountManager
		BuySide  *TreeNode
		SellSide *TreeNode
	}
	type args struct {
		ctx       context.Context
		fillOrder Order
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		assertions func(t *testing.T, fields fields, args args, m *market)
	}{
		{
			name: "should fill a simple matching order",
			fields: fields{
				Accounts: &accounts.InMemoryManager{
					Accounts: map[string]*accounts.UserAccount{
						"seller@test.com": {
							Email:          "seller@test.com",
							CurrentBalance: SELLER_STARTING_BALANCE,
						},
						"buyer@test.com": {
							Email:          "buyer@test.com",
							CurrentBalance: BUYER_STARTING_BALANCE,
						},
					},
				},
				BuySide: &TreeNode{
					val: 50.0,
					orders: []Order{
						&MarketOrder{
							Asset: AssetInfo{
								Name:       "ETH",
								Underlying: "USD",
							},
							UserAccount: &accounts.UserAccount{
								Email: "buyer@test.com",
							},
							UUID:           "0xBUY",
							OpenQuantity:   1,
							FilledQuantity: 0,
							PlacedAt:       time.Time{},
							MarketPrice:    50.0,
							done:           make(chan Order, 1),
						},
					},
				},
				SellSide: &TreeNode{
					val: 50.0,
					orders: []Order{
						&MarketOrder{
							Asset: AssetInfo{
								Name:       "ETH",
								Underlying: "USD",
							},
							UserAccount: &accounts.UserAccount{
								Email: "seller@test.com",
							},
							UUID:           "0xSELL",
							OpenQuantity:   1,
							FilledQuantity: 0,
							PlacedAt:       time.Time{},
							MarketPrice:    50.0,
							done:           make(chan Order, 1),
						},
					},
				},
				asset: &AssetInfo{
					Name:       "ETH",
					Underlying: "USD",
				},
			},
			args: args{
				ctx: context.Background(),
				fillOrder: &MarketOrder{
					UserAccount: &accounts.UserAccount{
						Email: "buyer@test.com",
					},
					Asset: AssetInfo{
						Name:       "ETH",
						Underlying: "USD",
					},
					UUID:           "0xBUY",
					OpenQuantity:   1,
					FilledQuantity: 0,
					PlacedAt:       time.Now(),
					MarketPrice:    50,
					done:           make(chan Order),
				},
			},
			assertions: func(t *testing.T, fields fields, args args, m *market) {
				got := <-args.fillOrder.Done()
				is.True(got.ID() == args.fillOrder.ID())
				buyerAcct, err := fields.Accounts.Get(got.Owner().UserID())
				is.NoErr(err)
				is.True(buyerAcct.Balance() == BUYER_STARTING_BALANCE-args.fillOrder.Price())
				orders, err := m.BuySide.Orders(args.fillOrder.Price())
				is.NoErr(err)
				is.Equal(len(orders), 0)
			},
		},
		{
			name: "should partially fill an order",
			fields: fields{
				asset: &AssetInfo{
					Name:       "ETH",
					Underlying: "USD",
				},
				Accounts: &accounts.InMemoryManager{
					Accounts: map[string]*accounts.UserAccount{
						"seller@test.com": {
							Email:          "seller@test.com",
							CurrentBalance: SELLER_STARTING_BALANCE,
						},
						"buyer@test.com": {
							Email:          "buyer@test.com",
							CurrentBalance: BUYER_STARTING_BALANCE,
						},
					},
				},
				SellSide: &TreeNode{
					val: 50.0,
					orders: []Order{
						&MarketOrder{
							Asset: AssetInfo{
								Name:       "ETH",
								Underlying: "USD",
							},
							UserAccount: &accounts.UserAccount{
								Email: "seller@test.com",
							},
							UUID:           "0xSELL",
							OpenQuantity:   5,
							FilledQuantity: 0,
							PlacedAt:       time.Time{},
							MarketPrice:    50.0,
							done:           make(chan Order, 1),
						},
					},
				},
				BuySide: &TreeNode{
					val: 50.0,
					orders: []Order{
						&MarketOrder{
							Asset: AssetInfo{
								Name:       "ETH",
								Underlying: "USD",
							},
							UserAccount: &accounts.UserAccount{
								Email: "buyer@test.com",
							},
							UUID:           "0xBUY",
							OpenQuantity:   1,
							FilledQuantity: 0,
							PlacedAt:       time.Time{},
							MarketPrice:    50.0,
							done:           make(chan Order, 1),
						},
					},
				},
			},
			args: args{
				ctx: context.Background(),
				fillOrder: &MarketOrder{
					UserAccount: &accounts.UserAccount{
						Email: "buyer@test.com",
					},
					Asset: AssetInfo{
						Name:       "ETH",
						Underlying: "USD",
					},
					UUID:           "0xBUY",
					OpenQuantity:   2,
					FilledQuantity: 0,
					PlacedAt:       time.Now(),
					MarketPrice:    50,
					done:           make(chan Order),
				},
			},

			assertions: func(t *testing.T, fields fields, args args, m *market) {
				got := <-args.fillOrder.Done()
				is.Equal(got.ID(), args.fillOrder.ID())
				is.True(got.CreatedAt().Before(time.Now()))
				is.Equal(got.Owner().UserID(), args.fillOrder.Owner().UserID())
				sellerAcct, err := m.Accounts.Get("seller@test.com")
				is.NoErr(err)
				is.Equal(sellerAcct.Balance(), SELLER_STARTING_BALANCE+(2*args.fillOrder.Price()))
				order, err := m.SellSide.Orders(args.fillOrder.Price())
				is.NoErr(err)
				is.True(len(order) == 1)
				is.Equal(order[0].Quantity(), int64(3))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := &market{
				asset:    tt.fields.asset,
				Accounts: tt.fields.Accounts,
				BuySide:  tt.fields.BuySide,
				SellSide: tt.fields.SellSide,
			}

			go tt.assertions(t, tt.fields, tt.args, fm)

			fm.Fill(tt.args.ctx, tt.args.fillOrder)

			// NB: This could miss failures if they occur after 1 second,
			// but is necessary to hold open.
			time.Sleep(1 * time.Second)
		})
	}
}
