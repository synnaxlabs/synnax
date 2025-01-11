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
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"io/fs"
	"math/rand"
	"strconv"
	"time"
)

type GCConfig struct {
	// MaxGoroutine is the maximum number of GoRoutines that can be launched for
	// each try of garbage collection.
	MaxGoroutine int64

	// GCTryInterval is the interval of time between two tries of garbage collection
	// are started.
	GCTryInterval time.Duration

	// GCThreshold is the minimum tombstone proportion of the Filesize to trigger a GC.
	// Must be in (0, 1].
	// Note: Setting this value to 0 will have NO EFFECT as it is the default value.
	// instead, set it to a very small number greater than 0.
	// [OPTIONAL] Default: 0.2
	GCThreshold float32
}

var DefaultGCConfig = GCConfig{
	MaxGoroutine:  10,
	GCTryInterval: 30 * time.Second,
	GCThreshold:   0.2,
}

func keyToDirName(ch ChannelKey) string {
	return strconv.Itoa(int(ch))
}

// DeleteChannel deletes a channel by its key.
// This method returns an error if there are other channels depending on the current
// channel, or if the current channel is being written to or read from.
// DeleteChannel is idempotent.
func (db *DB) DeleteChannel(ch ChannelKey) error {
	if db.closed.Load() {
		return errDBClosed
	}
	// Rename the file first, so we can avoid hogging the mutex while deleting the
	// directory, which may take a longer time.
	// Rename the file to have a random suffix in case the channel is repeatedly created
	// and deleted.
	oldName := keyToDirName(ch)
	newName := oldName + "-DELETE-" + strconv.Itoa(rand.Int())
	if err := (func() error {
		db.mu.Lock()
		defer db.mu.Unlock()
		if err := db.removeChannel(ch); err != nil {
			return err
		}
		err := db.fs.Rename(oldName, newName)
		if errors.Is(err, fs.ErrNotExist) {
			return nil
		}
		return err
	})(); err != nil {
		return err
	}
	return db.fs.Remove(newName)
}

// DeleteChannels deletes many channels by their keys.
// This operation is not guaranteed to be atomic: it is possible some channels in chs
// are deleted and some are not.
func (db *DB) DeleteChannels(chs []ChannelKey) (err error) {
	if db.closed.Load() {
		return errDBClosed
	}
	var (
		indexChannels       = make([]ChannelKey, 0, len(chs))
		directoriesToRemove = make([]string, 0, len(chs))
	)
	db.mu.Lock()
	// This 'defer' statement does a best-effort removal of all renamed directories
	// to ensure that all DBs deleted from db.unaryDBs and db.virtualDBs are also deleted
	// on FS.
	defer func() {
		db.mu.Unlock()
		c := errors.NewCatcher(errors.WithAggregation())
		for _, name := range directoriesToRemove {
			c.Exec(func() error { return db.fs.Remove(name) })
		}
		err = errors.Combine(err, c.Error())
	}()

	// Do a pass first to remove all non-index channels
	for _, ch := range chs {
		udb, uok := db.unaryDBs[ch]

		if !uok || udb.Channel().IsIndex {
			if udb.Channel().IsIndex {
				indexChannels = append(indexChannels, ch)
			}
			continue
		}

		err = db.removeChannel(ch)
		if err != nil {
			return
		}

		// Rename the files first, so we can avoid hogging the mutex while deleting
		// the directory, which may take a longer time.
		oldName := keyToDirName(ch)
		newName := oldName + "-DELETE-" + strconv.Itoa(rand.Int())
		err = db.fs.Rename(oldName, newName)
		if err != nil {
			return
		}

		directoriesToRemove = append(directoriesToRemove, newName)
	}

	// Do another pass to remove all index channels
	for _, ch := range indexChannels {
		err = db.removeChannel(ch)
		if err != nil {
			return
		}

		oldName := keyToDirName(ch)
		newName := oldName + "-DELETE-" + strconv.Itoa(rand.Int())
		err = db.fs.Rename(oldName, newName)
		if err != nil {
			return
		}

		directoriesToRemove = append(directoriesToRemove, newName)
	}

	return
}

// removeChannel removes ch from db.unaryDBs or db.virtualDBs. If the key does not exist
// or if there is an open entity on the specified database.
func (db *DB) removeChannel(ch ChannelKey) error {
	udb, uok := db.unaryDBs[ch]
	if uok {
		if udb.Channel().IsIndex {
			for otherDBKey := range db.unaryDBs {
				if otherDBKey == ch {
					continue
				}
				otherDB := db.unaryDBs[otherDBKey]
				if otherDB.Channel().Index == udb.Channel().Key {
					return errors.Newf(
						"cannot delete channel %v "+
							"because it indexes data in channel %v",
						udb.Channel(),
						otherDB.Channel(),
					)
				}
			}
		}

		if err := udb.Close(); err != nil {
			return err
		}
		delete(db.unaryDBs, ch)
		return nil
	}
	vdb, vok := db.virtualDBs[ch]
	if vok {
		if err := vdb.Close(); err != nil {
			return err
		}
		delete(db.virtualDBs, ch)
		return nil
	}

	return nil
}

// DeleteTimeRange deletes a time range of data in the database in the given channels
// This method return an error if the channel to be deleted is an index channel and
// there are other channels depending on it in the time range.
// DeleteTimeRange is idempotent, but when the channel does not exist, it returns
// ErrChannelNotFound.
func (db *DB) DeleteTimeRange(
	ctx context.Context,
	chs []ChannelKey,
	tr telem.TimeRange,
) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	var (
		indexChannels = make([]ChannelKey, 0, len(chs))
		dataChannels  = make([]ChannelKey, 0, len(chs))
	)

	for _, ch := range chs {
		udb, uok := db.unaryDBs[ch]
		if !uok {
			if vdb, vok := db.virtualDBs[ch]; vok {
				return errors.Newf(
					"cannot delete time range from virtual channel %v",
					vdb.Channel(),
				)
			}
			return errors.Wrapf(ErrChannelNotFound, "channel key %d not found", ch)
		}

		// Cannot delete an index channel that other channels rely on.
		if udb.Channel().IsIndex {
			indexChannels = append(indexChannels, ch)
			continue
		}

		dataChannels = append(dataChannels, ch)
	}

	for _, ch := range dataChannels {
		udb := db.unaryDBs[ch]
		if err := udb.Delete(ctx, tr); err != nil {
			return err
		}
	}

	for _, ch := range indexChannels {
		udb := db.unaryDBs[ch]
		// Cannot delete an index channel that other channels rely on.
		for otherDBKey, otherDB := range db.unaryDBs {
			if otherDBKey == ch || otherDB.Channel().Index != ch {
				continue
			}
			hasOverlap, err := otherDB.HasDataFor(ctx, tr)
			if err != nil || hasOverlap {
				return errors.Newf(
					"cannot delete index channel %v "+
						"with channel %v depending on it on the time range %s",
					udb.Channel(),
					otherDB.Channel(),
					tr,
				)
			}
		}

		if err := udb.Delete(ctx, tr); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) garbageCollect(ctx context.Context, maxGoRoutine int64) error {
	_, span := db.T.Debug(ctx, "garbage_collect")
	defer span.End()
	db.mu.RLock()
	var (
		sem     = semaphore.NewWeighted(maxGoRoutine)
		sCtx, _ = signal.Isolated()
	)
	for _, udb := range db.unaryDBs {
		if err := sem.Acquire(ctx, 1); err != nil {
			db.mu.RUnlock()
			return err
		}
		udb := udb
		sCtx.Go(func(_ctx context.Context) error {
			defer sem.Release(1)
			return udb.GarbageCollect(_ctx)
		}, signal.RecoverWithErrOnPanic())
	}
	db.mu.RUnlock()
	return sCtx.Wait()
}

func (db *DB) startGC(sCtx signal.Context, opts *options) {
	signal.GoTick(sCtx, opts.gcCfg.GCTryInterval, func(ctx context.Context, time time.Time) error {
		err := db.garbageCollect(ctx, opts.gcCfg.MaxGoroutine)
		if err != nil {
			db.L.Error("garbage collection error", zap.Error(err))
		}
		return nil
	}, signal.WithRetryOnPanic(10), signal.RecoverWithoutErrOnPanic())
}
