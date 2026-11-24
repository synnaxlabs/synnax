// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package convert

import (
	"cmp"
	"encoding/binary"
	"slices"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

// stampSize is the size of a stamp.
const stampSize = 8

// clock is an old index: its pointers in time order, with the alignment number and the
// first and last stamp of each. cut is true for a pointer that lost stamps to the bound
// check. A pointer that lost them all stays as a mark with no stamps over its time
// range, so that data samples never take the stamps of a later pointer.
type clock struct {
	key         uint32
	ptrs        []pointer
	nums        []uint32
	first, last []telem.TimeStamp
	cut         []bool
}

// count returns the number of stamps of pointer k.
func (c *clock) count(k int) uint32 { return c.ptrs[k].size / stampSize }

// indexes reads the first and last stamp of each pointer of each index, readWidth
// indexes at a time. Then it gives each pointer the next alignment number, in rising
// order of key and then of time, and adds it.
func (r *reader) indexes() error {
	var clocks []*clock
	for _, ch := range r.channels {
		if ch.IsIndex {
			ptrs := r.ptrs[ch.Key]
			c := &clock{
				key:   ch.Key,
				ptrs:  ptrs,
				first: make([]telem.TimeStamp, len(ptrs)),
				last:  make([]telem.TimeStamp, len(ptrs)),
			}
			r.clocks[ch.Key], clocks = c, append(clocks, c)
		}
	}
	if err := each(len(clocks), func(i int) error {
		h := &handle{fs: r.FS, l: r.L}
		return errors.Join(h.bounds(clocks[i]), h.close())
	}); err != nil {
		return err
	}
	for _, c := range clocks {
		for k, p := range c.ptrs {
			if p.size == 0 {
				c.nums = append(c.nums, 0)
				continue
			}
			c.nums = append(c.nums, uint32(r.next))
			r.add(c.key, p.file, state.Pointer{
				First:     c.first[k],
				Last:      c.last[k],
				Alignment: telem.NewAlignment(uint32(r.next), 0),
				Offset:    p.offset,
				Length:    p.size,
				Samples:   c.count(k),
			})
			r.next++
		}
	}
	return nil
}

// bounds reads the first and last stamp of each pointer of c, in file-offset order
// within each old file, then checks them. A mark takes the bounds of its time range.
func (h *handle) bounds(c *clock) error {
	order := make([]int, len(c.ptrs))
	for k := range order {
		order[k] = k
	}
	slices.SortFunc(order, func(a, b int) int {
		return cmp.Or(
			cmp.Compare(c.ptrs[a].file, c.ptrs[b].file),
			cmp.Compare(c.ptrs[a].offset, c.ptrs[b].offset),
		)
	})
	var err error
	for _, k := range order {
		if c.count(k) == 0 {
			c.first[k], c.last[k] = c.ptrs[k].start, c.ptrs[k].end-1
			continue
		}
		if c.first[k], err = h.stamp(c, k, 0); err != nil {
			return err
		}
		if c.last[k], err = h.stamp(c, k, c.count(k)-1); err != nil {
			return err
		}
	}
	return h.check(c)
}

// check checks the first and last stamp of each pointer of c against its time range.
// A crash can leave zeros inside an old file, and a zero stamp is never inside the
// range. A pointer whose last stamp is outside is cut after its last stamp inside, by a
// binary search. A pointer whose first stamp is outside loses every stamp.
func (h *handle) check(c *clock) error {
	c.cut = make([]bool, len(c.ptrs))
	for k, p := range c.ptrs {
		inside := func(s telem.TimeStamp) bool { return s >= p.start && s < p.end }
		if inside(c.first[k]) && inside(c.last[k]) {
			continue
		}
		name := p.name(c.key)
		c.cut[k] = true
		if !inside(c.first[k]) {
			h.l.Warn(
				"dropped a pointer of an index, whose first stamp is outside its time "+
					"range",
				zap.Uint32("key", c.key),
				zap.String("file", name),
				zap.Uint32("offset", p.offset),
				zap.Uint32("stamps", c.count(k)),
			)
			c.ptrs[k].size, c.first[k], c.last[k] = 0, p.start, p.end-1
			continue
		}
		lo, hi, last := uint32(0), c.count(k)-1, c.first[k]
		for hi-lo > 1 {
			mid := lo + (hi-lo)/2
			s, err := h.stamp(c, k, mid)
			if err != nil {
				return err
			}
			if inside(s) {
				lo, last = mid, s
			} else {
				hi = mid
			}
		}
		h.l.Warn(
			"cut a pointer of an index after its last stamp inside its time range",
			zap.Uint32("key", c.key),
			zap.String("file", name),
			zap.Uint32("offset", p.offset),
			zap.Uint32("stamps", c.count(k)),
			zap.Uint32("kept", lo+1),
		)
		c.ptrs[k].size, c.last[k] = (lo+1)*stampSize, last
	}
	return nil
}

// stamp reads the stamp at position pos of pointer k of c.
func (h *handle) stamp(c *clock, k int, pos uint32) (telem.TimeStamp, error) {
	p := c.ptrs[k]
	b, err := h.agreed(
		p.name(c.key),
		int64(p.offset)+int64(pos)*stampSize,
		stampSize,
	)
	if err != nil {
		return 0, err
	}
	return telem.TimeStamp(binary.LittleEndian.Uint64(b)), nil
}

// search returns the first position from lo in pointer k of c whose stamp is at least
// t, and that stamp. The last stamp of k must be at least t.
func (h *handle) search(
	c *clock,
	k int,
	lo uint32,
	t telem.TimeStamp,
) (uint32, telem.TimeStamp, error) {
	hi, at := c.count(k)-1, c.last[k]
	for lo < hi {
		mid := lo + (hi-lo)/2
		s, err := h.stamp(c, k, mid)
		if err != nil {
			return 0, 0, err
		}
		if s >= t {
			hi, at = mid, s
		} else {
			lo = mid + 1
		}
	}
	return hi, at, nil
}

// datas splits the pointers of each data key, readWidth keys at a time, and adds the
// pieces.
func (r *reader) datas() error {
	var chs []channel.Channel
	for _, ch := range r.channels {
		if !ch.IsIndex {
			chs = append(chs, ch)
		}
	}
	pieces := make([]map[uint16][]state.Pointer, len(chs))
	if err := each(len(chs), func(i int) (err error) {
		h := &handle{fs: r.FS, l: r.L}
		pieces[i], err = h.data(chs[i], r.clocks[chs[i].Index], r.ptrs[chs[i].Key])
		return errors.Join(err, h.close())
	}); err != nil {
		return err
	}
	for i, ch := range chs {
		if len(pieces[i]) > 0 {
			r.segs[ch.Key] = pieces[i]
		}
	}
	return nil
}

// data returns the pieces of ptrs, the pointers of data key ch, by old file number:
// each pointer split at the start of each pointer of c that it crosses.
func (h *handle) data(
	ch channel.Channel,
	c *clock,
	ptrs []pointer,
) (map[uint16][]state.Pointer, error) {
	if c == nil {
		return nil, errors.Newf(
			"key %d is timed by key %d, which is no index of the old layout",
			ch.Key,
			ch.Index,
		)
	}
	var density telem.Density
	if !ch.DataType.IsVariable() {
		density = ch.DataType.Density()
	}
	pieces := make(map[uint16][]state.Pointer)
	for _, d := range ptrs {
		if err := h.split(pieces, ch.Key, density, c, d); err != nil {
			return nil, err
		}
	}
	return pieces, nil
}

// split adds to pieces the pieces of data pointer d of key, one for each pointer of c
// whose stamps time its samples. Sample i of d takes stamp i of c from the first stamp
// at or after the start of d, and every stamp it takes is before the end of d. A
// density of zero is a variable type.
func (h *handle) split(
	pieces map[uint16][]state.Pointer,
	key uint32,
	density telem.Density,
	c *clock,
	d pointer,
) error {
	name, off, end := d.name(key), d.offset, d.offset+d.size
	k, pos, first, err := h.start(c, d)
	if err != nil {
		return err
	}
	for off < end {
		if k == len(c.ptrs) || c.count(k) == 0 {
			h.cut(key, name, off, end, c.key)
			return nil
		}
		p := piece{c: c, k: k, pos: pos, first: first, d: d, density: density}
		if err = h.piece(&p, name, off, end); err != nil {
			return err
		}
		if p.n == 0 {
			h.cut(key, name, off, end, c.key)
			return nil
		}
		n, length, last, short := p.n, p.length, p.last, p.short
		pieces[d.file] = append(pieces[d.file], state.Pointer{
			First:     first,
			Last:      last,
			Alignment: telem.NewAlignment(c.nums[k], pos),
			Offset:    off,
			Length:    length,
			Samples:   n,
		})
		off += length
		// The samples after a piece whose stamps the bound checks cut have none.
		if (short || c.cut[k]) && off < end {
			h.cut(key, name, off, end, c.key)
			return nil
		}
		if k++; k < len(c.ptrs) {
			pos, first = 0, c.first[k]
		}
	}
	return nil
}

// piece is the piece of data pointer d that the stamps of pointer k of c from position
// pos time, whose first stamp is first. piece fills n, length, and last, its samples,
// bytes, and last stamp, and short, which is true when a stamp outside k ended it.
type piece struct {
	c       *clock
	k       int
	pos     uint32
	first   telem.TimeStamp
	d       pointer
	density telem.Density
	n       uint32
	length  uint32
	last    telem.TimeStamp
	short   bool
}

// piece fills p for the samples of the file name from stream offset off to end. It
// leaves p.n zero when no stamp times the sample at off.
func (h *handle) piece(p *piece, name string, off, end uint32) error {
	n, final, err := h.take(p.c, p.k, p.pos, p.d, p.density, off)
	if err != nil || n == 0 {
		return err
	}
	// A whole pointer that ends by 1 ns after its last stamp has a stamp for each
	// sample.
	whole := final && !p.d.cut && p.c.last[p.k] >= p.d.end-1
	if p.n, p.length, err = h.span(name, off, end, n, p.density, whole); err != nil ||
		p.n == 0 {
		return err
	}
	var timed uint32
	if p.last, timed, err = h.lastOf(p.c, p.k, p.pos, p.n, p.first); err != nil ||
		timed == p.n {
		return err
	}
	p.short = true
	p.n, p.length, err = h.span(name, off, end, timed, p.density, false)
	return err
}

// span returns how many of n samples of density at off of the file name end by end, and
// their bytes. A density of zero is a variable type, whose prefixes it walks unless
// whole is true: then the samples take every byte up to end.
func (h *handle) span(
	name string,
	off, end, n uint32,
	density telem.Density,
	whole bool,
) (uint32, uint32, error) {
	switch {
	case density != 0:
		return n, n * uint32(density), nil
	case whole:
		return n, end - off, nil
	}
	return h.walk(name, off, end, n)
}

// lastOf returns the stamp of the last of the n samples from position pos of pointer k
// of c, a piece whose first stamp is first, and how many of the n samples are timed. A
// stamp outside first to the last stamp of k, from a lost block, ends the piece at the
// last stamp inside before it, by a binary search.
func (h *handle) lastOf(
	c *clock,
	k int,
	pos, n uint32,
	first telem.TimeStamp,
) (telem.TimeStamp, uint32, error) {
	if pos+n == c.count(k) {
		return c.last[k], n, nil
	}
	inside := func(s telem.TimeStamp) bool { return s >= first && s <= c.last[k] }
	s, err := h.stamp(c, k, pos+n-1)
	if err != nil || inside(s) {
		return s, n, err
	}
	lo, hi, last := pos, pos+n-1, first
	for hi-lo > 1 {
		mid := lo + (hi-lo)/2
		if s, err = h.stamp(c, k, mid); err != nil {
			return 0, 0, err
		}
		if inside(s) {
			lo, last = mid, s
		} else {
			hi = mid
		}
	}
	return last, lo - pos + 1, nil
}

// take returns how many stamps of pointer k of c, from position pos, time the samples
// of d from stream offset off. final is true when the samples of a variable type end in
// k.
func (h *handle) take(
	c *clock,
	k int,
	pos uint32,
	d pointer,
	density telem.Density,
	off uint32,
) (uint32, bool, error) {
	n, end := c.count(k)-pos, d.offset+d.size
	switch {
	case density != 0 && c.last[k] >= d.end:
		n, err := h.below(c, k, pos, min(n, (end-off)/uint32(density)), d.end)
		return n, false, err
	case density != 0:
		return min(n, (end-off)/uint32(density)), false, nil
	case c.last[k] >= d.end:
		e, _, err := h.search(c, k, pos, d.end)
		return e - pos, true, err
	}
	return n, k+1 == len(c.ptrs) || c.first[k+1] >= d.end, nil
}

// below returns how many of the n stamps from position pos of pointer k of c are before
// t. It reads one stamp when all of them are, and searches otherwise.
func (h *handle) below(
	c *clock,
	k int,
	pos, n uint32,
	t telem.TimeStamp,
) (uint32, error) {
	last, err := h.stamp(c, k, pos+n-1)
	if err != nil || last < t {
		return n, err
	}
	e, _, err := h.search(c, k, pos, t)
	return e - pos, err
}

// start returns the pointer k of c and the position in it of the first stamp at or
// after the start of d, and that stamp. k is len(c.ptrs) when c holds no such stamp, or
// a mark when d starts in the range of a pointer that lost every stamp.
func (h *handle) start(c *clock, d pointer) (int, uint32, telem.TimeStamp, error) {
	k, _ := slices.BinarySearch(c.last, d.start)
	switch {
	case k == len(c.ptrs) || c.count(k) == 0:
		return k, 0, 0, nil
	case c.first[k] >= d.start:
		return k, 0, c.first[k], nil
	}
	pos, first, err := h.search(c, k, 1, d.start)
	return k, pos, first, err
}

// cut logs the cut of bytes off to end of the file name of key, whose samples have no
// stamps in clock.
func (h *handle) cut(key uint32, name string, off, end, clock uint32) {
	h.l.Warn(
		"cut the samples of a data pointer that its index has no stamps for",
		zap.Uint32("key", key),
		zap.String("file", name),
		zap.Uint32("offset", off),
		zap.Uint32("bytes", end-off),
		zap.Uint32("index", clock),
	)
}

// walkChunk is the most bytes that one read of a walk takes.
const walkChunk = 1 << 20

// walk returns how many of the n variable-type samples at off of the file name end by
// end, from their length prefixes, and their bytes.
func (h *handle) walk(name string, off, end, n uint32) (uint32, uint32, error) {
	at, left := int64(off), n
	for left > 0 && int64(end)-at >= 4 {
		b, err := h.agreed(name, at, int(min(walkChunk, int64(end)-at)))
		if err != nil {
			return 0, 0, err
		}
		i := int64(0)
		for left > 0 && i+4 <= int64(len(b)) {
			next := i + 4 + int64(binary.LittleEndian.Uint32(b[i:]))
			if next > int64(len(b)) {
				break
			}
			i, left = next, left-1
		}
		if i == 0 {
			// The sample at at is longer than a read, or passes end.
			if i = 4 + int64(binary.LittleEndian.Uint32(b)); at+i > int64(end) {
				break
			}
			left--
		}
		at += i
	}
	return n - left, uint32(at - int64(off)), nil
}
