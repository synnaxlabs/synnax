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
// of another key on the key's clock lies in the range, and each cut names the live
// pointer that holds an end of the range, inside it.
func (c *core) checkTombstone(t *log.Tombstone, users []User) error {
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
	if err := c.checkClock(t, users); err != nil {
		return err
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

// tombstone applies t to both views, which a durable Tombstone changes at once, and
// buries each sealed segment that no live pointer is left in.
func (c *core) tombstone(t *log.Tombstone, pre []Preload) error {
	users := usersOf(pre, t.Key)
	if ch := c.channels[t.Key]; ch != nil {
		c.putBack(t.Key, recordsOf(pre, t.Key))
		if users == nil {
			users = ch.deleteUsers(t)
		}
	}
	if err := c.checkTombstone(t, users); err != nil {
		return err
	}
	ch := c.channels[t.Key]
	for _, s := range slices.Clone(ch.segs) {
		before := s.dur.pointers
		s.vis.Pointers = cut(s.vis.Pointers, t)
		s.dur.pointers = cut(s.dur.pointers, t)
		// A part may hold the record of a sealed segment before the cut. Replay keeps
		// only the durable pointers.
		if !slices.Equal(before, s.dur.pointers) {
			s.vis.Part = 0
			if s.dur.sealed {
				ch.kills = append(ch.kills, Kill{
					Key:        t.Key,
					Segment:    s.vis.ID,
					Generation: c.generation,
				})
				ch.repart()
			}
			c.test(s)
		}
		c.bury(s)
	}
	ch.publish()
	return nil
}

// checkClock refuses t when a segment of another key that names the key of t as its
// clock holds data in the range of t. A segment that a part holds counts through its
// record in users.
func (c *core) checkClock(t *log.Tombstone, users []User) error {
	start, end := telem.TimeStamp(t.Start), telem.TimeStamp(t.End)
	uses := func(key uint32, ptrs []Pointer) error {
		for _, p := range ptrs {
			if p.First < end && p.Last >= start {
				return refuse("tombstone of clock %d from %d to %d, which key %d uses "+
					"from %d to %d", t.Key, start, end, key, p.First, p.Last)
			}
		}
		return nil
	}
	for _, s := range c.segments {
		if s.key == t.Key || s.vis.Clock != t.Key || !s.dur.created || s.vis.Part != 0 {
			continue
		}
		if err := uses(s.key, s.dur.pointers); err != nil {
			return err
		}
	}
	for _, u := range users {
		// The records of a key that a Drop took out are dead.
		if c.channels[u.Key] == nil {
			continue
		}
		if err := uses(u.Key, u.Segment.Pointers); err != nil {
			return err
		}
	}
	return nil
}

// deleteUsers returns the users that the delete claim of the range of t holds.
func (c *chanState) deleteUsers(t *log.Tombstone) []User {
	for _, cl := range c.claims {
		if cl.w == nil && cl.start == telem.TimeStamp(t.Start) &&
			cl.end == telem.TimeStamp(t.End)-1 {
			return cl.users
		}
	}
	return nil
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
	// pre holds the preload of the key, in preload, or none.
	pre     []Preload
	preload [1]Preload
	// Left and Right are the durable pointers that hold the start and the end of the
	// range, or nil when no pointer holds that end.
	Left, Right *Edge
	// Empty is true when no durable pointer of the key lies in the range.
	Empty bool
}

type beginDeleteRequest struct {
	d          *Delete
	start, end telem.TimeStamp
}

func (r *beginDeleteRequest) fail(err error) { r.d.done <- err }

func (r *beginDeleteRequest) run(s *State) {
	if s.deferred(r, r.d.key) {
		return
	}
	ch := s.core.channels[r.d.key]
	if ch == nil {
		r.d.Empty = true
		r.d.done <- nil
		return
	}
	last := r.end - 1
	for _, cl := range ch.claims {
		if cl.reaches(r.start, last) {
			r.d.done <- errors.Wrapf(ErrConflict, "delete of key %d from %d to %d "+
				"reaches the claim of %s from %d", r.d.key, r.start, r.end, cl.owner(),
				cl.start)
			return
		}
	}
	records, err := s.core.preloadOf(ch, r.d.pre)
	if err != nil {
		r.d.done <- err
		return
	}
	r.d.Empty = true
	// A segment that a part holds counts through its record in the preload.
	for _, seg := range ch.segs {
		if seg.vis.Part == 0 {
			r.edges(&seg.vis, seg.dur.pointers, last)
		}
	}
	for i := range records {
		r.edges(&records[i], records[i].Pointers, last)
	}
	// The Tombstone cuts the records that the delete reaches as head segments.
	s.core.putBack(r.d.key, records)
	r.d.c = &claim{
		start:   r.start,
		end:     last,
		bounded: true,
		users:   usersOf(r.d.pre, r.d.key),
	}
	ch.claims = append(ch.claims, r.d.c)
	r.d.done <- nil
}

// edges notes on the delete of r each pointer of ptrs, of segment seg, in the range up
// to last, and the pointers that hold the ends of the range.
func (r *beginDeleteRequest) edges(seg *Segment, ptrs []Pointer, last telem.TimeStamp) {
	for _, p := range ptrs {
		if p.First > last || p.Last < r.start {
			continue
		}
		r.d.Empty = false
		if p.First < r.start {
			r.d.Left = &Edge{Segment: *seg, Pointer: p}
		}
		if p.Last >= r.end {
			r.d.Right = &Edge{Segment: *seg, Pointer: p}
		}
	}
}

// BeginDelete claims the time range from start to end of key for a delete, and returns
// the durable pointers at its ends, from which the caller builds the cuts of a
// Tombstone. pre is the preload of key over the range. Until Release, a commit or a
// writer start that reaches the range gets an error that wraps ErrConflict, so the
// durable view of the range does not change. It returns an error that wraps ErrConflict
// when the claim of a writer or another delete reaches the range, and ErrStale when pre
// is stale. It waits for a Replace in flight of a segment of key or one that key times.
func (s *State) BeginDelete(
	key uint32,
	start, end telem.TimeStamp,
	pre ...Preload,
) (*Delete, error) {
	d := &Delete{s: s, key: key, done: make(chan error, 1)}
	if i, found := search(pre, key); found {
		d.preload[0] = pre[i]
		d.pre = d.preload[:]
	}
	r := &beginDeleteRequest{d: d, start: start, end: end}
	if err := s.enqueue(r); err != nil {
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

func (r *checkDeleteRequest) run(s *State) {
	r.d.done <- s.core.checkTombstone(r.t, r.d.c.users)
}

// Check checks t with the rule that a durable Tombstone must pass, so the caller logs
// only a Tombstone that the state applies. It returns an error that wraps ErrRefused
// for a rule that t breaks.
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
