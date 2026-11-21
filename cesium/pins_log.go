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
	"sync"

	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
)

// pins counts the readers of each segment, and holds the dead segments that wait for
// their last reader before the engine unlinks their files.
type pins struct {
	mu      sync.Mutex
	readers map[uint64]int
	dead    set.Set[uint64]
}

// load returns the visible Channel of key from e, and pins each of its segments under
// the same lock that release takes, so release never unlinks a segment that a Channel
// it returned still names. It unpins held, the segments of an earlier load, and returns
// the segments it pinned.
func (e *engine) load(key ChannelKey, held []uint64) (*state.Channel, []uint64, error) {
	e.pins.mu.Lock()
	ch := e.state.Channel(key)
	var ids []uint64
	if ch != nil {
		ids = make([]uint64, len(ch.Segments))
		for i, seg := range ch.Segments {
			ids[i] = seg.ID
		}
	}
	e.pins.pin(ids)
	free := e.pins.unpin(held)
	e.pins.mu.Unlock()
	return ch, ids, e.unlink(free)
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

// release unlinks each segment of ids, which left the state, once no reader pins it.
func (e *engine) release(ids []uint64) error {
	var free []uint64
	e.pins.mu.Lock()
	for _, id := range ids {
		if e.pins.readers[id] > 0 {
			e.pins.dead.Add(id)
		} else {
			free = append(free, id)
		}
	}
	e.pins.mu.Unlock()
	return e.unlink(free)
}

// unlink removes the file of each segment of ids.
func (e *engine) unlink(ids []uint64) error {
	e.keys.Lock()
	defer e.keys.Unlock()
	for _, id := range ids {
		key, ok := e.keys.of[id]
		if !ok {
			return errors.Newf("no key of dead segment %d", id)
		}
		delete(e.keys.of, id)
		if err := e.fs.Remove(segmentName(key, id)); err != nil {
			return err
		}
	}
	return nil
}

// pin adds a reader of each segment of ids. The caller holds p.mu.
func (p *pins) pin(ids []uint64) {
	for _, id := range ids {
		p.readers[id]++
	}
}

// unpin drops a reader of each segment of ids, and returns the dead segments that no
// reader pins after it. The caller holds p.mu.
func (p *pins) unpin(ids []uint64) []uint64 {
	var free []uint64
	for _, id := range ids {
		if p.readers[id]--; p.readers[id] > 0 {
			continue
		}
		delete(p.readers, id)
		if p.dead.Contains(id) {
			p.dead.Remove(id)
			free = append(free, id)
		}
	}
	return free
}
