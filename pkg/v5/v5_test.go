package v5

import (
	"context"
	"log"
	"testing"
)

func TestListen(t *testing.T) {
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	reads := make(<-chan OpRead)
	writes := make(<-chan OpWrite)
	out := make(chan Book)

	// Listen kicks off and processes reads and writes concurrently
	go Listen(ctx, reads, writes, out)

	// Listens for processed updates to the books
	for _, update := range out {
		log.Printf("[update] %+v", update)
		// return after we process one
		return
	}
}
