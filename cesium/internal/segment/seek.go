// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package segment

import (
	"encoding/binary"
	"io"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
)

// Seek returns the stream offset where the sample at position pos of a variable-density
// segment starts, or length when pos is samples. length and samples are the committed
// length and sample count of the segment in r. buf is scratch space of at least 16
// bytes, and Seek reads up to len(buf) bytes at once. Seek reads the trailers of a
// logarithmic number of blocks, then walks the sample lengths of at most one block and
// the tail. It checks no checksum, and returns an error that wraps place.ErrCorrupt
// when a trailer or a length does not fit the segment.
func (c Config) Seek(
	r io.ReaderAt,
	buf []byte,
	pos uint32,
	length int64,
	samples uint32,
) (int64, error) {
	if pos > samples {
		return 0, errors.Newf("position %d is past the %d samples of the segment", pos,
			samples)
	}
	if pos == samples {
		return length, nil
	}
	x, p, err := c.seekBlock(r, buf, pos, length/BlockSize, samples)
	if err != nil {
		return 0, err
	}
	return c.walk(r, buf, x, p, pos, length)
}

// seekBlock returns the stream offset and position of a sample that starts at or before
// the sample at pos, in the last block of the n full blocks where such a sample starts.
func (c Config) seekBlock(
	r io.ReaderAt,
	buf []byte,
	pos uint32,
	n int64,
	samples uint32,
) (int64, uint32, error) {
	// first returns the first full block whose first_pos passes ok, or n.
	first := func(ok func(firstPos uint32) bool) (int64, error) {
		lo, hi := int64(0), n
		for lo < hi {
			mid := lo + (hi-lo)/2
			firstPos, _, err := c.trailerAt(r, buf, mid, samples)
			if err != nil {
				return 0, err
			}
			if ok(firstPos) {
				hi = mid
			} else {
				lo = mid + 1
			}
		}
		return lo, nil
	}
	after, err := first(func(fp uint32) bool { return fp > pos })
	if err != nil || n == 0 {
		return 0, 0, err
	}
	if after == 0 {
		return 0, 0, errors.Wrap(place.ErrCorrupt, "segment block 0 counts samples "+
			"before it")
	}
	fp, off, err := c.trailerAt(r, buf, after-1, samples)
	if err != nil || off != NoStart {
		return (after-1)*BlockSize + int64(off), fp, err
	}
	// A sample spans the block, so start at the last start of the block before the run
	// of blocks with no start.
	run, err := first(func(p uint32) bool { return p >= fp })
	if err != nil {
		return 0, 0, err
	}
	if run == 0 {
		return 0, 0, errors.Wrap(place.ErrCorrupt, "segment block 0 starts no sample")
	}
	fp, off, err = c.trailerAt(r, buf, run-1, samples)
	if err == nil && off == NoStart {
		err = errors.Wrapf(place.ErrCorrupt, "segment block %d starts no sample, but "+
			"the block after it counts more", run-1)
	}
	return (run-1)*BlockSize + int64(off), fp, err
}

// trailerAt reads first_pos and first_off of the trailer of full block b.
func (c Config) trailerAt(
	r io.ReaderAt,
	buf []byte,
	b int64,
	samples uint32,
) (firstPos, firstOff uint32, err error) {
	at := c.FileOffset((b+1)*BlockSize) - VariableTrailer
	if _, err = r.ReadAt(buf[:8], at); err != nil {
		return 0, 0, errors.Wrapf(err, "read of the trailer of segment block %d", b)
	}
	firstPos = binary.LittleEndian.Uint32(buf)
	firstOff = binary.LittleEndian.Uint32(buf[4:])
	if firstPos > samples || (firstOff >= BlockSize && firstOff != NoStart) {
		return 0, 0, errors.Wrapf(place.ErrCorrupt, "trailer of segment block %d has "+
			"first_pos %d and first_off %d in a segment of %d samples", b, firstPos,
			firstOff, samples)
	}
	return firstPos, firstOff, nil
}

// walk returns the stream offset of the sample at pos, from the sample at position p
// that starts at stream offset x.
func (c Config) walk(
	r io.ReaderAt,
	buf []byte,
	x int64,
	p, pos uint32,
	length int64,
) (int64, error) {
	var lo, hi int64
	for ; ; p++ {
		if x+lengthSize > length {
			return 0, errors.Wrapf(place.ErrCorrupt, "sample %d at stream offset %d "+
				"starts past the length %d", p, x, length)
		}
		if p == pos {
			break
		}
		if x+lengthSize > hi {
			n := min(int64(len(buf)), length-x)
			if err := c.ReadAt(r, buf[:n], x); err != nil {
				return 0, err
			}
			lo, hi = x, x+n
		}
		x += lengthSize + int64(binary.LittleEndian.Uint32(buf[x-lo:]))
	}
	return x, nil
}
