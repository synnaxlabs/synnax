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
	"context"
	"slices"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// ErrNotDurable is returned by CheckStamps while the clock holds a stamp that a writer
// of data against stored stamps needs, and that is visible but not durable.
var ErrNotDurable = errors.New("stamps are not durable")

type stampWaitRequest struct {
	w    *Writer
	need []telem.Alignment
	park bool
	done chan error
}

func (r *stampWaitRequest) fail(err error) { r.done <- err }

func (r *stampWaitRequest) run(s *State) {
	switch {
	case r.w.pulled != nil:
		r.done <- r.w.pulled
	case s.durableStamps(r.w.cfg.Clock, r.need):
		r.done <- nil
	case !r.park:
		r.done <- ErrNotDurable
	default:
		s.waits = append(s.waits, r)
	}
}

type unwaitRequest struct{ r *stampWaitRequest }

func (*unwaitRequest) fail(error) {}

func (u *unwaitRequest) run(s *State) {
	s.waits = slices.DeleteFunc(
		s.waits,
		func(r *stampWaitRequest) bool { return r == u.r },
	)
}

// WaitStamps waits until the clock of w, a writer that does not write it, holds a
// durable stamp at each alignment of need. It returns the error of the take back when a
// failed sync of the clock's writer took the stamps back, and an error that wraps
// ctx.Err() when ctx ends first.
func (w *Writer) WaitStamps(ctx context.Context, need []telem.Alignment) error {
	r := &stampWaitRequest{w: w, need: need, park: true, done: make(chan error, 1)}
	if err := w.s.enqueue(r); err != nil {
		return err
	}
	select {
	case err := <-r.done:
		return err
	case <-ctx.Done():
		return errors.Join(ctx.Err(), w.s.enqueue(&unwaitRequest{r: r}))
	case <-w.s.exit:
		return wait(w.s, r.done)
	}
}

// CheckStamps is WaitStamps with no wait: it returns ErrNotDurable when the clock does
// not yet hold a durable stamp at each alignment of need.
func (w *Writer) CheckStamps(need []telem.Alignment) error {
	r := &stampWaitRequest{w: w, need: need, done: make(chan error, 1)}
	if err := w.s.enqueue(r); err != nil {
		return err
	}
	return wait(w.s, r.done)
}

// durableStamps reports whether clock holds no stamp at an alignment of need that is
// visible and not durable. A stamp that left the head is in a part, which holds only
// durable stamps.
func (s *State) durableStamps(clock uint32, need []telem.Alignment) bool {
	ch := s.core.channels[clock]
	if ch == nil {
		return true
	}
	for _, a := range need {
		for _, seg := range ch.segs {
			if seg.vis.Part != 0 {
				continue
			}
			number, i := a.DomainIndex(), a.SampleIndex()
			if _, visible := pointerAt(seg.vis.Pointers, number, i); !visible {
				continue
			}
			if _, durable := pointerAt(seg.dur.pointers, number, i); !durable {
				return false
			}
		}
	}
	return true
}

// wakeStampWaits runs the parked stamp waits again.
func (s *State) wakeStampWaits() {
	waits := s.waits
	s.waits = nil
	for _, r := range waits {
		r.run(s)
	}
}

// span is the sample indexes from lo to hi of an alignment number.
type span struct{ number, lo, hi uint32 }

// undurable appends to dst the samples of seg that are visible and not durable. Visible
// samples that are not durable are a suffix of the visible pointers.
func undurable(seg *segState, dst []span) []span {
	for i, p := range seg.vis.Pointers {
		lo := p.Alignment.SampleIndex()
		if i < len(seg.dur.pointers) {
			d := seg.dur.pointers[i]
			if d.Samples == p.Samples {
				continue
			}
			lo = d.end() + 1
		}
		dst = append(dst, span{number: p.Alignment.DomainIndex(), lo: lo, hi: p.end()})
	}
	return dst
}

// pullBack takes back, in the step of the take back of w, the visible data of each
// other writer that pins the clock of w and whose data that is not durable reaches the
// stamps that the take back pulled, taken. Each later call of such a writer returns the
// error of the take back.
func (s *State) pullBack(w *Writer, taken []span) {
	for _, c := range s.core.channels[w.cfg.Clock].claims {
		d := c.w
		if d == nil || d == w || !c.pinned || d.pulled != nil {
			continue
		}
		first, last, ok := d.reaches(taken)
		if !ok {
			continue
		}
		d.pulledSeals = s.takeBack(d)
		d.pulled = errors.Newf("clock %d took back the stamps from %d to %d",
			w.cfg.Clock, first, last)
	}
}

// reaches returns the first and last stamps of the data of w that is not durable, and
// true when that data reaches a stamp of taken.
func (w *Writer) reaches(taken []span) (first, last telem.TimeStamp, ok bool) {
	for _, seg := range w.held {
		for i, p := range seg.vis.Pointers {
			if i < len(seg.dur.pointers) && seg.dur.pointers[i].Samples == p.Samples {
				continue
			}
			if !reachesSpan(taken, p) {
				continue
			}
			if !ok || p.First < first {
				first = p.First
			}
			last, ok = max(last, p.Last), true
		}
	}
	return first, last, ok
}

func reachesSpan(taken []span, p Pointer) bool {
	for _, t := range taken {
		if t.number == p.Alignment.DomainIndex() && t.lo <= p.end() &&
			p.Alignment.SampleIndex() <= t.hi {
			return true
		}
	}
	return false
}

// takeBackPulled returns the Seals of the take back of a writer that a take back of its
// clock pulled, and of any segment that it holds since.
func (s *State) takeBackPulled(w *Writer) []log.Seal {
	seals := slices.Concat(w.pulledSeals, s.takeBack(w))
	w.pulledSeals = nil
	sortSeals(seals)
	return seals
}
