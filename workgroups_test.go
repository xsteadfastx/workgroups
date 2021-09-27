//nolint:paralleltest
package workgroups_test

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
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

	d, ctx := workgroups.NewDispatcher(context.Background(), runtime.GOMAXPROCS(0))
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
	work := func(ctx context.Context) error {
		return fmt.Errorf("this is an error") //nolint:goerr113
	}

	d, ctx := workgroups.NewDispatcher(context.Background(), runtime.GOMAXPROCS(0))
	d.Start()
	d.Append(workgroups.NewJob(ctx, work))
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

	d, ctx := workgroups.NewDispatcher(ctx, runtime.GOMAXPROCS(0))
	d.Start()
	d.Append(workgroups.NewJob(ctx, work))
	d.Close()
	err := d.Wait()
	require.EqualError(err, "error on waiting: got error from context: context deadline exceeded")
}
