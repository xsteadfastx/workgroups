# workgroups

Package workgroups is a little helper for creating workers
with the help of [sync.Errgroup](https://pkg.go.dev/golang.org/x/sync/errgroup).

![build](https://ci.xsfx.dev/api/badges/xsteadfastx/workgroups/status.svg)
![coverage](https://codecov.io/gh/xsteadfastx/workgroups/branch/main/graph/badge.svg?token=RZE1ZWJSYA)
![report](https://goreportcard.com/badge/go.xsfx.dev/workgroups)

![readme](https://git.xsfx.dev/xsteadfastx/workgroups/raw/branch/main/README.gif)

## Links

* [build](https://ci.xsfx.dev/xsteadfastx/workgroups)
* [coverage](https://codecov.io/github/xsteadfastx/workgroups/)
* [report](https://goreportcard.com/report/go.xsfx.dev/workgroups)
* [reference](https://pkg.go.dev/go.xsfx.dev/workgroups)

## Variables

ErrInWorker is an error that gets returned if there is a error
in the work function.

```golang
var ErrInWorker = errors.New("received error in worker")
```

## Types

### type [Dispatcher](/workgroups.go#L52)

`type Dispatcher struct { ... }`

Dispatcher carries the job queue, the errgroup and the number of workers
to start.

#### func (*Dispatcher) [Append](/workgroups.go#L102)

`func (d *Dispatcher) Append(job Job)`

Append adds a job to the work queue.

#### func (*Dispatcher) [Close](/workgroups.go#L108)

`func (d *Dispatcher) Close()`

Close closes the queue channel.

#### func (*Dispatcher) [Start](/workgroups.go#L71)

`func (d *Dispatcher) Start()`

Start starts the configured number of workers and waits for jobs.

#### func (*Dispatcher) [Wait](/workgroups.go#L114)

`func (d *Dispatcher) Wait() error`

Wait for all jobs to finnish.

### type [Job](/workgroups.go#L40)

`type Job struct { ... }`

Job carries a job with everything it needs.
I know know that contexts shouldnt be stored in a struct.
Here is an exception, because its a short living object.
The context is only used as argument for the Work function.
Please use the NewJob function to get around this context in struct shenanigans.

### type [Work](/workgroups.go#L33)

`type Work func(ctx context.Context) error`

Work is a type that defines worker work.

## Examples

```golang
package main

import (
	"context"
	"fmt"
	"go.xsfx.dev/workgroups"
	"log"
	"runtime"
)

func main() {
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
	d.Start()

	// Feeding the workers some work.
	d.Append(workgroups.NewJob(ctx, work))

	// Closing the channel for work.
	d.Close()

	// Waiting to finnish everything.
	if err := d.Wait(); err != nil {
		log.Fatal(err)
	}

}

```

 Output:

```
hello world from work
```
