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
	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// withPreloads runs step with the preload of each key of keys over [first, last],
// appended to pre[:0], and loads them again while step returns state.ErrStale. keys are
// in rising order. It returns the preloads of the last run, so the caller may reuse
// their slice.
func (e *engine) withPreloads(
	pre []state.Preload,
	keys []ChannelKey,
	first, last telem.TimeStamp,
	step func([]state.Preload) error,
) ([]state.Preload, error) {
	for {
		pre = pre[:0]
		for _, key := range keys {
			p, err := e.preload(key, first, last)
			if err != nil {
				return pre, err
			}
			pre = append(pre, p)
		}
		if err := step(pre); !errors.Is(err, state.ErrStale) {
			return pre, err
		}
	}
}

// preload returns the live part records of key whose pointers reach [first, last].
// Only a part whose stamps reach the range needs its records, and only a read of a part
// that the part cache lacks pins it.
func (e *engine) preload(
	key ChannelKey,
	first, last telem.TimeStamp,
) (state.Preload, error) {
	ch := e.state.Channel(key)
	if ch == nil {
		return state.Preload{Key: key}, nil
	}
	pre := state.Preload{Key: key, Generation: partsGeneration(ch)}
	parts, err := e.partsOf(ch)
	if err != nil {
		return pre, err
	}
	var buf [4]checkpoint.Part
	if reach := reaching(parts, first, last, buf[:0]); len(reach) == 0 ||
		e.cached(&pre, ch, reach, key, first, last) {
		return pre, nil
	}
	ch, h, err := e.pinReaching(key, first, last, buf[:0])
	if err != nil || ch == nil {
		return state.Preload{Key: key}, err
	}
	pre.Generation = partsGeneration(ch)
	for _, p := range h.parts {
		r, err := e.cache.Records(p, key)
		if err != nil {
			e.unpin(h)
			return pre, err
		}
		addLive(&pre, ch, p, r, first, last)
	}
	e.unpin(h)
	return pre, nil
}

// cached adds to pre the live records of key in reach, parts of ch, that reach [first,
// last], when the part cache holds the records of every part of reach. It reports
// whether it did.
func (e *engine) cached(
	pre *state.Preload,
	ch *state.Channel,
	reach []checkpoint.Part,
	key ChannelKey,
	first, last telem.TimeStamp,
) bool {
	for _, p := range reach {
		r, ok := e.cache.Cached(p, key)
		if !ok {
			pre.Segments = pre.Segments[:0]
			return false
		}
		addLive(pre, ch, p, r, first, last)
	}
	return true
}

// addLive adds to pre each record of r, the records in part p of the key of ch, that is
// live and reaches [first, last].
func addLive(
	pre *state.Preload,
	ch *state.Channel,
	p checkpoint.Part,
	r checkpoint.Records,
	first, last telem.TimeStamp,
) {
	var buf [16]int
	for _, i := range r.Reaching(first, last, buf[:0]) {
		seg := r.Segments[i]
		if p.Live(r.First+i) && !killed(ch.Parts.Kills, p, seg.ID) &&
			reaches(seg.Pointers, first, last) {
			pre.Segments = append(pre.Segments, seg)
		}
	}
}

// reaching appends to buf the parts of parts whose stamps reach [first, last].
func reaching(
	parts []checkpoint.Part,
	first, last telem.TimeStamp,
	buf []checkpoint.Part,
) []checkpoint.Part {
	for _, p := range parts {
		if p.First <= last && first <= p.Last {
			buf = append(buf, p)
		}
	}
	return buf
}

// partsGeneration returns the generation of the head whose parts ch names, or 0 when
// none holds records of its key.
func partsGeneration(ch *state.Channel) uint64 {
	if ch.Parts == nil {
		return 0
	}
	return ch.Parts.Generation
}

// pinReaching returns the Channel of key and pins the parts of its range whose stamps
// reach [first, last], appended to buf, under the lock that the pins of a load take.
func (e *engine) pinReaching(
	key ChannelKey,
	first, last telem.TimeStamp,
	buf []checkpoint.Part,
) (*state.Channel, held, error) {
	e.pins.mu.Lock()
	defer e.pins.mu.Unlock()
	ch := e.state.Channel(key)
	if ch == nil {
		return nil, held{}, nil
	}
	parts, err := e.partsOf(ch)
	if err != nil {
		return ch, held{}, err
	}
	h := held{parts: reaching(parts, first, last, buf)}
	e.pins.pin(h)
	return ch, h, nil
}

// reaches reports whether a pointer of ptrs shares a stamp with [first, last].
func reaches(ptrs []state.Pointer, first, last telem.TimeStamp) bool {
	for _, p := range ptrs {
		if p.First <= last && first <= p.Last {
			return true
		}
	}
	return false
}
