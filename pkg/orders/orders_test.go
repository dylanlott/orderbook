package orders

import (
	"testing"
	"time"

	"github.com/dylanlott/orderbook/pkg/accounts"
	"github.com/matryer/is"
)

func TestOrders(t *testing.T) {
	t.Run("should add an order to the order list", func(t *testing.T) {
		m := &market{}
		_, err := m.Place(&MarketOrder{
			Asset: AssetInfo{
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
		})
		// TODO: make order fulfill interface correctly
		if err != nil {
			t.Errorf("failed to place market order: %v", err)
		}
	})
}

func TestFilling(t *testing.T) {
	t.Run("should go fill when order is placed", func(t *testing.T) {
		is := is.New(t)
		m := &market{
			Accounts: &accounts.InMemoryManager{
				Accounts: map[string]*accounts.UserAccount{
					"buyer@test.com": &accounts.UserAccount{
						Email:          "buyer@test.com",
						CurrentBalance: 2000.0,
					},
					"seller@test.com": &accounts.UserAccount{
						Email:          "seller@test.com",
						CurrentBalance: 1000.0,
					},
				},
			},
		}
		ord, err := m.Place(&MarketOrder{
			UUID: "buyer456",
			Asset: AssetInfo{
				Underlying: "USD",
				Name:       "ETH",
			},
			UserAccount: &accounts.UserAccount{
				Email:          "buyer@test.com",
				CurrentBalance: 1200.0,
			},
			OpenQuantity:   1,
			FilledQuantity: 0,
			PlacedAt:       time.Now(),
			MarketPrice:    50.0,
		})
		is.NoErr(err)
		is.Equal(ord.ID(), "buyer456")
	})
}

func TestAttemptFill(t *testing.T) {
	is := is.New(t)

	t.Run("should fully fill an order", func(t *testing.T) {
		m := &market{
			Accounts: &accounts.InMemoryManager{
				Accounts: map[string]*accounts.UserAccount{
					"buyer@test.com": &accounts.UserAccount{
						Email:          "buyer@test.com",
						CurrentBalance: 2000.0,
					},
					"seller@test.com": &accounts.UserAccount{
						Email:          "seller@test.com",
						CurrentBalance: 1000.0,
					},
				},
			},
			OrderTrie: setupTree(t),
		}

		bookOrder := &MarketOrder{
			UUID: "seller456",
			Asset: AssetInfo{
				Underlying: "ETH",
				Name:       "USD",
			},
			UserAccount: &accounts.UserAccount{
				Email: "seller@test.com",
			},
			OpenQuantity:   1,
			FilledQuantity: 0,
			PlacedAt:       time.Now(),
			MarketPrice:    50.0,
		}

		_, err := m.Place(bookOrder)
		is.NoErr(err)

		fillOrder := &MarketOrder{
			UUID: "buyer456",
			Asset: AssetInfo{
				Underlying: "USD",
				Name:       "ETH",
			},
			UserAccount: &accounts.UserAccount{
				Email: "buyer@test.com",
			},
			OpenQuantity:   1,
			FilledQuantity: 0,
			PlacedAt:       time.Now(),
			MarketPrice:    50.0,
		}

		err = m.attemptFill(fillOrder, bookOrder)
		is.NoErr(err)
		buyer, err := m.Accounts.Get("buyer@test.com")
		is.NoErr(err)
		is.Equal(buyer.Balance(), float64(1950))
	})
}
