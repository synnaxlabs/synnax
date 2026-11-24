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
	"slices"

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
	// version and threshold are the format version and the dead share of the candidate
	// test.
	version   uint16
	threshold float32
	// rewritable0 makes a checked version 0 segment a candidate too.
	rewritable0 bool
	// repairing makes load bury each sealed segment with no pointer. buried counts
	// them, at the start of dead.
	repairing bool
	buried    int
	// candidates holds the segments that pass the candidate test.
	candidates set.Set[uint64]
	// checking is the segment of the Check in flight, which a switch keeps, or 0.
	checking uint64
	// count counts the publishes of the owner for readers of more than one Channel.
	count    *pubCount
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
	dead []Dead
	keys set.Set[uint32]
	// generation is the generation of the log of the records that the core applies, or
	// of the head before it for replayed records. It stamps each kill.
	generation uint64
	// dropped holds the kills of the keys that a Drop took out of the state.
	dropped []Kill
	// head is the generation of the head whose parts the state reads, which a preload
	// must come from.
	head uint64
	// dropUsers holds, for each key that BeginDrop took, the part records that name it
	// as their clock, for the durable apply of its Drop.
	dropUsers map[uint32][]User
	// dropRecords holds, for each key that BeginDrop took, the part records of the key,
	// whose files its Drop gives to TakeDead.
	dropRecords map[uint32][]Segment
	// putBacks holds the keys with put-back records that a Parted has yet to take out.
	putBacks set.Set[uint32]
	// checked holds, for each segment with visible commits that are not durable yet,
	// the part records that those commits were checked against, which their durable
	// apply checks again. Most segments have none, so the map holds few.
	checked map[uint64][]Segment
}

func newCore(cfg Config) *core {
	return &core{
		versions:    cfg.Versions,
		version:     cfg.Version,
		threshold:   cfg.Threshold,
		rewritable0: cfg.Version0Rewritable,
		repairing:   cfg.Repairing,
		candidates:  set.Set[uint64]{},
		count:       &pubCount{},
		channels:    map[uint32]*chanState{},
		segments:    map[uint64]*segState{},
		intents:     set.Set[uint32]{},
		keys:        set.Set[uint32]{},
		dropUsers:   map[uint32][]User{},
		dropRecords: map[uint32][]Segment{},
		checked:     map[uint64][]Segment{},
	}
}

func (c *core) channel(key uint32) *chanState {
	ch, ok := c.channels[key]
	if !ok {
		ch = &chanState{key: key, count: c.count}
		c.channels[key] = ch
		c.keysChanged = true
	}
	return ch
}

// apply applies the records of the durable entry of sequence, in order, or returns an
// error that wraps ErrRefused, or log.ErrUnknown for a record this binary does not
// apply. A refused entry may be applied in part.
func (c *core) apply(sequence uint64, records []log.Record, pre []Preload) error {
	if sequence != c.sequence+1 {
		return refuse("entry %d follows entry %d", sequence, c.sequence)
	}
	for _, r := range records {
		if err := c.record(r, pre); err != nil {
			return errors.Wrapf(err, "entry %d", sequence)
		}
	}
	c.sequence = sequence
	return nil
}

func (c *core) record(r log.Record, pre []Preload) error {
	switch r := r.(type) {
	case *log.Reserve:
		return c.reserve(r)
	case *log.Create:
		return c.create(r)
	case *log.Commit:
		return c.commit(r, pre)
	case *log.Seal:
		return c.seal(r)
	case *log.Drop:
		return c.drop(r.Key, pre)
	case *log.Restart:
		c.restart()
		return nil
	case *log.Intent:
		return c.intent(r)
	case *log.Tombstone:
		return c.tombstone(r, pre)
	case *log.Replace:
		return c.replace(r)
	case *log.Check:
		return c.check(r, pre)
	default:
		return errors.Wrapf(log.ErrUnknown, "the state does not apply a %T record", r)
	}
}

// check turns on the checks of a sealed version 0 segment that now has its check file.
// A part may hold its record from before, so the record goes back to the head from
// pre, the preload of its key in replay, with a kill of the copy that a part holds.
func (c *core) check(r *log.Check, pre []Preload) error {
	records := recordsOf(pre, r.Key)
	if i := slices.IndexFunc(records, func(s Segment) bool {
		return s.ID == r.Segment
	}); i >= 0 && c.channels[r.Key] != nil {
		c.putBack(r.Key, records[i:i+1])
	}
	s := c.segments[r.Segment]
	switch {
	case s == nil || !s.dur.created:
		return refuse("check of segment %d, which no create made", r.Segment)
	case s.key != r.Key:
		return refuse("check of segment %d of key %d, which key %d holds", r.Segment,
			r.Key, s.key)
	case s.vis.Version != 0 || !s.dur.sealed:
		return refuse("check of segment %d, which is not a sealed version 0 segment",
			r.Segment)
	case s.vis.Checked:
		return refuse("check of segment %d, which has its checks", r.Segment)
	}
	ch := c.channels[s.key]
	s.vis.Checked, s.vis.Part = true, 0
	ch.kills = append(ch.kills, Kill{
		Key:        s.key,
		Segment:    s.vis.ID,
		Generation: c.generation,
	})
	ch.repart()
	c.test(s)
	ch.publish()
	return nil
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
	case r.Version == 0:
		return refuse("create of segment %d of version 0, which only a conversion "+
			"makes", r.Segment)
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

func (c *core) commit(r *log.Commit, pre []Preload) error {
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
		if p, ok := c.durableOverlap(s, first, last, pre); ok {
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
		if s.dur.length == s.vis.Length {
			delete(c.checked, s.vis.ID)
			ch := c.channels[s.key]
			ch.pending = slices.DeleteFunc(ch.pending, func(o *segState) bool {
				return o == s
			})
		}
	}
	return nil
}

// durableOverlap returns the first durable pointer of the key of s that shares a stamp
// with [first, last]. It reads the part records that the visible commits of s
// preloaded, and those of the key in pre.
func (c *core) durableOverlap(
	s *segState,
	first, last telem.TimeStamp,
	pre []Preload,
) (Pointer, bool) {
	return c.channels[s.key].overlapsDurable(
		first,
		last,
		c.checked[s.vis.ID],
		recordsOf(pre, s.key),
	)
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
	c.test(s)
	c.bury(s)
	return nil
}

// test adds s to the candidates when it is sealed in the durable view, untagged, of the
// format version of new segments or a checked version 0 segment that may be rewritten,
// and its dead share is above the threshold, and takes it out when not. A segment that
// a part holds gets no test until a Tombstone or a Check puts its record back in the
// head.
func (c *core) test(s *segState) {
	var live uint64
	for _, p := range s.dur.pointers {
		live += uint64(p.Length)
	}
	length := uint64(s.dur.length)
	rewritable := s.vis.Version == c.version ||
		c.rewritable0 && s.vis.Version == 0 && s.vis.Checked
	if s.dur.sealed && s.vis.Part == 0 && rewritable &&
		float64(length-live) > float64(c.threshold)*float64(length) {
		c.candidates.Add(s.vis.ID)
		return
	}
	c.candidates.Remove(s.vis.ID)
}

// replace takes the old segment of r out of the state, and seals in its place the new
// segment that the entry of r created, with the pointers of r. The copy keeps the time
// range, the alignment, the samples, and the length of each pointer.
func (c *core) replace(r *log.Replace) error {
	old, s := c.segments[r.Old], c.segments[r.New]
	switch {
	case old == nil || !old.dur.sealed:
		return refuse("replace of segment %d, which is not sealed", r.Old)
	case s == nil || !s.dur.created || s.dur.length != 0 || s.holder != nil:
		return refuse("replace of segment %d by %d, which is not a new segment",
			r.Old, r.New)
	case s.key != old.key || s.vis.Clock != old.vis.Clock ||
		s.vis.DataType != old.vis.DataType:
		return refuse("replace of segment %d by %d, which differs in key, clock, or "+
			"type", r.Old, r.New)
	case !copied(old.dur.pointers, r.Pointers):
		return refuse("replace of segment %d, whose pointers differ from the copy",
			r.Old)
	case !rising(r.Pointers):
		return refuse("replace of segment %d by %d, whose pointers overlap or fall "+
			"out of stream order", r.Old, r.New)
	case end(r.Pointers[len(r.Pointers)-1]) > uint64(r.Length):
		return refuse("replace of segment %d by %d, whose pointers end past its "+
			"length %d", r.Old, r.New, r.Length)
	// A candidate is never in a part, and live and replay name the same candidates,
	// while a part that a switch writes may still take a segment that is none.
	case !c.candidates.Contains(r.Old):
		return refuse("replace of segment %d, which is no candidate of a rewrite",
			r.Old)
	}
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
	}
	s.dur.sealed, s.dur.length, s.dur.tailCRC, s.dur.pointers = true, r.Length,
		r.TailCRC, ptrs
	s.vis.Sealed, s.vis.Length, s.vis.TailCRC, s.vis.Pointers = true, r.Length,
		r.TailCRC, ptrs
	old.dur.pointers = nil
	c.bury(old)
	return nil
}

// copied reports whether ptrs, the durable pointers of a segment, and dst, the pointers
// of its copy, name the same ranges at any offsets.
func copied(ptrs []Pointer, dst []log.Pointer) bool {
	return slices.EqualFunc(ptrs, dst, func(p Pointer, q log.Pointer) bool {
		return int64(p.First) == q.First && int64(p.Last) == q.Last &&
			uint64(p.Alignment) == q.Alignment && p.Length == q.Length &&
			p.Samples == q.Samples
	})
}

// rising reports whether ptrs rise in stream order with no overlap.
func rising(ptrs []log.Pointer) bool {
	for i := 1; i < len(ptrs); i++ {
		if uint64(ptrs[i].Offset) < end(ptrs[i-1]) {
			return false
		}
	}
	return true
}

func end(p log.Pointer) uint64 { return uint64(p.Offset) + uint64(p.Length) }

// bury takes s out of the state when it is sealed in the durable view with no durable
// pointer, and gives it to TakeDead: its file holds no live byte. It reads only the
// durable view, so replay and the owner bury the same segments.
func (c *core) bury(s *segState) {
	if !s.dur.sealed || len(s.dur.pointers) > 0 {
		return
	}
	ch := c.channels[s.key]
	ch.remove(s.vis.ID)
	delete(c.segments, s.vis.ID)
	c.candidates.Remove(s.vis.ID)
	c.dead = append(c.dead, Dead{Segment: s.vis.ID, Key: s.key, File: s.vis.File})
	ch.publish()
}

func (c *core) drop(key uint32, pre []Preload) error {
	users := usersOf(pre, key)
	if users == nil {
		users = c.dropUsers[key]
	}
	delete(c.dropUsers, key)
	records := recordsOf(pre, key)
	if records == nil {
		records = c.dropRecords[key]
	}
	delete(c.dropRecords, key)
	// A segment that a part holds counts through its record in users.
	for _, s := range c.segments {
		if s.key != key && s.vis.Clock == key && s.dur.created && s.vis.Part == 0 {
			return refuse("drop of key %d, the clock of segment %d of key %d", key,
				s.vis.ID, s.key)
		}
	}
	for _, u := range users {
		// The records of a key that a Drop took out are dead.
		if u.Key != key && c.channels[u.Key] != nil {
			return refuse("drop of key %d, the clock of segment %d of key %d", key,
				u.Segment.ID, u.Key)
		}
	}
	if ch, ok := c.channels[key]; ok {
		// A head names a key only while it holds a durable segment or parts, so a
		// replayed Drop of any other key finds no state of it.
		if ch.oldest != 0 || slices.ContainsFunc(ch.segs, func(s *segState) bool {
			return s.dur.created
		}) {
			c.dropped = append(
				c.dropped,
				Kill{Key: key, All: true, Generation: c.generation},
			)
		}
		// The database unlinks the file of each segment once no reader pins it. A
		// record whose segment the state holds goes with that segment.
		for _, rec := range records {
			if _, held := c.segments[rec.ID]; !held {
				c.dead = append(c.dead, Dead{Segment: rec.ID, Key: key, File: rec.File})
			}
		}
		for _, s := range ch.segs {
			delete(c.segments, s.vis.ID)
			c.candidates.Remove(s.vis.ID)
			c.dead = append(c.dead, Dead{Segment: s.vis.ID, Key: key, File: s.vis.File})
		}
		delete(c.channels, key)
		c.count.begin()
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
			c.test(s)
			c.bury(s)
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
