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
	"slices"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// KeptChannel is a channel that Sweep keeps: it holds samples but has no record and no
// delete intent. Writers and iterators cannot open on it until an intent and a Drop of
// its key remove it.
type KeptChannel struct {
	Key ChannelKey
	// Index is the key of the channel's index, or zero for an index.
	Index ChannelKey
	Size  telem.Size
}

// Sweep reconciles the DB with keys, the keys of every channel record, and must run
// before any writer opens. A key with an intent and a record lost the commit of its
// delete, so Sweep clears the intent and opens the channel. Sweep drops a key with an
// intent and no record, and a key with no record and no samples. It keeps a key with
// samples, no record, and no intent: writers and iterators on it fail, Sweep returns
// it, and only an intent and a Drop of its key remove it. An index that a kept or
// recorded channel uses stays, with any intent of the index. Each drop is a durable
// intent first. Sweep also returns the highest key that the DB holds in any form, so
// that no new channel takes it. Only the first Sweep in one open of the DB decides. A
// later call ignores keys, drops nothing, and returns the channels that the first kept
// and that are still kept, with the highest key. When the first fails, every later
// call returns its error.
func (db *DB) Sweep(
	ctx context.Context,
	keys []ChannelKey,
) ([]KeptChannel, ChannelKey, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed.Load() {
		return nil, 0, ErrDBClosed
	}
	if db.mu.swept {
		if db.mu.sweepErr != nil {
			return nil, 0, db.mu.sweepErr
		}
		return db.keptChannels(db.mu.kept), db.highestKey(), nil
	}
	db.mu.swept = true
	kept, err := db.decide(ctx, set.New(keys...))
	if err != nil {
		db.mu.sweepErr = err
		return nil, 0, err
	}
	db.mu.kept = kept
	return db.keptChannels(kept), db.highestKey(), nil
}

// decide drops, keeps, and hides the channels of the DB against records, and returns
// the keys it keeps. The caller must hold db.mu.
func (db *DB) decide(
	ctx context.Context,
	records set.Set[ChannelKey],
) (set.Set[ChannelKey], error) {
	if err := db.reopenFailedDeletes(ctx, records); err != nil {
		return nil, err
	}
	kept := set.New[ChannelKey]()
	var drop []ChannelKey
	for k := range db.mu.virtualDirs {
		if !records.Contains(k) {
			drop = append(drop, k)
		}
	}
	for k, u := range db.mu.dbs.unary {
		if records.Contains(k) {
			continue
		}
		if db.intents.has(k) || u.Size() == 0 {
			drop = append(drop, k)
		} else {
			kept.Add(k)
		}
	}
	hidden := db.usedIndexes(records, kept)
	drop = slices.DeleteFunc(drop, hidden.Contains)
	hidden.Add(kept.Slice()...)
	for k := range hidden {
		if u, ok := db.mu.dbs.unary[k]; ok {
			db.mu.dbs.hidden[k] = u
			delete(db.mu.dbs.unary, k)
		}
	}
	if err := db.intendDrops(drop); err != nil {
		return nil, err
	}
	err := errors.Join(db.dropLocked(drop), db.removeOrphanIntents(records, hidden))
	if err == nil {
		err = db.intents.truncateIfEmpty()
	}
	db.mu.skipped = set.New[ChannelKey]()
	return kept, err
}

// keptChannels returns the channels of kept that are still hidden, in rising order of
// key. The caller must hold db.mu.
func (db *DB) keptChannels(kept set.Set[ChannelKey]) []KeptChannel {
	out := make([]KeptChannel, 0, len(kept))
	for k := range kept {
		u, ok := db.mu.dbs.hidden[k]
		if !ok {
			continue
		}
		out = append(out, KeptChannel{Key: k, Index: u.Channel().Index, Size: u.Size()})
	}
	slices.SortFunc(out, func(a, b KeptChannel) int { return int(a.Key) - int(b.Key) })
	return out
}

// intendDrops makes an intent durable for each key of drop that has none, so every drop
// is an intent first. The caller must hold db.mu.
func (db *DB) intendDrops(drop []ChannelKey) error {
	unintended := slices.DeleteFunc(slices.Clone(drop), db.intents.has)
	if len(unintended) == 0 {
		return nil
	}
	return db.intents.add(unintended)
}

// highestKey returns the highest key that the DB holds: open, kept, with an intent, or
// with the directory of a virtual channel. The caller must hold db.mu.
func (db *DB) highestKey() ChannelKey {
	var highest ChannelKey
	for k := range db.mu.dbs.unary {
		highest = max(highest, k)
	}
	for k := range db.mu.dbs.hidden {
		highest = max(highest, k)
	}
	for _, k := range db.intents.all() {
		highest = max(highest, k)
	}
	for k := range db.mu.virtualDirs {
		highest = max(highest, k)
	}
	return highest
}

// removeOrphanIntents removes the files of each key with an intent that has no record,
// is not in hidden, and has no open channel: files that a crash left, or none. Removing
// them clears the intent. The caller must hold db.mu.
func (db *DB) removeOrphanIntents(records, hidden set.Set[ChannelKey]) error {
	var (
		err error
		rm  = &removals{}
	)
	for _, k := range db.intents.all() {
		if records.Contains(k) || hidden.Contains(k) {
			continue
		}
		if _, open := db.mu.dbs.unary[k]; open {
			continue
		}
		if db.mu.virtualDirs.Contains(k) {
			continue
		}
		err = errors.Join(err, rm.remove(db, k))
	}
	return errors.Join(err, rm.close(db))
}

// reopenFailedDeletes clears the intent of each key in records and opens its channel if
// Open skipped it. The caller must hold db.mu.
func (db *DB) reopenFailedDeletes(
	ctx context.Context,
	records set.Set[ChannelKey],
) error {
	var failed []ChannelKey
	for _, k := range db.intents.all() {
		if !records.Contains(k) {
			continue
		}
		failed = append(failed, k)
		if db.mu.skipped.Contains(k) {
			db.mu.skipped.Remove(k)
			if err := db.openChannel(ctx, Channel{Key: k}); err != nil {
				return err
			}
		}
	}
	return db.intents.clear(failed...)
}

// usedIndexes returns the indexes outside records that a channel in records or in kept
// uses. The caller must hold db.mu.
func (db *DB) usedIndexes(records, kept set.Set[ChannelKey]) set.Set[ChannelKey] {
	used := set.New[ChannelKey]()
	for k, u := range db.mu.dbs.unary {
		ch := u.Channel()
		if ch.IsIndex || ch.Index == 0 || records.Contains(ch.Index) {
			continue
		}
		if records.Contains(k) || kept.Contains(k) {
			used.Add(ch.Index)
		}
	}
	return used
}
