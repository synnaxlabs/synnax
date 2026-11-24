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
	"encoding/binary"
	"io"
	"os"
	"slices"
	"strconv"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// The reasons of the ranges that a repair does not place.
const (
	twoClocksReason = "its clock has more than one segment with bytes in the gap"
	noClockReason   = "its clock has no bytes in the gap"
	openReason      = "no later record ends it, so its count is not known"
	countReason     = "it holds another number of samples than its clock"
	riseReason      = "the stamps of its clock do not rise inside the pointers " +
		"around them"
	prefixReason = "past the end of the samples that every range of its clock write " +
		"holds in checked blocks"
	cutReason    = "from a block that fails its check"
	tailReason   = "the last block of a lost commit, whose checksum was in the bad frame"
	lengthReason = "a later record ends it before its length before the damage"
	walkReason   = "its samples do not end at the length of the later Commit"
)

// span is the bytes of a segment in the gap: from its committed length before the
// damage on.
type span struct {
	key ChannelKey
	cfg segment.Config
	// start and end bound the bytes of the span in checked blocks, which hold samples
	// samples.
	start, end, samples int64
	// closed is true when a later Commit or Seal ends the span, which then holds want
	// samples up to bound past any block that fails.
	closed      bool
	want, bound int64
	// tail is the number of stream bytes past end, in a last block that nothing checks.
	tail int64
	// prev is the last stamp of the pointers of the segment, or TimeStampMin, and next
	// the first stamp of the later Commit that ends it, or TimeStampMax.
	prev, next telem.TimeStamp
	// reason says why the span holds nothing to place, or is empty.
	reason string
}

func (s span) clock() bool { return s.cfg.Clock == s.key }

func (s span) rangeOf(samples, end int64, reason string) Range {
	return Range{
		Key:     s.key,
		Segment: s.cfg.ID,
		Bytes:   telem.Size(end - s.start),
		Samples: samples,
		Open:    !s.closed,
		Reason:  reason,
	}
}

// group is a clock write that a repair rebuilds: the span of a clock, the spans of the
// data that it times, and the samples that each of them holds.
type group struct {
	clock   span
	data    []span
	samples int64
	// reason says why the group places nothing, or is empty.
	reason string
	// ends holds the end of the samples of each span, the clock first.
	ends      []int64
	timeRange telem.TimeRange
}

// plan is what a repair rebuilds of the gap: the span of each segment with bytes in it,
// the clock writes of the spans, and the ranges that the report lists.
type plan struct {
	spans                    []span
	groups                   []group
	rebuilt, unplaced, tails []Range
}

// rebuild returns the plan of the gap: the clock writes that its bytes hold, the ranges
// that it places, those that it does not, and the tails that nothing checks.
func (e *engine) rebuild(l *later, placed []placement) (plan, error) {
	spans, err := e.spans(l, placed)
	if err != nil {
		return plan{}, err
	}
	var groups []group
	var rebuilt []Range
	clocks, unplaced, tails := listSpans(spans)
	members, more := pairData(spans, clocks)
	unplaced = append(unplaced, more...)
	for _, cs := range clocks {
		if len(cs) > 1 {
			for _, c := range cs {
				unplaced = append(
					unplaced,
					c.rangeOf(c.samples, c.end, twoClocksReason),
				)
			}
			continue
		}
		g, err := e.groupOf(cs[0], members[cs[0].cfg.ID])
		if err != nil {
			return plan{}, err
		}
		r, u := g.ranges()
		if g.reason != "" {
			unplaced = append(unplaced, u...)
			continue
		}
		groups, rebuilt, unplaced = append(groups, g), append(rebuilt, r...),
			append(unplaced, u...)
	}
	for _, list := range [][]Range{rebuilt, unplaced, tails} {
		slices.SortFunc(list, func(a, b Range) int {
			return cmp.Or(cmp.Compare(a.Key, b.Key), cmp.Compare(a.Segment, b.Segment))
		})
	}
	slices.SortFunc(groups, func(a, b group) int {
		return cmp.Compare(a.clock.cfg.ID, b.clock.cfg.ID)
	})
	return plan{
		spans:    spans,
		groups:   groups,
		rebuilt:  rebuilt,
		unplaced: unplaced,
		tails:    tails,
	}, nil
}

// listSpans returns the spans of clocks of spans that hold samples by key, and lists
// the
// tails of spans and the bytes of spans that it does not place.
func listSpans(spans []span) (clocks map[ChannelKey][]span, unplaced, tails []Range) {
	clocks = make(map[ChannelKey][]span)
	for _, s := range spans {
		if s.tail > 0 {
			tails = append(tails, Range{
				Key:     s.key,
				Segment: s.cfg.ID,
				Bytes:   telem.Size(s.tail),
				Open:    true,
				Reason:  tailReason,
			})
		}
		if s.closed && s.reason == "" && s.want > s.samples {
			unplaced = append(unplaced, Range{
				Key:     s.key,
				Segment: s.cfg.ID,
				Bytes:   telem.Size(s.bound - s.end),
				Samples: s.want - s.samples,
				Reason:  cutReason,
			})
		}
		if s.reason != "" {
			unplaced = append(unplaced, s.rangeOf(s.samples, s.end, s.reason))
		} else if s.clock() && s.samples > 0 {
			clocks[s.key] = append(clocks[s.key], s)
		}
	}
	return clocks, unplaced, tails
}

// pairData returns the spans of data of spans that pair with the span of their clock in
// clocks, by the segment of the clock, and lists those that do not.
func pairData(
	spans []span,
	clocks map[ChannelKey][]span,
) (members map[uint64][]span, unplaced []Range) {
	members = make(map[uint64][]span)
	for _, s := range spans {
		if s.reason != "" || s.clock() || s.samples == 0 {
			continue
		}
		reason := ""
		switch cs := clocks[s.cfg.Clock]; {
		case len(cs) == 0:
			reason = noClockReason
		case len(cs) > 1:
			reason = twoClocksReason
		case !cs[0].closed || !s.closed:
			reason = openReason
		case cs[0].want != s.want:
			reason = countReason
		default:
			members[cs[0].cfg.ID] = append(members[cs[0].cfg.ID], s)
		}
		if reason != "" {
			unplaced = append(unplaced, s.rangeOf(s.samples, s.end, reason))
		}
	}
	return members, unplaced
}

// groupOf returns the clock write of the span of clock c and the spans of data that
// pair with it: as many samples as each of them holds in checked blocks, when the
// stamps of the clock rise inside the pointers around them, or none.
func (e *engine) groupOf(c span, data []span) (group, error) {
	g := group{clock: c, data: data, samples: c.samples}
	for _, d := range data {
		g.samples = min(g.samples, d.samples)
	}
	if g.samples == 0 {
		g.reason = prefixReason
		return g, nil
	}
	first, last, rising, err := e.clockStamps(c, g.samples)
	if err != nil || !rising {
		g.samples, g.reason = 0, riseReason
		return g, err
	}
	g.timeRange = telem.TimeRange{Start: first, End: last + 1}
	for _, s := range append([]span{c}, data...) {
		end, err := e.endOf(s, g.samples)
		if err != nil {
			return group{}, err
		}
		g.ends = append(g.ends, end)
	}
	return g, nil
}

// ranges returns the rebuilt ranges of g, and the ranges of its spans that it does not
// place.
func (g group) ranges() (rebuilt, unplaced []Range) {
	spans := append([]span{g.clock}, g.data...)
	for i, s := range spans {
		if g.reason != "" {
			unplaced = append(unplaced, s.rangeOf(s.samples, s.end, g.reason))
			continue
		}
		r := s.rangeOf(g.samples, g.ends[i], "")
		r.TimeRange = g.timeRange
		rebuilt = append(rebuilt, r)
		if s.samples > g.samples {
			rest := s.rangeOf(s.samples-g.samples, s.end, prefixReason)
			rest.Bytes = telem.Size(s.end - g.ends[i])
			unplaced = append(unplaced, rest)
		}
	}
	return rebuilt, unplaced
}

// spans returns the span of each segment with bytes in the gap: each open segment of
// the state with bytes past its committed length, and each placed orphan. A segment
// that a later Replace retires has none, since the copy holds its bytes.
func (e *engine) spans(l *later, placed []placement) ([]span, error) {
	keys, err := e.keyDirs()
	if err != nil {
		return nil, err
	}
	var out []span
	add := func(key ChannelKey, cfg segment.Config, start int64, prev telem.TimeStamp) error {
		s, ok, err := e.span(l, key, cfg, start, prev)
		if ok {
			out = append(out, s)
		}
		return err
	}
	for _, key := range keys {
		for _, seg := range e.state.ReplaySegments(key) {
			if seg.Sealed || l.replaced.Contains(seg.ID) {
				continue
			}
			prev := telem.TimeStampMin
			if n := len(seg.Pointers); n > 0 {
				prev = seg.Pointers[n-1].Last
			}
			err = add(key, e.segmentConfig(seg), int64(seg.Length), prev)
			if err != nil {
				return nil, err
			}
		}
	}
	for _, p := range placed {
		if err = add(p.key, p.cfg, 0, telem.TimeStampMin); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// span returns the span of the segment of key with cfg from start, and false when the
// segment has no byte past start.
func (e *engine) span(
	l *later,
	key ChannelKey,
	cfg segment.Config,
	start int64,
	prev telem.TimeStamp,
) (s span, ok bool, err error) {
	f, err := e.fs.Open(segmentName(key, cfg.ID, 0), os.O_RDONLY)
	if err != nil {
		return span{}, false, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	info, err := f.Stat()
	if err != nil {
		return span{}, false, err
	}
	s = span{
		key:   key,
		cfg:   cfg,
		start: start,
		end:   start,
		prev:  prev,
		next:  telem.TimeStampMax,
	}
	length, tailCRC := cfg.StreamLength(info.Size()), uint32(0)
	c := l.named[cfg.ID]
	if end, crc, ok := l.end(cfg.ID); ok {
		length, tailCRC, s.closed = end, crc, true
	}
	if c != nil {
		s.next = telem.TimeStamp(c.First)
	}
	switch {
	case length < start:
		s.reason = lengthReason
		return s, true, nil
	case length == start:
		return s, false, nil
	}
	if s, err = s.measure(f, length, tailCRC, c); err != nil {
		return span{}, false, err
	}
	return s, true, nil
}

// measure returns s with its end, its counts, and its tail, from the bytes of its
// segment in f up to length, the length at the later Commit c when c is not nil.
func (s span) measure(
	f io.ReaderAt,
	length int64,
	tailCRC uint32,
	c *log.Commit,
) (span, error) {
	r := &streamReader{cfg: s.cfg, f: f, length: length}
	checked, err := checkedEnd(f, s.cfg, s.start, length, tailCRC, s.closed)
	if err != nil {
		return span{}, err
	}
	end := length
	if c != nil {
		if end, err = r.before(s.start, length, int64(c.Samples)); err != nil {
			return span{}, err
		}
		if end < 0 {
			s.reason, s.end = walkReason, length
			return s, nil
		}
	}
	if s.closed {
		if s.want, s.bound, err = r.count(s.start, end); err != nil {
			return span{}, err
		}
	} else {
		s.tail = length - checked
	}
	s.samples, s.end, err = r.count(s.start, min(end, checked))
	return s, err
}

// end returns the length and the tail checksum of segment at the first Commit after the
// damage that names it, or at its first Seal, and false when neither names it.
func (l *later) end(segment uint64) (int64, uint32, bool) {
	if c := l.named[segment]; c != nil {
		slot := c.Slots[slices.IndexFunc(c.Slots, func(s log.Slot) bool {
			return s.Segment == segment
		})]
		return int64(slot.Length), slot.TailCRC, true
	}
	if seal := l.sealed[segment]; seal != nil {
		return int64(seal.Length), seal.TailCRC, true
	}
	return 0, 0, false
}

// checkedEnd returns the end of the bytes from start to length of the segment in f with
// cfg whose blocks pass their checks: the last block against tailCRC when closed, and
// otherwise only full blocks.
func checkedEnd(
	f io.ReaderAt,
	cfg segment.Config,
	start, length int64,
	tailCRC uint32,
	closed bool,
) (int64, error) {
	buf := make([]byte, cfg.FileOffset(segment.BlockSize))
	end := start
	for k := start / segment.BlockSize; k*segment.BlockSize < length; k++ {
		limit := length
		if (k+1)*segment.BlockSize <= length {
			limit = (k + 1) * segment.BlockSize
		} else if !closed {
			return end, nil
		}
		_, err := cfg.ReadChecked(f, buf, k, length, tailCRC)
		if errors.Is(err, place.ErrCorrupt) {
			return end, nil
		}
		if err != nil {
			return 0, err
		}
		end = limit
	}
	return end, nil
}

// clockStamps returns the first and the last of the first n stamps of the span of clock
// c,
// and whether they rise inside the pointers around c.
func (e *engine) clockStamps(
	c span,
	n int64,
) (first, last telem.TimeStamp, rising bool, err error) {
	f, err := e.fs.Open(segmentName(c.key, c.cfg.ID, 0), os.O_RDONLY)
	if err != nil {
		return 0, 0, false, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	r := &streamReader{cfg: c.cfg, f: f, length: c.end}
	prev := c.prev
	for i := range n {
		b, err := r.at(c.start+i*8, 8)
		if err != nil {
			return 0, 0, false, err
		}
		stamp := telem.TimeStamp(binary.LittleEndian.Uint64(b))
		if stamp <= prev && (i > 0 || prev != telem.TimeStampMin) {
			return 0, 0, false, nil
		}
		if i == 0 {
			first = stamp
		}
		prev = stamp
	}
	return first, prev, prev < c.next, nil
}

// endOf returns the end of the first n samples of s.
func (e *engine) endOf(s span, n int64) (end int64, err error) {
	if !s.cfg.Variable {
		return s.start + n*density(s.cfg), nil
	}
	f, err := e.fs.Open(segmentName(s.key, s.cfg.ID, 0), os.O_RDONLY)
	if err != nil {
		return 0, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	r := &streamReader{cfg: s.cfg, f: f, length: s.end}
	end = s.start
	for range n {
		if end, err = r.next(end); err != nil {
			return 0, err
		}
	}
	return end, nil
}

// density returns the bytes of a sample of a fixed-density segment with cfg.
func density(cfg segment.Config) int64 {
	return int64(telem.DataTypeCode(cfg.DataType).DataType().Density())
}

// streamReader reads the stream bytes of a segment up to length through a buffer of up
// to one block.
type streamReader struct {
	cfg    segment.Config
	f      io.ReaderAt
	length int64
	buf    []byte
	// off is the stream offset of buf.
	off int64
}

// at returns the n stream bytes at off, valid until the next call. It returns an error
// that wraps io.ErrUnexpectedEOF for bytes past the length.
func (r *streamReader) at(off int64, n int) ([]byte, error) {
	if off+int64(n) > r.length {
		return nil, errors.Wrapf(io.ErrUnexpectedEOF, "read of %d bytes at %d, past "+
			"%d", n, off, r.length)
	}
	if off < r.off || off+int64(n) > r.off+int64(len(r.buf)) {
		size := max(int64(n), min(segment.BlockSize, r.length-off))
		r.buf = slices.Grow(r.buf[:0], int(size))[:size]
		if err := r.cfg.ReadAt(r.f, r.buf, off); err != nil {
			r.buf = r.buf[:0]
			return nil, err
		}
		r.off = off
	}
	return r.buf[off-r.off : off-r.off+int64(n)], nil
}

// next returns the end of the variable-density sample at off, or a value past the
// length when its length does not fit.
func (r *streamReader) next(off int64) (int64, error) {
	if off+4 > r.length {
		return r.length + 1, nil
	}
	b, err := r.at(off, 4)
	if err != nil {
		return 0, err
	}
	return off + 4 + int64(binary.LittleEndian.Uint32(b)), nil
}

// count returns the number of whole samples from start to at most end, and their end.
func (r *streamReader) count(start, end int64) (int64, int64, error) {
	if !r.cfg.Variable {
		d := density(r.cfg)
		n := (end - start) / d
		return n, start + n*d, nil
	}
	var n int64
	for at := start; at < end; n++ {
		next, err := r.next(at)
		if err != nil || next > end {
			return n, at, err
		}
		at = next
	}
	return n, end, nil
}

// before returns where the last n samples before length start, from samples that start
// at start, or -1 when the samples from start do not end at length.
func (r *streamReader) before(start, length, n int64) (int64, error) {
	if !r.cfg.Variable {
		at := length - n*density(r.cfg)
		if at < start {
			return -1, nil
		}
		return at, nil
	}
	var starts []int64
	at := start
	for at < length {
		starts = append(starts, at)
		next, err := r.next(at)
		if err != nil {
			return 0, err
		}
		at = next
	}
	if at != length || int64(len(starts)) < n {
		return -1, nil
	}
	if n == 0 {
		return length, nil
	}
	return starts[int64(len(starts))-n], nil
}

// keyDirs returns the keys of the channel directories of the root, in rising order.
func (e *engine) keyDirs() ([]ChannelKey, error) {
	dirs, err := e.fs.List("")
	if err != nil {
		return nil, err
	}
	var keys []ChannelKey
	for _, dir := range dirs {
		key, err := strconv.ParseUint(dir.Name(), 10, 32)
		if dir.IsDir() && err == nil {
			keys = append(keys, ChannelKey(key))
		}
	}
	slices.Sort(keys)
	return keys, nil
}
