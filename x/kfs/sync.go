package kfs

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/signal"
	"time"
)

// Sync is a synchronization utility that periodically flushes the contents of "idle" files to disk.
// It synchronizes files on two conditions:
//
//  1. When the file is "idle" i.e. the file is not locked.
//  2. The files "age" i.e. the time since the file was last synced exceeds Sync.MaxAge.
//
// Sequential struct fields must be initialized before the Sync is started using Sync.Start().
type Sync[T comparable] struct {
	// FS is the file system to sync.
	FS FS[T]
	// Interval is the time between syncs.
	Interval time.Duration
	// MaxAge sets the maximum age of a file before it is synced.
	MaxAge time.Duration
	// Conductor is used to fork and close goroutines.
}

// Start starts a goroutine that periodically calls Sync.
// Shuts down based on the Sync.Shutter.
// When sync.Shutter.Shutdown is called, the Sync executes a forced sync ON all files and then exits.
func (s *Sync[T]) Start(ctx signal.Context) <-chan error {
	errs := make(chan error)
	t := time.NewTicker(s.Interval)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return errors.CombineErrors(s.forceSync(), ctx.Err())
			case <-t.C:
				if err := s.sync(); err != nil {
					errs <- err
				}
			}
		}
	})
	return errs
}

func (s *Sync[T]) sync() error {
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, v := range s.FS.OpenFiles() {
		if v.Age() > s.MaxAge && v.TryLock() {
			c.Exec(v.Sync)
			v.Unlock()
		}
	}
	return c.Error()
}

func (s *Sync[T]) forceSync() error {
	c := errutil.NewCatch(errutil.WithAggregation())
	for _, v := range s.FS.OpenFiles() {
		v.Lock()
		c.Exec(v.Sync)
		v.Unlock()
	}
	return c.Error()
}
