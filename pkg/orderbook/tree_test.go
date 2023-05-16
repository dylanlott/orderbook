package orderbook

import (
	"testing"

	"github.com/matryer/is"
)

func TestInsert(t *testing.T) {
	is := is.New(t)
	root := NewNode(10)
	seedRootTree(root)
	ten := root.Find(10)
	is.Equal(len(ten.Orders), 0)
}

func TestFind(t *testing.T) {
	is := is.New(t)

	root := NewNode(10)
	seedRootTree(root)

	n := root.Find(12)
	is.Equal(len(n.Orders), 3)

	n = root.Find(9)
	n.Print()
	is.Equal(n.Price, uint64(9))
	is.Equal(len(n.Orders), 0)

	highest := root.FindMax()
	lowest := root.FindMin()
	is.Equal(highest.Price, uint64(15))
	is.Equal(lowest.Price, uint64(5))
}

func TestList(t *testing.T) {
	is := is.New(t)
	root := NewNode(10)
	seedRootTree(root)
	orderlist := root.List()
	is.Equal(len(orderlist), 6)
	is.Equal(orderlist[0].Price, uint64(5))
	is.Equal(orderlist[5].Price, uint64(15))
}

func seedRootTree(root *Node) {
	order1 := &Order{ID: "1", Price: 5, Side: "buy"}
	order2 := &Order{ID: "2", Price: 15, Side: "buy"}
	order3 := &Order{ID: "3", Price: 8, Side: "buy"}
	order4 := &Order{ID: "4", Price: 12, Side: "buy"}
	order5 := &Order{ID: "5", Price: 12, Side: "buy"}
	order6 := &Order{ID: "6", Price: 12, Side: "buy"}

	root.Insert(order1)
	root.Insert(order2)
	root.Insert(order3)
	root.Insert(order4)
	root.Insert(order5)
	root.Insert(order6)
}

func TestNode_RemoveOrder(t *testing.T) {
	is := is.New(t)
	root := NewNode(10)
	seedRootTree(root)
	is.True(root.RemoveOrder(&Order{ID: "5", Price: 12}))
}
