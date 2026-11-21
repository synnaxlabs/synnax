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
)

// stampSize is the size of a stamp.
const stampSize = 8

// clock is an old index: its pointers in time order, with the alignment number and the
// first and last stamp of each.
type clock struct {
	key         uint32
	ptrs        []pointer
	nums        []uint32
	first, last []telem.TimeStamp
}

// count returns the number of stamps of pointer k.
func (c *clock) count(k int) uint32 { return c.ptrs[k].size / stampSize }

// index gives each pointer of index ch the next alignment number, reads its first and
// last stamp, and adds it. The stamp reads go in file-offset order within each old
// file.
func (r *reader) index(ch channel.Channel) error {
	ptrs := r.ptrs[ch.Key]
	c := &clock{
		key:   ch.Key,
		ptrs:  ptrs,
		first: make([]telem.TimeStamp, len(ptrs)),
		last:  make([]telem.TimeStamp, len(ptrs)),
	}
	r.clocks[ch.Key] = c
	order := make([]int, len(ptrs))
	for k := range order {
		order[k] = k
	}
	slices.SortFunc(order, func(a, b int) int {
		return cmp.Or(
			cmp.Compare(ptrs[a].file, ptrs[b].file),
			cmp.Compare(ptrs[a].offset, ptrs[b].offset),
		)
	})
	var err error
	for _, k := range order {
		if c.first[k], err = r.stamp(c, k, 0); err != nil {
			return err
		}
		if c.last[k], err = r.stamp(c, k, c.count(k)-1); err != nil {
			return err
		}
	}
	for k, p := range ptrs {
		c.nums = append(c.nums, uint32(r.next))
		r.add(ch.Key, p.file, state.Pointer{
			First:     c.first[k],
			Last:      c.last[k],
			Alignment: telem.NewAlignment(uint32(r.next), 0),
			Offset:    p.offset,
			Length:    p.size,
			Samples:   c.count(k),
		})
		r.next++
	}
	return nil
}

// stamp reads the stamp at position pos of pointer k of c.
func (r *reader) stamp(c *clock, k int, pos uint32) (telem.TimeStamp, error) {
	p := c.ptrs[k]
	b, err := r.agreed(
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
func (r *reader) search(
	c *clock,
	k int,
	lo uint32,
	t telem.TimeStamp,
) (uint32, telem.TimeStamp, error) {
	hi, at := c.count(k)-1, c.last[k]
	for lo < hi {
		mid := lo + (hi-lo)/2
		s, err := r.stamp(c, k, mid)
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

// data adds the pointers of data key ch, each split at the start of each index pointer
// that it crosses.
func (r *reader) data(ch channel.Channel) error {
	c := r.clocks[ch.Index]
	if c == nil {
		return errors.Newf(
			"key %d is timed by key %d, which is no index of the old layout",
			ch.Key,
			ch.Index,
		)
	}
	var density telem.Density
	if !ch.DataType.IsVariable() {
		density = ch.DataType.Density()
	}
	for _, d := range r.ptrs[ch.Key] {
		if err := r.split(ch.Key, density, c, d); err != nil {
			return err
		}
	}
	return nil
}

// split adds the pieces of data pointer d of key, one for each pointer of c whose
// stamps time its samples. Sample i of d takes stamp i of c from the first stamp at or
// after the start of d, and every stamp it takes is before the end of d. A density of
// zero is a variable type.
func (r *reader) split(key uint32, density telem.Density, c *clock, d pointer) error {
	name, off, end := fileName(key, d.file), d.offset, d.offset+d.size
	k, pos, first, err := r.start(c, d)
	if err != nil {
		return err
	}
	for off < end {
		if k == len(c.ptrs) {
			return noStamps(name, off, end, c.key)
		}
		n, final, err := r.take(c, k, pos, d, density, off)
		if err != nil {
			return err
		}
		if n == 0 {
			return noStamps(name, off, end, c.key)
		}
		last := c.last[k]
		if pos+n < c.count(k) {
			if last, err = r.stamp(c, k, pos+n-1); err != nil {
				return err
			}
		}
		length := n * uint32(density)
		switch {
		case density == 0 && final:
			length = end - off
		case density == 0:
			if length, err = r.walk(name, off, end, n); err != nil {
				return err
			}
		}
		r.add(key, d.file, state.Pointer{
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
func (r *reader) take(
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
		n, err := r.below(c, k, pos, min(n, (end-off)/uint32(density)), d.end)
		return n, false, err
	case density != 0:
		return min(n, (end-off)/uint32(density)), false, nil
	case c.last[k] >= d.end:
		e, _, err := r.search(c, k, pos, d.end)
		return e - pos, true, err
	}
	return n, k+1 == len(c.ptrs) || c.first[k+1] >= d.end, nil
}

// below returns how many of the n stamps from position pos of pointer k of c are before
// t. It reads one stamp when all of them are, and searches otherwise.
func (r *reader) below(
	c *clock,
	k int,
	pos, n uint32,
	t telem.TimeStamp,
) (uint32, error) {
	last, err := r.stamp(c, k, pos+n-1)
	if err != nil || last < t {
		return n, err
	}
	e, _, err := r.search(c, k, pos, t)
	return e - pos, err
}

// start returns the pointer k of c and the position in it of the first stamp at or
// after the start of d, and that stamp. k is len(c.ptrs) when c holds no such stamp.
func (r *reader) start(c *clock, d pointer) (int, uint32, telem.TimeStamp, error) {
	k, _ := slices.BinarySearch(c.last, d.start)
	switch {
	case k == len(c.ptrs):
		return k, 0, 0, nil
	case c.first[k] >= d.start:
		return k, 0, c.first[k], nil
	}
	pos, first, err := r.search(c, k, 1, d.start)
	return k, pos, first, err
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
func (r *reader) walk(name string, off, end, n uint32) (uint32, error) {
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
		b, err := r.agreed(name, at, int(min(walkChunk, int64(end)-at)))
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
