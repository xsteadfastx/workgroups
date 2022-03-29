package workgroups_test

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"time"

	"go.xsfx.dev/workgroups"
)

func Example() {
	d, ctx := workgroups.NewDispatcher(
		context.Background(),
		runtime.GOMAXPROCS(0), // This starts as much worker as maximal processes are allowed for go.
		10,                    // Capacity of the queue.
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
	d.Start()

	// Feeding the workers some work.
	d.Append(workgroups.NewJob(ctx, work))

	// Closing the channel for work.
	d.Close()

	// Waiting to finnish everything.
	if err := d.Wait(); err != nil {
		log.Fatal(err)
	}

	// Output:
	// hello world from work
}

func ExampleRetry() {
	d, ctx := workgroups.NewDispatcher(
		context.Background(),
		runtime.GOMAXPROCS(0), // This starts as much worker as maximal processes are allowed for go.
		10,                    // Capacity of the queue.
	)

	// Just returning some error. So it can retry.
	failFunc := func() error {
		fmt.Print("fail ")

		return errors.New("fail") //nolint:goerr113
	}

	work := func(ctx context.Context) error {
		// Check if context already expired.
		// Return if its the case, else just go forward.
		select {
		case <-ctx.Done():
			return fmt.Errorf("got error from context: %w", ctx.Err())
		default:
		}

		if err := failFunc(); err != nil {
			return err
		}

		return nil
	}

	// Starting up the workers.
	d.Start()

	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	// Feeding the workers some work.
	d.Append(
		workgroups.NewJob(
			ctx,
			workgroups.Retry(ctx, time.Second/2)(work), // This will retry after a half second.
		),
	)

	// Closing the channel for work.
	d.Close()

	// Waiting to finnish everything.
	if err := d.Wait(); err != nil {
		fmt.Print(err)
	}

	// Output:
	// fail fail error on waiting: got error from context: context deadline exceeded
}
