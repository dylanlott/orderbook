package orderbook

import (
	"fmt"
)

// Node is the atomic unit of the price index
// for the Orderbook. Multiple Nodes are bound
// together in a binary tree. Each Node has a
// left and right node. Nodes without any orders
// are removed from the index.
type Node struct {
	Price  uint64
	Orders []*Order
	Left   *Node
	Right  *Node
}

// NewNode is a constructor function for returning
// a new default Node.
func NewNode(price uint64) *Node {
	return &Node{
		Price:  price,
		Orders: make([]*Order, 0),
	}
}

// Insert adds an Order into the tree and returns
// the Node that it was inserted into.
func (n *Node) Insert(order *Order) *Node {
	if n == nil {
		n = NewNode(order.Price)
		return n.Insert(order)
	}

	switch {
	case order.Price < n.Price:
		n.Left = n.Left.Insert(order)
	case order.Price > n.Price:
		n.Right = n.Right.Insert(order)
	default:
		n.Orders = append(n.Orders, order)
	}
	return n
}

// RemoveOrder removes an Order from a Node's list of Orders.
// * Must be called on the correct node.
func (n *Node) RemoveOrder(order *Order) bool {
	found := n.Find(order.Price)

	if found.Price == order.Price {
		for i, o := range found.Orders {
			if order.ID == o.ID {
				// slice the order out of the found nodes orderlist
				n.Orders = append(found.Orders[:i], found.Orders[i+1:]...)
				return true
			}
		}
	}
	return false
}

// Find returns the node for a given price.
func (n *Node) Find(price uint64) *Node {
	if n == nil {
		n := NewNode(price)
		return n
	}
	if price == n.Price {
		return n
	} else if price < n.Price {
		return n.Left.Find(price)
	} else {
		return n.Right.Find(price)
	}
}

// List returns a list of all the Orders in the tree.
func (n *Node) List() []*Order {
	if n == nil {
		return nil
	}
	left := n.Left.List()
	right := n.Right.List()
	orders := make([]*Order, 0, len(n.Orders)+len(left)+len(right))
	orders = append(orders, left...)
	orders = append(orders, n.Orders...)
	orders = append(orders, right...)
	return orders
}

// Print prints the contents of the tree to stdout
func (n *Node) Print() {
	if n == nil {
		return
	}
	n.Left.Print()
	fmt.Printf("Price: %d, Orders: %v\n", n.Price, n.Orders)
	n.Right.Print()
}

// FindMin returns the lowest price in the tree.
func (n *Node) FindMin() *Node {
	if n == nil {
		return nil
	}
	if n.Left == nil {
		return n
	}
	return n.Left.FindMin()
}

// FindMax returns the highest price node in the tree
func (n *Node) FindMax() *Node {
	if n == nil {
		return nil
	}
	if n.Right == nil {
		return n
	}
	return n.Right.FindMax()
}
