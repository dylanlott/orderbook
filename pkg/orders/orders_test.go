package orders

import "testing"

func TestOrders(t *testing.T) {
	t.Run("should add an order to the order list", func(t *testing.T) {
		m := &market{}
		o := &MarketOrder{}
		placed, err := m.Place(o) // TODO: make ORder fulfill interface correctly
		if err != nil {
			t.Errorf("failed to place market order")
		}
		t.Logf("placed order successfully %v", placed)
	})

	t.Run("should cancel an order", func(t *testing.T) {
		m := &market{}
		o := &MarketOrder{}
		placed, err := m.Place(o)
		if err != nil {
			t.Errorf("failed to place order: %s", err)
		}
		err = m.Cancel(placed.ID()) // TODO: Make Cancel take an ID
		if err != nil {
			t.Errorf("failed to place order: %s", err)
		}
	})

	t.Run("should fill an order", func(t *testing.T) {
		m := &market{}
		o := &MarketOrder{}
		placed, err := m.Place(o)
		if err != nil {
			t.Errorf("failed to place order: %s", err)
		}
		if placed.ID() == "" {
			t.Errorf("failed to assign ID to ordr")
		}
	})
}
