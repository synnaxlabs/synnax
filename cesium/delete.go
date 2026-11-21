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
	"path"
	"strconv"
	"sync"
	"time"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/meta"
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

// Drop removes the channels with the given keys and skips keys with no channel. A new
// writer or iterator on a dropped key fails at once. Drop removes the files of a unary
// channel when no writer or iterator is open on it: in Drop, or in the Close of the
// last one. A virtual channel holds no files for its writers, so its directory goes at
// once. The files of an index go after the files of its dropped data channels, and stay
// when the removal of one of those fails. Drop keeps an index that a channel outside
// keys still uses. A failed removal does not stop Drop, which returns the error of each
// removal that runs in Drop and logs a later one. The next Open removes a directory
// that a failed removal renamed. A directory that keeps its name loads again at Open.
func (db *DB) Drop(keys []ChannelKey) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed.Load() {
		return ErrDBClosed
	}
	return db.dropLocked(keys)
}

// dropLocked drops keys as Drop does. The caller must hold db.mu.
func (db *DB) dropLocked(keys []ChannelKey) error {
	keys = lo.Uniq(keys)
	// A key that Sweep keeps drops as any other.
	for _, k := range keys {
		if u, ok := db.mu.dbs.hidden[k]; ok {
			db.mu.dbs.unary[k] = u
			delete(db.mu.dbs.hidden, k)
		}
	}
	dropping := set.New(keys...)
	indexes := make(map[ChannelKey]*latch)
	for _, key := range keys {
		uDB, ok := db.mu.dbs.unary[key]
		if !ok || !uDB.Channel().IsIndex || db.indexesKept(key, dropping) {
			continue
		}
		// The index holds one count, given up after its data channels drop.
		indexes[key] = &latch{n: 1}
	}
	var (
		err error
		rm  = &removals{}
	)
	for _, key := range keys {
		if db.mu.virtualDirs.Contains(key) {
			db.mu.virtualDirs.Remove(key)
			err = errors.Join(err, rm.remove(db, key))
			continue
		}
		uDB, ok := db.mu.dbs.unary[key]
		if !ok || uDB.Channel().IsIndex {
			continue
		}
		idx, waits := indexes[uDB.Channel().Index]
		if waits {
			idx.add()
		}
		delete(db.mu.dbs.unary, key)
		err = errors.Join(err, uDB.Drop(func() error {
			rmErr := rm.remove(db, key)
			if !waits {
				return rmErr
			}
			if rmErr != nil {
				idx.keep()
			}
			return errors.Join(rmErr, idx.done())
		}))
	}
	// An iterator on a data channel reads its index, so the index drops only once its
	// dropped data channels have freed their files.
	for key, idx := range indexes {
		uDB := db.mu.dbs.unary[key]
		delete(db.mu.dbs.unary, key)
		idx.run = func(keep bool) error {
			return uDB.Drop(func() error {
				if keep {
					return nil
				}
				return rm.remove(db, key)
			})
		}
		err = errors.Join(err, idx.done())
	}
	return errors.Join(err, rm.close(db))
}

// indexesKept reports whether a channel outside dropping uses index as its index. The
// caller must hold db.mu.
func (db *DB) indexesKept(index ChannelKey, dropping set.Set[ChannelKey]) bool {
	for _, dbs := range []map[ChannelKey]unary.DB{db.mu.dbs.unary, db.mu.dbs.hidden} {
		for key, other := range dbs {
			if other.Channel().Index == index && !dropping.Contains(key) {
				return true
			}
		}
	}
	return false
}

// latch runs run in the call to done that brings n to zero. run gets true when keep
// was called.
type latch struct {
	run  func(keep bool) error
	mu   sync.Mutex
	n    int
	kept bool
}

func (l *latch) keep() {
	l.mu.Lock()
	l.kept = true
	l.mu.Unlock()
}

func (l *latch) add() {
	l.mu.Lock()
	l.n++
	l.mu.Unlock()
}

func (l *latch) done() error {
	l.mu.Lock()
	l.n--
	last, kept := l.n == 0, l.kept
	l.mu.Unlock()
	if !last {
		return nil
	}
	return l.run(kept)
}

// deletedDirSuffix marks the directory of a dropped channel that waits for removal.
// Keys are never reused, so only a leftover of the same key can hold the name.
const deletedDirSuffix = "-DELETE"

// removals collects the keys whose files a drop removes in line, so their intents clear
// in one append. A removal that runs after the batch closes clears its own intent.
type removals struct {
	keys   []ChannelKey
	mu     sync.Mutex
	closed bool
}

// remove removes the files of key and adds key to the batch, or clears its intent when
// the batch has closed.
func (r *removals) remove(db *DB, key ChannelKey) error {
	if err := db.removeDir(key); err != nil {
		return err
	}
	r.mu.Lock()
	if !r.closed {
		r.keys = append(r.keys, key)
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()
	return db.clearRemoved(key)
}

// close closes the batch and clears the intents of its keys.
func (r *removals) close(db *DB) error {
	r.mu.Lock()
	r.closed = true
	keys := r.keys
	r.mu.Unlock()
	return db.clearRemoved(keys...)
}

// clearRemoved syncs the root, so that the removals of keys are durable, then clears
// their intents in one append. A crash then never brings back files that no intent
// names.
func (db *DB) clearRemoved(keys ...ChannelKey) error {
	if len(keys) == 0 {
		return nil
	}
	if err := db.fs.SyncDir(""); err != nil {
		return err
	}
	return db.intents.clear(keys...)
}

// removeDir renames the directory of key to its deleted name, then removes its data
// files, its metadata file, and the directory, in that order. It first removes a
// leftover of an earlier removal of key.
func (db *DB) removeDir(key ChannelKey) error {
	name := keyToDirName(key)
	deleted := name + deletedDirSuffix
	if err := db.fs.Remove(deleted); err != nil {
		return err
	}
	exists, err := db.fs.Exists(name)
	if err != nil {
		return err
	}
	if exists {
		if err = db.fs.Rename(name, deleted); err != nil {
			return err
		}
		return db.removeFiles(deleted)
	}
	return nil
}

// removeFiles removes the files of the channel directory dir other than its metadata
// file, then the metadata file, then dir.
func (db *DB) removeFiles(dir string) error {
	infos, err := db.fs.List(dir)
	if err != nil {
		return err
	}
	for _, f := range infos {
		if f.Name() == meta.File {
			continue
		}
		if err = db.fs.Remove(path.Join(dir, f.Name())); err != nil {
			return err
		}
	}
	if err = db.fs.Remove(path.Join(dir, meta.File)); err != nil {
		return err
	}
	return db.fs.Remove(dir)
}

// Intend makes delete intents for keys durable, with one write and one sync of the
// intents file. Open skips the directory of a key with an intent, Sweep decides its
// fate, and Drop clears the intent after it removes the files of the key.
func (db *DB) Intend(keys []ChannelKey) error {
	if db.closed.Load() {
		return ErrDBClosed
	}
	if len(keys) == 0 {
		return nil
	}
	return db.intents.add(keys)
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
				return channel.Wrap(ch, errors.Newf(
					"cannot delete the index with channel %v depending on it on the "+
						"time range %s",
					otherDB.Channel(),
					tr,
				))
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
