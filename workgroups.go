// Package workgroups is a little helper for creating workers
// with the help of (sync.Errgroup) https://pkg.go.dev/golang.org/x/sync/errgroup.
//
// (image/build) https://ci.xsfx.dev/api/badges/xsteadfastx/workgroups/status.svg
// (image/coverage) https://codecov.io/gh/xsteadfastx/workgroups/branch/main/graph/badge.svg?token=RZE1ZWJSYA
// (image/report) https://goreportcard.com/badge/go.xsfx.dev/workgroups
//
// (image/readme) https://git.xsfx.dev/xsteadfastx/workgroups/raw/branch/main/README.gif
//
// Links
//
// * (build) https://ci.xsfx.dev/xsteadfastx/workgroups
// * (coverage) https://codecov.io/github/xsteadfastx/workgroups/
// * (report) https://goreportcard.com/report/go.xsfx.dev/workgroups
// * (reference) https://pkg.go.dev/go.xsfx.dev/workgroups
//
package workgroups

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	"golang.org/x/sync/errgroup"
)

// ErrInWorker is an error that gets returned if there is a error
// in the work function.
var ErrInWorker = errors.New("received error in worker")

// Work is a type that defines worker work.
type Work func(ctx context.Context) error

// Job carries a job with everything it needs.
// I know know that contexts shouldnt be stored in a struct.
// Here is an exception, because its a short living object.
// The context is only used as argument for the Work function.
// Please use the NewJob function to get around this context in struct shenanigans.
type Job struct {
	ctx  context.Context //nolint:containedctx
	work Work
}

// NewJob creates a new Job to send down the work queue with context and all that stuff.
func NewJob(ctx context.Context, work Work) Job {
	return Job{ctx, work}
}

// Dispatcher carries the job queue, the errgroup and the number of workers
// to start.
type Dispatcher struct {
	queue      chan Job
	eg         *errgroup.Group
	numWorkers int
	log        logr.Logger
}

// NewDispatcher creates a new Dispatcher.
// It takes a context and adds it to the errgroup creation and returns it again.
func NewDispatcher(ctx context.Context, log logr.Logger, numWorkers, workLength int) (*Dispatcher, context.Context) {
	eg, ctx := errgroup.WithContext(ctx)

	return &Dispatcher{
		queue:      make(chan Job, workLength),
		eg:         eg,
		numWorkers: numWorkers,
		log:        log,
	}, ctx
}

// Start starts the configured number of workers and waits for jobs.
func (d *Dispatcher) Start() {
	for i := 0; i < d.numWorkers; i++ {
		d.log.V(1).Info("starting worker", "worker", i)
		i := i

		d.eg.Go(func() error {
			for j := range d.queue {
				errChan := make(chan error)

				go func() {
					err := j.work(j.ctx)
					select {
					case <-j.ctx.Done():
						return
					default:
						errChan <- err
					}
				}()

				select {
				case <-j.ctx.Done():
					close(errChan)

					return fmt.Errorf("got error from context: %w", j.ctx.Err())
				case err := <-errChan:
					if err != nil {
						return fmt.Errorf("go error from work function: %w", err)
					}
				}
			}

			d.log.V(1).Info("no work. returning...", "worker", i)

			return nil
		})
	}
}

// Append adds a job to the work queue.
func (d *Dispatcher) Append(job Job) {
	d.log.V(1).Info("adds job")
	d.queue <- job
}

// Close closes the queue channel.
func (d *Dispatcher) Close() {
	d.log.V(1).Info("closing queue")
	close(d.queue)
}

// Wait for all jobs to finnish.
func (d *Dispatcher) Wait() error {
	d.log.V(1).Info("waiting for jobs to finnish")

	if err := d.eg.Wait(); err != nil {
		return fmt.Errorf("error on waiting: %w", err)
	}

	return nil
}

// Retry is a middleware for doing a retry in executing job work.
func Retry(ctx context.Context, wait time.Duration) func(Work) Work {
	return func(next Work) Work {
		return func(ctx context.Context) error {
			for {
				if err := next(ctx); err == nil {
					return nil
				}

				select {
				case <-ctx.Done():
					return fmt.Errorf("timeout while fetching information (last error: %w)", ctx.Err())
				case <-time.After(wait):
				}
			}
		}
	}
}
