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
	"math"
	"slices"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// load sets c to the durable state s of a head. It returns an error
// that wraps ErrRefused when a value of s breaks a rule, or log.ErrUnknown for a
// segment format version that this binary does not read.
func (c *core) load(s *Snapshot) error {
	if s.Limits.Alignments > uint64(telem.LeadingDomain) {
		return refuse(
			"head has alignment number limit %d, above %d",
			s.Limits.Alignments,
			telem.LeadingDomain,
		)
	}
	c.sequence, c.limits, c.reserved = s.Sequence, s.Limits, s.Limits
	for _, ch := range s.Channels {
		cs := c.channel(ch.Key)
		for _, seg := range ch.Segments {
			if err := c.checkSegment(ch.Key, seg); err != nil {
				return err
			}
			st := &segState{key: ch.Key, vis: seg, dur: durable{
				created:  true,
				sealed:   seg.Sealed,
				length:   seg.Length,
				tailCRC:  seg.TailCRC,
				pointers: slices.Clone(seg.Pointers),
			}}
			c.segments[seg.ID] = st
			cs.segs = append(cs.segs, st)
		}
		if err := checkApart(ch); err != nil {
			return err
		}
	}
	c.intents.Add(s.Intents...)
	return nil
}

func (c *core) checkSegment(key uint32, seg Segment) error {
	switch {
	case seg.ID == 0 || seg.ID >= c.limits.Segments:
		return refuse("head names segment %d, not from 1 to below the limit %d", seg.ID,
			c.limits.Segments)
	case c.segments[seg.ID] != nil:
		return refuse("head names segment %d twice", seg.ID)
	case !c.versions.Contains(seg.Version):
		return errors.Wrapf(log.ErrUnknown, "segment %d has format version %d", seg.ID,
			seg.Version)
	}
	var end uint32
	for _, p := range seg.Pointers {
		a := p.Alignment
		switch {
		case p.Samples == 0 || p.Length < p.Samples || p.First > p.Last ||
			(p.Samples == 1 && p.First != p.Last):
			return refuse(
				"segment %d of key %d has a pointer of %d samples in %d bytes "+
					"from %d to %d",
				seg.ID,
				key,
				p.Samples,
				p.Length,
				p.First,
				p.Last,
			)
		case p.Offset < end:
			return refuse(
				"segment %d has a pointer at %d, before the end %d of the one "+
					"before",
				seg.ID,
				p.Offset,
				end,
			)
		case a.DomainIndex() == 0 || uint64(a.DomainIndex()) >= c.limits.Alignments ||
			uint64(a.SampleIndex())+uint64(p.Samples)-1 > math.MaxUint32:
			return refuse("segment %d has a pointer with alignment %d of %d samples",
				seg.ID, uint64(a), p.Samples)
		case uint64(p.Offset)+uint64(p.Length) > uint64(seg.Length):
			return refuse("segment %d has a pointer past its length %d", seg.ID,
				seg.Length)
		}
		end = p.Offset + p.Length
	}
	return nil
}

// checkApart checks that the pointers of ch lie apart in time, across its segments.
func checkApart(ch *Channel) error {
	var all []Pointer
	for _, seg := range ch.Segments {
		all = append(all, seg.Pointers...)
	}
	slices.SortFunc(
		all,
		func(a, b Pointer) int { return cmp.Compare(a.First, b.First) },
	)
	for i := 1; i < len(all); i++ {
		if all[i].First <= all[i-1].Last {
			return refuse(
				"key %d has pointers that overlap at %d",
				ch.Key,
				all[i].First,
			)
		}
	}
	return nil
}

// endReplay makes the visible view of every segment its durable view, and publishes
// each key.
func (c *core) endReplay() {
	for _, s := range c.segments {
		s.vis.Sealed, s.vis.Length, s.vis.TailCRC = s.dur.sealed, s.dur.length,
			s.dur.tailCRC
		s.vis.Pointers = slices.Clone(s.dur.pointers)
		if s.dur.sealed {
			s.dur.pointers = s.vis.Pointers
		}
	}
	for _, ch := range c.channels {
		ch.publish()
	}
}

// snapshot returns the durable state. A segment that a durable Create made is in it,
// with its durable length and pointers.
func (c *core) snapshot() *Snapshot {
	s := &Snapshot{Sequence: c.sequence, Limits: c.limits}
	keys := make([]uint32, 0, len(c.channels))
	for k := range c.channels {
		keys = append(keys, k)
	}
	slices.Sort(keys)
	for _, k := range keys {
		ch := &Channel{Key: k}
		for _, seg := range c.channels[k].segs {
			if !seg.dur.created {
				continue
			}
			d := seg.vis
			d.Sealed, d.Length, d.TailCRC = seg.dur.sealed, seg.dur.length,
				seg.dur.tailCRC
			d.Pointers = seg.dur.pointers
			if !seg.dur.sealed {
				d.Pointers = slices.Clone(seg.dur.pointers)
			}
			ch.Segments = append(ch.Segments, d)
		}
		if len(ch.Segments) > 0 {
			s.Channels = append(s.Channels, ch)
		}
	}
	s.Intents = c.intents.Slice()
	slices.Sort(s.Intents)
	if len(s.Intents) == 0 {
		s.Intents = nil
	}
	return s
}
