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

	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/set"
)

// plan is what a switch writes: the parts that it keeps, the parts that it writes, and
// the channels of the head.
type plan struct {
	index  *Index
	writes []partWrite
	head   []*state.Channel
}

type partWrite struct {
	id       uint64
	channels []*state.Channel
}

// plan diffs s, the whole state at a switch, against the index of the last head. A part
// record that s no longer holds is dead. A part with half its records dead, or with a
// record whose segment s holds cut, is written again under a new identity. Sealed
// segments that no part holds go to a new part once their records pass the part size.
// The head holds the rest.
func (c *Checkpointer) plan(s *state.Snapshot) plan {
	live := make(map[uint64]state.Segment)
	for _, ch := range s.Channels {
		for _, seg := range ch.Segments {
			live[seg.ID] = seg
		}
	}
	pl := plan{index: &Index{limit: max(c.index.limit, s.Limits.Parts, 1)}}
	parted := set.New[uint64]()
	for _, p := range c.index.parts {
		q := p
		q.Dead = slices.Clone(p.Dead)
		cut := false
		for i, seg := range q.segs {
			if q.dead(i) {
				continue
			}
			cur, ok := live[seg.ID]
			if !ok {
				q.kill(i)
			} else if !sameSegment(cur, seg) {
				cut = true
			}
		}
		dead := q.deadCount()
		switch {
		case dead == len(q.segs):
		case cut || 2*dead >= len(q.segs):
			pl.add(liveChannels(q, live))
		default:
			pl.index.parts = append(pl.index.parts, q)
			for i, seg := range q.segs {
				if !q.dead(i) {
					parted.Add(seg.ID)
				}
			}
		}
	}
	for _, w := range pl.writes {
		for _, ch := range w.channels {
			for _, seg := range ch.Segments {
				parted.Add(seg.ID)
			}
		}
	}
	sealed, head := split(s.Channels, parted)
	if recordSize(sealed) > c.cfg.PartSize {
		pl.add(sealed)
		sealed = nil
	}
	pl.head = merge(head, sealed)
	return pl
}

// add queues a new part of channels.
func (pl *plan) add(channels []*state.Channel) {
	pl.writes = append(pl.writes, partWrite{id: pl.index.limit, channels: channels})
	pl.index.limit++
}

// liveChannels returns the segments of the records of p that are not dead, as s holds
// them, in record order.
func liveChannels(p indexed, live map[uint64]state.Segment) []*state.Channel {
	var out []*state.Channel
	for i, seg := range p.segs {
		if p.dead(i) {
			continue
		}
		if len(out) == 0 || out[len(out)-1].Key != p.keys[i] {
			out = append(out, &state.Channel{Key: p.keys[i]})
		}
		c := out[len(out)-1]
		c.Segments = append(c.Segments, live[seg.ID])
	}
	return out
}

// split returns the sealed segments of channels that no part holds, and the open ones.
func split(
	channels []*state.Channel,
	parted set.Set[uint64],
) (sealed, open []*state.Channel) {
	for _, ch := range channels {
		var s, o []state.Segment
		for _, seg := range ch.Segments {
			switch {
			case parted.Contains(seg.ID):
			case seg.Sealed:
				s = append(s, seg)
			default:
				o = append(o, seg)
			}
		}
		if len(s) > 0 {
			sealed = append(sealed, &state.Channel{Key: ch.Key, Segments: s})
		}
		if len(o) > 0 {
			open = append(open, &state.Channel{Key: ch.Key, Segments: o})
		}
	}
	return sealed, open
}

// merge returns the channels of a and b, in rising order of key, with the segments of
// each key in rising order of identity.
func merge(a, b []*state.Channel) []*state.Channel {
	byKey := make(map[uint32]*state.Channel, len(a)+len(b))
	for _, ch := range slices.Concat(a, b) {
		c, ok := byKey[ch.Key]
		if !ok {
			c = &state.Channel{Key: ch.Key}
			byKey[ch.Key] = c
		}
		c.Segments = append(c.Segments, ch.Segments...)
	}
	out := make([]*state.Channel, 0, len(byKey))
	for _, c := range byKey {
		slices.SortFunc(c.Segments, func(x, y state.Segment) int {
			return cmp.Compare(x.ID, y.ID)
		})
		out = append(out, c)
	}
	slices.SortFunc(out, func(x, y *state.Channel) int {
		return cmp.Compare(x.Key, y.Key)
	})
	return out
}

// recordSize returns the size of the channel, segment, and pointer records of channels
// in a head.
func recordSize(channels []*state.Channel) int {
	size := 0
	for _, ch := range channels {
		size += int(headFormat.sizes[sectionChans])
		for _, seg := range ch.Segments {
			size += int(headFormat.sizes[sectionSegs]) +
				len(seg.Pointers)*int(headFormat.sizes[sectionPtrs])
		}
	}
	return size
}

func sameSegment(a, b state.Segment) bool {
	return a.Length == b.Length && a.TailCRC == b.TailCRC && a.Sealed == b.Sealed &&
		slices.Equal(a.Pointers, b.Pointers)
}
