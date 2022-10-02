package orders

import (
	"testing"
	"time"

	"github.com/dylanlott/orderbook/pkg/accounts"

	"github.com/matryer/is"
)

func TestAttemptFill(t *testing.T) {
	is := is.New(t)

	t.Run("should fill an order", func(t *testing.T) {
		tree := setupTree(t)
		m := &market{
			Accounts: &accounts.InMemoryManager{
				Accounts: map[string]*accounts.UserAccount{
					"buyer@test.com": {
						Email:          "buyer@test.com",
						CurrentBalance: 2000.0,
					},
					"seller@test.com": {
						Email:          "seller@test.com",
						CurrentBalance: 1000.0,
					},
				},
			},
			OrderTrie: tree,
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

		err := m.OrderTrie.Insert(bookOrder)
		is.NoErr(err)

		err = m.OrderTrie.Insert(fillOrder)
		is.NoErr(err)

		err = m.attemptFill(fillOrder, bookOrder)
		is.NoErr(err)

		m.OrderTrie.PrintInorder()
	})
}

func TestPlace(t *testing.T) {

}

func TestDone(t *testing.T) {
	t.Run("should send on done when filled", func(t *testing.T) {
		is := is.New(t)
		o := &MarketOrder{
			OpenQuantity:   1,
			FilledQuantity: 0,
			done:           make(chan Order),
		}
		go func() {
			got := <-o.Done()
			is.Equal(got.Quantity(), int64(0))
		}()
		_, err := o.Update(0, 1)
		is.NoErr(err)
	})
}
