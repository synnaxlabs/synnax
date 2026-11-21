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
	// pointers reach the time range of the step.
	Segments []Segment
}

// preloadOf returns the records of the preload of c in pre, which is in rising order of
// key. It returns ErrStale for a preload from the parts of a head that c no longer
// reads, and refuses a step with none when c has a part range.
func (c *chanState) preloadOf(pre []Preload) ([]Segment, error) {
	i, found := slices.BinarySearchFunc(pre, c.key, func(p Preload, key uint32) int {
		return cmp.Compare(p.Key, key)
	})
	switch {
	case !found && c.oldest == 0:
		return nil, nil
	case !found:
		return nil, refuse("no preload of key %d, which parts hold", c.key)
	case pre[i].Generation != c.partsGeneration():
		return nil, errors.Wrapf(ErrStale, "key %d reads the parts of head %d, not %d",
			c.key, c.partsGeneration(), pre[i].Generation)
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
