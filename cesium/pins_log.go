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
	"slices"
	"sync"

	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// pins counts the readers of each segment, and holds the key of each dead segment that
// waits for its last reader before the engine unlinks its file.
type pins struct {
	mu      sync.Mutex
	readers map[uint64]int
	dead    map[uint64]ChannelKey
}

// loadView sets each of chs to the visible Channel of the key of keys at its index,
// from one step of the state, and pins each of their segments under the same lock that
// release takes, so release never unlinks a segment that a Channel it returned still
// names. It pins a key that keys names twice once. It unpins held, the segments of an
// earlier load, and returns the publish count of the load, which State.Unchanged takes,
// and the segments it pinned, in buf when it has room. buf must not share an array with
// held.
func (e *engine) loadView(
	keys []ChannelKey,
	chs []*state.Channel,
	held, buf []uint64,
) (count uint64, ids []uint64, err error) {
	for {
		count = e.state.Load(keys, chs)
		e.pins.mu.Lock()
		// A segment leaves the state only on a step that publishes, so with the count
		// unchanged, release has unlinked no segment of chs.
		if e.state.Unchanged(count) {
			break
		}
		e.pins.mu.Unlock()
	}
	n := 0
	for _, ch := range chs {
		if ch != nil {
			n += len(ch.Segments)
		}
	}
	ids = buf[:0]
	if cap(ids) < n {
		ids = make([]uint64, 0, n)
	}
	for i, ch := range chs {
		if ch == nil || slices.Contains(keys[:i], keys[i]) {
			continue
		}
		for _, seg := range ch.Segments {
			ids = append(ids, seg.ID)
		}
	}
	e.pins.pin(ids)
	free := e.pins.unpin(held)
	e.pins.mu.Unlock()
	return count, ids, e.unlink(free)
}

// repin pins ids and unpins held under one lock, and unlinks each dead segment that no
// reader pins after it.
func (e *engine) repin(held, ids []uint64) error {
	e.pins.mu.Lock()
	e.pins.pin(ids)
	free := e.pins.unpin(held)
	e.pins.mu.Unlock()
	return e.unlink(free)
}

// pin pins segment id for a reader.
func (e *engine) pin(id uint64) {
	e.pins.mu.Lock()
	e.pins.pin([]uint64{id})
	e.pins.mu.Unlock()
}

// unpin drops a reader's pins of ids, and unlinks each dead segment that no reader
// pins after it.
func (e *engine) unpin(ids []uint64) error {
	e.pins.mu.Lock()
	free := e.pins.unpin(ids)
	e.pins.mu.Unlock()
	return e.unlink(free)
}

// release unlinks the file of each segment of dead, which left the state, once no
// reader pins it.
func (e *engine) release(dead []state.Dead) error {
	var free []state.Dead
	e.pins.mu.Lock()
	for _, d := range dead {
		if e.pins.readers[d.Segment] > 0 {
			e.pins.dead[d.Segment] = d.Key
		} else {
			free = append(free, d)
		}
	}
	e.pins.mu.Unlock()
	return e.unlink(free)
}

// pinnedFiles returns the file names of the segments of key that a reader pins. After a
// Drop, the engine unlinks each of them once the log covers the Drop and no reader pins
// it, so a file that waits for either stays.
func (e *engine) pinnedFiles(key ChannelKey) set.Set[string] {
	e.pins.mu.Lock()
	defer e.pins.mu.Unlock()
	names := set.New[string]()
	for id := range e.pins.readers {
		names.Add(segmentName(key, id))
	}
	return names
}

// unlink removes the file of each segment of dead, and tries every removal.
func (e *engine) unlink(dead []state.Dead) error {
	errs := make([]error, 0, len(dead))
	for _, d := range dead {
		errs = append(errs, e.fs.Remove(segmentName(d.Key, d.Segment)))
	}
	return errors.Join(errs...)
}

// pin adds a reader of each segment of ids. The caller holds p.mu.
func (p *pins) pin(ids []uint64) {
	for _, id := range ids {
		p.readers[id]++
	}
}

// unpin drops a reader of each segment of ids, and returns the dead segments that no
// reader pins after it. The caller holds p.mu.
func (p *pins) unpin(ids []uint64) []state.Dead {
	var free []state.Dead
	for _, id := range ids {
		if p.readers[id]--; p.readers[id] > 0 {
			continue
		}
		delete(p.readers, id)
		if key, dead := p.dead[id]; dead {
			delete(p.dead, id)
			free = append(free, state.Dead{Segment: id, Key: key})
		}
	}
	return free
}
