package orders

import (
	"testing"
	"time"

	"github.com/dylanlott/orderbook/pkg/accounts"
)

func TestOrders(t *testing.T) {
	t.Run("should add an order to the order list", func(t *testing.T) {
		m := &market{
			Orders: []Order{},
		}
		o := &MarketOrder{
			Asset: Asset{
				Underlying: "USD",
				Name:       "ETH",
			},
			UserAccount: &accounts.UserAccount{
				Email:          "shakezula@test.com",
				CurrentBalance: 1200.0,
			},
			UUID:           "abc123",
			OpenQuantity:   10,
			FilledQuantity: 0,
			PlacedAt:       time.Now(),
			MarketPrice:    50.0,
		}
		_, err := m.Place(o) // TODO: make ORder fulfill interface correctly
		if err != nil {
			t.Errorf("failed to place market order: %v", err)
		}
	})

	t.Run("should cancel an order", func(t *testing.T) {
		m := &market{
			Orders: []Order{
				&MarketOrder{
					Asset: Asset{
						Underlying: "USD",
						Name:       "ETH",
					},
					UserAccount: &accounts.UserAccount{
						Email:          "shakezula@test.com",
						CurrentBalance: 1200.0,
					},
					UUID:           "abc123",
					OpenQuantity:   10,
					FilledQuantity: 0,
					PlacedAt:       time.Now(),
					MarketPrice:    50.0,
				},
			},
		}
		err := m.Cancel("abc123")
		if err != nil {
			t.Errorf("failed to place order: %s", err)
		}
		for _, v := range m.Orders {
			// assert that order is removed from books.
			if v.ID() == "acb123" {
				t.Errorf("order should not exist in books after cancellation")
			}
		}
	})
}
