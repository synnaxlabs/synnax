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
	"sync/atomic"

	"github.com/synnaxlabs/x/telem"
)

// segState is the state of one segment. vis is what readers see, and dur is what a
// durable log record covers. Visible data that is not durable is always a suffix of
// vis.
type segState struct {
	key uint32
	vis Segment
	dur durable
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
}

// chanState is the state of one key: its segments, in rising order of identity, and
// the visible Channel that readers load.
type chanState struct {
	key    uint32
	segs   []*segState
	claims []*claim
	pub    atomic.Pointer[Channel]
}

// insert adds s to c in order of identity.
func (c *chanState) insert(s *segState) {
	i, _ := slices.BinarySearchFunc(c.segs, s.vis.ID, func(o *segState, id uint64) int {
		switch {
		case o.vis.ID < id:
			return -1
		case o.vis.ID > id:
			return 1
		}
		return 0
	})
	c.segs = slices.Insert(c.segs, i, s)
}

// remove takes the segment with identity id out of c.
func (c *chanState) remove(id uint64) {
	c.segs = slices.DeleteFunc(c.segs, func(s *segState) bool { return s.vis.ID == id })
}

// publish stores a new visible Channel of c for readers.
func (c *chanState) publish() {
	ch := &Channel{Key: c.key, Segments: make([]Segment, 0, len(c.segs))}
	for _, s := range c.segs {
		ch.Segments = append(ch.Segments, s.vis)
	}
	c.pub.Store(ch)
}

// overlaps returns the first pointer of c, in the durable view or the visible one, that
// shares a stamp with [first, last], and false when there is none.
func (c *chanState) overlaps(
	first, last telem.TimeStamp,
	durableView bool,
) (Pointer, bool) {
	for _, s := range c.segs {
		ptrs := s.vis.Pointers
		if durableView {
			ptrs = s.dur.pointers
		}
		i, _ := slices.BinarySearchFunc(
			ptrs,
			first,
			func(p Pointer, t telem.TimeStamp) int {
				if p.Last < t {
					return -1
				}
				return 1
			},
		)
		if i < len(ptrs) && ptrs[i].First <= last {
			return ptrs[i], true
		}
	}
	return Pointer{}, false
}

// extend returns ptrs with a commit of samples at alignment from first to last over
// the stream range [from, to), which extends the last pointer when the alignment
// follows it. It changes the last pointer in place only when inPlace is true.
func extend(
	ptrs []Pointer,
	alignment telem.Alignment,
	first, last telem.TimeStamp,
	from, to, samples uint32,
	inPlace bool,
) []Pointer {
	if n := len(ptrs); n > 0 {
		p := ptrs[n-1]
		if p.Alignment.AddSamples(p.Samples) == alignment && p.Offset+p.Length == from {
			p.Last = last
			p.Length += to - from
			p.Samples += samples
			if !inPlace {
				ptrs = slices.Clone(ptrs)
			}
			ptrs[n-1] = p
			return ptrs
		}
	}
	next := Pointer{
		First:     first,
		Last:      last,
		Alignment: alignment,
		Offset:    from,
		Length:    to - from,
		Samples:   samples,
	}
	if !inPlace {
		return append(slices.Clip(ptrs), next)
	}
	return append(ptrs, next)
}
