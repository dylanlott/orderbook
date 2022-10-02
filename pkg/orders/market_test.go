package orders

import (
	"context"
	"testing"
	"time"

	"github.com/matryer/is"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

func Test_market_Fill(t *testing.T) {
	is := is.New(t)
	type fields struct {
		asset     *AssetInfo
		Accounts  accounts.AccountManager
		BuySide   *TreeNode
		SellSide  *TreeNode
		OrderTrie *TreeNode
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
			name: "",
			fields: fields{
				Accounts: &accounts.InMemoryManager{
					Accounts: map[string]*accounts.UserAccount{
						"seller@test.com": {
							Email:          "seller@test.com",
							CurrentBalance: 1000.00,
						},
						"buyer@test.com": {
							Email:          "buyer@test.com",
							CurrentBalance: 500.00,
						},
					},
				},
				BuySide: &TreeNode{
					val:    0.0,
					orders: []Order{},
					right:  &TreeNode{},
					left:   &TreeNode{},
				},
				SellSide: &TreeNode{
					val:    0.0,
					orders: []Order{},
					right:  &TreeNode{},
					left:   &TreeNode{},
				},
				asset: &AssetInfo{
					Name:       "ETH",
					Underlying: "USD",
				},
			},
			args: args{
				ctx: context.Background(),
				fillOrder: &MarketOrder{
					Asset: AssetInfo{
						Name:       "ETH",
						Underlying: "USD",
					},
					UUID:           "foo",
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
			fm.Fill(tt.args.ctx, tt.args.fillOrder)
			got := <-tt.args.fillOrder.Done()
			is.True(got.ID() == tt.args.fillOrder.ID())
		})
	}
}
