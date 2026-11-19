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

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// KeptChannel is a channel that Sweep keeps: it holds samples but has no record and no
// delete intent. Writers and iterators cannot open on it until Purge removes it.
type KeptChannel struct {
	Key  ChannelKey
	Size telem.Size
}

// Sweep reconciles the DB with keys, the keys of every channel record, and must run
// before any writer opens. A key with an intent and a record lost the commit of its
// delete, so Sweep clears the intent and opens the channel. Sweep drops a key with an
// intent and no record, and a key with no record and no samples. It keeps a key with
// samples, no record, and no intent: writers and iterators on it fail, and Sweep
// returns it. An index that a kept or recorded channel uses stays, with any intent of
// the index.
func (db *DB) Sweep(ctx context.Context, keys []ChannelKey) ([]KeptChannel, error) {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed.Load() {
		return nil, ErrDBClosed
	}
	records := set.New(keys...)
	if err := db.reopenFailedDeletes(ctx, records); err != nil {
		return nil, err
	}
	kept := set.New[ChannelKey]()
	var drop []ChannelKey
	for k := range db.mu.dbs.virtual {
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
	err := errors.Join(db.dropLocked(drop), db.removeOrphanIntents(records, hidden))
	db.mu.skipped = set.New[ChannelKey]()
	if err != nil {
		return nil, err
	}
	out := make([]KeptChannel, 0, len(kept))
	for k := range kept {
		u := db.mu.dbs.hidden[k]
		out = append(out, KeptChannel{Key: k, Size: u.Size()})
	}
	slices.SortFunc(out, func(a, b KeptChannel) int { return int(a.Key) - int(b.Key) })
	return out, nil
}

// removeOrphanIntents removes the files of each key with an intent that has no record,
// is not in hidden, and has no open channel: files that a crash left, or none. Removing
// them clears the intent. The caller must hold db.mu.
func (db *DB) removeOrphanIntents(records, hidden set.Set[ChannelKey]) error {
	var err error
	for _, k := range db.intents.all() {
		if records.Contains(k) || hidden.Contains(k) {
			continue
		}
		if _, open := db.mu.dbs.unary[k]; open {
			continue
		}
		if _, open := db.mu.dbs.virtual[k]; open {
			continue
		}
		err = errors.Join(err, db.removeDir(k))
	}
	return err
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
			if err := db.openVirtualOrUnary(ctx, Channel{Key: k}); err != nil {
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

// Purge drops each key that Sweep kept. It returns an error naming a key that Sweep did
// not keep, and purges no key then.
func (db *DB) Purge(keys []ChannelKey) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	if db.closed.Load() {
		return ErrDBClosed
	}
	for _, k := range keys {
		if _, ok := db.mu.dbs.hidden[k]; !ok {
			return channel.Wrap(k, errors.Wrap(query.ErrNotFound, "not a kept channel"))
		}
	}
	for _, k := range keys {
		db.mu.dbs.unary[k] = db.mu.dbs.hidden[k]
		delete(db.mu.dbs.hidden, k)
	}
	return db.dropLocked(keys)
}
