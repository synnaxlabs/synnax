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
	"maps"
	"slices"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// Rebuild is what a repair puts in the state at a bad frame of the log, after the
// entries before it.
type Rebuild struct {
	// Sequence is the sequence of the last entry of the bad frame.
	Sequence uint64
	// Limits are the limits of segment identities and alignment numbers that the state
	// raises its limits to, when they are above its own.
	Limits Limits
	// Creates make the segments that the repair places.
	Creates []log.Create
	// Commits put the rebuilt ranges in the state by the rules of a Commit, each with a
	// fresh alignment number in place of its own.
	Commits []log.Commit
	// Ends move each segment with bytes in the gap to its length and tail checksum at
	// the end of the gap.
	Ends []log.Slot
}

// Rebuild applies r in replay mode, before Run: the limits and the sequence, the
// Creates, each Commit, then the Ends. Each Commit takes the next alignment number at
// the limit, which rises past it. pre holds, in rising order of key, the part records
// that the Commits reach. It returns the error of each Commit that breaks a rule, which
// applies nothing, or nil for each that applies. It returns an error when the sequence
// of r is before the state's, when the alignment number limit passes the domain limit,
// or when a Create or an End breaks a rule.
func (s *State) Rebuild(r Rebuild, pre ...Preload) ([]error, error) {
	defer s.core.count.end()
	return s.core.rebuild(r, pre)
}

func (c *core) rebuild(r Rebuild, pre []Preload) ([]error, error) {
	limits := Limits{
		Segments:   max(c.limits.Segments, r.Limits.Segments),
		Alignments: max(c.limits.Alignments, r.Limits.Alignments, 1),
	}
	switch fresh := limits.Alignments + uint64(len(r.Commits)); {
	case r.Sequence < c.sequence:
		return nil, refuse("rebuild at entry %d, before entry %d", r.Sequence,
			c.sequence)
	case fresh > uint64(telem.LeadingDomain):
		return nil, refuse("rebuild of alignment number limit %d, above %d", fresh,
			telem.LeadingDomain)
	}
	c.sequence = r.Sequence
	c.limits.Segments, c.limits.Alignments = limits.Segments, limits.Alignments
	for i := range r.Creates {
		if err := c.create(&r.Creates[i]); err != nil {
			return nil, err
		}
	}
	errs := make([]error, len(r.Commits))
	for i := range r.Commits {
		cm := r.Commits[i]
		cm.Alignment = uint64(telem.NewAlignment(uint32(c.limits.Alignments), 0))
		c.limits.Alignments++
		errs[i] = c.commit(&cm, pre)
	}
	c.reserved.Segments = max(c.reserved.Segments, c.limits.Segments)
	c.reserved.Alignments = max(c.reserved.Alignments, c.limits.Alignments)
	for _, e := range r.Ends {
		s := c.segments[e.Segment]
		switch {
		case s == nil || !s.dur.created:
			return nil, refuse("end of segment %d, which no create made", e.Segment)
		case s.dur.sealed:
			return nil, refuse("end of segment %d, which is sealed", e.Segment)
		case e.Length < s.dur.length:
			return nil, refuse("end of segment %d at length %d, below %d", e.Segment,
				e.Length, s.dur.length)
		}
		s.dur.length, s.dur.tailCRC = e.Length, e.TailCRC
	}
	return errs, nil
}

// Skipped is a record that ReplayEach skips, and the rule that it breaks.
type Skipped struct {
	Record log.Record
	Err    error
}

// ReplayEach applies the records of the durable entry of sequence one at a time, in
// replay mode, before Run, and returns each one that breaks a rule. Such a record
// changes nothing, but for the part records that its preload put back, which keep
// their tags. pre holds, in rising order of key, the part records that the records
// reach. It returns an error when sequence does not follow the entry of the state, or
// one that wraps log.ErrUnknown for a record that this binary does not apply.
func (s *State) ReplayEach(
	sequence uint64,
	records []log.Record,
	pre ...Preload,
) ([]Skipped, error) {
	defer s.core.count.end()
	return s.core.applyEach(sequence, records, pre)
}

func (c *core) applyEach(
	sequence uint64,
	records []log.Record,
	pre []Preload,
) ([]Skipped, error) {
	if sequence != c.sequence+1 {
		return nil, refuse("entry %d follows entry %d", sequence, c.sequence)
	}
	var skipped []Skipped
	for _, r := range records {
		err := c.record(r, pre)
		if errors.Is(err, errRefused) {
			skipped = append(skipped, Skipped{Record: r, Err: err})
			continue
		}
		if err != nil {
			return nil, errors.Wrapf(err, "entry %d", sequence)
		}
	}
	c.sequence = sequence
	return skipped, nil
}

// ReplayKeys returns the keys that the state holds, in rising order. Call it in replay
// mode, before Run.
func (s *State) ReplayKeys() []uint32 {
	return slices.Sorted(maps.Keys(s.core.channels))
}

// Remove takes segment of key out of the state for a repair whose file is gone, in
// replay mode, before Run. A segment that a part holds gets a kill, when parted is true
// or the state holds the record of the part. It returns an error when the state holds
// neither the segment nor key.
func (s *State) Remove(key uint32, segment uint64, parted bool) error {
	defer s.core.count.end()
	return s.core.remove(key, segment, parted)
}

func (c *core) remove(key uint32, id uint64, parted bool) error {
	ch := c.channels[key]
	seg := c.segments[id]
	switch {
	case ch == nil:
		return refuse("removal of segment %d of key %d, which the state does not hold",
			id, key)
	case seg != nil && seg.key == key:
		ch.remove(id)
		delete(c.segments, id)
		c.candidates.Remove(id)
		parted = parted || seg.vis.Part != 0
	case !parted:
		return refuse("removal of segment %d of key %d, which the state does not hold",
			id, key)
	}
	if parted {
		ch.kills = append(
			ch.kills,
			Kill{Key: key, Segment: id, Generation: c.generation},
		)
		ch.repart()
	}
	ch.publish()
	return nil
}

// Gone marks the file of segment gone, in repair mode, before Run. A later Replace of
// segment then takes it out of the state, and gives the copy the pointers that the
// Replace names by the pointer rules of a Commit alone.
func (s *State) Gone(segment uint64) { s.core.gone.Add(segment) }

// replaceGone applies r, whose old segment has no file: the copy takes the pointers of
// r when they rise inside its length and no other pointer of the key shares a stamp
// with them, and the old segment leaves the state.
func (c *core) replaceGone(r *log.Replace, pre []Preload) error {
	old, s := c.segments[r.Old], c.segments[r.New]
	switch {
	case old == nil:
		return refuse("replace of segment %d, which the state does not hold", r.Old)
	case s == nil || !s.dur.created || s.dur.length != 0 || s.holder != nil:
		return refuse("replace of segment %d by %d, which is not a new segment",
			r.Old, r.New)
	case s.key != old.key || s.vis.Clock != old.vis.Clock ||
		s.vis.DataType != old.vis.DataType:
		return refuse("replace of segment %d by %d, which differs in key, clock, or "+
			"type", r.Old, r.New)
	case len(r.Pointers) == 0 || !rising(r.Pointers):
		return refuse("replace of segment %d by %d, whose pointers are none, overlap, "+
			"or fall out of stream order", r.Old, r.New)
	case end(r.Pointers[len(r.Pointers)-1]) > uint64(r.Length):
		return refuse("replace of segment %d by %d, whose pointers end past its "+
			"length %d", r.Old, r.New, r.Length)
	}
	ch := c.channels[s.key]
	ptrs := make([]Pointer, len(r.Pointers))
	for i, p := range r.Pointers {
		ptrs[i] = Pointer{
			First:     telem.TimeStamp(p.First),
			Last:      telem.TimeStamp(p.Last),
			Alignment: telem.Alignment(p.Alignment),
			Offset:    p.Offset,
			Length:    p.Length,
			Samples:   p.Samples,
		}
		if err := checkShape(r.New, s.key, ptrs[i]); err != nil {
			return err
		}
		if q, ok := ch.overlapsOther(ptrs[i], recordsOf(pre, s.key), r.Old, r.New); ok {
			return refuse("replace of segment %d by %d, whose pointer from %d to %d "+
				"overlaps a pointer of key %d from %d to %d", r.Old, r.New,
				ptrs[i].First, ptrs[i].Last, s.key, q.First, q.Last)
		}
	}
	parted := old.vis.Part != 0
	ch.remove(r.Old)
	delete(c.segments, r.Old)
	c.candidates.Remove(r.Old)
	if parted {
		ch.kills = append(
			ch.kills,
			Kill{Key: s.key, Segment: r.Old, Generation: c.generation},
		)
		ch.repart()
	}
	s.dur.sealed, s.dur.length, s.dur.tailCRC, s.dur.pointers = true, r.Length,
		r.TailCRC, ptrs
	s.vis.Sealed, s.vis.Length, s.vis.TailCRC, s.vis.Pointers = true, r.Length,
		r.TailCRC, ptrs
	c.test(s)
	ch.publish()
	return nil
}

// overlapsOther returns the first durable pointer of c, or of records, that shares a
// stamp with p, of a segment other than those of skip.
func (c *chanState) overlapsOther(
	p Pointer,
	records []Segment,
	skip ...uint64,
) (Pointer, bool) {
	for _, s := range c.segs {
		if s.vis.Part != 0 || slices.Contains(skip, s.vis.ID) {
			continue
		}
		if q, ok := overlap(s.dur.pointers, p.First, p.Last); ok {
			return q, true
		}
	}
	for _, s := range records {
		if slices.Contains(skip, s.ID) {
			continue
		}
		if q, ok := overlap(s.Pointers, p.First, p.Last); ok {
			return q, true
		}
	}
	return Pointer{}, false
}

// ReplaySnapshot returns the durable state as replay has left it, which a repair
// writes as a head. Call it in replay mode, before Run.
func (s *State) ReplaySnapshot() *Snapshot { return s.core.snapshot() }
