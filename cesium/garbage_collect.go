package cesium

import (
	"context"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/telem"
	"golang.org/x/sync/semaphore"
	"sync"
	"time"
)

type GCConfig struct {
	maxSizeRead  uint32
	maxGoroutine int64
	gcInterval   time.Duration
}

var DefaultGCConfig = GCConfig{
	maxSizeRead:  uint32(100 * telem.ByteSize),
	maxGoroutine: 10,
	gcInterval:   5 * time.Minute,
}

func (db *DB) garbageCollect(ctx context.Context, maxsizeRead uint32, maxGoRoutine int64) (err error) {
	_, span := db.T.Debug(ctx, "Garbage Collect")
	defer span.End()
	db.mu.RLock()
	defer db.mu.RUnlock()
	var (
		sem = semaphore.NewWeighted(maxGoRoutine)
		wg  = &sync.WaitGroup{}
		c   = errutil.NewCatch(errutil.WithAggregation())
	)

	for _, udb := range db.unaryDBs {
		if err = sem.Acquire(ctx, 1); err != nil {
			return err
		}
		wg.Add(1)
		udb := udb
		go func() {
			defer func() {
				sem.Release(1)
				wg.Done()
			}()
			c.Exec(func() error {
				err := udb.GarbageCollect(ctx, maxsizeRead)
				return err
			})
		}()
	}
	wg.Wait()
	return c.Error()
}
