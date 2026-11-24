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
	"slices"

	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

type splitRequest struct {
	w, d *Writer
	keys []uint32
}

func (r *splitRequest) fail(err error) { r.w.done <- err }

func (r *splitRequest) run(s *State) {
	w, d := r.w, r.d
	if w.pulled != nil {
		w.done <- w.pulled
		return
	}
	for _, key := range r.keys {
		if _, written := w.types[key]; !written || key == w.cfg.Clock {
			w.done <- errors.Newf("writer of clock %d writes no key %d to split off",
				w.cfg.Clock, key)
			return
		}
	}
	for _, key := range r.keys {
		d.types[key] = w.types[key]
		delete(w.types, key)
		c := w.claims[key]
		delete(w.claims, key)
		c.w, d.claims[key] = d, c
		if seg, ok := w.held[key]; ok {
			delete(w.held, key)
			seg.holder, d.held[key] = d, seg
		}
	}
	clock := w.claims[w.cfg.Clock]
	pin := &claim{
		w:       d,
		start:   clock.start,
		end:     clock.end,
		bounded: clock.bounded,
		pinned:  true,
	}
	d.claims[w.cfg.Clock] = pin
	ch := s.core.channel(w.cfg.Clock)
	ch.claims = append(ch.claims, pin)
	w.done <- nil
}

// Split moves keys out of w, with their claims and the open segments that w holds for
// them, to a new writer of data against the stored stamps of the clock of w. The new
// writer pins the clock over the range that w claims. keys must not hold the clock.
func (w *Writer) Split(keys []uint32) (*Writer, error) {
	d := &Writer{
		s: w.s,
		cfg: WriterConfig{
			Clock:   w.cfg.Clock,
			Start:   w.cfg.Start,
			Bounded: w.cfg.Bounded,
		},
		done:   make(chan error, 1),
		claims: map[uint32]*claim{},
		held:   map[uint32]*segState{},
		types:  map[uint32]uint8{},
	}
	if err := w.call(&splitRequest{w: w, d: d, keys: keys}); err != nil {
		return nil, err
	}
	return d, nil
}

type joinRequest struct{ w, d *Writer }

func (r *joinRequest) fail(err error) { r.w.done <- err }

func (r *joinRequest) run(s *State) {
	w, d := r.w, r.d
	switch {
	case w.pulled != nil:
		w.done <- w.pulled
		return
	case d.pulled != nil:
		w.done <- d.pulled
		return
	case d.cfg.Clock != w.cfg.Clock:
		w.done <- errors.Newf("writer of clock %d joins no writer of clock %d",
			w.cfg.Clock, d.cfg.Clock)
		return
	}
	for key := range d.types {
		if _, written := w.types[key]; written {
			w.done <- errors.Newf("writer of clock %d already writes key %d",
				w.cfg.Clock, key)
			return
		}
		if err := s.checkClaims(w, d, key, d.claims[key].start); err != nil {
			w.done <- err
			return
		}
	}
	clock := w.claims[w.cfg.Clock]
	for key, t := range d.types {
		w.types[key] = t
		c := d.claims[key]
		c.w, c.end, c.bounded = w, clock.end, clock.bounded
		w.claims[key] = c
		if seg, ok := d.held[key]; ok {
			seg.holder, w.held[key] = w, seg
		}
	}
	pin := d.claims[d.cfg.Clock]
	ch := s.core.channels[d.cfg.Clock]
	ch.claims = slices.DeleteFunc(ch.claims, func(o *claim) bool { return o == pin })
	clear(d.types)
	clear(d.claims)
	clear(d.held)
	w.done <- nil
}

// Join moves every key of d, a writer of data against the stamps of the clock of w,
// with its claims and open segments, into w, and drops the pin of d on the clock. Each
// claim ends where the claim of w on the clock ends, and Join refuses a claim that then
// reaches the claim of another writer, as OpenWriter does. d takes no later call.
func (w *Writer) Join(d *Writer) error { return w.call(&joinRequest{w: w, d: d}) }

type boundRequest struct {
	w           *Writer
	first, last telem.TimeStamp
}

func (r *boundRequest) fail(err error) { r.w.done <- err }

func (r *boundRequest) run(*State) {
	if r.w.pulled != nil {
		r.w.done <- r.w.pulled
		return
	}
	for _, c := range r.w.claims {
		c.start, c.end, c.bounded = r.first, r.last, true
	}
	r.w.done <- nil
}

// Bound limits each claim of w, and its pin on the clock, to the stamps from first to
// last, so that other writers may claim the keys of w outside them.
func (w *Writer) Bound(first, last telem.TimeStamp) error {
	return w.call(&boundRequest{w: w, first: first, last: last})
}

type addRequest struct {
	w     *Writer
	keys  []WriterKey
	start telem.TimeStamp
	pre   []Preload
}

func (r *addRequest) fail(err error) { r.w.done <- err }

func (r *addRequest) run(s *State) {
	w := r.w
	if w.pulled != nil {
		w.done <- w.pulled
		return
	}
	start := r.start
	for _, k := range r.keys {
		if _, written := w.types[k.Key]; written {
			w.done <- errors.Newf("writer of clock %d already writes key %d",
				w.cfg.Clock, k.Key)
			return
		}
		if err := s.checkOpen(w, k.Key, start, r.pre); err != nil {
			w.done <- err
			return
		}
	}
	for _, k := range r.keys {
		w.types[k.Key] = k.DataType
		s.claim(w, k.Key, start, false)
	}
	s.storeKeysIfChanged()
	w.done <- nil
}

// Add makes w write keys too, with claims from start, after the checks that OpenWriter
// makes at start. pre holds the preload of each key of keys at start, in rising order
// of key. It returns the errors of OpenWriter.
func (w *Writer) Add(keys []WriterKey, start telem.TimeStamp, pre ...Preload) error {
	return w.call(&addRequest{w: w, keys: keys, start: start, pre: pre})
}

type stampedRequest struct {
	w           *Writer
	a           telem.Alignment
	n           uint32
	first, last telem.TimeStamp
	pre         []Preload
	held        bool
}

func (r *stampedRequest) fail(err error) { r.w.done <- err }

func (r *stampedRequest) run(s *State) {
	r.w.done <- s.stamped(r)
}

func (s *State) stamped(r *stampedRequest) error {
	ch := s.core.channels[r.w.cfg.Clock]
	if ch == nil {
		return nil
	}
	records, err := s.core.preloadOf(ch, r.pre)
	if err != nil {
		return err
	}
	if _, ok := ch.pointerAt(r.a.DomainIndex(), r.a.SampleIndex(), records); !ok {
		return nil
	}
	if r.held = ch.holds(r.a, r.n, r.first, r.last, records); !r.held {
		return errors.Newf(
			"clock %d holds other stamps than the %d samples from %d to %d",
			r.w.cfg.Clock,
			r.n,
			r.first,
			r.last,
		)
	}
	return nil
}

// Stamped reports whether the clock of w holds visible stamps for the n samples from
// alignment a, with first and last as the stamps of the first and last. It returns
// false while the clock holds no stamp at a, and an error when it holds other stamps
// there. pre holds the preload of the clock from first to last.
func (w *Writer) Stamped(
	a telem.Alignment,
	n uint32,
	first, last telem.TimeStamp,
	pre ...Preload,
) (bool, error) {
	r := &stampedRequest{w: w, a: a, n: n, first: first, last: last, pre: pre}
	if err := w.call(r); err != nil {
		return false, err
	}
	return r.held, nil
}
