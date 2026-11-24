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
// first and last stamp of each, and the cut of each: the number of its first stamps at
// or below the last stamp of a pointer before it, which a writer wrote again after a
// roll. first is the first stamp past the cut.
type clock struct {
	key         uint32
	ptrs        []pointer
	nums        []uint32
	first, last []telem.TimeStamp
	cut         []uint32
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
		h := &handle{fs: r.FS}
		err := h.bounds(clocks[i])
		if err == nil {
			err = h.cuts(clocks[i])
		}
		return errors.Join(err, h.close())
	}); err != nil {
		return err
	}
	for _, c := range clocks {
		for k, p := range c.ptrs {
			c.nums = append(c.nums, uint32(r.next))
			n := c.cut[k]
			if n > 0 {
				r.warnCut(c, k)
			}
			if n < c.count(k) {
				r.add(c.key, p.file, state.Pointer{
					First:     c.first[k],
					Last:      c.last[k],
					Alignment: telem.NewAlignment(uint32(r.next), n),
					Offset:    p.offset + n*stampSize,
					Length:    p.size - n*stampSize,
					Samples:   c.count(k) - n,
				})
			}
			r.next++
		}
	}
	return nil
}

// cuts sets the cut of each pointer of c whose index range starts after the last stamp
// of the pointers before it while its first stamps do not: those stamps, which a writer
// wrote again in a new file after a roll. It returns an error that names the file of a
// pointer whose first stamps and index range both reach back that far, which no cut
// can place.
func (h *handle) cuts(c *clock) error {
	c.cut = make([]uint32, len(c.ptrs))
	if len(c.ptrs) == 0 {
		return nil
	}
	prev := c.last[0]
	for k := 1; k < len(c.ptrs); k++ {
		switch {
		case c.first[k] > prev:
		case c.ptrs[k].start <= prev:
			return errors.Newf(
				"the index pointer at offset %d of %s holds stamps from %d, at or "+
					"below the last stamp %d before it, inside its own time range",
				c.ptrs[k].offset,
				fileName(c.key, c.ptrs[k].file),
				c.first[k],
				prev,
			)
		case c.last[k] <= prev:
			c.cut[k], c.last[k] = c.count(k), prev
		default:
			n, first, err := h.search(c, k, 0, prev+1)
			if err != nil {
				return err
			}
			c.cut[k], c.first[k] = n, first
		}
		prev = max(prev, c.last[k])
	}
	return nil
}

// warnCut logs the cut of pointer k of c, with the keys that c times.
func (r *reader) warnCut(c *clock, k int) {
	var keys []uint32
	for _, ch := range r.channels {
		if !ch.IsIndex && ch.Index == c.key {
			keys = append(keys, ch.Key)
		}
	}
	r.L.Warn(
		"cut the samples of an old file that repeat the stamps of the file before it",
		zap.Uint32("clock", c.key),
		zap.Uint32s("keys", keys),
		zap.String("file", fileName(c.key, c.ptrs[k].file)),
		zap.Int64("stamp", int64(c.first[k])),
		zap.Uint32("samples", c.cut[k]),
	)
}

// bounds reads the first and last stamp of each pointer of c, in file-offset order
// within each old file.
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
		if c.first[k], err = h.stamp(c, k, 0); err != nil {
			return err
		}
		if c.last[k], err = h.stamp(c, k, c.count(k)-1); err != nil {
			return err
		}
	}
	return nil
}

// stamp reads the stamp at position pos of pointer k of c.
func (h *handle) stamp(c *clock, k int, pos uint32) (telem.TimeStamp, error) {
	p := c.ptrs[k]
	b, err := h.agreed(
		fileName(c.key, p.file),
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
		h := &handle{fs: r.FS}
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
	name, off, end := fileName(key, d.file), d.offset, d.offset+d.size
	k, pos, first, err := h.start(c, d)
	if err != nil {
		return err
	}
	for off < end {
		if k == len(c.ptrs) {
			return noStamps(name, off, end, c.key)
		}
		if pos < c.cut[k] {
			if off, k, pos, first, err = h.pastCut(
				c,
				k,
				pos,
				name,
				density,
				off,
				end,
			); err != nil {
				return err
			}
			continue
		}
		n, final, err := h.take(c, k, pos, d, density, off)
		if err != nil {
			return err
		}
		if n == 0 {
			return noStamps(name, off, end, c.key)
		}
		last := c.last[k]
		if pos+n < c.count(k) {
			if last, err = h.stamp(c, k, pos+n-1); err != nil {
				return err
			}
		}
		length, err := h.length(name, density, off, end, n, final)
		if err != nil {
			return err
		}
		pieces[d.file] = append(pieces[d.file], state.Pointer{
			First:     first,
			Last:      last,
			Alignment: telem.NewAlignment(c.nums[k], pos),
			Offset:    off,
			Length:    length,
			Samples:   n,
		})
		off += length
		if k++; k < len(c.ptrs) {
			pos, first = 0, c.first[k]
		}
	}
	return nil
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
// after the start of d, and that stamp. k is len(c.ptrs) when c holds no such stamp. A
// data pointer that starts where pointer k starts takes the samples of k from its first
// position, those that repeat earlier stamps included.
func (h *handle) start(c *clock, d pointer) (int, uint32, telem.TimeStamp, error) {
	k, _ := slices.BinarySearch(c.last, d.start)
	switch {
	case k == len(c.ptrs):
		return k, 0, 0, nil
	case c.ptrs[k].start == d.start:
		return k, 0, c.first[k], nil
	case c.first[k] >= d.start:
		return k, c.cut[k], c.first[k], nil
	}
	pos, first, err := h.search(c, k, max(1, c.cut[k]), d.start)
	return k, pos, first, err
}

// length returns the bytes of the n samples of the file name at off, which end at end
// when final is true for a variable type, a density of zero.
func (h *handle) length(
	name string,
	density telem.Density,
	off, end, n uint32,
	final bool,
) (uint32, error) {
	switch {
	case density != 0:
		return n * uint32(density), nil
	case final:
		return end - off, nil
	}
	return h.walk(name, off, end, n)
}

// pastCut moves past the samples of the file name from off to end that the cut of
// pointer k of c times from position pos, since they go with the stamps that repeat.
// It returns the offset after them, and the pointer, position, and stamp that follow.
func (h *handle) pastCut(
	c *clock,
	k int,
	pos uint32,
	name string,
	density telem.Density,
	off, end uint32,
) (uint32, int, uint32, telem.TimeStamp, error) {
	skipped, err := h.skip(name, density, off, end, c.cut[k]-pos)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	off, pos = off+skipped, c.cut[k]
	if pos < c.count(k) {
		return off, k, pos, c.first[k], nil
	}
	if k++; k < len(c.ptrs) {
		return off, k, 0, c.first[k], nil
	}
	return off, k, 0, 0, nil
}

// skip returns the bytes of up to n samples of the file name from off to end, a
// density of zero being a variable type.
func (h *handle) skip(
	name string,
	density telem.Density,
	off, end, n uint32,
) (uint32, error) {
	if density != 0 {
		return min(n, (end-off)/uint32(density)) * uint32(density), nil
	}
	at := off
	for range n {
		if at >= end {
			break
		}
		length, err := h.walk(name, at, end, 1)
		if err != nil {
			return 0, err
		}
		at += length
	}
	return at - off, nil
}

// noStamps returns the error of bytes off to end of the file name that hold samples
// with no stamps in clock.
func noStamps(name string, off, end, clock uint32) error {
	return errors.Newf(
		"bytes %d to %d of %s have no stamps in key %d",
		off,
		end,
		name,
		clock,
	)
}

// walkChunk is the most bytes that one read of a walk takes.
const walkChunk = 1 << 20

// walk returns the bytes of the n variable-type samples at off of the file name, from
// their length prefixes. It returns an error when the samples pass end.
func (h *handle) walk(name string, off, end, n uint32) (uint32, error) {
	at := int64(off)
	for left := n; left > 0; {
		if int64(end)-at < 4 {
			return 0, errors.Newf(
				"%d variable samples at %d of %s pass the end %d of their pointer",
				n,
				off,
				name,
				end,
			)
		}
		b, err := h.agreed(name, at, int(min(walkChunk, int64(end)-at)))
		if err != nil {
			return 0, err
		}
		i := int64(0)
		for left > 0 && i+4 <= int64(len(b)) {
			next := i + 4 + int64(binary.LittleEndian.Uint32(b[i:]))
			if next > int64(len(b)) {
				// A sample longer than a read moves the walk past it.
				if i == 0 {
					i, left = next, left-1
				}
				break
			}
			i, left = next, left-1
		}
		at += i
	}
	if at > int64(end) {
		return 0, errors.Newf(
			"%d variable samples at %d of %s pass the end %d of their pointer",
			n,
			off,
			name,
			end,
		)
	}
	return uint32(at - int64(off)), nil
}
