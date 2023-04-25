package orderbook

import "fmt"

type Node struct {
	Price  uint64
	Orders []*Order
	Left   *Node
	Right  *Node
}

func NewNode(price uint64) *Node {
	return &Node{
		Price:  price,
		Orders: make([]*Order, 0),
	}
}

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

// Removes an Order from a Node's list of Orders.
// * Must be called on the correct node.
func (n *Node) RemoveOrder(orderID string) bool {
	for i, order := range n.Orders {
		if order.ID == orderID {
			n.Orders = append(n.Orders[:i], n.Orders[i+1:]...)
			return true
		}
	}
	return false
}

func (n *Node) AddOrder(order *Order) {
	n.Orders = append(n.Orders, order)
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

func (n *Node) Print() {
	if n == nil {
		return
	}
	n.Left.Print()
	fmt.Printf("Price: %d, Orders: %v\n", n.Price, n.Orders)
	n.Right.Print()
}

func (n *Node) Rightmost() *Node {
	if n == nil {
		return nil
	}
	if n.Right != nil {
		return n.Right.Rightmost()
	}
	return n
}

func (n *Node) Leftmost() *Node {
	if n == nil {
		return nil
	}
	if n.Left != nil {
		return n.Left.Leftmost()
	}
	return n
}

func (n *Node) HighestPrice() uint64 {
	if n == nil {
		return 0
	}
	if len(n.Orders) > 0 {
		return n.Price
	}
	return n.Right.HighestPrice()
}

func (n *Node) LowestPrice() uint64 {
	if n == nil {
		return 0
	}
	if len(n.Orders) > 0 {
		return n.Price
	}
	return n.Left.LowestPrice()
}
