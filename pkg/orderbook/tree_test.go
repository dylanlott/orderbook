package orderbook

import (
	"testing"

	"github.com/matryer/is"
)

func TestInsert(t *testing.T) {
	is := is.New(t)
	root := NewNode(10)

	order1 := &Order{ID: "1", Price: 5}
	order2 := &Order{ID: "2", Price: 15}
	order3 := &Order{ID: "3", Price: 8}
	order4 := &Order{ID: "4", Price: 12}
	order5 := &Order{ID: "5", Price: 12}
	order6 := &Order{ID: "6", Price: 12}

	root.Insert(order1)
	root.Insert(order2)
	root.Insert(order3)
	root.Insert(order4)
	root.Insert(order5)
	root.Insert(order6)

	root.Print()

	ten := root.Find(10)
	is.Equal(len(ten.Orders), 0)
}

func TestFind(t *testing.T) {
	is := is.New(t)
	root := NewNode(10)

	order1 := &Order{ID: "1", Price: 5}
	order2 := &Order{ID: "2", Price: 15}
	order3 := &Order{ID: "3", Price: 8}
	order4 := &Order{ID: "4", Price: 12}
	order5 := &Order{ID: "5", Price: 12}
	order6 := &Order{ID: "6", Price: 12}

	root.Insert(order1)
	root.Insert(order2)
	root.Insert(order3)
	root.Insert(order4)
	root.Insert(order5)
	root.Insert(order6)

	n := root.Find(12)
	is.Equal(len(n.Orders), 3)

	// Finding a root that doesn't exist should create it and return the root.
	n = root.Find(9)
	n.Print()
	is.Equal(n.Price, uint64(9))
	is.Equal(len(n.Orders), 0)
}
