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
	"slices"

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
// of another key on the key's clock keeps a sample at an alignment that t takes out of
// the clock, and each cut names a live pointer that the range reaches, inside it. A
// pointer that holds an end of the range by time must be the one that the cut of that
// end names.
func (c *core) checkTombstone(t *log.Tombstone) error {
	start, end := telem.TimeStamp(t.Start), telem.TimeStamp(t.End)
	ch := c.channels[t.Key]
	var left, right, leftEnd, rightEnd cutAt
	switch {
	case start >= end:
		return refuse("tombstone of key %d from %d to %d is empty", t.Key,
			start, end)
	case ch == nil:
		return refuse("tombstone of key %d, which holds no segment", t.Key)
	}
	gone := removed(ch, t)
	for _, s := range c.segments {
		if s.key == t.Key || s.vis.Clock != t.Key || !s.dur.created {
			continue
		}
		for _, p := range s.dur.pointers {
			if overlaps(p, gone) {
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
				leftEnd = cutAt{s, i}
			}
			if p.First < end && p.Last >= end {
				rightEnd = cutAt{s, i}
			}
			if p.First < end && p.Last >= start {
				if names(t.Left, s.vis.ID, p) {
					left = cutAt{s, i}
				}
				if names(t.Right, s.vis.ID, p) {
					right = cutAt{s, i}
				}
			}
		}
	}
	if err := checkCut(t.Key, "left", t.Left, leftEnd, left); err != nil {
		return err
	}
	if err := checkCut(t.Key, "right", t.Right, rightEnd, right); err != nil {
		return err
	}
	if l, r := t.Left, t.Right; l != nil && r != nil && left == right &&
		(l.Offset > r.Offset || l.Alignment > r.Alignment) {
		return refuse("tombstone of key %d cuts its right end before its "+
			"left end", t.Key)
	}
	return nil
}

// checkCut checks that cut names the pointer at, the pointer of the range that holds
// the alignment of cut, and lies inside it. end is the pointer that holds that end of
// the range by time, which cut must name.
func checkCut(key uint32, side string, cut *log.Cut, end, at cutAt) error {
	switch {
	case cut == nil && end.seg == nil:
		return nil
	case cut == nil:
		return refuse(
			"tombstone of key %d has no %s cut, but segment %d holds that end",
			key,
			side,
			end.seg.vis.ID,
		)
	case at.seg == nil || (end.seg != nil && end != at):
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

// names reports whether cut names p, a pointer of segment seg: whether the sample at
// its alignment lies in p.
func names(cut *log.Cut, seg uint64, p Pointer) bool {
	if cut == nil || cut.Segment != seg {
		return false
	}
	a, first := telem.Alignment(cut.Alignment), p.Alignment.SampleIndex()
	return a.DomainIndex() == p.Alignment.DomainIndex() && a.SampleIndex() >= first &&
		a.SampleIndex() < first+p.Samples
}

// span is the samples from alignment lo up to hi.
type span struct{ lo, hi telem.Alignment }

// removed returns the samples of ch, the channel of the key of t, that t takes out.
func removed(ch *chanState, t *log.Tombstone) []span {
	start, end := telem.TimeStamp(t.Start), telem.TimeStamp(t.End)
	var out []span
	for _, s := range ch.segs {
		for _, p := range s.dur.pointers {
			left, right := names(t.Left, s.vis.ID, p), names(t.Right, s.vis.ID, p)
			if !left && !right && (p.Last < start || p.First >= end) {
				continue
			}
			sp := span{lo: p.Alignment, hi: p.Alignment.AddSamples(p.Samples)}
			if left {
				sp.lo = telem.Alignment(t.Left.Alignment)
			}
			if right {
				sp.hi = telem.Alignment(t.Right.Alignment)
			}
			out = append(out, sp)
		}
	}
	return out
}

// overlaps reports whether p holds a sample in one of spans.
func overlaps(p Pointer, spans []span) bool {
	lo, hi := p.Alignment, p.Alignment.AddSamples(p.Samples)
	return slices.ContainsFunc(spans, func(sp span) bool {
		return lo < sp.hi && hi > sp.lo
	})
}

// tombstone applies t to both views, which a durable Tombstone changes at once, and
// buries each sealed segment that no live pointer is left in.
func (c *core) tombstone(t *log.Tombstone) error {
	if err := c.checkTombstone(t); err != nil {
		return err
	}
	ch := c.channels[t.Key]
	for _, s := range slices.Clone(ch.segs) {
		s.vis.Pointers = cut(s.vis.ID, s.vis.Pointers, t)
		s.dur.pointers = cut(s.vis.ID, s.dur.pointers, t)
		c.bury(s)
	}
	ch.publish()
	return nil
}

// cut returns a new slice of ptrs, the pointers of segment seg, with the samples of t's
// range taken out. A pointer that a cut names keeps its samples on the far side of the
// cut. An open writer's claim keeps a delete from the range of its visible commits, so
// the same cut applies to the visible and the durable view.
func cut(seg uint64, ptrs []Pointer, t *log.Tombstone) []Pointer {
	start, end := telem.TimeStamp(t.Start), telem.TimeStamp(t.End)
	out := make([]Pointer, 0, len(ptrs)+1)
	for _, p := range ptrs {
		left, right := names(t.Left, seg, p), names(t.Right, seg, p)
		if !left && !right && (p.Last < start || p.First >= end) {
			out = append(out, p)
			continue
		}
		if left {
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
		if right {
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

// Edge is the durable pointer that holds an end of a deleted range, and its segment.
type Edge struct {
	// Segment is the segment of the pointer, as readers see it.
	Segment Segment
	// Pointer is the durable pointer.
	Pointer Pointer
}

// Delete is a delete of a time range of one key that the state holds a claim for. Its
// calls block until the owner runs them. A Delete is not safe for concurrent use.
type Delete struct {
	s    *State
	c    *claim
	key  uint32
	done chan error
	// Left and Right are the durable pointers that hold the start and the end of the
	// range, or nil when no pointer holds that end.
	Left, Right *Edge
	// Reached holds each durable pointer of the key that the range reaches, so a caller
	// can cut one by alignment.
	Reached []Edge
	// Empty is true when no durable pointer of the key lies in the range.
	Empty bool
}

type beginDeleteRequest struct {
	d          *Delete
	start, end telem.TimeStamp
}

func (r *beginDeleteRequest) fail(err error) { r.d.done <- err }

func (r *beginDeleteRequest) run(s *State) {
	ch := s.core.channels[r.d.key]
	if ch == nil {
		r.d.Empty = true
		r.d.done <- nil
		return
	}
	last := r.end - 1
	for _, cl := range ch.claims {
		if cl.reaches(r.start, last) {
			r.d.done <- errors.Wrapf(errConflict, "delete of key %d from %d to %d "+
				"reaches the claim of %s from %d", r.d.key, r.start, r.end, cl.owner(),
				cl.start)
			return
		}
	}
	r.d.Empty = true
	for _, seg := range ch.segs {
		for _, p := range seg.dur.pointers {
			if p.First > last || p.Last < r.start {
				continue
			}
			r.d.Empty = false
			r.d.Reached = append(r.d.Reached, Edge{Segment: seg.vis, Pointer: p})
			if p.First < r.start {
				r.d.Left = &Edge{Segment: seg.vis, Pointer: p}
			}
			if p.Last >= r.end {
				r.d.Right = &Edge{Segment: seg.vis, Pointer: p}
			}
		}
	}
	r.d.c = &claim{start: r.start, end: last, bounded: true}
	ch.claims = append(ch.claims, r.d.c)
	r.d.done <- nil
}

// BeginDelete claims the time range from start to end of key for a delete, and returns
// the durable pointers at its ends, from which the caller builds the cuts of a
// Tombstone. Until Release, a commit or a writer start that reaches the range gets an
// error that wraps validate.ErrValidation, so the durable view of the range does not
// change. It returns an error that wraps validate.ErrValidation when the claim of a
// writer or another delete reaches the range.
func (s *State) BeginDelete(key uint32, start, end telem.TimeStamp) (*Delete, error) {
	d := &Delete{s: s, key: key, done: make(chan error, 1)}
	if err := s.enqueue(&beginDeleteRequest{d: d, start: start, end: end}); err != nil {
		return nil, err
	}
	if err := wait(s, d.done); err != nil {
		return nil, err
	}
	return d, nil
}

type checkDeleteRequest struct {
	d *Delete
	t *log.Tombstone
}

func (r *checkDeleteRequest) fail(err error) { r.d.done <- err }

func (r *checkDeleteRequest) run(s *State) { r.d.done <- s.core.checkTombstone(r.t) }

// Check checks t with the rule that a durable Tombstone must pass, so the caller logs
// only a Tombstone that the state applies. It returns an error for a rule that t
// breaks.
func (d *Delete) Check(t *log.Tombstone) error {
	if err := d.s.enqueue(&checkDeleteRequest{d: d, t: t}); err != nil {
		return err
	}
	return wait(d.s, d.done)
}

type releaseDeleteRequest struct{ d *Delete }

func (r *releaseDeleteRequest) fail(err error) { r.d.done <- err }

func (r *releaseDeleteRequest) run(s *State) {
	if ch := s.core.channels[r.d.key]; ch != nil && r.d.c != nil {
		ch.claims = slices.DeleteFunc(
			ch.claims,
			func(c *claim) bool { return c == r.d.c },
		)
	}
	r.d.done <- nil
}

// Release drops the claim of d. The caller releases it after the ack of the Tombstone,
// which comes after the state applies it.
func (d *Delete) Release() error {
	if err := d.s.enqueue(&releaseDeleteRequest{d: d}); err != nil {
		return err
	}
	return wait(d.s, d.done)
}
