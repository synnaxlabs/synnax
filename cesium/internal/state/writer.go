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
	"maps"
	"math"
	"slices"

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
	pinned     bool
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

func (r *openRequest) fail(err error) { r.w.done <- err }

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
			case o.pinned:
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
	keys := append(slices.Collect(maps.Keys(w.types)), w.cfg.Clock)
	for _, key := range keys {
		if _, ok := w.claims[key]; ok {
			continue
		}
		_, written := w.types[key]
		c := &claim{
			w:       w,
			start:   start,
			end:     end,
			bounded: w.cfg.Bounded,
			pinned:  !written,
		}
		w.claims[key] = c
		ch := s.core.channel(key)
		ch.claims = append(ch.claims, c)
	}
	s.takeOver(w)
	s.storeKeysIfChanged()
	return nil
}

// takeOver gives w, for each key it writes, the first open segment that no writer holds
// and whose commits are all durable, with the clock, data type, and version of w.
func (s *State) takeOver(w *Writer) {
	for key, t := range w.types {
		for _, seg := range s.core.channels[key].segs {
			if seg.holder == nil && !seg.vis.Sealed && seg.vis.Clock == w.cfg.Clock &&
				seg.vis.DataType == t && seg.vis.Version == s.cfg.Version &&
				seg.dur.created && seg.dur.length == seg.vis.Length {
				seg.holder = w
				w.held[key] = seg
				break
			}
		}
	}
}

// Held returns the open segment that w holds for key, and false when it holds none. It
// returns the error of a failed or closed state.
func (w *Writer) Held(key uint32) (Segment, bool, error) {
	r := &heldRequest{w: w, key: key}
	if err := w.call(r); err != nil {
		return Segment{}, false, err
	}
	return r.seg, r.ok, nil
}

type heldRequest struct {
	w   *Writer
	key uint32
	seg Segment
	ok  bool
}

func (r *heldRequest) fail(err error) { r.w.done <- err }

func (r *heldRequest) run(*State) {
	if seg, ok := r.w.held[r.key]; ok {
		r.seg, r.ok = seg.vis, true
	}
	r.w.done <- nil
}

type segmentRequest struct {
	w        *Writer
	key      uint32
	dataType uint8
	id       uint64
}

func (r *segmentRequest) fail(err error) { r.w.done <- err }

func (r *segmentRequest) run(s *State) {
	if err := s.checkSegment(r); err != nil {
		r.w.done <- err
		return
	}
	if s.nextID >= s.core.limits.Segments {
		s.park(r)
		return
	}
	s.newSegment(r)
	s.reserveAhead()
	r.w.done <- nil
}

type alignmentRequest struct {
	w      *Writer
	number uint64
}

func (r *alignmentRequest) fail(err error) { r.w.done <- err }

func (r *alignmentRequest) run(s *State) {
	switch {
	case s.nextAlign < s.core.limits.Alignments:
		r.number = s.nextAlign
		s.nextAlign++
		s.reserveAhead()
		r.w.done <- nil
	case s.core.limits.Alignments >= uint64(telem.LeadingDomain):
		r.w.done <- ErrExhausted
	default:
		s.park(r)
	}
}

// NewSegment gives out a segment identity, registers a new open segment of key that w
// holds, and returns its identity. The caller logs its Create.
func (w *Writer) NewSegment(key uint32) (uint64, error) {
	r := &segmentRequest{w: w, key: key}
	if err := w.call(r); err != nil {
		return 0, err
	}
	return r.id, nil
}

func (s *State) checkSegment(r *segmentRequest) error {
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
	r.dataType = t
	return nil
}

func (s *State) newSegment(r *segmentRequest) {
	id := s.nextID
	s.nextID++
	seg := &segState{key: r.key, holder: r.w, vis: Segment{
		ID:       id,
		Clock:    r.w.cfg.Clock,
		Version:  s.cfg.Version,
		DataType: r.dataType,
	}}
	s.core.segments[id] = seg
	ch := s.core.channel(r.key)
	ch.insert(seg)
	ch.publish()
	s.storeKeysIfChanged()
	r.w.held[r.key] = seg
	r.id = id
}

func block(n int) uint64 {
	if n == 1 {
		return firstBlock
	}
	return laterBlock
}

// room returns how many values from next are below limit.
func room(next, limit uint64) uint64 {
	if next >= limit {
		return 0
	}
	return limit - next
}

// reserveAhead appends a Reserve when none is in flight and a kind of value has at most
// half a block left below its durable limit.
func (s *State) reserveAhead() {
	l := s.core.limits
	if s.core.reserved != l {
		return
	}
	next, blocks := l, [2]int{s.idBlocks, s.alignBlocks}
	if room(s.nextID, l.Segments) <= block(s.idBlocks)/2 {
		s.idBlocks++
		next.Segments = max(l.Segments, s.nextID) + block(s.idBlocks)
	}
	zero := uint64(telem.LeadingDomain)
	if l.Alignments < zero &&
		room(s.nextAlign, l.Alignments) <= block(s.alignBlocks)/2 {
		s.alignBlocks++
		next.Alignments = min(max(l.Alignments, s.nextAlign)+block(s.alignBlocks), zero)
	}
	if next == l {
		return
	}
	s.core.reserved, s.refusedBlocks = next, blocks
	s.entry.Reset()
	s.entry.Add(&log.Reserve{Segments: next.Segments, Alignments: next.Alignments})
	s.cfg.Append(&s.entry)
}

// park holds r until a durable Reserve raises the limit that it needs, or the log
// refuses the Reserve.
func (s *State) park(r request) {
	s.parked = append(s.parked, r)
	s.reserveAhead()
}

// unpark runs the parked requests again.
func (s *State) unpark() {
	parked := s.parked
	s.parked = nil
	for _, r := range parked {
		r.run(s)
	}
}

// failParked fails every parked request with err.
func (s *State) failParked(err error) {
	for _, r := range s.parked {
		r.fail(err)
	}
	s.parked = nil
}

type refusedRequest struct {
	records []log.Record
	err     error
}

func (*refusedRequest) fail(error) {}

func (r *refusedRequest) run(s *State) {
	for _, rec := range r.records {
		if _, ok := rec.(*log.Reserve); ok {
			s.core.reserved = s.core.limits
			s.idBlocks, s.alignBlocks = s.refusedBlocks[0], s.refusedBlocks[1]
			s.failParked(r.err)
			return
		}
	}
}

// Refused queues the refusal of an entry of records by the log, with the error err that
// refused it. A refused Reserve fails every request that waits for its values, with
// err. Refused returns at once.
func (s *State) Refused(records []log.Record, err error) error {
	return s.enqueue(&refusedRequest{records: records, err: err})
}

// NewAlignment gives out the alignment number of a new stretch. It returns
// ErrExhausted when every number below the numbers of unstored data is given out.
func (w *Writer) NewAlignment() (uint32, error) {
	r := &alignmentRequest{w: w}
	if err := w.call(r); err != nil {
		return 0, err
	}
	return uint32(r.number), nil
}

type commitRequest struct {
	w *Writer
	c *log.Commit
}

func (r *commitRequest) fail(err error) { r.w.done <- err }

func (r *commitRequest) run(s *State) { r.w.done <- s.commit(r.w, r.c) }

// Commit checks c against the claims and the stored data of its keys, and makes its
// samples visible. The caller logs c at its next sync. It returns an error that wraps
// ErrConflict when c overlaps stored data of a key or reaches another writer's claim,
// and an error that wraps ErrRefused when c breaks a value rule.
func (w *Writer) Commit(c *log.Commit) error {
	return w.call(&commitRequest{w: w, c: c})
}

func (s *State) checkVisible(w *Writer, c *log.Commit) error {
	if err := s.core.checkCommit(c); err != nil {
		return err
	}
	first, last := telem.TimeStamp(c.First), telem.TimeStamp(c.Last)
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
			if o.w != w && !o.pinned && o.reaches(first, last) {
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
		if cl.bounded || cl.pinned {
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
	w    *Writer
	key  uint32
	seal log.Seal
}

func (r *sealRequest) fail(err error) { r.w.done <- err }

type takeBackRequest struct {
	w     *Writer
	seals []log.Seal
}

func (r *takeBackRequest) fail(err error) { r.w.done <- err }

func (r *takeBackRequest) run(s *State) {
	r.seals = s.takeBack(r.w)
	r.w.done <- nil
}

func (r *sealRequest) run(s *State) {
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
	r := &takeBackRequest{w: w}
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

type switchRequest struct{ generation uint64 }

// run seals each open segment that no writer holds, takes the durable snapshot that the
// head of the new log holds, appends the Seals, and hands the snapshot over. An unheld
// open segment that no durable Create made has no durable byte, so it leaves the state
// for the database to unlink.
func (r *switchRequest) run(s *State) {
	var seals []log.Seal
	for id, seg := range s.core.segments {
		if seg.holder != nil || seg.vis.Sealed {
			continue
		}
		if !seg.dur.created {
			s.core.channels[seg.key].remove(id)
			delete(s.core.segments, id)
			s.core.dead = append(s.core.dead, id)
			s.core.channels[seg.key].publish()
			continue
		}
		seg.vis.Sealed = true
		seals = append(seals, log.Seal{
			Segment: id,
			Length:  seg.vis.Length,
			TailCRC: seg.vis.TailCRC,
		})
		s.core.channels[seg.key].publish()
	}
	sortSeals(seals)
	snap := s.core.snapshot()
	if len(seals) > 0 {
		var e log.Entry
		for i := range seals {
			e.Add(&seals[i])
		}
		s.cfg.Append(&e)
	}
	s.cfg.OnHead(r.generation, snap)
}

func (*switchRequest) fail(error) {}

// Switch queues the switch of the state to the log of generation, which the log owner
// asks for right after the last entry of the log before it went to Durable. The owner
// takes the durable snapshot there, so it covers exactly the entries before the
// switch, and hands it to the OnHead hook. It seals every open segment that no writer
// holds, and appends their Seals to the new log. Switch returns at once.
func (s *State) Switch(generation uint64) error {
	return s.enqueue(&switchRequest{generation: generation})
}

type closeRequest struct{ w *Writer }

func (r *closeRequest) fail(err error) { r.w.done <- err }

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
