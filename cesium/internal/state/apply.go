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
	"math"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
)

// ErrRefused is the error of a log record or a head that breaks a rule of the state.
var ErrRefused = errors.New("refused by the state")

func refuse(format string, args ...any) error {
	return errors.Wrapf(ErrRefused, format, args...)
}

// core is the data of the state and the rules that change it. Only the owner uses it.
type core struct {
	versions set.Set[uint16]
	channels map[uint32]*chanState
	segments map[uint64]*segState
	intents  set.Set[uint32]
	// sequence is the sequence of the last durable entry.
	sequence uint64
	// limits are the durable limits, below which the state gives out values. reserved
	// holds the limits of the last Reserve appended, above limits while that Reserve
	// waits for its frame.
	limits, reserved Limits
	// keysChanged is true when a key joined or left channels since the owner last
	// published the keys.
	keysChanged bool
	// dead holds the segments that left the state with a file left to unlink once no
	// reader pins it.
	dead []uint64
	keys set.Set[uint32]
}

func newCore(versions set.Set[uint16]) *core {
	return &core{
		versions: versions,
		channels: map[uint32]*chanState{},
		segments: map[uint64]*segState{},
		intents:  set.Set[uint32]{},
		keys:     set.Set[uint32]{},
	}
}

func (c *core) channel(key uint32) *chanState {
	ch, ok := c.channels[key]
	if !ok {
		ch = &chanState{key: key}
		c.channels[key] = ch
		c.keysChanged = true
	}
	return ch
}

// apply applies the records of the durable entry of sequence, in order, or returns an
// error that wraps ErrRefused, or log.ErrUnknown for a record this binary does not
// apply. A refused entry may be applied in part.
func (c *core) apply(sequence uint64, records []log.Record) error {
	if sequence != c.sequence+1 {
		return refuse("entry %d follows entry %d", sequence, c.sequence)
	}
	for _, r := range records {
		if err := c.record(r); err != nil {
			return errors.Wrapf(err, "entry %d", sequence)
		}
	}
	c.sequence = sequence
	return nil
}

func (c *core) record(r log.Record) error {
	switch r := r.(type) {
	case *log.Reserve:
		return c.reserve(r)
	case *log.Create:
		return c.create(r)
	case *log.Commit:
		return c.commit(r)
	case *log.Seal:
		return c.seal(r)
	case *log.Drop:
		return c.drop(r.Key)
	case *log.Restart:
		c.restart()
		return nil
	case *log.Intent:
		return c.intent(r)
	case *log.Tombstone:
		dead, err := c.tombstone(r)
		c.dead = append(c.dead, dead...)
		return err
	default:
		return errors.Wrapf(log.ErrUnknown, "the state does not apply a %T record", r)
	}
}

func (c *core) reserve(r *log.Reserve) error {
	switch {
	case r.Alignments > uint64(telem.LeadingDomain):
		return refuse("reserve of alignment number limit %d, above %d", r.Alignments,
			telem.LeadingDomain)
	case r.Segments < c.limits.Segments || r.Alignments < c.limits.Alignments:
		return refuse("reserve of limits %d and %d, below %d and %d", r.Segments,
			r.Alignments, c.limits.Segments, c.limits.Alignments)
	}
	c.limits.Segments, c.limits.Alignments = r.Segments, r.Alignments
	c.reserved.Segments = max(c.reserved.Segments, r.Segments)
	c.reserved.Alignments = max(c.reserved.Alignments, r.Alignments)
	return nil
}

func (c *core) create(r *log.Create) error {
	switch s := c.segments[r.Segment]; {
	case r.Segment == 0 || r.Segment >= c.limits.Segments:
		return refuse("create of segment %d, not from 1 to below the limit %d",
			r.Segment, c.limits.Segments)
	case s != nil && s.dur.created:
		return refuse("create of segment %d, which exists", r.Segment)
	case !c.versions.Contains(r.Version):
		return errors.Wrapf(log.ErrUnknown, "segment %d has format version %d",
			r.Segment, r.Version)
	case s != nil && (s.key != r.Key || s.vis.Clock != r.Clock ||
		s.vis.Version != r.Version || s.vis.DataType != r.DataType):
		return refuse("create of segment %d differs from the segment it names",
			r.Segment)
	case s != nil:
		s.dur.created = true
		return nil
	}
	s := &segState{key: r.Key, vis: Segment{
		ID:       r.Segment,
		Clock:    r.Clock,
		Version:  r.Version,
		DataType: r.DataType,
	}, dur: durable{created: true}}
	c.segments[r.Segment] = s
	c.channel(r.Key).insert(s)
	return nil
}

// checkCommit checks the value rules of a Commit that do not depend on its slots.
func (c *core) checkCommit(r *log.Commit) error {
	a := telem.Alignment(r.Alignment)
	switch {
	case len(r.Slots) == 0:
		return refuse("commit with no slot")
	case r.Samples == 0 || r.First > r.Last || (r.Samples == 1 && r.First != r.Last):
		return refuse("commit of %d samples from %d to %d", r.Samples, r.First, r.Last)
	case a.DomainIndex() == 0 || uint64(a.DomainIndex()) >= c.limits.Alignments:
		return refuse("commit of alignment number %d, not from 1 to below the limit %d",
			a.DomainIndex(), c.limits.Alignments)
	case uint64(a.SampleIndex())+uint64(r.Samples)-1 > math.MaxUint32:
		return refuse("commit of %d samples from sample index %d", r.Samples,
			a.SampleIndex())
	}
	return nil
}

func (c *core) commit(r *log.Commit) error {
	if err := c.checkCommit(r); err != nil {
		return err
	}
	first, last := telem.TimeStamp(r.First), telem.TimeStamp(r.Last)
	clear(c.keys)
	var clock uint32
	for i, slot := range r.Slots {
		s := c.segments[slot.Segment]
		switch {
		case i > 0 && slot.Segment <= r.Slots[i-1].Segment:
			return refuse("commit slot %d names segment %d, not above segment %d", i,
				slot.Segment, r.Slots[i-1].Segment)
		case s == nil || !s.dur.created:
			return refuse("commit names segment %d, which no create made", slot.Segment)
		case s.dur.sealed:
			return refuse("commit names segment %d, which is sealed", slot.Segment)
		case i > 0 && s.vis.Clock != clock:
			return refuse("commit names segments of clocks %d and %d", clock,
				s.vis.Clock)
		case c.keys.Contains(s.key):
			return refuse("commit names two segments of key %d", s.key)
		case slot.Length <= s.dur.length:
			return refuse("commit gives segment %d length %d, not above %d",
				slot.Segment, slot.Length, s.dur.length)
		}
		err := checkGrow(slot.Segment, s.key, s.dur.pointers, r, s.dur.length,
			slot.Length)
		if err != nil {
			return err
		}
		if p, ok := c.channels[s.key].overlaps(first, last, true); ok {
			return refuse(
				"commit of key %d from %d to %d overlaps its pointer from %d "+
					"to %d",
				s.key,
				first,
				last,
				p.First,
				p.Last,
			)
		}
		clock = s.vis.Clock
		c.keys.Add(s.key)
	}
	for _, slot := range r.Slots {
		s := c.segments[slot.Segment]
		p, extended := grow(s.dur.pointers, telem.Alignment(r.Alignment), first, last,
			s.dur.length, slot.Length, r.Samples)
		s.dur.pointers = extend(s.dur.pointers, p, extended, true)
		s.dur.length, s.dur.tailCRC = slot.Length, slot.TailCRC
	}
	return nil
}

func (c *core) seal(r *log.Seal) error {
	s := c.segments[r.Segment]
	switch {
	case s == nil || !s.dur.created:
		return refuse("seal of segment %d, which no create made", r.Segment)
	case s.dur.sealed:
		return refuse("seal of segment %d, which is sealed", r.Segment)
	case r.Length != s.dur.length || r.TailCRC != s.dur.tailCRC:
		return refuse(
			"seal of segment %d at length %d and tail %#x, not its committed "+
				"%d and %#x",
			r.Segment,
			r.Length,
			r.TailCRC,
			s.dur.length,
			s.dur.tailCRC,
		)
	}
	s.dur.sealed = true
	if s.vis.Sealed && s.vis.Length == s.dur.length {
		s.dur.pointers = s.vis.Pointers
	}
	return nil
}

func (c *core) drop(key uint32) error {
	for _, s := range c.segments {
		if s.key != key && s.vis.Clock == key && s.dur.created {
			return refuse("drop of key %d, the clock of segment %d of key %d", key,
				s.vis.ID, s.key)
		}
	}
	if ch, ok := c.channels[key]; ok {
		for _, s := range ch.segs {
			delete(c.segments, s.vis.ID)
		}
		delete(c.channels, key)
		ch.pub.Store(nil)
		c.keysChanged = true
	}
	c.intents.Remove(key)
	return nil
}

func (c *core) restart() {
	for _, s := range c.segments {
		if s.dur.created && !s.dur.sealed {
			s.dur.sealed = true
		}
	}
}

func (c *core) intent(r *log.Intent) error {
	if r.Cleared {
		if !c.intents.Contains(r.Key) {
			return refuse("intent of key %d cleared, but it has none", r.Key)
		}
		c.intents.Remove(r.Key)
		return nil
	}
	c.intents.Add(r.Key)
	return nil
}
