package v4

import (
	"fmt"
	"sync"
)

type books struct {
	buy  *sync.Map
	sell *sync.Map
}

type order struct {
	ID    string
	Price uint64
	Side  bool
}

func newBooks() *books {
	b := &books{
		buy:  &sync.Map{},
		sell: &sync.Map{},
	}
	return b
}

// Push pushes order [ord] into the books.
func (b *books) Push(ord *order) error {
	if ord.Side {
		// push buy side
		loaded, ok := b.buy.Load(ord.Price)
		if !ok {
			// nothing exists at this price, push it into map
			b.buy.Store(ord.Price, []*order{ord})
			return nil
		}
		if val, ok := loaded.([]*order); ok {
			val = append(val, ord)
		} else {
			return fmt.Errorf("ErrNotImpl for type %T", loaded)
		}
	}
	// push sell side
	loaded, ok := b.sell.Load(ord.Price)
	if !ok {
		// nothing exists at this price, push it into the map at the price
		b.sell.Store(ord.Price, []*order{ord})
		return nil
	}
	if val, ok := loaded.([]*order); ok {
		val = append(val, ord)
	} else {
		return fmt.Errorf("ErrNotImpl for type %T", loaded)
	}
	return nil
}
