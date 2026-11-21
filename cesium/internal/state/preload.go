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
	// Generation is the Generation of the Channel that the caller loaded the records
	// for.
	Generation uint64
	// Segments are the live records of the key in the parts of that Channel whose
	// pointers reach the time range of the step, each tagged with its part.
	Segments []Segment
}

// preloadOf returns the records of the preload of c in pre, which is in rising order of
// key. It returns ErrStale for a preload from the parts of a head that c no longer
// reads, and refuses a step with none when c has a part range.
func (c *chanState) preloadOf(pre []Preload) ([]Segment, error) {
	i, found := search(pre, c.key)
	switch {
	case !found && c.oldest == 0:
		return nil, nil
	case !found:
		return nil, refuse("no preload of key %d, which parts hold", c.key)
	case pre[i].Generation != c.generation:
		return nil, errors.Wrapf(ErrStale, "key %d reads the parts of head %d, not %d",
			c.key, c.generation, pre[i].Generation)
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
