// Package workgroups is a little helper for creating workers
// with the help of sync.Errgroup.
//
// (image/build) https://ci.xsfx.dev/api/badges/xsteadfastx/workgroups/status.svg
// (image/coverage) https://codecov.io/gh/xsteadfastx/workgroups/branch/main/graph/badge.svg?token=RZE1ZWJSYA
// (image/report) https://goreportcard.com/badge/go.xsfx.dev/workgroups
// (image/reference) https://pkg.go.dev/badge/go.xsfx.dev/workgroups.svg
//
// (image/readme) ./README.gif
package workgroups

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
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
	ctx  context.Context
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
}

// NewDispatcher creates a new Dispatcher.
// It takes a context and adds it to the errgroup creation and returns it again.
func NewDispatcher(ctx context.Context, numWorkers int) (*Dispatcher, context.Context) {
	eg, ctx := errgroup.WithContext(ctx)

	return &Dispatcher{
		queue:      make(chan Job, numWorkers),
		eg:         eg,
		numWorkers: numWorkers,
	}, ctx
}

// Start starts the configured number of workers and waits for jobs.
func (d *Dispatcher) Start() {
	for i := 0; i < d.numWorkers; i++ {
		logger := log.With().Caller().Int("worker", i).Logger()
		logger.Info().Msg("starting worker")

		d.eg.Go(func() error {
			for j := range d.queue {
				errChan := make(chan error)

				go func() {
					errChan <- j.work(j.ctx)
				}()

				select {
				case <-j.ctx.Done():
					return fmt.Errorf("got error from context: %w", j.ctx.Err())
				case err := <-errChan:
					if err != nil {
						return fmt.Errorf("go error from work function: %w", err)
					}
				}
			}

			logger.Info().Msg("no work. returning...")

			return nil
		})
	}
}

// Append adds a job to the work queue.
func (d *Dispatcher) Append(job Job) {
	log.Debug().Msg("adds job")
	d.queue <- job
}

// Close closes the queue channel.
func (d *Dispatcher) Close() {
	log.Debug().Msg("closing queue")
	close(d.queue)
}

func (d *Dispatcher) Wait() error {
	log.Debug().Msg("waiting for jobs to finnish")

	if err := d.eg.Wait(); err != nil {
		return fmt.Errorf("error on waiting: %w", err)
	}

	return nil
}
