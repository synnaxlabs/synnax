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
// in rising order. With timed, each preload also holds the users of its key as a clock.
// It returns the preloads of the last run, so the caller may reuse their slice.
func (e *engine) withPreloads(
	pre []state.Preload,
	keys []ChannelKey,
	first, last telem.TimeStamp,
	timed bool,
	step func([]state.Preload) error,
) ([]state.Preload, error) {
	for {
		pre = pre[:0]
		for _, key := range keys {
			p, err := e.preload(key, first, last, timed)
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

// addPreload adds the part records of key whose pointers reach [first, last], and with
// timed the users of key as a clock, to the preload of key in pre.
func (e *engine) addPreload(
	pre []state.Preload,
	key ChannelKey,
	first, last telem.TimeStamp,
	timed bool,
) ([]state.Preload, error) {
	p, err := e.preload(key, first, last, timed)
	if err != nil || (len(p.Segments) == 0 && len(p.Users) == 0) {
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
		for _, u := range p.Users {
			if !slices.ContainsFunc(pre[i].Users, func(o state.User) bool {
				return o.Segment.ID == u.Segment.ID
			}) {
				pre[i].Users = append(pre[i].Users, u)
			}
		}
		return pre, nil
	}
	return append(pre, p), nil
}

// preload returns the live part records of key whose pointers reach [first, last], and
// with timed the users of key as a clock.
func (e *engine) preload(
	key ChannelKey,
	first, last telem.TimeStamp,
	timed bool,
) (state.Preload, error) {
	pre, err := e.records(key, first, last)
	if err != nil || !timed {
		return pre, err
	}
	pre.Users, err = e.clockUsers(key, first, last)
	return pre, err
}

// records returns the live part records of key whose pointers reach [first, last].
// Only a part whose stamps reach the range needs its records, and only a read of a part
// that the part cache lacks pins it.
func (e *engine) records(
	key ChannelKey,
	first, last telem.TimeStamp,
) (state.Preload, error) {
	pre := state.Preload{Key: key, Generation: e.parts.Load().live.generation}
	kp, ok := e.keyParts(key)
	if !ok || kp == nil {
		return pre, nil
	}
	pre.Generation = kp.Generation
	parts, err := e.partsOf(key, kp)
	if err != nil {
		return pre, err
	}
	var buf [4]checkpoint.Part
	if reach := reaching(parts, first, last, buf[:0]); len(reach) == 0 ||
		e.cached(&pre, kp, reach, key, first, last) {
		return pre, nil
	}
	kp, h, err := e.pinReaching(key, first, last, buf[:0])
	if err != nil || kp == nil {
		return pre, err
	}
	pre.Generation = kp.Generation
	for _, p := range h.parts {
		r, err := e.cache.Records(p, key)
		if err != nil {
			e.unpin(h)
			return pre, err
		}
		addLive(&pre, kp, p, r, first, last)
	}
	e.unpin(h)
	return pre, nil
}

// cached adds to pre the live records of key in reach, parts of kp, that reach [first,
// last], when the part cache holds the records of every part of reach. It reports
// whether it did.
func (e *engine) cached(
	pre *state.Preload,
	kp *state.Parts,
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
		addLive(pre, kp, p, r, first, last)
	}
	return true
}

// addLive adds to pre each record of r, the records in part p of the key of kp, that is
// live and reaches [first, last], tagged with p.
func addLive(
	pre *state.Preload,
	kp *state.Parts,
	p checkpoint.Part,
	r checkpoint.Records,
	first, last telem.TimeStamp,
) {
	var buf [16]int
	for _, i := range r.Reaching(first, last, buf[:0]) {
		seg := r.Segments[i]
		if p.Live(r.First+i) && !killed(kp.Kills, p, seg.ID) &&
			reaches(seg.Pointers, first, last) {
			seg.Part = p.ID
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

// keyParts returns the parts of key, and false when the state does not hold key. It
// reads the core during replay, since the state publishes no Channel before Run.
func (e *engine) keyParts(key ChannelKey) (*state.Parts, bool) {
	if e.replaying {
		return e.state.ReplayParts(key)
	}
	ch := e.state.Channel(key)
	if ch == nil {
		return nil, false
	}
	return ch.Parts, true
}

// clockUsers returns the live part records of keys other than clock that name clock as
// their clock, and whose pointers reach [first, last]. Section 8 of a part tells
// whether its records need a read, and it pins the parts that it reads.
func (e *engine) clockUsers(
	clock ChannelKey,
	first, last telem.TimeStamp,
) ([]state.User, error) {
	var h held
	e.pins.mu.Lock()
	for _, p := range e.parts.Load().live.parts {
		if p.First <= last && first <= p.Last {
			h.parts = append(h.parts, p)
		}
	}
	e.pins.pin(h)
	e.pins.mu.Unlock()
	var users []state.User
	for _, p := range h.parts {
		timed, err := e.cache.Timed(p, clock)
		if err != nil {
			return nil, errors.Join(err, e.unpin(h))
		}
		for _, t := range timed {
			kp, ok := e.keyParts(t.Key)
			if !ok || kp == nil || !p.Live(t.Index) ||
				killed(
					kp.Kills,
					p,
					t.Segment.ID,
				) || !reaches(t.Segment.Pointers, first, last) {
				continue
			}
			seg := t.Segment
			seg.Part = p.ID
			users = append(users, state.User{Key: t.Key, Segment: seg})
		}
	}
	return users, e.unpin(h)
}

// pinReaching returns the parts of key and pins those of their range whose stamps
// reach [first, last], under the lock that the pins of a load take.
func (e *engine) pinReaching(
	key ChannelKey,
	first, last telem.TimeStamp,
	buf []checkpoint.Part,
) (*state.Parts, held, error) {
	e.pins.mu.Lock()
	defer e.pins.mu.Unlock()
	kp, ok := e.keyParts(key)
	if !ok || kp == nil {
		return nil, held{}, nil
	}
	parts, err := e.partsOf(key, kp)
	if err != nil {
		return kp, held{}, err
	}
	h := held{parts: reaching(parts, first, last, buf)}
	e.pins.pin(h)
	return kp, h, nil
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
