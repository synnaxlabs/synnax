// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/x/errutil"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"golang.org/x/sync/semaphore"
	"strconv"
	"sync"
	"time"
)

type GCConfig struct {
	readChunkSize uint32
	maxGoroutine  int64
	gcInterval    time.Duration
}

var DefaultGCConfig = GCConfig{
	readChunkSize: uint32(20 * telem.Megabyte),
	maxGoroutine:  10,
	gcInterval:    30 * time.Second,
}

func (db *DB) DeleteChannel(ch ChannelKey) error {
	db.mu.Lock()
	udb, uok := db.unaryDBs[ch]
	if uok {
		if udb.Config.Channel.IsIndex {
			for otherDBKey := range db.unaryDBs {
				if otherDBKey == ch {
					continue
				}
				otherDB := db.unaryDBs[otherDBKey]
				if otherDB.Channel.Index == udb.Config.Channel.Key {
					return errors.New("[cesium] - could not delete index channel with other channels depending on it")
				}
			}
		}

		if err := udb.TryClose(); err != nil {
			return err
		}
		delete(db.unaryDBs, ch)
		db.mu.Unlock()
		return db.fs.Remove(strconv.Itoa(int(ch)))
	}
	vdb, vok := db.virtualDBs[ch]
	if vok {
		if db.digests.key == ch {
			return errors.New("[cesium] - cannot delete update digest channel")
		}
		if err := vdb.TryClose(); err != nil {
			return err
		}
		delete(db.virtualDBs, ch)
		db.mu.Unlock()
		return db.fs.Remove(strconv.Itoa(int(ch)))
	}

	db.mu.Unlock()
	return ChannelNotFound
}

func (db *DB) DeleteTimeRange(ctx context.Context, ch ChannelKey, tr telem.TimeRange) error {
	db.mu.RLock()
	defer db.mu.RUnlock()
	udb, uok := db.unaryDBs[ch]
	if !uok {
		return ChannelNotFound
	}

	// cannot delete an index channel that other channels rely on
	if udb.Config.Channel.IsIndex {
		for otherDBKey := range db.unaryDBs {
			if otherDBKey == ch || db.unaryDBs[otherDBKey].Channel.Index != udb.Config.Channel.Key {
				continue
			}
			otherDB := db.unaryDBs[otherDBKey]
			// we must determine whether there is another db that has data in the timerange tr
			i := otherDB.Domain.NewIterator(domain.IterRange(otherDB.Domain.GetBounds()))

			if i.SeekGE(ctx, tr.Start) && i.TimeRange().OverlapsWith(tr) {
				return errors.New("[cesium] - could not delete index channel with other channels depending on it")
			}
			if i.SeekLE(ctx, tr.End) && i.TimeRange().OverlapsWith(tr) {
				return errors.New("[cesium] - could not delete index channel with other channels depending on it")
			}

		}
	}

	return udb.Delete(ctx, tr)
}

func (db *DB) garbageCollect(ctx context.Context, readChunkSize uint32, maxGoRoutine int64) (err error) {
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
				err := udb.GarbageCollect(ctx, readChunkSize)
				return err
			})
		}()
	}
	wg.Wait()
	return c.Error()
}

func (db *DB) startGC(opts *options) {
	sCtx, cancel := signal.Isolated(signal.WithInstrumentation(db.Instrumentation))
	signal.GoTick(sCtx, opts.gcCfg.gcInterval, func(ctx context.Context, time time.Time) error {
		return db.garbageCollect(ctx, opts.gcCfg.readChunkSize, opts.gcCfg.maxGoroutine)
	})

	db.shutdown = signal.NewShutdown(sCtx, cancel)
}
