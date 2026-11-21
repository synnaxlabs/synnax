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
)

// pins counts the readers of each segment, and holds each dead segment that waits for
// its last reader before the engine unlinks its file.
type pins struct {
	mu      sync.Mutex
	readers map[uint64]int
	dead    map[uint64]state.Dead
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

// release unlinks the file of each segment of dead, which left the state, once no
// reader pins it.
func (e *engine) release(dead []state.Dead) error {
	var free []state.Dead
	e.pins.mu.Lock()
	for _, d := range dead {
		if e.pins.readers[d.Segment] > 0 {
			e.pins.dead[d.Segment] = d
		} else {
			free = append(free, d)
		}
	}
	e.pins.mu.Unlock()
	return e.unlink(free)
}

// unlink removes the file of each segment of dead, and tries every removal.
func (e *engine) unlink(dead []state.Dead) error {
	errs := make([]error, 0, len(dead))
	for _, d := range dead {
		errs = append(errs, e.fs.Remove(segmentName(d.Key, d.Segment, d.File)))
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
		if d, dead := p.dead[id]; dead {
			delete(p.dead, id)
			free = append(free, d)
		}
	}
	return free
}
