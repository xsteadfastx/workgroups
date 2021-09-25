// Package workgroups is a little helper for creating workers
// with the help of sync.Errgroup.
package workgroups

import (
	"context"
	"errors"
	"fmt"

	"github.com/rs/zerolog/log"
	"golang.org/x/sync/errgroup"
)

var ErrInWorker = errors.New("received error in worker")

type Job func(ctx context.Context) error

type Dispatcher struct {
	queue      chan Job
	eg         *errgroup.Group
	numWorkers int
}

func NewDispatcher(ctx context.Context, numWorkers int) (*Dispatcher, context.Context) {
	eg, ctx := errgroup.WithContext(ctx)

	return &Dispatcher{
		queue:      make(chan Job, numWorkers),
		eg:         eg,
		numWorkers: numWorkers,
	}, ctx
}

func (d *Dispatcher) Start(ctx context.Context) {
	for i := 0; i < d.numWorkers; i++ {
		logger := log.With().Caller().Int("worker", i).Logger()
		logger.Info().Msg("starting worker")

		d.eg.Go(func() error {
			for j := range d.queue {
				errChan := make(chan error)

				go func() {
					errChan <- j(ctx)
				}()

				select {
				case <-ctx.Done():
					return fmt.Errorf("got error from context: %w", ctx.Err())
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

// Append adds a job to the queue.
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
