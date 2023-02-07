package orders

import (
	"context"
	"testing"
	"time"

	"github.com/matryer/is"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

// Starting balances for the Buyer and Seller
const BuyerStartingBalance = 500.00
const SellerStartingBalance = 1000.00

func TestFill(t *testing.T) {
	is := is.New(t)

	t.Run("fill handle equal want", func(t *testing.T) {
		ctx := context.Background()

		fm := &market{
			Accounts: &accounts.InMemoryManager{},
			BuySide: &TreeNode{
				val: 50.0,
				orders: []Order{
					&MarketOrder{
						Asset: AssetInfo{
							Name: "ETH",
						},
						side: "BUY",
						UserAccount: &accounts.UserAccount{
							Email: "buyer@test.com",
						},
						UUID:           "0xBUY",
						OpenQuantity:   1,
						FilledQuantity: 0,
						PlacedAt:       time.Now(),
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
							Name: "ETH",
						},
						side: "SELL",
						UserAccount: &accounts.UserAccount{
							Email: "seller@test.com",
						},
						UUID:           "0xSELL",
						OpenQuantity:   1,
						FilledQuantity: 0,
						PlacedAt:       time.Time{},
						MarketPrice:    50.0,
						done:           make(chan Order, 1), // must be unbuffered channel.
					},
				},
			},
		}

		fillOrder := &MarketOrder{
			UserAccount: &accounts.UserAccount{
				Email: "buyer@test.com",
			},
			Asset: AssetInfo{
				Name: "ETH",
			},
			UUID:           "0xBUY",
			side:           "BUY",
			OpenQuantity:   1,
			FilledQuantity: 0,
			PlacedAt:       time.Now(),
			MarketPrice:    50,
			done:           make(chan Order, 1),
		}

		// NB: no load-bearing sleeps.

		// go wait for order to be filled and assert on it
		go func() {
			got := <-fillOrder.Done()
			is.True(got.ID() == fillOrder.ID())
		}()

		fm.Fill(ctx, fillOrder)
	})
}
