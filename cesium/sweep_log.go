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
	"os"
	"path"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// intend logs a delete intent for each of keys, or clears it when cleared is true, in
// one log entry, and returns once the entry is durable.
func (e *engine) intend(keys []ChannelKey, cleared bool) error {
	if len(keys) == 0 {
		return nil
	}
	var entry log.Entry
	for _, k := range keys {
		entry.Add(&log.Intent{Key: k, Cleared: cleared})
	}
	_, err := e.log.Append(&entry).Wait()
	return err
}

// dropLog drops keys on the pointer log layout, as Drop does: one durable Drop record
// per key, then, once the log covers it, the removal of its files and its metadata. It
// keeps a clock that a key outside keys uses, and logs the Drop of each clock after the
// Drops of the keys it times. The caller holds db.mu.
func (db *DB) dropLog(ctx context.Context, keys []ChannelKey) error {
	keys = lo.Uniq(keys)
	dropping := set.New(keys...)
	snapshot, err := db.engine.state.Snapshot()
	if err != nil {
		return err
	}
	var dropped, clocks []ChannelKey
	for _, k := range keys {
		ch, live := db.mu.channels[k]
		if !live {
			ch, live = db.mu.hiddenChannels[k]
		}
		stored := db.engine.state.Channel(k)
		if !live && stored == nil {
			continue
		}
		own, err := db.engine.preload(k, telem.TimeStampMin, telem.TimeStampMax, true)
		if err != nil {
			return err
		}
		if !ch.IsIndex && !timesItself(stored, own.Segments, k) {
			dropped = append(dropped, k)
			continue
		}
		if !db.indexUsedOutside(k, dropping) &&
			!clockStoredOutside(snapshot, own.Users, k, dropping) {
			clocks = append(clocks, k)
		}
	}
	dropped = append(dropped, clocks...)
	if len(dropped) == 0 {
		return nil
	}
	// The Drop of each key checks the part records that name it as their clock.
	for _, k := range dropped {
		if _, err = db.engine.withPreloads(
			nil,
			[]ChannelKey{k},
			telem.TimeStampMin,
			telem.TimeStampMax,
			true,
			func(pre []state.Preload) error { return db.engine.state.BeginDrop(k, pre...) },
		); err != nil {
			return err
		}
	}
	var entry log.Entry
	for _, k := range dropped {
		entry.Add(&log.Drop{Key: k})
	}
	sequence, err := db.engine.log.Append(&entry).Wait()
	if err != nil {
		return err
	}
	for _, k := range dropped {
		delete(db.mu.channels, k)
		delete(db.mu.hiddenChannels, k)
	}
	if err = db.engine.waitCovered(ctx, sequence); err != nil {
		return err
	}
	err = db.engine.reap()
	for _, k := range dropped {
		err = errors.Join(err, db.removeLogDir(k))
	}
	return err
}

// removeLogDir removes the files in the directory of a dropped key that no reader pins,
// and the directory once it is empty. The Drop gave the segments of the key to the
// engine, which unlinks each pinned file once the log covers the Drop and no reader
// pins it. A directory that a reader keeps goes at a later Sweep, as the directory of a
// torn key.
func (db *DB) removeLogDir(key ChannelKey) error {
	dir := keyToDirName(key)
	infos, err := db.fs.List(dir)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	pinned := db.engine.pinnedFiles(key)
	kept := false
	for _, info := range infos {
		name := path.Join(dir, info.Name())
		if pinned.Contains(name) {
			kept = true
			continue
		}
		if err = db.fs.Remove(name); err != nil {
			return err
		}
	}
	if kept {
		return db.fs.SyncDir(dir)
	}
	if err = db.fs.Remove(dir); err != nil {
		return err
	}
	return db.fs.SyncDir("")
}

// timesItself reports whether a stored segment of key has key as its clock: a segment
// of ch, the stored channel of key, that no part holds, or a part record of parted.
func timesItself(ch *state.Channel, parted []state.Segment, key ChannelKey) bool {
	clocked := func(seg state.Segment) bool { return seg.Clock == key }
	if ch != nil && slices.ContainsFunc(ch.Segments, func(seg state.Segment) bool {
		return seg.Part == 0 && clocked(seg)
	}) {
		return true
	}
	return slices.ContainsFunc(parted, clocked)
}

// clockStoredOutside reports whether a stored segment of a key outside dropping has
// clock as its clock: a segment of s that no part holds, or a part record of users.
func clockStoredOutside(
	s *state.Snapshot,
	users []state.User,
	clock ChannelKey,
	dropping set.Set[ChannelKey],
) bool {
	for _, ch := range s.Channels {
		if ch.Key == clock || dropping.Contains(ch.Key) {
			continue
		}
		for _, seg := range ch.Segments {
			if seg.Part == 0 && seg.Clock == clock {
				return true
			}
		}
	}
	return slices.ContainsFunc(users, func(u state.User) bool {
		return !dropping.Contains(u.Key)
	})
}

// indexUsedOutside reports whether a channel outside dropping, open or kept, uses
// index. The caller holds db.mu.
func (db *DB) indexUsedOutside(index ChannelKey, dropping set.Set[ChannelKey]) bool {
	for _, chs := range []map[ChannelKey]Channel{db.mu.channels, db.mu.hiddenChannels} {
		for k, ch := range chs {
			if !ch.IsIndex && ch.Index == index && !dropping.Contains(k) {
				return true
			}
		}
	}
	return false
}

// decideLog decides the fate of each key of the pointer log layout against records, as
// decide does on the old layout, and returns the keys it keeps. It removes the
// directory of each torn key outside records. The caller holds db.mu.
func (db *DB) decideLog(
	ctx context.Context,
	records set.Set[ChannelKey],
) (set.Set[ChannelKey], error) {
	if err := db.removeTorn(records); err != nil {
		return nil, err
	}
	snapshot, err := db.engine.state.Snapshot()
	if err != nil {
		return nil, err
	}
	intents := set.New(snapshot.Intents...)
	var cleared []ChannelKey
	for k := range intents {
		if records.Contains(k) {
			cleared = append(cleared, k)
		}
	}
	slices.Sort(cleared)
	if err = db.engine.intend(cleared, true); err != nil {
		return nil, err
	}
	sizes := storedSizes(snapshot)
	keys := set.New[ChannelKey]()
	for k := range db.mu.channels {
		keys.Add(k)
	}
	for k := range sizes {
		keys.Add(k)
	}
	kept := set.New[ChannelKey]()
	var drop []ChannelKey
	for k := range keys {
		switch {
		case records.Contains(k):
		case intents.Contains(k) || sizes[k] == 0:
			drop = append(drop, k)
		default:
			kept.Add(k)
		}
	}
	hidden := db.usedLogIndexes(records, kept)
	drop = slices.DeleteFunc(drop, hidden.Contains)
	hidden.Add(kept.Slice()...)
	for k := range hidden {
		if ch, ok := db.mu.channels[k]; ok {
			db.mu.hiddenChannels[k] = ch
			delete(db.mu.channels, k)
		}
	}
	slices.Sort(drop)
	if err = db.engine.intend(drop, false); err != nil {
		return nil, err
	}
	return kept, db.dropLog(ctx, drop)
}

// removeTorn removes the directory of each torn key outside records. The caller holds
// db.mu.
func (db *DB) removeTorn(records set.Set[ChannelKey]) error {
	for k := range db.mu.torn {
		if records.Contains(k) {
			continue
		}
		if err := db.removeLogDir(k); err != nil {
			return err
		}
		db.mu.torn.Remove(k)
	}
	return nil
}

// usedLogIndexes returns the indexes outside records that a channel in records or in
// kept uses. The caller holds db.mu.
func (db *DB) usedLogIndexes(records, kept set.Set[ChannelKey]) set.Set[ChannelKey] {
	used := set.New[ChannelKey]()
	for k, ch := range db.mu.channels {
		if ch.IsIndex || ch.Index == 0 || records.Contains(ch.Index) {
			continue
		}
		if records.Contains(k) || kept.Contains(k) {
			used.Add(ch.Index)
		}
	}
	return used
}

// storedSizes returns the committed bytes of each key that the state holds.
func storedSizes(s *state.Snapshot) map[ChannelKey]telem.Size {
	sizes := make(map[ChannelKey]telem.Size, len(s.Channels))
	for _, ch := range s.Channels {
		var size telem.Size
		for _, seg := range ch.Segments {
			for _, ptr := range seg.Pointers {
				size += telem.Size(ptr.Length)
			}
		}
		sizes[ch.Key] = size
	}
	return sizes
}

// keptLogChannels returns the channels of kept that are still hidden, in rising order
// of key. The caller holds db.mu.
func (db *DB) keptLogChannels(kept set.Set[ChannelKey]) ([]KeptChannel, error) {
	snapshot, err := db.engine.state.Snapshot()
	if err != nil {
		return nil, err
	}
	sizes := storedSizes(snapshot)
	out := make([]KeptChannel, 0, len(kept))
	for k := range kept {
		ch, ok := db.mu.hiddenChannels[k]
		if !ok {
			if _, stored := sizes[k]; !stored {
				continue
			}
		}
		index := ch.Index
		if ch.IsIndex {
			index = 0
		}
		out = append(out, KeptChannel{Key: k, Index: index, Size: sizes[k]})
	}
	slices.SortFunc(out, func(a, b KeptChannel) int { return int(a.Key) - int(b.Key) })
	return out, nil
}

// highestLogKey returns the highest key that the pointer log layout holds: open, kept,
// stored, or with an intent. The caller holds db.mu.
func (db *DB) highestLogKey() (ChannelKey, error) {
	snapshot, err := db.engine.state.Snapshot()
	if err != nil {
		return 0, err
	}
	var highest ChannelKey
	for k := range db.mu.channels {
		highest = max(highest, k)
	}
	for k := range db.mu.hiddenChannels {
		highest = max(highest, k)
	}
	for _, ch := range snapshot.Channels {
		highest = max(highest, ch.Key)
	}
	for _, k := range snapshot.Intents {
		highest = max(highest, k)
	}
	return highest, nil
}
