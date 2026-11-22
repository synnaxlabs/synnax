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

// pins counts the readers of each segment, and holds the key of each dead segment that
// waits for its last reader before the engine unlinks its file.
type pins struct {
	mu      sync.Mutex
	readers map[uint64]int
	dead    map[uint64]ChannelKey
}

// loadView returns the visible Channels of key and then of its clock from e, and pins
// each of their segments under the same lock that release takes, so release never
// unlinks a segment that a Channel it returned still names. A commit publishes its
// clock before its data, so the clock has stamps for each sample of the key. It unpins
// held, the segments of an earlier load, and returns the segments it pinned.
func (e *engine) loadView(
	key, clock ChannelKey,
	held []uint64,
) (ch, clocks *state.Channel, ids []uint64, err error) {
	e.pins.mu.Lock()
	ch, clocks = e.state.Channel(key), e.state.Channel(clock)
	for _, c := range []*state.Channel{ch, clocks} {
		if c == nil || (c == clocks && clock == key) {
			continue
		}
		for _, seg := range c.Segments {
			ids = append(ids, seg.ID)
		}
	}
	e.pins.pin(ids)
	free := e.pins.unpin(held)
	e.pins.mu.Unlock()
	return ch, clocks, ids, e.unlink(free)
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

// pinnedDead returns the file names of the dead segments of key that wait for their
// last reader.
func (e *engine) pinnedDead(key ChannelKey) set.Set[string] {
	e.pins.mu.Lock()
	defer e.pins.mu.Unlock()
	names := set.New[string]()
	for id, k := range e.pins.dead {
		if k == key {
			names.Add(segmentName(k, id))
		}
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
