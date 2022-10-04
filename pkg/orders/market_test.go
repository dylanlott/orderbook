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
		name   string
		fields fields
		args   args
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
			go func() {
				got := <-tt.args.fillOrder.Done()
				is.True(got.ID() == tt.args.fillOrder.ID())
				buyerAcct, err := tt.fields.Accounts.Get(got.Owner().UserID())
				is.NoErr(err)
				is.True(buyerAcct.Balance() == BUYER_STARTING_BALANCE-tt.args.fillOrder.Price())
				orders, err := fm.BuySide.Orders(tt.args.fillOrder.Price())
				is.NoErr(err)
				is.Equal(len(orders), 0)
			}()
			fm.Fill(tt.args.ctx, tt.args.fillOrder)
			time.Sleep(1 * time.Second) // NB: This could miss failures if they occur after 1 second.
		})
	}
}
