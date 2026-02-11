// Copyright 2026 Synnax Labs, Inc.
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
	"io/fs"
	"math/rand"
	"strconv"
	"time"

	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
	"golang.org/x/sync/semaphore"
)

type GCConfig struct {
	// MaxGoroutine is the maximum number of Goroutines that can be launched for each
	// try of garbage collection.
	MaxGoroutine uint
	// TryInterval is the interval of time between two tries of garbage collection are
	// started.
	TryInterval time.Duration
	// Threshold is the minimum tombstone proportion of the Filesize to trigger a GC.
	// Must be in (0, 1]. Note: Setting this value to 0 will have NO EFFECT as it is the
	// default value. instead, set it to a very small number greater than 0.
	// [OPTIONAL] Default: 0.2
	Threshold float32
}

var (
	_               config.Config[GCConfig] = GCConfig{}
	DefaultGCConfig                         = GCConfig{
		MaxGoroutine: 10,
		TryInterval:  30 * time.Second,
		Threshold:    0.2,
	}
)

// Override implements config.Config.
func (cfg GCConfig) Override(other GCConfig) GCConfig {
	cfg.TryInterval = override.Numeric(cfg.TryInterval, other.TryInterval)
	cfg.Threshold = override.Numeric(cfg.Threshold, other.Threshold)
	cfg.MaxGoroutine = override.Numeric(cfg.MaxGoroutine, other.MaxGoroutine)
	return cfg
}

// Validate implements config.Config.
func (cfg GCConfig) Validate() error {
	v := validate.New("cesium.gc_config")
	validate.Positive(v, "gc_try_interval", cfg.TryInterval)
	validate.Positive(v, "gc_threshold", cfg.Threshold)
	validate.Positive(v, "max_goroutine", cfg.MaxGoroutine)
	return v.Error()
}

func keyToDirName(ch ChannelKey) string {
	return strconv.Itoa(int(ch))
}

// DeleteChannel deletes a channel by its key.
//
// This method returns an error if there are other channels depending on the current
// channel, or if the current channel is being written to or read from.
//
// DeleteChannel is idempotent.
func (db *DB) DeleteChannel(ch ChannelKey) error {
	if db.closed.Load() {
		return errDBClosed
	}
	// Rename the file first, so we can avoid hogging the mutex while deleting the
	// directory, which may take a longer time. Rename the file to have a random suffix
	// in case the channel is repeatedly created and deleted.
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

// DeleteChannels deletes many channels by their keys. This operation is not guaranteed
// to be atomic: it is possible some channels in chs are deleted and some are not.
func (db *DB) DeleteChannels(chs []ChannelKey) (err error) {
	if db.closed.Load() {
		return errDBClosed
	}
	var (
		indexChannels       = make([]ChannelKey, 0, len(chs))
		directoriesToRemove = make([]string, 0, len(chs))
	)
	db.mu.Lock()
	// This 'defer' statement does a best-effort removal of all renamed directories to
	// ensure that all DBs deleted from db.mu.unaryDBs and db.mu.virtualDBs are also
	// deleted on FS.
	defer func() {
		db.mu.Unlock()
		var errRemove error
		for _, name := range directoriesToRemove {
			errRemove = errors.Join(errRemove, db.fs.Remove(name))
		}
		err = errors.Combine(err, errRemove)
	}()

	// Do a pass first to remove all non-index channels
	for _, ch := range chs {
		udb, uok := db.mu.unaryDBs[ch]

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

		// Rename the files first, so we can avoid hogging the mutex while deleting the
		// directory, which may take a longer time.
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

// removeChannel removes ch from db.mu.unaryDBs or db.mu.virtualDBs. If the channel or
// if there is an open resource on the specified database.
func (db *DB) removeChannel(ch ChannelKey) error {
	uDB, uOk := db.mu.unaryDBs[ch]
	if uOk {
		if uDB.Channel().IsIndex {
			for otherDBKey, otherDB := range db.mu.unaryDBs {
				if otherDBKey != ch && otherDB.Channel().Index == uDB.Channel().Key {
					return errors.Newf(
						"cannot delete channel %v "+
							"because it indexes data in channel %v",
						uDB.Channel(),
						otherDB.Channel(),
					)
				}
			}
		}

		if err := uDB.Close(); err != nil {
			return err
		}
		delete(db.mu.unaryDBs, ch)
		return nil
	}
	vDB, vOk := db.mu.virtualDBs[ch]
	if vOk {
		if err := vDB.Close(); err != nil {
			return err
		}
		delete(db.mu.virtualDBs, ch)
		return nil
	}

	return nil
}

// DeleteTimeRange deletes a time range of data in the database in the given channels
// This method return an error if the channel to be deleted is an index channel and
// there are other channels depending on it in the time range. DeleteTimeRange is
// idempotent, but when the channel does not exist, it returns ErrChannelNotFound.
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
		uDB, uOk := db.mu.unaryDBs[ch]
		if !uOk {
			// If the channel is virtual, delete is a no-op but we don't return an
			// error.
			if _, vOk := db.mu.virtualDBs[ch]; vOk {
				continue
			}
			return errors.Wrapf(ErrChannelNotFound, "channel key %d not found", ch)
		}

		// Cannot delete an index channel that other channels rely on.
		if uDB.Channel().IsIndex {
			indexChannels = append(indexChannels, ch)
			continue
		}

		dataChannels = append(dataChannels, ch)
	}

	for _, ch := range dataChannels {
		udb := db.mu.unaryDBs[ch]
		if err := udb.Delete(ctx, tr); err != nil {
			return err
		}
	}

	for _, ch := range indexChannels {
		udb := db.mu.unaryDBs[ch]
		// Cannot delete an index channel that other channels rely on.
		for otherDBKey, otherDB := range db.mu.unaryDBs {
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

func (db *DB) garbageCollect(ctx context.Context, maxGoRoutine uint) error {
	_, span := db.T.Debug(ctx, "garbage_collect")
	defer span.End()
	db.mu.RLock()
	var (
		sem          = semaphore.NewWeighted(int64(maxGoRoutine))
		sCtx, cancel = signal.WithCancel(ctx)
	)
	defer cancel()
	for _, uDB := range db.mu.unaryDBs {
		if err := sem.Acquire(ctx, 1); err != nil {
			db.mu.RUnlock()
			return err
		}
		uDB := uDB
		sCtx.Go(func(_ctx context.Context) error {
			defer sem.Release(1)
			return uDB.GarbageCollect(_ctx)
		}, signal.RecoverWithErrOnPanic(), signal.WithKeyf("garbage_collect_%v", uDB.Channel()))
	}
	db.mu.RUnlock()
	return sCtx.Wait()
}

func (db *DB) startGC(sCtx signal.Context, opts *options) {
	signal.GoTick(sCtx, opts.gcCfg.TryInterval, func(ctx context.Context, time time.Time) error {
		err := db.garbageCollect(ctx, opts.gcCfg.MaxGoroutine)
		if err != nil {
			db.L.Error("garbage collection error", zap.Error(err))
		}
		return nil
	},
		signal.WithRetryOnPanic(10),
		signal.RecoverWithoutErrOnPanic(),
		signal.WithKey("gc-ticker"),
	)
}
