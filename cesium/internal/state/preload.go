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
	"cmp"
	"slices"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// ErrStale is the error of a step whose preload came from the parts of a head that the
// key no longer reads. The caller loads again and retries.
var ErrStale = errors.New("preload from the parts of an older head")

// Preload holds part records of a key that a caller loads through the part cache
// before a step of the owner that needs them, since a step does no I/O.
type Preload struct {
	Key uint32
	// Generation is the generation of the head whose parts hold the records.
	Generation uint64
	// Segments are the live records of the key in the parts of that Channel whose
	// pointers reach the time range of the step, each tagged with its part.
	Segments []Segment
	// Users are the live part records of other keys that name the key as their clock,
	// and whose pointers reach the time range of the step.
	Users []User
}

// User is a part record of a key that names another key as its clock.
type User struct {
	Key     uint32
	Segment Segment
}

// ReplayParts returns the parts of key as replay has left them, and false when the
// state does not hold key. Call it in replay mode, before Run.
func (s *State) ReplayParts(key uint32) (*Parts, bool) {
	ch := s.core.channels[key]
	if ch == nil {
		return nil, false
	}
	return ch.parts, true
}

// usersOf returns the users of the preload of key in pre, or none.
func usersOf(pre []Preload, key uint32) []User {
	if i, found := search(pre, key); found {
		return pre[i].Users
	}
	return nil
}

// preloadOf returns the records of the preload of ch in pre, which is in rising order
// of key. It returns ErrStale for a preload from the parts of a head other than the
// head of the core, and refuses a step with none when ch has a part range.
func (c *core) preloadOf(ch *chanState, pre []Preload) ([]Segment, error) {
	i, found := search(pre, ch.key)
	switch {
	case !found && ch.oldest == 0:
		return nil, nil
	case !found:
		return nil, refuse("no preload of key %d, which parts hold", ch.key)
	case pre[i].Generation != c.head:
		return nil, errors.Wrapf(ErrStale, "key %d reads the parts of head %d, not %d",
			ch.key, c.head, pre[i].Generation)
	}
	return pre[i].Segments, nil
}

// stored returns the first visible pointer of c that shares a stamp with [first,
// last], and false when there is none. A segment that a part holds counts through its
// record in pre, the live part records of c that reach the range.
func (c *chanState) stored(first, last telem.TimeStamp, pre []Segment) (Pointer, bool) {
	for _, s := range c.segs {
		if s.vis.Part != 0 {
			continue
		}
		if p, ok := overlap(s.vis.Pointers, first, last); ok {
			return p, true
		}
	}
	for _, s := range pre {
		if p, ok := overlap(s.Pointers, first, last); ok {
			return p, true
		}
	}
	return Pointer{}, false
}

type beginDropRequest struct {
	key  uint32
	pre  []Preload
	done chan error
}

func (r *beginDropRequest) fail(err error) { r.done <- err }

func (r *beginDropRequest) run(s *State) {
	if i, found := search(r.pre, r.key); found && r.pre[i].Generation != s.core.head {
		r.done <- errors.Wrapf(ErrStale, "key %d reads the parts of head %d, not %d",
			r.key, s.core.head, r.pre[i].Generation)
		return
	}
	s.core.dropUsers[r.key] = usersOf(r.pre, r.key)
	r.done <- nil
}

// BeginDrop holds the part records that name key as their clock, from the preload of
// key in pre, for the durable apply of the Drop of key that the caller logs next. It
// returns ErrStale when the preload is stale.
func (s *State) BeginDrop(key uint32, pre ...Preload) error {
	r := &beginDropRequest{key: key, pre: pre, done: make(chan error, 1)}
	if err := s.enqueue(r); err != nil {
		return err
	}
	return <-r.done
}

// carry hands seg, which a part now holds, to each step in progress that took its part
// records before seg was tagged: a delete or a Drop of the clock of seg, and the
// durable apply of a visible commit of the key of seg. Each would miss seg otherwise,
// since its checks skip tagged segments.
func (c *core) carry(seg *segState) {
	user := User{Key: seg.key, Segment: seg.vis}
	if seg.vis.Clock != seg.key {
		if ch := c.channels[seg.vis.Clock]; ch != nil {
			for _, cl := range ch.claims {
				if cl.w == nil && overlaps(seg.vis.Pointers, cl.start, cl.end) {
					cl.users = append(cl.users, user)
				}
			}
		}
		if users, ok := c.dropUsers[seg.vis.Clock]; ok {
			c.dropUsers[seg.vis.Clock] = append(users, user)
		}
	}
	for _, s := range c.channels[seg.key].segs {
		if s != seg && s.dur.length < s.vis.Length {
			s.pre = union(s.pre, []Segment{seg.vis})
		}
	}
}

// forgetTagged takes each tagged segment of ch out of the state, with no file to
// unlink,
// and reports whether any was. It keeps a segment that a delete in progress put back
// for the cut of its Tombstone.
func (c *core) forgetTagged(ch *chanState) bool {
	cutting := func(s *segState) bool {
		return slices.ContainsFunc(ch.claims, func(cl *claim) bool {
			return cl.w == nil && overlaps(s.vis.Pointers, cl.start, cl.end)
		})
	}
	kept := ch.segs[:0]
	for _, s := range ch.segs {
		if s.vis.Part != 0 && !cutting(s) {
			delete(c.segments, s.vis.ID)
			continue
		}
		kept = append(kept, s)
	}
	forgot := len(kept) < len(ch.segs)
	clear(ch.segs[len(kept):])
	ch.segs = kept
	return forgot
}

// overlaps reports whether a pointer of ptrs shares a stamp with [first, last].
func overlaps(ptrs []Pointer, first, last telem.TimeStamp) bool {
	_, ok := overlap(ptrs, first, last)
	return ok
}

// search returns the index of the preload of key in pre, which is in rising order of
// key, and whether pre holds one.
func search(pre []Preload, key uint32) (int, bool) {
	return slices.BinarySearchFunc(pre, key, func(p Preload, key uint32) int {
		return cmp.Compare(p.Key, key)
	})
}

// recordsOf returns the records of the preload of key in pre, or none.
func recordsOf(pre []Preload, key uint32) []Segment {
	if i, found := search(pre, key); found {
		return pre[i].Segments
	}
	return nil
}

// union returns segs with each record of more whose segment segs lacks.
func union(segs, more []Segment) []Segment {
	for _, rec := range more {
		if !slices.ContainsFunc(segs, func(s Segment) bool { return s.ID == rec.ID }) {
			segs = append(segs, rec)
		}
	}
	return segs
}

// putBack puts each record of records, which parts of key hold, back into the state as
// a tagged segment, so a cut can change it. A tagged segment that the state still holds
// takes its record, and a segment that the state holds untagged keeps its own.
func (c *core) putBack(key uint32, records []Segment) {
	ch := c.channels[key]
	for _, rec := range records {
		seg := c.segments[rec.ID]
		switch {
		case seg == nil:
			seg = &segState{key: key}
			c.segments[rec.ID] = seg
			ch.insert(seg)
		case seg.vis.Part == 0:
			continue
		}
		ptrs := slices.Clone(rec.Pointers)
		seg.vis, seg.vis.Pointers = rec, ptrs
		seg.dur = durable{
			created:  true,
			sealed:   rec.Sealed,
			length:   rec.Length,
			tailCRC:  rec.TailCRC,
			pointers: ptrs,
		}
	}
	if len(records) > 0 {
		ch.publish()
	}
}

// overlap returns the first pointer of ptrs that shares a stamp with [first, last]. It
// reads every pointer, since a backfill leaves the pointers of a segment out of time
// order.
func overlap(ptrs []Pointer, first, last telem.TimeStamp) (Pointer, bool) {
	for _, p := range ptrs {
		if p.First <= last && first <= p.Last {
			return p, true
		}
	}
	return Pointer{}, false
}
