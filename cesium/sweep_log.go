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
	"os"
	"path"
	"slices"

	"github.com/samber/lo"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/meta"
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
// per key, then the removal of its files and its metadata. It keeps an index that a
// channel outside keys uses. The caller holds db.mu.
func (db *DB) dropLog(keys []ChannelKey) error {
	keys = lo.Uniq(keys)
	dropping := set.New(keys...)
	var (
		entry   log.Entry
		dropped []ChannelKey
	)
	for _, k := range keys {
		ch, live := db.mu.channels[k]
		if !live {
			ch, live = db.mu.hiddenChannels[k]
		}
		if !live && db.engine.state.Channel(k) == nil {
			continue
		}
		if ch.IsIndex && db.indexUsedOutside(k, dropping) {
			continue
		}
		entry.Add(&log.Drop{Key: k})
		dropped = append(dropped, k)
	}
	if entry.Len() == 0 {
		return nil
	}
	if _, err := db.engine.log.Append(&entry).Wait(); err != nil {
		return err
	}
	err := db.engine.reap()
	for _, k := range dropped {
		delete(db.mu.channels, k)
		delete(db.mu.hiddenChannels, k)
		err = errors.Join(err, db.removeLogDir(k))
	}
	return err
}

// removeLogDir removes the metadata file and the directory of key, when they exist.
func (db *DB) removeLogDir(key ChannelKey) error {
	dir := keyToDirName(key)
	for _, name := range []string{path.Join(dir, meta.File), dir} {
		if err := db.fs.Remove(name); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	return db.fs.SyncDir("")
}

// indexUsedOutside reports whether a channel outside dropping, open or kept, uses
// index.
// The caller holds db.mu.
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

// decideLog decides the fate of each key of the pointer log layout against records,
// as decide does on the old layout, and returns the keys it keeps. The caller holds
// db.mu.
func (db *DB) decideLog(records set.Set[ChannelKey]) (set.Set[ChannelKey], error) {
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
	return kept, db.dropLog(drop)
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
