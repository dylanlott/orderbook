package orderbook

import "testing"

func TestInsert(t *testing.T) {
	root := NewNode(10)
	order1 := &Order{ID: "1", Price: 5}
	order2 := &Order{ID: "2", Price: 15}
	order3 := &Order{ID: "3", Price: 8}
	order4 := &Order{ID: "4", Price: 12}

	root.Insert(order1)
	root.Insert(order2)
	root.Insert(order3)
	root.Insert(order4)

	if root.Price != 10 {
		t.Errorf("Expected root price to be 10, got %v", root.Price)
	}

	if len(root.Orders) != 0 {
		t.Errorf("Expected root orders to be empty, got %v", root.Orders)
	}

	left := root.Left
	if left == nil || left.Price != 5 {
		t.Errorf("Expected node with price 5 to exist on the left of root")
	}

	if len(left.Orders) != 1 || left.Orders[0].ID != "1" {
		t.Errorf("Expected node with price 5 to have order with ID 1")
	}

	right := root.Right
	if right == nil || right.Price != 15 {
		t.Errorf("Expected node with price 15 to exist on the right of root")
	}

	if len(right.Orders) != 0 {
		t.Errorf("Expected node with price 15 to have no orders")
	}

	rightLeft := right.Left
	if rightLeft == nil || rightLeft.Price != 12 {
		t.Errorf("Expected node with price 12 to exist on the left of node with price 15")
	}

	if len(rightLeft.Orders) != 1 || rightLeft.Orders[0].ID != "4" {
		t.Errorf("Expected node with price 12 to have order with ID 4")
	}

	rightRight := right.Right
	if rightRight == nil || rightRight.Price != 8 {
		t.Errorf("Expected node with price 8 to exist on the right of node with price 15")
	}

	if len(rightRight.Orders) != 1 || rightRight.Orders[0].ID != "3" {
		t.Errorf("Expected node with price 8 to have order with ID 3")
	}
}
