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
}

func TestNode_Remove(t *testing.T) {
	var five = &Node{
		Price: 5,
		Right: nil,
		Left:  nil,
	}
	var one = &Node{
		Price: 1,
		Right: nil,
		Left:  nil,
	}
	type fields struct {
		Price  uint64
		Orders []*Order
		Left   *Node
		Right  *Node
	}
	type args struct {
		value uint64
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Node
	}{
		{
			name: "should remove a childless node",
			fields: fields{
				Price: 2,
				Left:  one,
				Right: five,
			},
			args: args{
				value: 5,
			},
			want: &Node{
				Price: 2,
				Left:  one,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := &Node{
				Price:  tt.fields.Price,
				Orders: tt.fields.Orders,
				Left:   tt.fields.Left,
				Right:  tt.fields.Right,
			}
			got := n.Remove(tt.args.value)
			if got.Price != tt.want.Price {
				t.Fail()
			}
			if tt.want.Right != nil && got.Right.Price != tt.want.Right.Price {
				t.Fail()
			}
			if tt.want.Left != nil && got.Left.Price != tt.want.Left.Price {
				t.Fail()
			}
		})
	}
}
