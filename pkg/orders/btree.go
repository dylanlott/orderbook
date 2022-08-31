package orders

import (
	"fmt"
	"sort"
)

// TreeNode represents a tree of nodes that maintain lists of Orders at that price.
type TreeNode struct {
	val    float64 // to represent price
	orders []Order
	right  *TreeNode
	left   *TreeNode
}

// Insert will add an Order to the Tree.
func (t *TreeNode) Insert(o Order) error {
	if t == nil {
		t = &TreeNode{val: o.Price()}
	}

	if t.val == o.Price() {
		// when we find a price match for the order,
		// insert the order into this node's order list.
		if t.orders == nil {
			t.orders = make([]Order, 0)
		}
		t.orders = append(t.orders, o)
		return nil
	}

	if t.val > o.Price() {
		if t.left == nil {
			t.left = &TreeNode{val: o.Price()}
			return t.left.Insert(o)
		}
		return t.left.Insert(o)
	}

	if t.val < o.Price() {
		if t.right == nil {
			t.right = &TreeNode{val: o.Price()}
			return t.right.Insert(o)
		}
		return t.right.Insert(o)
	}

	panic("should not get here; this smells like a bug")
}

// Find returns the highest priority order for a given price point.
// It returns the Order or an error.
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

	return nil, fmt.Errorf("ErrFind")
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
