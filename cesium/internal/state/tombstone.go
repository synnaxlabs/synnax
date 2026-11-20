// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package state

import (
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// cutAt is a pointer that a Tombstone cuts: its segment and its index there.
type cutAt struct {
	seg *segState
	i   int
}

// checkTombstone checks t against the durable view: the key is live, no live pointer
// of another key on the key's clock lies in the range, and each cut names the live
// pointer that holds an end of the range, inside it.
func (c *core) checkTombstone(t *log.Tombstone) error {
	start, end := telem.TimeStamp(t.Start), telem.TimeStamp(t.End)
	ch := c.channels[t.Key]
	var left, right cutAt
	switch {
	case start >= end:
		return refuse("tombstone of key %d from %d to %d is empty", t.Key,
			start, end)
	case ch == nil:
		return refuse("tombstone of key %d, which holds no segment", t.Key)
	}
	for _, s := range c.segments {
		if s.key == t.Key || s.vis.Clock != t.Key || !s.dur.created {
			continue
		}
		for _, p := range s.dur.pointers {
			if p.First < end && p.Last >= start {
				return refuse(
					"tombstone of clock %d from %d to %d, which "+
						"key %d uses from %d to %d",
					t.Key,
					start,
					end,
					s.key,
					p.First,
					p.Last,
				)
			}
		}
	}
	for _, s := range ch.segs {
		for i, p := range s.dur.pointers {
			if p.First < start && p.Last >= start {
				left = cutAt{s, i}
			}
			if p.First < end && p.Last >= end {
				right = cutAt{s, i}
			}
		}
	}
	if err := checkCut(t.Key, "left", t.Left, left); err != nil {
		return err
	}
	if err := checkCut(t.Key, "right", t.Right, right); err != nil {
		return err
	}
	if l, r := t.Left, t.Right; l != nil && r != nil && left == right &&
		(l.Offset > r.Offset || l.Alignment > r.Alignment) {
		return refuse("tombstone of key %d cuts its right end before its "+
			"left end", t.Key)
	}
	return nil
}

// checkCut checks that cut names the pointer at, which holds an end of the range, and
// lies inside it.
func checkCut(key uint32, side string, cut *log.Cut, at cutAt) error {
	switch {
	case cut == nil && at.seg == nil:
		return nil
	case cut == nil:
		return refuse(
			"tombstone of key %d has no %s cut, but segment %d holds that end",
			key,
			side,
			at.seg.vis.ID,
		)
	case at.seg == nil || at.seg.vis.ID != cut.Segment:
		return refuse("tombstone of key %d has a stale %s cut in segment %d", key, side,
			cut.Segment)
	}
	p := at.seg.dur.pointers[at.i]
	a := telem.Alignment(cut.Alignment)
	first, last := p.Alignment.SampleIndex(), p.Alignment.SampleIndex()+p.Samples-1
	stamp := telem.TimeStamp(cut.Stamp)
	if a.DomainIndex() != p.Alignment.DomainIndex() || a.SampleIndex() <= first ||
		a.SampleIndex() > last || cut.Offset <= p.Offset ||
		cut.Offset >= p.Offset+p.Length || stamp < p.First || stamp > p.Last {
		return refuse("tombstone of key %d has a %s cut at offset %d and alignment %d "+
			"outside its pointer from %d to %d", key, side, cut.Offset, uint64(a),
			p.First, p.Last)
	}
	return nil
}

// tombstone applies t to both views, which a durable Tombstone changes at once. It
// removes a sealed segment that no live pointer is left in, and returns it.
func (c *core) tombstone(t *log.Tombstone) ([]uint64, error) {
	if err := c.checkTombstone(t); err != nil {
		return nil, err
	}
	ch := c.channels[t.Key]
	var dead []uint64
	for _, s := range ch.segs {
		s.vis.Pointers = cut(s.vis.Pointers, t)
		s.dur.pointers = cut(s.dur.pointers, t)
		if s.vis.Sealed && s.dur.sealed && len(s.vis.Pointers) == 0 {
			dead = append(dead, s.vis.ID)
		}
	}
	for _, id := range dead {
		ch.remove(id)
		delete(c.segments, id)
	}
	ch.publish()
	return dead, nil
}

// cut returns a new slice of ptrs with the samples of t's range taken out. An open
// writer's claim keeps a delete from the range of its visible commits, so the same cut
// applies to the visible and the durable view.
func cut(ptrs []Pointer, t *log.Tombstone) []Pointer {
	start, end := telem.TimeStamp(t.Start), telem.TimeStamp(t.End)
	out := make([]Pointer, 0, len(ptrs)+1)
	for _, p := range ptrs {
		if p.Last < start || p.First >= end {
			out = append(out, p)
			continue
		}
		if p.First < start {
			l := t.Left
			kept := telem.Alignment(l.Alignment).
				SampleIndex() -
				p.Alignment.SampleIndex()
			out = append(out, Pointer{
				First:     p.First,
				Last:      telem.TimeStamp(l.Stamp),
				Alignment: p.Alignment,
				Offset:    p.Offset,
				Length:    l.Offset - p.Offset,
				Samples:   kept,
			})
		}
		if p.Last >= end {
			r := t.Right
			gone := telem.Alignment(r.Alignment).
				SampleIndex() -
				p.Alignment.SampleIndex()
			out = append(out, Pointer{
				First:     telem.TimeStamp(r.Stamp),
				Last:      p.Last,
				Alignment: telem.Alignment(r.Alignment),
				Offset:    r.Offset,
				Length:    p.Offset + p.Length - r.Offset,
				Samples:   p.Samples - gone,
			})
		}
	}
	return out
}

type tombstoneRequest struct {
	t    *log.Tombstone
	dead []uint64
	take bool
	done chan error
}

func (r *tombstoneRequest) run(s *State) {
	if r.take {
		r.dead, s.core.dead = s.core.dead, nil
		r.done <- nil
		return
	}
	if err := s.core.checkTombstone(r.t); err != nil {
		r.done <- err
		return
	}
	start, end := telem.TimeStamp(r.t.Start), telem.TimeStamp(r.t.End)
	for _, cl := range s.core.channels[r.t.Key].claims {
		if cl.reaches(start, end-1) {
			r.done <- errors.Wrapf(ErrConflict, "tombstone of key %d from %d to %d "+
				"reaches the claim of a writer from %d", r.t.Key, start, end, cl.start)
			return
		}
	}
	r.done <- nil
}

// CheckTombstone checks t with the rule that a durable Tombstone must pass, and against
// the claims of open writers, so the caller logs only a Tombstone that the state
// applies. It returns an error that wraps ErrRefused for a rule that t breaks, and
// ErrConflict when a writer's claim or pin reaches its range.
func (s *State) CheckTombstone(t *log.Tombstone) error {
	r := &tombstoneRequest{t: t, done: make(chan error, 1)}
	if err := s.enqueue(r); err != nil {
		return err
	}
	return wait(s, r.done)
}

// TakeDead returns the sealed segments that durable Tombstones left with no live
// pointer since the last call, which the database unlinks once no reader pins them.
func (s *State) TakeDead() ([]uint64, error) {
	r := &tombstoneRequest{take: true, done: make(chan error, 1)}
	if err := s.enqueue(r); err != nil {
		return nil, err
	}
	if err := wait(s, r.done); err != nil {
		return nil, err
	}
	return r.dead, nil
}
