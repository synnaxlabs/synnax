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
	"github.com/synnaxlabs/x/set"
	"go.uber.org/zap"
)

// pins counts the readers of each segment, and holds the key of each dead segment that
// waits for its last reader before the engine unlinks its file.
type pins struct {
	mu      sync.Mutex
	readers map[uint64]int
	dead    map[uint64]ChannelKey
	// parts counts the readers of each part, which a switch keeps.
	parts map[uint64]int
}

// held is what a load pins: the segments of its Channel, and the parts of its range.
type held struct {
	segments []uint64
	parts    []checkpoint.Part
}

// loadView sets each of chs to the visible Channel of the key of keys at its index, and
// each of parts to the parts of its range, from one step of the state. It pins each of
// their segments and parts under the same lock that release takes, so release never
// unlinks a segment that a Channel it returned still names, and a switch keeps each
// part. It pins a key that keys names twice once. It unpins prev, the pins of an earlier
// load, as unpin does, and returns the new pins and the error of parts that e no longer
// holds. It puts the segments that it pins in buf when it has room, which must not
// share an array with prev.
func (e *engine) loadView(
	keys []ChannelKey,
	chs []*state.Channel,
	parts [][]checkpoint.Part,
	prev held,
	buf []uint64,
) (h held, err error) {
	for {
		count := e.state.Load(keys, chs)
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
	h.segments = buf[:0]
	if cap(h.segments) < n {
		h.segments = make([]uint64, 0, n)
	}
	for i, ch := range chs {
		if j := slices.Index(keys[:i], keys[i]); j >= 0 {
			parts[i] = parts[j]
			continue
		}
		parts[i] = nil
		if ch == nil {
			continue
		}
		for _, seg := range ch.Segments {
			h.segments = append(h.segments, seg.ID)
		}
		p, partErr := e.partsOf(keys[i], ch.Parts)
		parts[i], err = p, errors.Join(err, partErr)
		h.parts = append(h.parts, p...)
	}
	e.pins.pin(h)
	free := e.pins.unpin(prev)
	e.pins.mu.Unlock()
	e.unlinkLate(free)
	return h, err
}

// repin pins next and unpins prev under one lock, and unlinks each dead segment that no
// reader pins after it, as unlinkLate does.
func (e *engine) repin(prev, next held) {
	e.pins.mu.Lock()
	e.pins.pin(next)
	free := e.pins.unpin(prev)
	e.pins.mu.Unlock()
	e.unlinkLate(free)
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
// after it, as unlinkLate does.
func (e *engine) unpin(h held) {
	e.pins.mu.Lock()
	free := e.pins.unpin(h)
	e.pins.mu.Unlock()
	e.unlinkLate(free)
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

// unlinkLate removes the file of each segment of dead that its last reader unpinned.
// The removal belongs to no caller then, so a failure is logged with the key and the
// file, and the file waits for the orphan pass of the next open.
func (e *engine) unlinkLate(dead []state.Dead) {
	for _, d := range dead {
		name := segmentName(d.Key, d.Segment)
		if err := e.fs.Remove(name); err != nil {
			e.ins.L.Error(
				"Failed to remove the file of a dead segment",
				zap.Uint32("key", d.Key),
				zap.String("file", name),
				zap.Error(err),
			)
		}
	}
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
}

// unpin drops a reader of each segment and part of h, and returns the dead segments
// that no reader pins after it. The caller holds p.mu.
func (p *pins) unpin(h held) []state.Dead {
	for _, part := range h.parts {
		if p.parts[part.ID]--; p.parts[part.ID] == 0 {
			delete(p.parts, part.ID)
		}
	}
	var free []state.Dead
	for _, id := range h.segments {
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
