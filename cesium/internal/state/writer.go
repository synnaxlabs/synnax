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

	"github.com/synnaxlabs/cesium/internal/alignment"
	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

var (
	// ErrConflict is the error of a writer open or a commit that reaches stored data or
	// another writer's claim.
	ErrConflict = errors.Wrap(validate.ErrValidation, "write conflict")
	// ErrExhausted is the error of a new stretch when every stored alignment number is
	// given out.
	ErrExhausted = errors.New("alignment numbers are exhausted")
)

const (
	// firstBlock and laterBlock are how many values a Reserve gives out: the first
	// of an open, then each after it.
	firstBlock = 16
	laterBlock = 1024
)

// WriterKey is a key that a writer writes, with the data type code of its samples.
type WriterKey struct {
	Key      uint32
	DataType uint8
}

// WriterConfig is the configuration of a writer.
type WriterConfig struct {
	// Keys are the keys that the writer writes. Each is timed by Clock.
	Keys []WriterKey
	// Clock is the clock key of every key. A writer that writes data against stored
	// stamps does not list its clock in Keys, and pins the clock over its commits.
	Clock uint32
	// Start is the first stamp that the writer may write.
	Start telem.TimeStamp
	// Bounded makes each claim cover one nanosecond at Start and grow to the end of
	// each commit. A claim that is not bounded runs from Start to the end of time.
	Bounded bool
}

// claim is the time range that a writer may write on one key. A pin keeps stamps that
// a writer writes against from a delete, and conflicts with no writer.
type claim struct {
	w          *Writer
	start, end telem.TimeStamp
	bounded    bool
	pin        bool
}

func (c *claim) reaches(first, last telem.TimeStamp) bool {
	return first <= c.end && last >= c.start
}

// Writer is the state of one Cesium writer: its claims and the open segment that it
// holds for each key. Its calls block until the owner runs them. A Writer is not safe
// for concurrent use.
type Writer struct {
	s    *State
	cfg  WriterConfig
	done chan error
	// The fields below belong to the owner.
	claims map[uint32]*claim
	held   map[uint32]*segState
	types  map[uint32]uint8
}

// call queues r for the owner and waits for its error.
func (w *Writer) call(r request) error {
	if err := w.s.enqueue(r); err != nil {
		return err
	}
	return wait(w.s, w.done)
}

type openRequest struct{ w *Writer }

func (r *openRequest) run(s *State) { r.w.done <- s.openWriter(r.w) }

// OpenWriter opens a writer of cfg. It returns an error that wraps ErrConflict when the
// start lies in stored data of one of its keys, when a bounded claim starts inside
// another claim, or when another writer that is not bounded claims one of its keys.
// The writer takes over each open segment of its keys that no writer holds, when it
// has the same clock, type, and version, and every commit of it is durable.
func (s *State) OpenWriter(cfg WriterConfig) (*Writer, error) {
	w := &Writer{
		s:      s,
		cfg:    cfg,
		done:   make(chan error, 1),
		claims: map[uint32]*claim{},
		held:   map[uint32]*segState{},
		types:  map[uint32]uint8{},
	}
	for _, k := range cfg.Keys {
		w.types[k.Key] = k.DataType
	}
	if err := w.call(&openRequest{w: w}); err != nil {
		return nil, err
	}
	return w, nil
}

func (s *State) openWriter(w *Writer) error {
	start := w.cfg.Start
	for key := range w.types {
		ch := s.core.channels[key]
		if ch == nil {
			continue
		}
		if p, ok := ch.overlaps(start, start, false); ok {
			return errors.Wrapf(
				ErrConflict,
				"writer of key %d starts at %d, inside its "+
					"stored data from %d to %d",
				key,
				start,
				p.First,
				p.Last,
			)
		}
		for _, o := range ch.claims {
			refused := false
			switch {
			case o.pin:
			case w.cfg.Bounded:
				refused = o.reaches(start, start)
			default:
				refused = !o.bounded
			}
			if refused {
				return errors.Wrapf(ErrConflict, "writer of key %d starting at %d "+
					"reaches the claim of another writer from %d", key, start, o.start)
			}
		}
	}
	end := telem.TimeStamp(math.MaxInt64)
	if w.cfg.Bounded {
		end = start
	}
	keys := append(slices.Collect(mapsKeys(w.types)), w.cfg.Clock)
	for _, key := range keys {
		if _, ok := w.claims[key]; ok {
			continue
		}
		_, written := w.types[key]
		c := &claim{w: w, start: start, end: end, bounded: w.cfg.Bounded, pin: !written}
		w.claims[key] = c
		ch := s.core.channel(key)
		ch.claims = append(ch.claims, c)
	}
	for key, t := range w.types {
		for _, seg := range s.core.channels[key].segs {
			if seg.holder == nil && !seg.vis.Sealed && seg.vis.Clock == w.cfg.Clock &&
				seg.vis.DataType == t && seg.vis.Version == s.cfg.Version &&
				seg.dur.created && seg.dur.length == seg.vis.Length {
				seg.holder = w
				w.held[key] = seg
			}
		}
	}
	s.storeKeysIfChanged()
	return nil
}

func mapsKeys[V any](m map[uint32]V) func(func(uint32) bool) {
	return func(yield func(uint32) bool) {
		for k := range m {
			if !yield(k) {
				return
			}
		}
	}
}

// Held returns the open segment that w holds for key, and false when it holds none.
func (w *Writer) Held(key uint32) (Segment, bool) {
	r := &heldRequest{w: w, key: key}
	if err := w.call(r); err != nil {
		return Segment{}, false
	}
	return r.seg, r.ok
}

type heldRequest struct {
	w   *Writer
	key uint32
	seg Segment
	ok  bool
}

func (r *heldRequest) run(*State) {
	if seg, ok := r.w.held[r.key]; ok {
		r.seg, r.ok = seg.vis, true
	}
	r.w.done <- nil
}

type valueRequest struct {
	w     *Writer
	key   uint32
	value uint64
	align bool
}

func (r *valueRequest) run(s *State) {
	if r.align {
		var err error
		r.value, err = s.giveAlignment()
		r.w.done <- err
		return
	}
	r.w.done <- s.newSegment(r)
}

// NewSegment gives out a segment identity, registers a new open segment of key that w
// holds, and returns its identity. The caller logs its Create.
func (w *Writer) NewSegment(key uint32) (uint64, error) {
	r := &valueRequest{w: w, key: key}
	if err := w.call(r); err != nil {
		return 0, err
	}
	return r.value, nil
}

func (s *State) newSegment(r *valueRequest) error {
	t, ok := r.w.types[r.key]
	switch {
	case !ok:
		return errors.Newf("writer does not write key %d", r.key)
	case r.w.held[r.key] != nil:
		return errors.Newf(
			"writer holds open segment %d of key %d",
			r.w.held[r.key].vis.ID,
			r.key,
		)
	}
	if s.nextID >= s.core.reserved.Segments {
		s.idBlocks++
		s.reserve(block(s.idBlocks)+s.nextID, s.core.reserved.Alignments)
	}
	id := s.nextID
	s.nextID++
	seg := &segState{key: r.key, holder: r.w, vis: Segment{
		ID:       id,
		Clock:    r.w.cfg.Clock,
		Version:  s.cfg.Version,
		DataType: t,
	}}
	s.core.segments[id] = seg
	ch := s.core.channel(r.key)
	ch.insert(seg)
	ch.publish()
	s.storeKeysIfChanged()
	r.w.held[r.key] = seg
	r.value = id
	return nil
}

func block(n int) uint64 {
	if n == 1 {
		return firstBlock
	}
	return laterBlock
}

// reserve raises the reserved limits and appends their Reserve to the log.
func (s *State) reserve(segments, alignments uint64) {
	s.core.reserved = Limits{Segments: segments, Alignments: alignments}
	s.entry.Reset()
	s.entry.Add(&log.Reserve{Segments: segments, Alignments: alignments})
	s.cfg.Append(&s.entry)
}

// NewAlignment gives out the alignment number of a new stretch. It returns
// ErrExhausted when every number below the numbers of unstored data is given out.
func (w *Writer) NewAlignment() (uint32, error) {
	r := &valueRequest{w: w, align: true}
	if err := w.call(r); err != nil {
		return 0, err
	}
	return uint32(r.value), nil
}

func (s *State) giveAlignment() (uint64, error) {
	if s.nextAlign >= s.core.reserved.Alignments {
		if s.core.reserved.Alignments >= uint64(alignment.ZeroLeading) {
			return 0, ErrExhausted
		}
		s.alignBlocks++
		limit := min(s.nextAlign+block(s.alignBlocks), uint64(alignment.ZeroLeading))
		s.reserve(s.core.reserved.Segments, limit)
	}
	n := s.nextAlign
	s.nextAlign++
	return n, nil
}

type commitRequest struct {
	w *Writer
	c *log.Commit
}

func (r *commitRequest) run(s *State) { r.w.done <- s.commit(r.w, r.c) }

// Commit checks c against the claims and the stored data of its keys, and makes its
// samples visible. The caller logs c at its next sync. It returns an error that wraps
// ErrConflict when c overlaps stored data of a key or reaches another writer's claim,
// and an error that wraps ErrRefused when c breaks a value rule.
func (w *Writer) Commit(c *log.Commit) error {
	return w.call(&commitRequest{w: w, c: c})
}

func (s *State) checkVisible(w *Writer, c *log.Commit) error {
	a := telem.Alignment(c.Alignment)
	first, last := telem.TimeStamp(c.First), telem.TimeStamp(c.Last)
	switch {
	case len(c.Slots) == 0:
		return refuse("commit with no slot")
	case c.Samples == 0 || first > last || (c.Samples == 1 && first != last):
		return refuse("commit of %d samples from %d to %d", c.Samples, first, last)
	case a.DomainIndex() == 0 || uint64(a.DomainIndex()) >= s.core.reserved.Alignments:
		return refuse("commit of alignment number %d, which the state did not give out",
			a.DomainIndex())
	case uint64(a.SampleIndex())+uint64(c.Samples)-1 > math.MaxUint32:
		return refuse("commit of %d samples from sample index %d", c.Samples,
			a.SampleIndex())
	}
	for i, slot := range c.Slots {
		seg := s.core.segments[slot.Segment]
		switch {
		case i > 0 && slot.Segment <= c.Slots[i-1].Segment:
			return refuse("commit slot %d names segment %d, not above segment %d", i,
				slot.Segment, c.Slots[i-1].Segment)
		case seg == nil || seg.holder != w || w.held[seg.key] != seg:
			return refuse("commit names segment %d, which the writer does not hold",
				slot.Segment)
		case slot.Length <= seg.vis.Length:
			return refuse("commit gives segment %d length %d, not above %d",
				slot.Segment, slot.Length, seg.vis.Length)
		}
		ch := s.core.channels[seg.key]
		if p, ok := ch.overlaps(first, last, false); ok {
			return errors.Wrapf(ErrConflict, "commit of key %d from %d to %d overlaps "+
				"its stored data from %d to %d", seg.key, first, last, p.First, p.Last)
		}
		for _, o := range ch.claims {
			if o.w != w && !o.pin && o.reaches(first, last) {
				return errors.Wrapf(
					ErrConflict,
					"commit of key %d from %d to %d reaches "+
						"the claim of another writer from %d",
					seg.key,
					first,
					last,
					o.start,
				)
			}
		}
	}
	return nil
}

func (s *State) commit(w *Writer, c *log.Commit) error {
	if err := s.checkVisible(w, c); err != nil {
		return err
	}
	first, last := telem.TimeStamp(c.First), telem.TimeStamp(c.Last)
	for _, cl := range w.claims {
		cl.start = min(cl.start, first)
		if cl.bounded || cl.pin {
			cl.end = max(cl.end, last)
		}
	}
	for _, slot := range c.Slots {
		seg := s.core.segments[slot.Segment]
		seg.vis.Pointers = extend(seg.vis.Pointers, telem.Alignment(c.Alignment), first,
			last, seg.vis.Length, slot.Length, c.Samples, false)
		seg.vis.Length, seg.vis.TailCRC = slot.Length, slot.TailCRC
		s.core.channels[seg.key].publish()
	}
	return nil
}

type sealRequest struct {
	w     *Writer
	key   uint32
	seal  log.Seal
	seals []log.Seal
	back  bool
}

func (r *sealRequest) run(s *State) {
	if r.back {
		r.seals = s.takeBack(r.w)
		r.w.done <- nil
		return
	}
	seg, ok := r.w.held[r.key]
	if !ok {
		r.w.done <- errors.Newf("writer holds no open segment of key %d", r.key)
		return
	}
	r.seal = s.sealVisible(r.w, seg)
	s.core.channels[r.key].publish()
	r.w.done <- nil
}

// sealVisible seals seg in the visible view, takes it from w, and returns its Seal.
func (s *State) sealVisible(w *Writer, seg *segState) log.Seal {
	seg.vis.Sealed, seg.holder = true, nil
	delete(w.held, seg.key)
	return log.Seal{
		Segment: seg.vis.ID,
		Length:  seg.vis.Length,
		TailCRC: seg.vis.TailCRC,
	}
}

// Seal seals the open segment that w holds for key at its committed length, and
// returns the Seal that the caller logs.
func (w *Writer) Seal(key uint32) (log.Seal, error) {
	r := &sealRequest{w: w, key: key}
	if err := w.call(r); err != nil {
		return log.Seal{}, err
	}
	return r.seal, nil
}

// TakeBack takes the samples that no durable record covers out of the visible view of
// every segment that w holds, seals each at its durable length, and returns their
// Seals, which the caller logs. A writer calls it after a failed sync.
func (w *Writer) TakeBack() ([]log.Seal, error) {
	r := &sealRequest{w: w, back: true}
	if err := w.call(r); err != nil {
		return nil, err
	}
	return r.seals, nil
}

func (s *State) takeBack(w *Writer) []log.Seal {
	seals := make([]log.Seal, 0, len(w.held))
	for _, seg := range w.held {
		seg.vis.Pointers = slices.Clone(seg.dur.pointers)
		seg.vis.Length, seg.vis.TailCRC = seg.dur.length, seg.dur.tailCRC
		seals = append(seals, s.sealVisible(w, seg))
		s.core.channels[seg.key].publish()
	}
	sortSeals(seals)
	return seals
}

func sortSeals(seals []log.Seal) {
	slices.SortFunc(
		seals,
		func(a, b log.Seal) int { return cmp.Compare(a.Segment, b.Segment) },
	)
}

type unheldRequest struct {
	seals []log.Seal
	done  chan error
}

func (r *unheldRequest) run(s *State) {
	for _, seg := range s.core.segments {
		if seg.holder == nil && !seg.vis.Sealed {
			seg.vis.Sealed = true
			r.seals = append(r.seals, log.Seal{
				Segment: seg.vis.ID,
				Length:  seg.vis.Length,
				TailCRC: seg.vis.TailCRC,
			})
			s.core.channels[seg.key].publish()
		}
	}
	sortSeals(r.seals)
	r.done <- nil
}

// SealUnheld seals every open segment that no writer holds, and returns their Seals in
// rising order of identity, which the log owner logs at a switch.
func (s *State) SealUnheld() ([]log.Seal, error) {
	r := &unheldRequest{done: make(chan error, 1)}
	if err := s.enqueue(r); err != nil {
		return nil, err
	}
	if err := wait(s, r.done); err != nil {
		return nil, err
	}
	return r.seals, nil
}

type closeRequest struct{ w *Writer }

func (r *closeRequest) run(s *State) {
	for key, c := range r.w.claims {
		ch := s.core.channels[key]
		ch.claims = slices.DeleteFunc(ch.claims, func(o *claim) bool { return o == c })
	}
	for _, seg := range r.w.held {
		seg.holder = nil
	}
	clear(r.w.held)
	clear(r.w.claims)
	r.w.done <- nil
}

// Close releases the claims of w and the open segments it holds, which a later writer
// may take over. Call it once the last entry of w is durable.
func (w *Writer) Close() error { return w.call(&closeRequest{w: w}) }
