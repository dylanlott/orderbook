package orderbook

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

func (n *Node) AddOrder(order *Order) {
	n.Orders = append(n.Orders, order)
}

func (n *Node) Insert(order *Order) *Node {
	if n == nil {
		node := NewNode(order.Price)
		return node.Insert(order)
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

func (n *Node) Find(price uint64) *Node {
	if n == nil {
		return nil
	}
	if price == n.Price {
		return n
	} else if price < n.Price {
		return n.Left.Find(price)
	} else {
		return n.Right.Find(price)
	}
}
