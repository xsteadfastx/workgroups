package workgroups_test

import (
	"context"
	"fmt"
	"log"
	"runtime"

	"go.xsfx.dev/workgroups"
)

func Example() {
	d, ctx := workgroups.NewDispatcher(
		context.Background(),
		runtime.GOMAXPROCS(0), // This starts as much worker as maximal processes are allowed for go.
	)

	work := func(ctx context.Context) error {
		// Check if context already expired.
		// Return if its the case, else just go forward.
		select {
		case <-ctx.Done():
			return fmt.Errorf("got error from context: %w", ctx.Err())
		default:
		}

		// Some wannebe work... printing something.
		fmt.Print("hello world from work")

		return nil
	}

	// Starting up the workers.
	d.Start(ctx)

	// Feeding the workers some work.
	d.Append(work)

	// Closing the channel for work.
	d.Close()

	// Waiting to finnish everything.
	if err := d.Wait(); err != nil {
		log.Fatal(err)
	}

	// Output:
	// hello world from work
}
