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
	"math/rand/v2"
	"strconv"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
	"go.uber.org/zap"
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
	//
	// [OPTIONAL] Default: 0.2
	Threshold float32
}

var _ config.Config[GCConfig] = GCConfig{}

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
	v.Positive("gc_try_interval", cfg.TryInterval)
	v.Positive("gc_threshold", cfg.Threshold)
	v.Positive("max_goroutine", cfg.MaxGoroutine)
	return v.Error()
}

func keyToDirName(ch ChannelKey) string { return strconv.FormatUint(uint64(ch), 10) }

func dirNameToKey(name string) (ChannelKey, error) {
	key, err := strconv.ParseUint(name, 10, 32)
	if err != nil {
		return 0, err
	}
	return ChannelKey(key), nil
}

// DeleteChannel deletes a channel by its key. It returns once the channel is detached
// and removes its data in the background. Close waits for the removal to finish.
//
// This method returns an error if there are other channels depending on the current
// channel, or if the current channel is being written to or read from.
//
// DeleteChannel is idempotent.
func (db *DB) DeleteChannel(ch ChannelKey) error {
	return db.DeleteChannels([]ChannelKey{ch})
}

// DeleteChannels deletes the channels with the given keys and skips keys with no
// channel. It returns an error and deletes none of them if one is being written to or
// read from, or is the index of a channel outside chs. A file system error stops it and
// can leave some of them deleted. It deletes index channels last. It returns once the
// channels are detached and removes their data in the background. Close waits for the
// removal to finish.
func (db *DB) DeleteChannels(chs []ChannelKey) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	// Close waits for removals under db.mu, so the check must hold the lock.
	if db.closed.Load() {
		return ErrDBClosed
	}
	// Index channels detach last, so that a failed detach never keeps a data channel
	// whose index it deleted.
	data, indexes := lo.FilterReject(lo.Uniq(chs), func(ch ChannelKey, _ int) bool {
		uDB, ok := db.mu.dbs.unary[ch]
		return !ok || !uDB.Channel().IsIndex
	})
	chs = append(data, indexes...)
	deleting := set.New(chs...)
	for i, ch := range chs {
		if err := db.markClosed(ch, deleting); err != nil {
			for _, marked := range chs[:i] {
				db.unmarkClosed(marked)
			}
			return err
		}
	}
	dirs := make([]string, 0, len(chs))
	defer func() { db.removeDirs(dirs...) }()
	for i, ch := range chs {
		dir, err := db.detach(ch)
		if err != nil {
			for _, marked := range chs[i+1:] {
				db.unmarkClosed(marked)
			}
			return err
		}
		if dir != "" {
			dirs = append(dirs, dir)
		}
	}
	return nil
}

// markClosed marks the DB of ch closed for a delete of the channels in deleting. It
// returns an error if ch is being written to or read from, or is the index of a channel
// outside deleting. The caller must hold db.mu.
func (db *DB) markClosed(ch ChannelKey, deleting set.Set[ChannelKey]) error {
	if vDB, ok := db.mu.dbs.virtual[ch]; ok {
		return vDB.MarkClosed()
	}
	uDB, ok := db.mu.dbs.unary[ch]
	if !ok {
		return nil
	}
	if uDB.Channel().IsIndex {
		for key, other := range db.mu.dbs.unary {
			if other.Channel().Index == ch && !deleting.Contains(key) {
				return errors.Newf(
					"cannot delete channel %v because it indexes data in channel %v",
					uDB.Channel(),
					other.Channel(),
				)
			}
		}
	}
	return uDB.MarkClosed()
}

// unmarkClosed opens the DB of ch that markClosed marked closed. The caller must hold
// db.mu.
func (db *DB) unmarkClosed(ch ChannelKey) {
	if vDB, ok := db.mu.dbs.virtual[ch]; ok {
		vDB.UnmarkClosed()
	} else if uDB, ok := db.mu.dbs.unary[ch]; ok {
		uDB.UnmarkClosed()
	}
}

// detach releases the DB of ch that markClosed marked closed, removes it from db, and
// renames its directory for removal. It returns the new name, or "" if ch does not
// exist. The caller must hold db.mu.
func (db *DB) detach(ch ChannelKey) (string, error) {
	if uDB, ok := db.mu.dbs.unary[ch]; ok {
		// The data of the channel is deleted anyway, so a close error loses nothing.
		if err := uDB.Release(); err != nil {
			db.L.Error(
				"failed to release a deleted channel",
				zap.Stringer("channel", uDB.Channel()),
				zap.Error(err),
			)
		}
		delete(db.mu.dbs.unary, ch)
	} else if _, ok = db.mu.dbs.virtual[ch]; ok {
		delete(db.mu.dbs.virtual, ch)
	} else {
		return "", nil
	}
	// The random suffix lets a channel be created and deleted again while an earlier
	// removal runs.
	name := keyToDirName(ch)
	detached := name + deletedDirInfix + strconv.Itoa(rand.Int())
	if err := db.fs.Rename(name, detached); err != nil {
		return "", err
	}
	return detached, nil
}

// deletedDirInfix marks a channel directory that a delete detached and that waits for
// removal.
const deletedDirInfix = "-DELETE-"

// removeDirs removes the named detached channel directories in the background. A
// failed removal logs the error, and the next Open retries it.
func (db *DB) removeDirs(names ...string) {
	if len(names) == 0 {
		return
	}
	db.removals.Go(func() {
		for _, name := range names {
			if err := db.fs.Remove(name); err != nil {
				db.L.Error(
					"failed to remove deleted channel data",
					zap.String("dir", name),
					zap.Error(err),
				)
			}
		}
	})
}

// DeleteTimeRange deletes a time range of data in the database in the given channels
// This method return an error if the channel to be deleted is an index channel and
// there are other channels depending on it in the time range. DeleteTimeRange is
// idempotent, but when the channel does not exist, it returns ErrChannelNotFound. It
// deletes nothing and returns ctx.Err() if ctx is already canceled.
func (db *DB) DeleteTimeRange(
	ctx context.Context,
	chs []ChannelKey,
	tr telem.TimeRange,
) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	var (
		indexChannels = make([]ChannelKey, 0, len(chs))
		dataChannels  = make([]ChannelKey, 0, len(chs))
	)

	for _, ch := range chs {
		if uDB, ok := db.mu.dbs.unary[ch]; ok {
			if uDB.Channel().IsIndex {
				indexChannels = append(indexChannels, ch)
			} else {
				dataChannels = append(dataChannels, ch)
			}
			continue
		}
		if _, ok := db.mu.dbs.virtual[ch]; ok {
			continue
		}
		return channel.NewNotFoundError(ch)
	}

	for _, ch := range dataChannels {
		if udb, ok := db.mu.dbs.unary[ch]; ok {
			if err := udb.Delete(ctx, tr); err != nil {
				return err
			}
		}
	}

	for _, ch := range indexChannels {
		idxDB := db.mu.dbs.unary[ch]
		for otherDBKey, otherDB := range db.mu.dbs.unary {
			if otherDBKey == ch || otherDB.Channel().Index != ch {
				continue
			}
			hasOverlap, err := otherDB.HasDataFor(ctx, tr)
			if err != nil || hasOverlap {
				return errors.Newf(
					"cannot delete index channel %v "+
						"with channel %v depending on it on the time range %s",
					idxDB.Channel(),
					otherDB.Channel(),
					tr,
				)
			}
		}
		if err := idxDB.Delete(ctx, tr); err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) garbageCollect(ctx context.Context, maxGoRoutine uint) error {
	_, span := db.T.Debug(ctx, "garbage_collect")
	defer span.End()
	sCtx, cancel := signal.WithCancel(ctx)
	defer cancel()
	db.mu.RLock()
	pending := make(chan unary.DB, len(db.mu.dbs.unary))
	for _, uDB := range db.mu.dbs.unary {
		pending <- uDB
	}
	db.mu.RUnlock()
	close(pending)
	for i := range min(int(maxGoRoutine), len(pending)) {
		sCtx.Go(func(ctx context.Context) error {
			for uDB := range pending {
				// A collection rewrites files and does not stop for ctx.
				if ctx.Err() != nil {
					return nil
				}
				if err := uDB.GarbageCollect(ctx); err != nil {
					return err
				}
			}
			return nil
		}, signal.RecoverWithErrOnPanic(), signal.WithKeyf("garbage_collect_%d", i))
	}
	return sCtx.Wait()
}

func (db *DB) startGC(sCtx signal.Context, opts *options) {
	signal.GoTick(
		sCtx,
		opts.gcCfg.TryInterval,
		func(ctx context.Context, time time.Time) error {
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
