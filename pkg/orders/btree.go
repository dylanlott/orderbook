package orders

import (
	"fmt"
	"sort"
)

// TreeNode represents a tree of nodes that maintain lists of Orders at that price.
// * Each TreeNode maintains an ordered list of Orders that share the same price.
// * This tree is a simple binary tree, where left nodes are lesser prices and right
// nodes are greater in price than the current node.
type TreeNode struct {
	val    float64 // to represent price
	orders []Order
	right  *TreeNode
	left   *TreeNode
}

// Insert will add an Order to the Tree. It traverses until it finds the right price
// or where the price should exist and creates a price node if it doesn't exist, then
// adds the Order to that price node.
func (t *TreeNode) Insert(o Order) error {
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
		return nil
	}

	if o.Price() < t.val {
		if t.left == nil {
			t.left = &TreeNode{val: o.Price()}
			return t.left.Insert(o)
		}
		return t.left.Insert(o)
	}

	if o.Price() > t.val {
		if t.right == nil {
			t.right = &TreeNode{val: o.Price()}
			return t.right.Insert(o)
		}
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

	if price == t.val {
		if len(t.orders) > 0 {
			return t.orders[0], nil
		}
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
// fillOrder and finds a bookOrder that matches its price.
func (t *TreeNode) Match(fillOrder Order, cb func(bookOrder Order)) {
	if t == nil {
		cb(nil)
		return
	}

	if fillOrder.Price() == t.val {
		// callback with first order in the list
		bookOrder := t.orders[0]
		cb(bookOrder)
		return
	}

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

//PrintInorder prints the elements in left-current-right order.
func (t *TreeNode) PrintInorder() {
	if t == nil {
		return
	}
	t.left.PrintInorder()
	fmt.Printf("%+v\n", t.val)
	t.right.PrintInorder()
}

// sortByTimePriority sorts orders by oldest to newest
func sortByTimePriority(orders []Order) []Order {
	sort.SliceStable(orders, func(i, j int) bool {
		return orders[i].CreatedAt().After(orders[j].CreatedAt())
	})
	return orders
}
