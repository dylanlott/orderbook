package v4

import (
	"testing"

	"github.com/matryer/is"
)

func TestV4(t *testing.T) {
	is := is.New(t)
	b := newBooks[*order]()
	is.True(b.buy != nil)
	is.True(b.sell != nil)
}
