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
	"go.uber.org/zap"
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

// held is what a load pins: the segments of its Channels, the parts of their ranges,
// and the keys whose parts it pins.
type held struct {
	keys     []ChannelKey
	segments []uint64
	parts    []checkpoint.Part
}

// load returns the visible Channel of key and the parts of its range, and pins them, as
// loadView does. It unpins prev, the pins of an earlier load.
func (e *engine) load(key ChannelKey, prev held) (*state.Channel, held, error) {
	chs, parts := []*state.Channel{nil}, [][]checkpoint.Part{nil}
	h, err := e.loadView([]ChannelKey{key}, chs, parts, prev, nil)
	return chs[0], h, err
}

// loadView sets each of chs to the visible Channel of the key of keys at its index, and
// each of parts to the parts of its range, from one step of the state. It pins each of
// their segments and parts under the same lock that release takes, so release never
// unlinks a segment that a Channel it returned still names, and a switch keeps each
// part. It pins a key that keys names twice once. It unpins prev, the pins of an
// earlier load, as unpin does, and returns the new pins and the error of parts that e
// no longer holds. It puts the segments that it pins in buf when it has room, which
// must not share an array with prev.
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
		key, dst := keys[i], &parts[i]
		if j := slices.Index(keys[:i], key); j >= 0 {
			*dst = parts[j]
			continue
		}
		*dst = nil
		if ch == nil {
			continue
		}
		for _, seg := range ch.Segments {
			h.segments = append(h.segments, seg.ID)
		}
		p, partErr := e.partsOf(key, ch.Parts)
		*dst, err = p, errors.Join(err, partErr)
		h.parts = append(h.parts, p...)
		if len(p) > 0 {
			h.keys = append(h.keys, key)
		}
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
		if d.Key == key {
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

// unlinkLate removes the file of each segment of dead that its last reader unpinned.
// The removal belongs to no caller then, so a failure is logged with the key and the
// file, and the file waits for the orphan pass of the next open.
func (e *engine) unlinkLate(dead []state.Dead) {
	for _, d := range dead {
		name := segmentName(d.Key, d.Segment, d.File)
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
		errs = append(errs, e.removeSegment(d.Key, d.Segment, d.File))
	}
	return errors.Join(errs...)
}

// pin adds a reader of each segment and part of h, and a load of each key of h. The
// caller holds p.mu.
func (p *pins) pin(h held) {
	for _, id := range h.segments {
		p.readers[id]++
	}
	for _, part := range h.parts {
		p.parts[part.ID]++
	}
	for _, k := range h.keys {
		p.keys[k]++
	}
}

// unpin drops a reader of each segment and part of h, and a load of each key of h, and
// returns the dead segments that nothing holds after it. The caller holds p.mu.
func (p *pins) unpin(h held) []state.Dead {
	for _, part := range h.parts {
		if p.parts[part.ID]--; p.parts[part.ID] == 0 {
			delete(p.parts, part.ID)
		}
	}
	var unpinned []ChannelKey
	for _, k := range h.keys {
		if p.keys[k]--; p.keys[k] == 0 {
			delete(p.keys, k)
			unpinned = append(unpinned, k)
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
	for _, d := range p.dead {
		if slices.Contains(unpinned, d.Key) {
			take(d)
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
