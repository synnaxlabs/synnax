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
	"maps"
	"slices"
	"sync"

	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
)

// pins counts the readers of each segment, and holds each dead segment that waits for
// its last reader before the engine unlinks its file.
type pins struct {
	mu      sync.Mutex
	readers map[uint64]int
	dead    map[uint64]state.Dead
	// parts counts the readers of each part, which a switch keeps.
	parts map[uint64]int
	// keys counts the loads of each key that pin parts. Such a load may read the file
	// of any segment that a part of its range held a record of, whichever part holds
	// that record now.
	keys map[ChannelKey]int
}

// held is what a load pins: the segments of its Channel, and the parts of its range.
type held struct {
	key      ChannelKey
	segments []uint64
	parts    []checkpoint.Part
}

// load returns the visible Channel of key from e and the parts of its range, or the
// error of parts that e no longer holds. It pins each segment and part under the same
// lock that release takes, so release never unlinks a segment that a Channel it
// returned still names, and a switch keeps each part. It unpins prev, the pins of an
// earlier load, and returns the new pins and the error of the unlinks that the unpin
// frees.
func (e *engine) load(
	key ChannelKey,
	prev held,
) (ch *state.Channel, h held, err, unlinkErr error) {
	e.pins.mu.Lock()
	h.key = key
	if ch = e.state.Channel(key); ch != nil {
		h.segments = make([]uint64, len(ch.Segments))
		for i, seg := range ch.Segments {
			h.segments[i] = seg.ID
		}
		h.parts, err = e.partsOf(key, ch.Parts)
	}
	e.pins.pin(h)
	free := e.pins.unpin(prev)
	e.pins.mu.Unlock()
	return ch, h, err, e.unlink(free)
}

// pinned returns the parts that readers pin.
func (e *engine) pinned() []uint64 {
	e.pins.mu.Lock()
	defer e.pins.mu.Unlock()
	return slices.Collect(maps.Keys(e.pins.parts))
}

// pin pins segment id for a reader.
func (e *engine) pin(id uint64) {
	e.pins.mu.Lock()
	e.pins.pin(held{segments: []uint64{id}})
	e.pins.mu.Unlock()
}

// unpin drops a reader's pins of h, and unlinks each dead segment that no reader pins
// after it.
func (e *engine) unpin(h held) error {
	e.pins.mu.Lock()
	free := e.pins.unpin(h)
	e.pins.mu.Unlock()
	return e.unlink(free)
}

// release unlinks the file of each segment of dead, which left the state, once no
// reader pins it, and no load of its key pins parts when a part held its record.
func (e *engine) release(dead []state.Dead) error {
	var free []state.Dead
	e.pins.mu.Lock()
	for _, d := range dead {
		if e.pins.held(d) {
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
		errs = append(errs, e.fs.Remove(segmentName(d.Key, d.Segment)))
	}
	return errors.Join(errs...)
}

// pin adds a reader of each segment and part of h. The caller holds p.mu.
func (p *pins) pin(h held) {
	for _, id := range h.segments {
		p.readers[id]++
	}
	for _, part := range h.parts {
		p.parts[part.ID]++
	}
	if len(h.parts) > 0 {
		p.keys[h.key]++
	}
}

// unpin drops a reader of each segment and part of h, and returns the dead segments
// that no reader pins after it. The caller holds p.mu.
func (p *pins) unpin(h held) []state.Dead {
	for _, part := range h.parts {
		if p.parts[part.ID]--; p.parts[part.ID] == 0 {
			delete(p.parts, part.ID)
		}
	}
	unpinned := false
	if len(h.parts) > 0 {
		if p.keys[h.key]--; p.keys[h.key] == 0 {
			delete(p.keys, h.key)
			unpinned = true
		}
	}
	for _, id := range h.segments {
		if p.readers[id]--; p.readers[id] == 0 {
			delete(p.readers, id)
		}
	}
	if len(p.dead) == 0 {
		return nil
	}
	var free []state.Dead
	take := func(d state.Dead) {
		if !p.held(d) {
			delete(p.dead, d.Segment)
			free = append(free, d)
		}
	}
	if unpinned {
		for _, d := range p.dead {
			if d.Key == h.key {
				take(d)
			}
		}
	}
	for _, id := range h.segments {
		if d, dead := p.dead[id]; dead {
			take(d)
		}
	}
	return free
}

// held reports whether a reader pins the dead segment d, or a load of its key pins
// parts when a part held its record. The caller holds p.mu.
func (p *pins) held(d state.Dead) bool {
	return p.readers[d.Segment] > 0 || (d.Part != 0 && p.keys[d.Key] > 0)
}
