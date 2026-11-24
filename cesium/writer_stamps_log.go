// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"cmp"
	"math"
	"slices"

	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// stampRun is a run of the stored stamps of a clock: the stamps of pointers whose
// alignments touch.
type stampRun struct {
	alignment   telem.Alignment
	samples     uint32
	first, last telem.TimeStamp
}

func (r stampRun) pointer() state.Pointer {
	return state.Pointer{
		First:     r.first,
		Last:      r.last,
		Alignment: r.alignment,
		Samples:   r.samples,
	}
}

// holds reports whether r holds the stamp of the sample at alignment a.
func (r stampRun) holds(a telem.Alignment) bool {
	return a.DomainIndex() == r.alignment.DomainIndex() &&
		a.SampleIndex() >= r.alignment.SampleIndex() &&
		a.SampleIndex()-r.alignment.SampleIndex() < r.samples
}

// stampRuns appends to dst the runs of the visible stamps of clock that end at or after
// from, in rising order of first stamp. A run that a part holds in part counts from its
// first pointer that reaches from.
func (e *engine) stampRuns(
	clock ChannelKey,
	from telem.TimeStamp,
	dst []stampRun,
) ([]stampRun, error) {
	ch, h, err, unlinkErr := e.load(clock, held{})
	if err = errors.Join(err, unlinkErr); err != nil || ch == nil {
		return dst, errors.Join(err, e.unpin(h))
	}
	pre := state.Preload{Key: clock}
	for _, part := range h.parts {
		if part.Last < from {
			continue
		}
		rec, err := e.cache.Records(part, clock)
		if err != nil {
			return dst, errors.Join(err, e.unpin(h))
		}
		addLive(&pre, ch.Parts, part, rec, from, math.MaxInt64)
	}
	if err = e.unpin(h); err != nil {
		return dst, err
	}
	var ptrs []state.Pointer
	for _, seg := range pre.Segments {
		ptrs = append(ptrs, seg.Pointers...)
	}
	for _, seg := range ch.Segments {
		if seg.Part == 0 {
			ptrs = append(ptrs, seg.Pointers...)
		}
	}
	slices.SortFunc(ptrs, func(a, b state.Pointer) int {
		return cmp.Compare(a.Alignment, b.Alignment)
	})
	start := len(dst)
	for _, p := range ptrs {
		if n := len(dst); n > start {
			if r := &dst[n-1]; r.alignment.AddSamples(r.samples) == p.Alignment {
				r.samples += p.Samples
				r.last = p.Last
				continue
			}
		}
		dst = append(dst, stampRun{
			alignment: p.Alignment,
			samples:   p.Samples,
			first:     p.First,
			last:      p.Last,
		})
	}
	runs := slices.DeleteFunc(
		dst[start:],
		func(r stampRun) bool { return r.last < from },
	)
	slices.SortFunc(
		runs,
		func(a, b stampRun) int { return cmp.Compare(a.first, b.first) },
	)
	return dst[:start+len(runs)], nil
}

// placer places the samples of a writer that does not write its clock on the stored
// stamps of the clock, one per sample, from the stamp that equals start on. Samples
// that reach the end of a run go on in the next run in time.
type placer struct {
	e     *engine
	clock ChannelKey
	start telem.TimeStamp
	// cur is the run that holds the stamp of the next sample, next the alignment of
	// that stamp, and placed the number of samples placed. started is false until the
	// first sample has a stamp.
	cur     stampRun
	next    telem.Alignment
	placed  int64
	started bool
	// err is the error of the first sample that had no stored stamp. No later sample
	// gets a stamp.
	err    error
	runs   []stampRun
	pieces []piece
}

// piece is the samples of a write that lie on one run of stored stamps.
type piece struct {
	alignment telem.Alignment
	samples   uint32
	stored    stampRun
}

// place places n samples, and returns the pieces of those that have stored stamps, in
// order. When the pieces hold fewer than n samples, the rest have no stored stamp, and
// err holds the error of the first of them.
func (p *placer) place(n uint32) ([]piece, error) {
	p.pieces = p.pieces[:0]
	for n > 0 && p.err == nil {
		left := p.left()
		if left == 0 {
			if err := p.advance(); err != nil {
				return nil, err
			}
			continue
		}
		k := min(n, left)
		p.pieces = append(p.pieces, piece{alignment: p.next, samples: k, stored: p.cur})
		p.next = p.next.AddSamples(k)
		p.placed += int64(k)
		n -= k
	}
	return p.pieces, nil
}

// left returns the number of stamps of cur from next on.
func (p *placer) left() uint32 {
	if !p.started {
		return 0
	}
	return p.cur.alignment.SampleIndex() + p.cur.samples - p.next.SampleIndex()
}

// advance moves cur to the run that holds the next stamp: the run of start at the first
// sample, cur again when its writer extended it, and otherwise the next run in time. It
// sets err when the clock holds no such stamp.
func (p *placer) advance() error {
	from := p.start
	if p.started {
		from = p.cur.last
	}
	runs, err := p.e.stampRuns(p.clock, from, p.runs[:0])
	p.runs = runs
	if err != nil {
		return err
	}
	if !p.started {
		return p.locate()
	}
	for _, r := range runs {
		if r.holds(p.next) {
			p.cur = r
			return nil
		}
		if r.first > p.cur.last {
			p.cur, p.next = r, r.alignment
			return nil
		}
	}
	p.err = index.NewDiscontinuousOffsetError(p.placed, p.placed)
	return nil
}

// locate places the first sample on the stored stamp that equals start.
func (p *placer) locate() error {
	for _, r := range p.runs {
		if r.first > p.start {
			break
		}
		sr, err := p.e.openStamps(p.clock, r.pointer())
		if err != nil {
			return err
		}
		k, stamp, err := sr.search(p.start, false)
		if err = errors.Join(err, sr.close()); err != nil {
			return err
		}
		if stamp == p.start {
			p.cur, p.next, p.started = r, r.alignment.AddSamples(uint32(k)), true
			return nil
		}
	}
	p.err = index.NewDiscontinuousStampError(p.start)
	return nil
}

// stampEnds returns the stamps of the first and last samples of r, a run of a writer
// that does not write clock.
func (e *engine) stampEnds(
	clock ChannelKey,
	r run,
) (first, last telem.TimeStamp, err error) {
	sr, err := e.openStamps(clock, state.Pointer{
		First:     r.stored.first,
		Last:      r.stored.last,
		Alignment: r.alignment,
		Samples:   r.samples,
	})
	if err != nil {
		return 0, 0, err
	}
	first, err = sr.at(0)
	if err == nil {
		last, err = sr.at(int(r.samples) - 1)
	}
	return first, last, errors.Join(err, sr.close())
}

// writeData writes the n samples of each series of fr, from a writer that does not
// write its clock, on the stored stamps of the clock, except the keys for which skip
// returns true. Each piece of a series that lies on one run of stamps streams as its
// own series with the alignment of its first stamp. The samples with no stored stamp
// stream with no alignment and are never stored.
func (w *logWriter) writeData(
	fr Frame,
	n uint32,
	skip func(ChannelKey) bool,
) (Frame, bool, error) {
	pieces, err := w.stamps.place(n)
	if err != nil {
		return fr, false, err
	}
	keys := fr.RawKeys()
	count := len(keys)
	w.cuts = slices.Grow(w.cuts[:0], count)[:count]
	for i, k := range keys {
		w.cuts[i] = -1
		lk, ok := w.byKey[k]
		if !ok || fr.ShouldExcludeRaw(i) {
			continue
		}
		if lk.skipped || (skip != nil && skip(k)) {
			lk.skipped = true
			w.excluded = append(w.excluded, k)
			continue
		}
		w.cuts[i] = 0
	}
	whole := w.whole[:0]
	for i := range count {
		whole = append(whole, fr.RawSeriesAt(i))
	}
	w.whole = whole
	split := func(j, i, to int, a telem.Alignment) {
		s := whole[i]
		s.Data, s.Alignment = s.Data[w.cuts[i]:to], a
		if j == 0 {
			fr.SetRawSeriesAt(i, s)
		} else {
			fr = fr.Append(keys[i], s)
		}
		w.cuts[i] = to
	}
	for j, p := range pieces {
		w.extendRuns(p)
		for i := range count {
			if w.cuts[i] < 0 {
				continue
			}
			lk, s := w.byKey[keys[i]], whole[i]
			to := w.cuts[i] + sampleBytes(s, w.cuts[i], p.samples)
			if !w.unstored {
				if err = w.append(lk, s.Data[w.cuts[i]:to]); err != nil {
					w.putBack()
					return fr, false, err
				}
			}
			split(j, i, to, p.alignment)
		}
	}
	if w.stamps.err != nil {
		for i := range count {
			if w.cuts[i] >= 0 {
				split(len(pieces), i, len(whole[i].Data), 0)
			}
		}
	}
	return fr.ExcludeKeys(w.excluded), len(w.excluded) > 0, nil
}

// extendRuns adds the samples of p to the open run when their alignments touch, and
// otherwise ends the open run and starts another.
func (w *logWriter) extendRuns(p piece) {
	if n := len(w.runs); n > 0 {
		r := &w.runs[n-1]
		if r.alignment.AddSamples(r.samples) == p.alignment {
			r.samples += p.samples
			r.stored.first = min(r.stored.first, p.stored.first)
			r.stored.last = max(r.stored.last, p.stored.last)
			return
		}
		r.ends = w.ends(r.ends, true)
	}
	w.runs = append(
		w.runs,
		run{alignment: p.alignment, samples: p.samples, stored: p.stored},
	)
}

// sampleBytes returns the number of bytes of the n samples of s that start at byte
// offset from.
func sampleBytes(s telem.Series, from int, n uint32) int {
	if !s.DataType.IsVariable() {
		return int(n) * int(s.DataType.Density())
	}
	at := from
	for range n {
		_, size := telem.UnmarshalVariableSample(s.Data[at:])
		at += size
	}
	return at - from
}
