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
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
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
	// of any segment of its key that a part held a record of, which the state no longer
	// tells apart.
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
	ch = e.state.Channel(key)
	h, err = e.pinLocked(key, ch)
	free := e.pins.unpin(prev)
	e.pins.mu.Unlock()
	return ch, h, err, e.unlink(free)
}

// loadView loads key and its clock as load does, from one step of the state, so the
// samples of key and their stamps come from one state. For an index, clock is key, and
// the clock half of the view is the same Channel with no pins of its own.
func (e *engine) loadView(
	key, clock ChannelKey,
	prev [2]held,
) (chs [2]*state.Channel, hs [2]held, err, unlinkErr error) {
	for {
		var count uint64
		chs[0], chs[1], count = e.state.Pair(key, clock)
		e.pins.mu.Lock()
		// A segment leaves the state only on a step that publishes, so with the count
		// unchanged, release has unlinked no segment of chs.
		if e.state.Unchanged(count) {
			break
		}
		e.pins.mu.Unlock()
	}
	hs[0], err = e.pinLocked(key, chs[0])
	if clock == key {
		chs[1] = chs[0]
	} else {
		var clockErr error
		hs[1], clockErr = e.pinLocked(clock, chs[1])
		err = errors.Join(err, clockErr)
	}
	free := append(e.pins.unpin(prev[0]), e.pins.unpin(prev[1])...)
	e.pins.mu.Unlock()
	return chs, hs, err, e.unlink(free)
}

// pinLocked pins each segment of ch, the visible Channel of key, and the parts of its
// range, and returns them, or the error of parts that e no longer holds. The caller
// holds e.pins.mu.
func (e *engine) pinLocked(key ChannelKey, ch *state.Channel) (h held, err error) {
	h.key = key
	if ch != nil {
		h.segments = make([]uint64, len(ch.Segments))
		for i, seg := range ch.Segments {
			h.segments[i] = seg.ID
		}
		h.parts, err = e.partsOf(key, ch.Parts)
	}
	e.pins.pin(h)
	return h, err
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
// reader pins it and no load of its key pins parts.
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

// pinnedFiles reports which files of the directory of key a reader may still read: the
// file of each dead segment of key that waits for its last reader, and every segment
// file of key while a load of key pins parts, since that load may read any segment of
// key that a part held a record of.
func (e *engine) pinnedFiles(key ChannelKey) func(name string) bool {
	e.pins.mu.Lock()
	defer e.pins.mu.Unlock()
	if e.pins.keys[key] > 0 {
		return isSegmentFile
	}
	names := set.New[string]()
	for _, d := range e.pins.dead {
		if ChannelKey(d.Key) == key {
			names.Add(segmentName(key, d.Segment, d.File))
		}
	}
	return names.Contains
}

// isSegmentFile reports whether name, a path in a channel directory, names the file of
// a segment: a version 1 segment file, or an old file of a version 0 segment.
func isSegmentFile(name string) bool {
	base := path.Base(name)
	if strings.HasSuffix(base, ".seg") {
		return true
	}
	n, ok := strings.CutSuffix(base, ".domain")
	if !ok {
		return false
	}
	_, err := strconv.ParseUint(n, 10, 16)
	return err == nil
}

// unlink removes the file of each segment of dead, and tries every removal.
func (e *engine) unlink(dead []state.Dead) error {
	errs := make([]error, 0, len(dead))
	for _, d := range dead {
		errs = append(errs, e.fs.Remove(segmentName(d.Key, d.Segment, d.File)))
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
// parts. The caller holds p.mu.
func (p *pins) held(d state.Dead) bool {
	return p.readers[d.Segment] > 0 || p.keys[d.Key] > 0
}
