package orders

import (
	"testing"

	"github.com/matryer/is"
)

func TestDone(t *testing.T) {
	t.Run("should send on done when filled", func(t *testing.T) {
		is := is.New(t)
		o := &MarketOrder{
			OpenQuantity:   1,
			FilledQuantity: 0,
			done:           make(chan Order),
		}
		go func() {
			got := <-o.Done()
			is.Equal(got.Quantity(), int64(0))
		}()
		_, err := o.Update(0, 1)
		is.NoErr(err)
	})
}
