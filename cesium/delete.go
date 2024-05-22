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
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
	"math/rand"
	"strconv"
	"sync"
	"time"
)

type GCConfig struct {
	// ReadChunkSize is the maximum number of bytes to be read into memory while garbage collecting
	ReadChunkSize uint32

	// MaxGoroutine is the maximum number of GoRoutines that can be launched for each try of garbage collection
	MaxGoroutine int64

	// GcTryInterval is the interval of time between two tries of garbage collection are started
	GcTryInterval time.Duration
}

var DefaultGCConfig = GCConfig{
	ReadChunkSize: uint32(20 * telem.Megabyte),
	MaxGoroutine:  10,
	GcTryInterval: 30 * time.Second,
}

func channelDirName(ch ChannelKey) string {
	return strconv.Itoa(int(ch))
}

// DeleteChannel deletes a channel by its key.
// This method returns an error if there are other channels depending on the current
// channel, or if the current channel is being written to or read from.
// Does nothing if channel does not exist.
func (db *DB) DeleteChannel(ch ChannelKey) error {
	db.mu.Lock()
	err := db.removeChannel(ch)
	if err != nil {
		db.mu.Unlock()
		return err
	}

	// Rename the file first, so we can avoid hogging the mutex while deleting the directory
	// may take a longer time.
	// Rename the file to have a random suffix in case the channel is repeatedly created
	// and deleted.
	oldName := channelDirName(ch)
	newName := oldName + "-DELETE-" + strconv.Itoa(rand.Int())
	err = db.fs.Rename(oldName, newName)
	if err != nil {
		db.mu.Unlock()
		return nil
	}

	db.mu.Unlock()
	return db.fs.Remove(newName)
}

func (db *DB) DeleteChannels(chs []ChannelKey) (err error) {
	db.mu.Lock()
	var (
		indexChannels       = make([]ChannelKey, 0)
		directoriesToRemove = make([]string, 0)
	)

	// This 'defer' statement does a best-effort removal of all renamed directories
	// to ensure that all DBs deleted from db.unaryDBs and db.virtualDBs are also deleted
	// on FS.
	defer func() {
		db.mu.Unlock()
		c := errors.NewCatcher(errors.WithAggregation())
		for _, name := range directoriesToRemove {
			c.Exec(func() error { return db.fs.Remove(name) })
		}
		err = errors.CombineErrors(err, c.Error())
	}()

	// Do a pass first to remove all non-index channels
	for _, ch := range chs {
		udb, uok := db.unaryDBs[ch]

		if !uok || udb.Channel.IsIndex {
			if udb.Channel.IsIndex {
				indexChannels = append(indexChannels, ch)
			}
			continue
		}

		err = db.removeChannel(ch)
		if err != nil {
			return
		}

		// Rename the files first, so we can avoid hogging the mutex while deleting the directory
		// may take a longer time.
		oldName := channelDirName(ch)
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

		oldName := channelDirName(ch)
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
		if udb.Channel.IsIndex {
			for otherDBKey := range db.unaryDBs {
				if otherDBKey == ch {
					continue
				}
				otherDB := db.unaryDBs[otherDBKey]
				if otherDB.Channel.Index == udb.Config.Channel.Key {
					return errors.Newf("cannot delete channel %d because it indexes data in channel %d", ch, otherDBKey)
				}
			}
		}

		if err := udb.TryClose(); err != nil {
			return err
		}
		delete(db.unaryDBs, ch)
		return nil
	}
	vdb, vok := db.virtualDBs[ch]
	if vok {
		if err := vdb.TryClose(); err != nil {
			return err
		}
		delete(db.virtualDBs, ch)
		return nil
	}

	return nil
}

// DeleteTimeRange deletes a timerange of data in the database in a given channel
// This method return an error if there are other channels depending on the timerange
// that we are trying to delete
func (db *DB) DeleteTimeRange(ctx context.Context, ch ChannelKey, tr telem.TimeRange) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	udb, uok := db.unaryDBs[ch]
	if !uok {
		return core.ChannelNotFound(ch)
	}

	// Cannot delete an index channel that other channels rely on.
	if udb.Config.Channel.IsIndex {
		for otherDBKey := range db.unaryDBs {
			if otherDBKey == ch || db.unaryDBs[otherDBKey].Channel.Index != udb.Config.Channel.Key {
				continue
			}
			otherDB := db.unaryDBs[otherDBKey]
			// We must determine whether there is an indexed db that has data in the timerange tr.
			hasOverlap, err := otherDB.HasDataFor(ctx, tr)
			if err != nil || hasOverlap {
				return errors.Newf("cannot delete channel %d because it indexes data in channel %d", ch, otherDBKey)
			}
		}
	}

	return udb.Delete(ctx, tr)
}

func (db *DB) garbageCollect(ctx context.Context, readChunkSize uint32, maxGoRoutine int64) error {
	_, span := db.T.Debug(ctx, "garbage_collect")
	defer span.End()
	db.mu.RLock()
	var (
		sem = semaphore.NewWeighted(maxGoRoutine)
		wg  = &sync.WaitGroup{}
		c   = errors.NewCatcher(errors.WithAggregation())
	)

	for _, udb := range db.unaryDBs {
		if err := sem.Acquire(ctx, 1); err != nil {
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

	db.mu.RUnlock()
	wg.Wait()
	return c.Error()
}

func (db *DB) startGC(sCtx signal.Context, opts *options) {
	signal.GoTick(sCtx, opts.gcCfg.GcTryInterval, func(ctx context.Context, time time.Time) error {
		err := db.garbageCollect(ctx, opts.gcCfg.ReadChunkSize, opts.gcCfg.MaxGoroutine)
		if err != nil {
			db.L.Error("garbage collection failure", zap.Error(err))
		}
		return nil
	})
}
