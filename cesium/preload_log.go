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

// addPreload adds the part records of key whose pointers reach [first, last] to the
// preload of key in pre.
func (e *engine) addPreload(
	pre []state.Preload,
	key ChannelKey,
	first, last telem.TimeStamp,
) ([]state.Preload, error) {
	p, err := e.preload(key, first, last)
	if err != nil || len(p.Segments) == 0 {
		return pre, err
	}
	for i := range pre {
		if pre[i].Key != key {
			continue
		}
		for _, rec := range p.Segments {
			if !slices.ContainsFunc(pre[i].Segments, func(s state.Segment) bool {
				return s.ID == rec.ID
			}) {
				pre[i].Segments = append(pre[i].Segments, rec)
			}
		}
		return pre, nil
	}
	return append(pre, p), nil
}

// preload returns the live part records of key whose pointers reach [first, last].
// Only a part whose stamps reach the range needs a read, which it pins.
func (e *engine) preload(
	key ChannelKey,
	first, last telem.TimeStamp,
) (state.Preload, error) {
	ch := e.state.Channel(key)
	if ch == nil {
		return state.Preload{Key: key}, nil
	}
	pre := state.Preload{Key: key, Generation: ch.Generation}
	parts, err := e.partsOf(ch)
	if err != nil || !reachesAny(parts, first, last) {
		return pre, err
	}
	ch, h, err := e.pinReaching(key, first, last)
	if err != nil || ch == nil {
		return state.Preload{Key: key}, err
	}
	pre.Generation = ch.Generation
	for _, p := range h.parts {
		r, err := e.cache.Records(p, key)
		if err != nil {
			return pre, errors.Join(err, e.unpin(h))
		}
		for i, seg := range r.Segments {
			if p.Live(r.First+i) && !killed(ch.Kills, p, seg.ID) &&
				reaches(seg.Pointers, first, last) {
				seg.Part = p.ID
				pre.Segments = append(pre.Segments, seg)
			}
		}
	}
	return pre, e.unpin(h)
}

// pinReaching returns the Channel of key and pins the parts of its range whose stamps
// reach [first, last], under the lock that the pins of a load take.
func (e *engine) pinReaching(
	key ChannelKey,
	first, last telem.TimeStamp,
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
	var h held
	for _, p := range parts {
		if p.First <= last && first <= p.Last {
			h.parts = append(h.parts, p)
		}
	}
	e.pins.pin(h)
	return ch, h, nil
}

// reachesAny reports whether the stamps of a part of parts reach [first, last].
func reachesAny(parts []checkpoint.Part, first, last telem.TimeStamp) bool {
	for _, p := range parts {
		if p.First <= last && first <= p.Last {
			return true
		}
	}
	return false
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
