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
	"sync/atomic"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/invariants"
	"github.com/synnaxlabs/x/telem"
)

// segEntry is the state of one segment. vis is what readers see, and dur is what a
// durable log record covers. Visible data that is not durable is always a suffix of
// vis.
type segEntry struct {
	key uint32
	vis Segment
	// visSpan is the span of vis.Pointers.
	visSpan span
	dur     durable
	// holder is the writer that holds the open segment, or nil.
	holder *Writer
}

// durable is the part of a segment that the log covers.
type durable struct {
	created bool
	sealed  bool
	length  uint32
	tailCRC uint32
	// pointers belong to the owner, which changes the last one in place. A sealed
	// segment whose visible data is all durable shares vis.Pointers.
	pointers []Pointer
	// span is the span of pointers.
	span span
}

// span is the time range from the least First to the greatest Last of a set of
// pointers.
type span struct{ first, last telem.TimeStamp }

// noSpan returns the span of no pointer, which overlaps nothing.
func noSpan() span { return span{first: telem.TimeStampMax, last: telem.TimeStampMin} }

func spanOf(ptrs []Pointer) span {
	s := noSpan()
	for _, p := range ptrs {
		s = s.cover(p)
	}
	return s
}

// cover returns s grown to cover p.
func (s span) cover(p Pointer) span {
	return s.join(span{first: p.First, last: p.Last})
}

func (s span) join(o span) span {
	return span{first: min(s.first, o.first), last: max(s.last, o.last)}
}

func (s span) overlaps(first, last telem.TimeStamp) bool {
	return s.first <= last && first <= s.last
}

// keyEntry is the state of one key: its segments, in rising order of identity, and the
// visible Channel that readers load.
type keyEntry struct {
	key  uint32
	segs []*segEntry
	// open holds the open segments and sealed the visible views of the others, each in
	// rising order of identity once replay ends. Published Channels share sealed: only
	// seal appends to it, and every other change makes a new array.
	open   []*segEntry
	sealed []Segment
	claims []*claim
	pub    atomic.Pointer[Channel]
	// visSpan and durSpan join the spans of every segment in each view.
	visSpan, durSpan span
}

func byID(o *segEntry, id uint64) int { return cmp.Compare(o.vis.ID, id) }

// insert adds s, which is open, to c in order of identity.
func (c *keyEntry) insert(s *segEntry) {
	i, _ := slices.BinarySearchFunc(c.segs, s.vis.ID, byID)
	c.segs = slices.Insert(c.segs, i, s)
	i, _ = slices.BinarySearchFunc(c.open, s.vis.ID, byID)
	c.open = slices.Insert(c.open, i, s)
}

// seal seals the visible view of s, and moves it from the open segments of c to the
// sealed ones. It appends to the sealed views when s comes after them, and makes a new
// array when it does not.
func (c *keyEntry) seal(s *segEntry) {
	s.vis.Sealed = true
	c.open = slices.DeleteFunc(c.open, func(o *segEntry) bool { return o == s })
	n := len(c.sealed)
	if n == 0 || c.sealed[n-1].ID < s.vis.ID {
		c.sealed = append(c.sealed, s.vis)
		return
	}
	i, _ := slices.BinarySearchFunc(
		c.sealed,
		s.vis.ID,
		func(o Segment, id uint64) int { return cmp.Compare(o.ID, id) },
	)
	c.sealed = slices.Concat(c.sealed[:i], []Segment{s.vis}, c.sealed[i:])
}

// remove takes s, which is open, out of c.
func (c *keyEntry) remove(s *segEntry) {
	c.segs = slices.DeleteFunc(c.segs, func(o *segEntry) bool { return o == s })
	c.open = slices.DeleteFunc(c.open, func(o *segEntry) bool { return o == s })
	c.respan()
}

// split sets the open segments and the sealed views of c from its segments, in new
// arrays.
func (c *keyEntry) split() {
	c.open, c.sealed = nil, nil
	for _, s := range c.segs {
		if s.vis.Sealed {
			c.sealed = append(c.sealed, s.vis)
		} else {
			c.open = append(c.open, s)
		}
	}
}

// respan sets the spans of c from the spans of its segments.
func (c *keyEntry) respan() {
	c.visSpan, c.durSpan = noSpan(), noSpan()
	for _, s := range c.segs {
		c.visSpan, c.durSpan = c.visSpan.join(s.visSpan), c.durSpan.join(s.dur.span)
	}
}

// checkViews panics when the views, spans, or open and sealed lists of c break an
// invariant. It runs only in a build with invariants.
func (c *keyEntry) checkViews() {
	if !invariants.Enabled {
		return
	}
	var (
		vis, dur []Segment
		opens    []*segEntry
		sealed   []Segment
	)
	visSpan, durSpan := noSpan(), noSpan()
	for _, s := range c.segs {
		if s.dur.length > s.vis.Length || len(s.dur.pointers) > len(s.vis.Pointers) {
			panic(errors.Newf("state: durable view of segment %d runs past its "+
				"visible view", s.vis.ID))
		}
		if s.visSpan != spanOf(s.vis.Pointers) || s.dur.span != spanOf(s.dur.pointers) {
			panic(errors.Newf("state: span of segment %d is stale", s.vis.ID))
		}
		visSpan, durSpan = visSpan.join(s.visSpan), durSpan.join(s.dur.span)
		if s.vis.Sealed {
			sealed = append(sealed, s.vis)
		} else {
			opens = append(opens, s)
		}
		vis = append(vis, s.vis)
		dur = append(dur, Segment{ID: s.vis.ID, Pointers: s.dur.pointers})
		for _, ptrs := range [][]Pointer{s.vis.Pointers, s.dur.pointers} {
			for i := 1; i < len(ptrs); i++ {
				if ptrs[i].Offset < ptrs[i-1].Offset+ptrs[i-1].Length {
					panic(errors.Newf("state: pointers of segment %d are out of "+
						"stream order", s.vis.ID))
				}
			}
		}
	}
	if c.visSpan != visSpan || c.durSpan != durSpan {
		panic(errors.Newf("state: span of key %d is stale", c.key))
	}
	if !slices.Equal(c.open, opens) ||
		!slices.EqualFunc(c.sealed, sealed, sameSegment) {
		panic(errors.Newf("state: open or sealed segments of key %d are stale", c.key))
	}
	for _, segs := range [][]Segment{vis, dur} {
		if err := checkApart(NewChannel(c.key, segs...)); err != nil {
			panic(err)
		}
	}
}

// sameSegment reports whether a and b hold the same fields and pointers.
func sameSegment(a, b Segment) bool {
	return a.ID == b.ID && a.Clock == b.Clock && a.Version == b.Version &&
		a.DataType == b.DataType && a.Sealed == b.Sealed && a.Length == b.Length &&
		a.TailCRC == b.TailCRC && slices.Equal(a.Pointers, b.Pointers)
}

// publish stores a new visible Channel of c for readers. The Channel shares the sealed
// views of c, and copies the open ones.
func (c *keyEntry) publish() {
	ch := &Channel{Key: c.key, sealed: c.sealed}
	ch.open = make([]Segment, 0, len(c.open))
	for _, s := range c.open {
		ch.open = append(ch.open, s.vis)
	}
	c.pub.Store(ch)
}

// overlaps returns the first pointer of c, in the durable view or the visible one, that
// shares a stamp with [first, last], and false when there is none. It reads no segment
// when the span of c misses, and every pointer of each segment whose span overlaps.
func (c *keyEntry) overlaps(
	first, last telem.TimeStamp,
	durableView bool,
) (Pointer, bool) {
	whole := c.visSpan
	if durableView {
		whole = c.durSpan
	}
	if !whole.overlaps(first, last) {
		return Pointer{}, false
	}
	for _, s := range c.segs {
		ptrs, sp := s.vis.Pointers, s.visSpan
		if durableView {
			ptrs, sp = s.dur.pointers, s.dur.span
		}
		if !sp.overlaps(first, last) {
			continue
		}
		for _, p := range ptrs {
			if p.First <= last && first <= p.Last {
				return p, true
			}
		}
	}
	return Pointer{}, false
}

// grow returns the pointer that a commit of samples at alignment from first to last
// over the stream range [from, to) leaves last in ptrs, and true when it extends the
// last pointer, which it does when the alignment and the range follow it.
func grow(
	ptrs []Pointer,
	alignment telem.Alignment,
	first, last telem.TimeStamp,
	from, to, samples uint32,
) (Pointer, bool) {
	if n := len(ptrs); n > 0 {
		p := ptrs[n-1]
		if p.Alignment.AddSamples(p.Samples) == alignment && p.Offset+p.Length == from {
			p.Last = last
			p.Length += to - from
			p.Samples += samples
			return p, true
		}
	}
	return Pointer{
		First:     first,
		Last:      last,
		Alignment: alignment,
		Offset:    from,
		Length:    to - from,
		Samples:   samples,
	}, false
}

// checkGrow checks the pointer that a commit leaves last in the pointers ptrs of
// segment id of key, which end at from: its stamps go on after the stretch that it
// extends, and its shape passes checkShape.
func checkGrow(
	id uint64,
	key uint32,
	ptrs []Pointer,
	c *log.Commit,
	from, to uint32,
) error {
	first := telem.TimeStamp(c.First)
	p, extended := grow(ptrs, telem.Alignment(c.Alignment), first,
		telem.TimeStamp(c.Last), from, to, c.Samples)
	if extended && first <= ptrs[len(ptrs)-1].Last {
		return refuse("commit of key %d at %d goes back inside its stretch, which "+
			"reaches %d", key, first, ptrs[len(ptrs)-1].Last)
	}
	return checkShape(id, key, p)
}

// checkShape checks that pointer p of segment id of key holds at least one byte for
// each sample, and stamps that do not go back.
func checkShape(id uint64, key uint32, p Pointer) error {
	if p.Samples == 0 || p.Length < p.Samples || p.First > p.Last ||
		(p.Samples == 1 && p.First != p.Last) {
		return refuse("segment %d of key %d has a pointer of %d samples in %d bytes "+
			"from %d to %d", id, key, p.Samples, p.Length, p.First, p.Last)
	}
	return nil
}

// extend returns ptrs with the pointer p that grow returned, which replaces the last
// pointer when extended is true. It changes the last pointer in place only when inPlace
// is true.
func extend(ptrs []Pointer, p Pointer, extended, inPlace bool) []Pointer {
	if extended {
		if !inPlace {
			ptrs = slices.Clone(ptrs)
		}
		ptrs[len(ptrs)-1] = p
		return ptrs
	}
	if !inPlace {
		return append(slices.Clip(ptrs), p)
	}
	return append(ptrs, p)
}
