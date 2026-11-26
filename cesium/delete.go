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
	"cmp"
	"context"
	"math/rand/v2"
	"slices"
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
	chs = slices.Concat(data, indexes)
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
//
// Writers that open on tr in a channel it deletes, or in a dependent of an index it
// deletes, wait until it returns. So does CreateChannel of a new dependent.
func (db *DB) DeleteTimeRange(
	ctx context.Context,
	chs []ChannelKey,
	tr telem.TimeRange,
) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	targets, end, err := db.startTimeRangeDelete(chs)
	if err != nil {
		return err
	}
	defer end()
	var (
		dataDels  = make([]func(context.Context) error, 0, len(targets))
		indexDels = make([]func(context.Context) error, 0, len(targets))
		releases  = make([]func(), 0, len(targets))
	)
	defer func() {
		for _, release := range releases {
			release()
		}
	}()
	// Take every channel before the first delete, so that a channel a writer controls
	// refuses the call before it deletes anything. The key order keeps two deletes from
	// waiting on each other.
	for _, t := range targets {
		if t.dependent {
			hasData, release, err := t.db.OpenHold(ctx, tr)
			if err != nil {
				return err
			}
			releases = append(releases, release)
			if hasData {
				return errors.Newf(
					"cannot delete index channel %v "+
						"with channel %v depending on it on the time range %s",
					t.index,
					t.db.Channel(),
					tr,
				)
			}
			continue
		}
		del, release, err := t.db.OpenDelete(tr)
		if err != nil {
			return err
		}
		releases = append(releases, release)
		if t.db.Channel().IsIndex {
			indexDels = append(indexDels, del)
		} else {
			dataDels = append(dataDels, del)
		}
	}
	// Data deletes resolve their offsets through the index, so they run first.
	for _, del := range slices.Concat(dataDels, indexDels) {
		if err := del(ctx); err != nil {
			return err
		}
	}
	return nil
}

// deleteTarget is a channel that DeleteTimeRange takes before it deletes.
type deleteTarget struct {
	db unary.DB
	// index is the channel that db depends on, set when dependent is true.
	index channel.Channel
	// dependent is true when DeleteTimeRange only holds db to keep data out of the
	// time range of an index it deletes.
	dependent bool
}

// indexDelete records the index channels that a DeleteTimeRange deletes from. done
// closes when the delete returns.
type indexDelete struct {
	indexes set.Set[ChannelKey]
	done    chan struct{}
}

// startTimeRangeDelete returns the unary channels in chs and the dependents of the
// index channels in chs, sorted by key. It records the index channels until end runs,
// so that CreateChannel waits to add a dependent of one of them.
func (db *DB) startTimeRangeDelete(
	chs []ChannelKey,
) (targets []deleteTarget, end func(), err error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	var (
		deleted = make(set.Set[ChannelKey], len(chs))
		indexes = make(set.Set[ChannelKey])
	)
	for _, ch := range lo.Uniq(chs) {
		if uDB, ok := db.mu.dbs.unary[ch]; ok {
			deleted.Add(ch)
			targets = append(targets, deleteTarget{db: uDB})
			if uDB.Channel().IsIndex {
				indexes.Add(ch)
			}
			continue
		}
		if _, ok := db.mu.dbs.virtual[ch]; ok {
			continue
		}
		return nil, nil, channel.NewNotFoundError(ch)
	}
	for key, uDB := range db.mu.dbs.unary {
		idx := uDB.Channel().Index
		if key == idx || !indexes.Contains(idx) || deleted.Contains(key) {
			continue
		}
		idxDB := db.mu.dbs.unary[idx]
		targets = append(targets, deleteTarget{
			db:        uDB,
			index:     idxDB.Channel(),
			dependent: true,
		})
	}
	slices.SortFunc(targets, func(a, b deleteTarget) int {
		return cmp.Compare(a.db.Channel().Key, b.db.Channel().Key)
	})
	if len(indexes) == 0 {
		return targets, func() {}, nil
	}
	del := &indexDelete{indexes: indexes, done: make(chan struct{})}
	db.mu.indexDeletes = append(db.mu.indexDeletes, del)
	return targets, func() {
		db.mu.Lock()
		defer db.mu.Unlock()
		db.mu.indexDeletes = slices.DeleteFunc(
			db.mu.indexDeletes,
			func(d *indexDelete) bool { return d == del },
		)
		close(del.done)
	}, nil
}

// indexDeleteDone returns a channel that closes when a running DeleteTimeRange on the
// index of a channel in chs returns, or nil if none runs. The caller must hold db.mu.
func (db *DB) indexDeleteDone(chs []Channel) <-chan struct{} {
	for _, del := range db.mu.indexDeletes {
		for _, ch := range chs {
			if !ch.IsIndex && del.indexes.Contains(ch.Index) {
				return del.done
			}
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
				select {
				case <-ctx.Done():
					return nil
				default:
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
