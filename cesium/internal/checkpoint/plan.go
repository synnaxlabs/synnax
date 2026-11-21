// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint

import (
	"cmp"
	"slices"
	"sort"

	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
)

// plan is what a switch writes: the parts that it keeps, the parts that it writes, and
// the channels of the head.
type plan struct {
	index  *Index
	writes []partWrite
	head   []*state.Channel
	// err is the error of each part that the switch keeps because a read of it for its
	// rewrite failed.
	err error
}

type partWrite struct {
	id       uint64
	channels []*state.Channel
}

// plan diffs s, the whole state at a switch, against the index of the last head. A part
// record that s no longer holds is dead. A part with half its records dead, or with a
// record whose segment s holds cut, is written again under a new identity. Sealed
// segments that no part holds go to a new part once their records pass the part size.
// The head holds the rest. Both s and each part keep their records in rising order of
// key, then identity, so plan walks them side by side.
func (c *Checkpointer) plan(s *state.Snapshot) plan {
	pl := plan{index: &Index{limit: max(c.index.limit, s.Limits.Parts, 1)}}
	f := flatten(s)
	for _, p := range c.index.parts {
		q := p
		q.Dead = slices.Clone(p.Dead)
		at := f.match(&q)
		dead, cut := q.deadCount(), false
		for i, j := range at {
			if j >= 0 && !sameSegment(f.segment(j), q.segs[i]) {
				cut = true
			}
		}
		switch {
		case dead == len(q.segs):
		case cut:
			pl.add(f.channels(at, q.keys))
			f.part(at)
		case 2*dead >= len(q.segs):
			// A part that fails a read for its rewrite stays, with its dead marks.
			if live, err := c.live(q); err != nil {
				pl.err = errors.Join(pl.err, err)
				pl.index.parts = append(pl.index.parts, q)
			} else {
				pl.add(live)
			}
			f.part(at)
		default:
			pl.index.parts = append(pl.index.parts, q)
			f.part(at)
		}
	}
	if f.sealedSize() > c.cfg.PartSize {
		pl.add(f.keep(true))
		f.part(f.sealedAt())
	}
	pl.head = f.keep(false)
	return pl
}

// live returns the records of p that are not dead, read through the cache.
func (c *Checkpointer) live(p indexed) ([]*state.Channel, error) {
	var out []*state.Channel
	for i := 0; i < len(p.keys); {
		key := p.keys[i]
		r, err := c.cfg.Cache.Records(p.Part, key)
		if err != nil {
			return nil, err
		}
		var segs []state.Segment
		for j, seg := range r.Segments {
			if !p.dead(r.First + j) {
				segs = append(segs, seg)
			}
		}
		if len(segs) > 0 {
			out = append(out, &state.Channel{Key: key, Segments: segs})
		}
		for i < len(p.keys) && p.keys[i] == key {
			i++
		}
	}
	return out, nil
}

// add queues a new part of channels.
func (pl *plan) add(channels []*state.Channel) {
	pl.writes = append(pl.writes, partWrite{id: pl.index.limit, channels: channels})
	pl.index.limit++
}

// flat numbers the segments of a state in record order, and marks the ones that a part
// holds.
type flat struct {
	s *state.Snapshot
	// first holds the number of the first segment of each channel.
	first  []int
	parted []bool
}

func flatten(s *state.Snapshot) *flat {
	f := &flat{s: s, first: make([]int, len(s.Channels)+1)}
	for i, ch := range s.Channels {
		f.first[i+1] = f.first[i] + len(ch.Segments)
	}
	f.parted = make([]bool, f.first[len(s.Channels)])
	return f
}

// segment returns segment number j.
func (f *flat) segment(j int) state.Segment {
	i := sort.SearchInts(f.first, j+1) - 1
	return f.s.Channels[i].Segments[j-f.first[i]]
}

// match returns the number of the segment of each record of p, or -1 for a dead record,
// and kills each live record whose segment the state no longer holds.
func (f *flat) match(p *indexed) []int {
	at := make([]int, len(p.segs))
	ci := 0
	for i, seg := range p.segs {
		at[i] = -1
		if p.dead(i) {
			continue
		}
		for ci < len(f.s.Channels) && f.s.Channels[ci].Key < p.keys[i] {
			ci++
		}
		if ci == len(f.s.Channels) || f.s.Channels[ci].Key != p.keys[i] {
			p.kill(i)
			continue
		}
		segs := f.s.Channels[ci].Segments
		k, ok := slices.BinarySearchFunc(segs, seg.ID, compareID)
		if !ok {
			p.kill(i)
			continue
		}
		at[i] = f.first[ci] + k
	}
	return at
}

func compareID(s state.Segment, id uint64) int { return cmp.Compare(s.ID, id) }

// part marks the segments at as held by a part.
func (f *flat) part(at []int) {
	for _, j := range at {
		if j >= 0 {
			f.parted[j] = true
		}
	}
}

// channels returns the segments at, with the key of each in keys, as channels.
func (f *flat) channels(at []int, keys []uint32) []*state.Channel {
	var out []*state.Channel
	for i, j := range at {
		if j < 0 {
			continue
		}
		if len(out) == 0 || out[len(out)-1].Key != keys[i] {
			out = append(out, &state.Channel{Key: keys[i]})
		}
		c := out[len(out)-1]
		c.Segments = append(c.Segments, f.segment(j))
	}
	return out
}

// keep returns the segments that no part holds, the sealed ones alone when sealed is
// true, as channels. A channel that keeps every segment is the state's channel, and the
// rest come from one slab of channels and one of segments.
func (f *flat) keep(sealed bool) []*state.Channel {
	want := func(j int, seg state.Segment) bool {
		return !f.parted[j] && (seg.Sealed || !sealed)
	}
	counts := make([]int, len(f.s.Channels))
	var emitted, cut, cutSegs int
	for i, ch := range f.s.Channels {
		for k, seg := range ch.Segments {
			if want(f.first[i]+k, seg) {
				counts[i]++
			}
		}
		if counts[i] > 0 {
			emitted++
		}
		if counts[i] > 0 && counts[i] < len(ch.Segments) {
			cut, cutSegs = cut+1, cutSegs+counts[i]
		}
	}
	out := make([]*state.Channel, 0, emitted)
	channels, segs := make([]state.Channel, cut), make([]state.Segment, 0, cutSegs)
	for i, ch := range f.s.Channels {
		switch counts[i] {
		case 0:
			continue
		case len(ch.Segments):
			out = append(out, ch)
			continue
		}
		start := len(segs)
		for k, seg := range ch.Segments {
			if want(f.first[i]+k, seg) {
				segs = append(segs, seg)
			}
		}
		c := &channels[0]
		channels = channels[1:]
		c.Key, c.Segments = ch.Key, segs[start:len(segs):len(segs)]
		out = append(out, c)
	}
	return out
}

// sealedAt returns the numbers of the sealed segments that no part holds.
func (f *flat) sealedAt() []int {
	var at []int
	for i, ch := range f.s.Channels {
		for k, seg := range ch.Segments {
			if j := f.first[i] + k; seg.Sealed && !f.parted[j] {
				at = append(at, j)
			}
		}
	}
	return at
}

// sealedSize returns the size in a head of the records of the sealed segments that no
// part holds.
func (f *flat) sealedSize() int {
	size := 0
	for i, ch := range f.s.Channels {
		held := false
		for k, seg := range ch.Segments {
			if seg.Sealed && !f.parted[f.first[i]+k] {
				held = true
				size += int(headFormat.sizes[sectionSegs]) +
					len(seg.Pointers)*int(headFormat.sizes[sectionPtrs])
			}
		}
		if held {
			size += int(headFormat.sizes[sectionChans])
		}
	}
	return size
}

func sameSegment(a, b state.Segment) bool {
	return a.Length == b.Length && a.TailCRC == b.TailCRC && a.Sealed == b.Sealed &&
		slices.Equal(a.Pointers, b.Pointers)
}
