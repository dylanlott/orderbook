package v4

import (
	"fmt"
	"sync"
)

type readOp struct{}
type writeOp struct{}

type books[T any] struct {
	buy  *sync.Map
	sell *sync.Map
}

type order struct {
	ID    string
	Price uint64
	Side  bool
}

func newBooks[Order any]() *books[Order] {
	b := &books[Order]{
		buy:  &sync.Map{},
		sell: &sync.Map{},
	}
	return b
}

func (b *books[T]) push(order *order) {

}

func (b *books[T]) match(o *order, cb func(matched *order)) error {
	return fmt.Errorf("not impl")
}

func (b *books[T]) pull() {

}
