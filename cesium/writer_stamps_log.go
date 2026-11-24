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
	"context"
	"math"
	"slices"

	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// holds reports whether p holds the stamp of the sample at alignment a.
func holds(p state.Pointer, a telem.Alignment) bool {
	return a.DomainIndex() == p.Alignment.DomainIndex() &&
		a.SampleIndex() >= p.Alignment.SampleIndex() &&
		a.SampleIndex()-p.Alignment.SampleIndex() < p.Samples
}

// stampPointers appends to dst the visible pointers of clock that end at or after from,
// in rising order of first stamp.
func (e *engine) stampPointers(
	clock ChannelKey,
	from telem.TimeStamp,
	dst []state.Pointer,
) ([]state.Pointer, error) {
	ch, h, err := e.load(clock, held{})
	if err != nil || ch == nil {
		e.unpin(h)
		return dst, err
	}
	pre := state.Preload{Key: clock}
	for _, part := range h.parts {
		if part.Last < from {
			continue
		}
		rec, err := e.cache.Records(part, clock)
		if err != nil {
			e.unpin(h)
			return dst, err
		}
		addLive(&pre, ch.Parts, part, rec, from, math.MaxInt64)
	}
	e.unpin(h)
	start := len(dst)
	for _, seg := range pre.Segments {
		dst = append(dst, seg.Pointers...)
	}
	for _, seg := range ch.Segments {
		if seg.Part == 0 {
			dst = append(dst, seg.Pointers...)
		}
	}
	ptrs := slices.DeleteFunc(
		dst[start:],
		func(p state.Pointer) bool { return p.Last < from },
	)
	slices.SortFunc(
		ptrs,
		func(a, b state.Pointer) int { return cmp.Compare(a.First, b.First) },
	)
	return dst[:start+len(ptrs)], nil
}

// placer places the samples of a writer that does not write its clock on the stored
// stamps of the clock, one per sample, from the stamp that equals start on. Samples
// that reach the end of a run go on in the next run in time.
type placer struct {
	e     *engine
	clock ChannelKey
	start telem.TimeStamp
	// cur is the pointer that holds the stamp of the next sample, next the alignment of
	// that stamp, and placed the number of samples placed. started is false until the
	// first sample has a stamp.
	cur     state.Pointer
	next    telem.Alignment
	placed  int64
	started bool
	// err is the error of the first sample that had no stored stamp. No later sample
	// gets a stamp. The samples with no stamp stream with alignment number unstored,
	// from the numbers of data that is not stored, and unstamped counts them.
	err                 error
	unstored, unstamped uint32
	ptrs                []state.Pointer
	pieces              []piece
}

// piece is the samples of a write whose stamps touch, and from is the first stamp of
// the stored pointer that holds the stamp of its first sample.
type piece struct {
	alignment telem.Alignment
	samples   uint32
	from      telem.TimeStamp
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
		p.extend(k)
		p.next = p.next.AddSamples(k)
		p.placed += int64(k)
		n -= k
	}
	return p.pieces, nil
}

// extend adds k samples from next to the last piece when they touch it, as after a
// reload that finds the writer of the clock extended cur, and otherwise starts a piece.
func (p *placer) extend(k uint32) {
	if n := len(p.pieces); n > 0 {
		last := &p.pieces[n-1]
		if last.alignment.AddSamples(last.samples) == p.next {
			last.samples += k
			return
		}
	}
	p.pieces = append(p.pieces, piece{
		alignment: p.next,
		samples:   k,
		from:      p.cur.First,
	})
}

// left returns the number of stamps of cur from next on.
func (p *placer) left() uint32 {
	if !p.started {
		return 0
	}
	return p.cur.Alignment.SampleIndex() + p.cur.Samples - p.next.SampleIndex()
}

// advance moves cur to the pointer that holds the next stamp: the pointer of start at
// the first sample, cur again when its writer extended it, and otherwise the next
// pointer in time. It sets err when the clock holds no such stamp.
func (p *placer) advance() error {
	from := p.start
	if p.started {
		from = p.cur.Last
	}
	ptrs, err := p.e.stampPointers(p.clock, from, p.ptrs[:0])
	p.ptrs = ptrs
	if err != nil {
		return err
	}
	if !p.started {
		return p.locate()
	}
	for _, ptr := range ptrs {
		if holds(ptr, p.next) {
			p.cur = ptr
			return nil
		}
		if ptr.First > p.cur.Last {
			p.cur, p.next = ptr, ptr.Alignment
			return nil
		}
	}
	p.err = index.NewDiscontinuousOffsetError(p.placed, p.placed)
	return nil
}

// locate places the first sample on the stored stamp that equals start.
func (p *placer) locate() error {
	for _, ptr := range p.ptrs {
		if ptr.First > p.start {
			break
		}
		sr, err := p.e.openStamps(p.clock, ptr)
		if err != nil {
			return err
		}
		k, stamp, err := sr.search(p.start, false)
		if err = errors.Join(err, sr.close()); err != nil {
			return err
		}
		if stamp == p.start {
			p.cur, p.next, p.started = ptr, ptr.Alignment.AddSamples(uint32(k)), true
			return nil
		}
	}
	p.err = index.NewDiscontinuousStampError(p.start)
	return nil
}

// stampEnds returns the stamps of the first and last samples of r, a run of a writer
// that does not write clock, whose stamps lie at or before to.
func (e *engine) stampEnds(
	clock ChannelKey,
	r run,
	to telem.TimeStamp,
) (first, last telem.TimeStamp, err error) {
	sr, err := e.openStamps(clock, state.Pointer{
		First:     r.from,
		Last:      to,
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
// stream with one alignment number of data that is not stored, and are never stored.
func (w *logWriter) writeData(
	ctx context.Context,
	fr Frame,
	n uint32,
	skip func(ChannelKey) bool,
) (Frame, bool, error) {
	pieces, err := w.stamps.place(n)
	if err != nil {
		return fr, false, err
	}
	keys, whole := fr.RawKeys(), w.cut(fr, skip)
	count := len(keys)
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
	left := n
	for j, p := range pieces {
		w.extendRuns(p)
		left -= p.samples
		for i := range count {
			if w.cuts[i] < 0 {
				continue
			}
			lk, s := w.byKey[keys[i]], whole[i]
			to := w.cuts[i] + sampleBytes(s, w.cuts[i], p.samples)
			if !w.unstored {
				if err = w.append(ctx, lk, s.Data[w.cuts[i]:to]); err != nil {
					w.putBack()
					return fr, false, err
				}
			}
			split(j, i, to, p.alignment)
		}
	}
	if left > 0 {
		a, err := w.stamps.unstoredAt(left)
		if err != nil {
			return fr, false, err
		}
		for i := range count {
			if w.cuts[i] >= 0 {
				split(len(pieces), i, len(whole[i].Data), a)
			}
		}
	}
	return fr.ExcludeKeys(w.excluded), len(w.excluded) > 0, nil
}

// cut sets cuts to 0 for each series of fr that the writer writes, and to -1 for the
// others and for the keys for which skip returns true, which it excludes. It returns
// the series of fr.
func (w *logWriter) cut(fr Frame, skip func(ChannelKey) bool) []telem.Series {
	w.cuts, w.whole = w.cuts[:0], w.whole[:0]
	for i, k := range fr.RawKeys() {
		w.whole = append(w.whole, fr.RawSeriesAt(i))
		lk, ok := w.byKey[k]
		switch {
		case !ok || fr.ShouldExcludeRaw(i):
			w.cuts = append(w.cuts, -1)
		case lk.skipped || (skip != nil && skip(k)):
			lk.skipped = true
			w.excluded = append(w.excluded, k)
			w.cuts = append(w.cuts, -1)
		default:
			w.cuts = append(w.cuts, 0)
		}
	}
	return w.whole
}

// unstoredAt returns the alignment of the next k samples with no stored stamp, in one
// number of data that is not stored for the writer.
func (p *placer) unstoredAt(k uint32) (telem.Alignment, error) {
	if p.unstored == 0 {
		a, err := p.e.unstored.Next()
		if err != nil {
			return 0, err
		}
		p.unstored = a.DomainIndex()
	}
	a := telem.NewAlignment(p.unstored, p.unstamped)
	p.unstamped += k
	return a, nil
}

// extendRuns adds the samples of p to the open run when their alignments touch, and
// otherwise ends the open run and starts another.
func (w *logWriter) extendRuns(p piece) {
	if n := len(w.runs); n > 0 {
		r := &w.runs[n-1]
		if r.alignment.AddSamples(r.samples) == p.alignment {
			r.samples += p.samples
			return
		}
		r.ends = w.ends(r.ends, true)
	}
	w.runs = append(
		w.runs,
		run{alignment: p.alignment, samples: p.samples, from: p.from},
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
