package v4

import (
	"sync"
	"testing"

	"github.com/matryer/is"
)

func TestV4(t *testing.T) {
	is := is.New(t)

	b := &books{
		buy:  &sync.Map{},
		sell: &sync.Map{},
	}

	err := b.Push(&order{
		ID:    "foo",
		Price: 1000,
		Side:  true,
	})
	is.NoErr(err)
	err = b.Push(&order{
		ID:    "bar",
		Price: 1000,
		Side:  false,
	})
	is.NoErr(err)
	err = b.Push(&order{
		ID:    "buz",
		Price: 1000,
		Side:  true,
	})
	is.NoErr(err)
}
