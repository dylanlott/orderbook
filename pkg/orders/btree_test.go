package orders

import (
	"testing"

	"github.com/matryer/is"
)

func TestTreeNodeInsert(t *testing.T) {
	t.Run("insert right", func(t *testing.T) {
		is := is.New(t)
		root := &TreeNode{val: 0.0}
		o := &MarketOrder{
			MarketPrice: 10.00,
		}
		err := root.Insert(o)
		is.NoErr(err)
		is.True(root.right != nil)
	})
	t.Run("insert left", func(t *testing.T) {
		is := is.New(t)
		root := &TreeNode{val: 10.0}
		o := &MarketOrder{
			MarketPrice: 5.00,
		}
		err := root.Insert(o)
		is.NoErr(err)
		is.True(root.left != nil)
	})
	t.Run("insert order at equal price", func(t *testing.T) {
		is := is.New(t)
		root := &TreeNode{val: 10.0}
		o := &MarketOrder{
			MarketPrice: 10.0,
		}
		err := root.Insert(o)
		is.NoErr(err)
		is.True(len(root.orders) > 0)
		is.True(root.orders[0] == o)
	})
}

func TestTreeNodeFind(t *testing.T) {
	is := is.New(t)
	root := &TreeNode{val: 10.0}
	err := root.Insert(&MarketOrder{
		UUID:        "0xACAB",
		MarketPrice: 10.0,
	})
	is.NoErr(err)
	err = root.Insert(&MarketOrder{
		UUID:        "0xDEAD",
		MarketPrice: 5.0,
	})
	is.NoErr(err)
	err = root.Insert(&MarketOrder{
		UUID:        "0xBEEF",
		MarketPrice: 15.0,
	})
	is.NoErr(err)
	err = root.Insert(&MarketOrder{
		UUID:        "0xDEED",
		MarketPrice: 13.0,
	})
	is.NoErr(err)
	err = root.Insert(&MarketOrder{
		UUID:        "0xDEED",
		MarketPrice: 8.5,
	})
	is.NoErr(err)
	root.PrintInorder()
	ord, err := root.Find(15.0)
	is.NoErr(err)
	is.True(ord.ID() == "0xBEEF")
}
