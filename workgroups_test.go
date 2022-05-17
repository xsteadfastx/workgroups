//nolint:paralleltest
package workgroups_test

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/stdr"
	"github.com/stretchr/testify/require"
	"github.com/tonglil/buflogr"
	"go.xsfx.dev/workgroups"
)

func TestDispatcher(t *testing.T) {
	require := require.New(t)

	collector := struct {
		sync.RWMutex
		results []int
	}{results: []int{}}

	work := func(ctx context.Context) error {
		log.Print("printing this from inner work function")
		collector.Lock()
		collector.results = append(collector.results, 1)
		collector.Unlock()

		return nil
	}

	d, ctx := workgroups.NewDispatcher(
		context.Background(),
		stdr.New(log.New(os.Stderr, "", log.Lshortfile)),
		runtime.GOMAXPROCS(0),
		20,
	)
	d.Start()

	for i := 0; i < 10; i++ {
		d.Append(workgroups.NewJob(ctx, work))
	}

	d.Close()
	err := d.Wait()

	require.NoError(err)

	require.Equal(10, len(collector.results))
}

func TestDispatcherError(t *testing.T) {
	require := require.New(t)
	errWork := func(ctx context.Context) error {
		return fmt.Errorf("this is an error") //nolint:goerr113
	}

	okWork := func(ctx context.Context) error {
		return nil
	}

	d, ctx := workgroups.NewDispatcher(
		context.Background(),
		stdr.New(log.New(os.Stderr, "", log.Lshortfile)),
		runtime.GOMAXPROCS(0),
		2,
	)
	d.Start()
	d.Append(workgroups.NewJob(ctx, errWork))
	d.Append(workgroups.NewJob(ctx, okWork))
	d.Close()
	err := d.Wait()

	require.EqualError(err, "error on waiting: go error from work function: this is an error")
}

func TestDispatcherErrorOneWorker(t *testing.T) {
	require := require.New(t)
	errWork := func(ctx context.Context) error {
		return fmt.Errorf("this is an error") //nolint:goerr113
	}

	okWork := func(ctx context.Context) error {
		return nil
	}

	d, ctx := workgroups.NewDispatcher(context.Background(), stdr.New(log.New(os.Stderr, "", log.Lshortfile)), 1, 1)
	d.Start()
	d.Append(workgroups.NewJob(ctx, errWork))
	d.Append(workgroups.NewJob(ctx, okWork))
	d.Close()
	err := d.Wait()

	require.EqualError(err, "error on waiting: go error from work function: this is an error")
}

func TestDispatcherTimeout(t *testing.T) {
	require := require.New(t)
	work := func(ctx context.Context) error {
		log.Print("sleeping...")
		time.Sleep(5 * time.Second)

		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	d, ctx := workgroups.NewDispatcher(ctx, stdr.New(log.New(os.Stderr, "", log.Lshortfile)), runtime.GOMAXPROCS(0), 1)
	d.Start()
	d.Append(workgroups.NewJob(ctx, work))
	d.Close()
	err := d.Wait()
	require.EqualError(err, "error on waiting: got error from context: context deadline exceeded")
}

var errTest = errors.New("just a test")

type counter struct {
	sync.Mutex
	count int
}

func (c *counter) Work(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return fmt.Errorf("got error from context: %w", ctx.Err())
	default:
	}

	c.Lock()
	c.count++
	c.Unlock()

	return errTest
}

func TestRetry(t *testing.T) {
	type args struct {
		timeout time.Duration
	}

	tests := []struct {
		name        string
		args        args
		greaterThan int
		err         error
	}{
		{
			"00",
			args{
				time.Second,
			},
			500,
			context.DeadlineExceeded,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.args.timeout)
			defer cancel()

			c := counter{}
			d, ctx := workgroups.NewDispatcher(
				ctx,
				stdr.New(log.New(os.Stderr, "", log.Lshortfile)),
				1,
				1,
			)

			d.Start()
			d.Append(
				workgroups.NewJob(
					ctx,
					workgroups.Retry(
						ctx,
						time.Millisecond,
					)(c.Work),
				),
			)
			d.Close()
			err := d.Wait()

			require.ErrorIs(t, err, tt.err)
			require.Greater(t, c.count, tt.greaterThan)
		})
	}
}

func TestErrChanNotUsed(t *testing.T) {
	var buf bytes.Buffer
	log := buflogr.NewWithBuffer(&buf)

	require := require.New(t)

	work := func(ctx context.Context) error {
		time.Sleep(5 * time.Second)

		return nil
	}

	ctx, cancel := context.WithCancel(context.Background())
	d, ctx := workgroups.NewDispatcher(ctx, log, runtime.GOMAXPROCS(0), 1)
	d.Start()
	d.Append(workgroups.NewJob(ctx, work))
	d.Close()

	go func() {
		time.Sleep(time.Second / 2)
		cancel()
	}()

	err := d.Wait()
	require.ErrorIs(err, context.Canceled)
	time.Sleep(10 * time.Second)

	// Breaking glass!
	s := log.GetSink()

	underlier, ok := s.(buflogr.Underlier)
	if !ok {
		t.FailNow()
	}

	bl := underlier.GetUnderlying()
	bl.Mutex().Lock()
	require.Contains(buf.String(), "received job return after canceled context")
	bl.Mutex().Unlock()
}
