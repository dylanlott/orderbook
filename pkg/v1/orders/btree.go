package orders

import (
	"fmt"
	"log"
	"sync"
)

// TreeNode represents a tree of nodes that maintain lists of Orders at that price.
// * Each TreeNode maintains an ordered list of Orders that share the same price.
// * This tree is a simple binary tree, where left nodes are lesser prices and right
// nodes are greater in price than the current node.
type TreeNode struct {
	sync.Mutex

	val    float64 // to represent price
	orders []Order
	right  *TreeNode
	left   *TreeNode
}

// Insert will add an Order to the Tree. It traverses until it finds the right price
// or where the price should exist and creates a price node if it doesn't exist, then
// adds the Order to that price node.
func (t *TreeNode) Insert(o Order) error {
	t.Lock()

	if t == nil {
		t = &TreeNode{val: o.Price()}
	}

	if t.val == o.Price() {
		// when we find a price match for the Order's price,
		// insert the Order into this node's Order list.
		if t.orders == nil {
			t.orders = make([]Order, 0)
		}
		t.orders = append(t.orders, o)
		t.Unlock()
		return nil
	}

	if o.Price() < t.val {
		if t.left == nil {
			t.left = &TreeNode{val: o.Price()}
			t.Unlock()
			return t.left.Insert(o)
		}
		t.Unlock()
		return t.left.Insert(o)
	}

	if o.Price() > t.val {
		if t.right == nil {
			t.right = &TreeNode{val: o.Price()}
			t.Unlock()
			return t.right.Insert(o)
		}
		t.Unlock()
		return t.right.Insert(o)
	}

	panic("should not get here; this smells like a bug")
}

// Find returns the highest priority Order for a given price point.
// * If it can't find an order at that exact price, it will search for
// a cheaper order if one exists.
func (t *TreeNode) Find(price float64) (Order, error) {
	if t == nil {
		return nil, fmt.Errorf("err no exist")
	}

	t.Lock()

	if price == t.val {
		if len(t.orders) > 0 {
			defer t.Unlock()
			return t.orders[0], nil
		}
		defer t.Unlock()
		return nil, fmt.Errorf("no orders at this price")
	}

	if price > t.val {
		if t.right != nil {
			return t.right.Find(price)
		}
	}

	if price < t.val {
		if t.left != nil {
			return t.left.Find(price)
		}
	}

	panic("should not get here; this smells like a bug")
}

// Match will iterate through the tree based on the price of the
// fillOrder and fijjnds a bookOrder that matches its price.
func (t *TreeNode) Match(fillOrder Order, cb func(bookOrder Order)) {
	if t == nil {
		cb(nil)
		return
	}

	t.Lock()

	if fillOrder.Price() == t.val {
		// callback with first order in the list
		bookOrder := t.orders[0]
		cb(bookOrder)
		t.Unlock()
		return
	}

	t.Unlock()

	if fillOrder.Price() > t.val {
		if t.right != nil {
			t.right.Match(fillOrder, cb)
			return
		}
	}

	if fillOrder.Price() < t.val {
		if t.left != nil {
			t.left.Match(fillOrder, cb)
			return
		}
	}

	panic("should not get here; this smells like a bug")
}

// Orders returns the list of Orders for a given price.
func (t *TreeNode) Orders(price float64) ([]Order, error) {
	if t == nil {
		return nil, fmt.Errorf("order tree is nil")
	}

	if t.val == price {
		// READ AT
		t.Lock()
		defer t.Unlock()
		return t.orders, nil
	}

	if price > t.val {
		if t.right != nil {
			return t.right.Orders(price)
		}
	}

	if price < t.val {
		if t.left != nil {
			return t.left.Orders(price)
		}
	}

	panic("should not get here; this smells like a bug")
}

func (t *TreeNode) InOrder() []Order {
	return inOrderTraversal(t)
}

func inOrderTraversal(root *TreeNode) []Order {
	if root == nil {
		return nil
	}

	left := inOrderTraversal(root.left)
	right := inOrderTraversal(root.right)

	output := make([]Order, 0)
	output = append(output, left...)
	output = append(output, root.orders...)
	output = append(output, right...)

	return output
}

// RemoveFromPriceList removes an order from the list of orders at a
// given price in our tree. It does not currently rebalance the tree.
// TODO: make this rebalance the tree at some threshold.
func (t *TreeNode) RemoveFromPriceList(order Order) error {
	if t == nil {
		return fmt.Errorf("order tree is nil")
	}

	if order == nil {
		log.Printf("Nil order detected")
		return fmt.Errorf("ErrNilOrder")
	}

	if order.Price() == t.val {
		for i, ord := range t.orders {
			t.Lock()
			if ord.ID() == order.ID() {
				t.orders = remove(t.orders, i)
				t.Unlock() // NB: make sure to unlock in both paths
				return nil
			}
			t.Unlock()
		}
		return fmt.Errorf("ErrNoExist")
	}

	if order.Price() > t.val {
		if t.right != nil {
			return t.right.RemoveFromPriceList(order)
		}
		return fmt.Errorf("ErrNoExist")
	}

	if order.Price() < t.val {
		if t.left != nil {
			return t.left.RemoveFromPriceList(order)
		}
		return fmt.Errorf("ErrNoExist")
	}

	panic("should not get here; this smells like a bug")
}

//PrintInorder prints the elements in left-current-right order.
func (t *TreeNode) PrintInorder() {
	if t == nil {
		return
	}
	t.left.PrintInorder()
	fmt.Printf("%+v\n", t.val)
	t.right.PrintInorder()
}
