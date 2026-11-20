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

// Seek returns the stream offset where the sample at position pos of the committed
// variable-density segment s starts, or s.Length when pos is samples, its sample count.
// It reads the trailers of a logarithmic number of blocks, each checked against its
// tcrc, then walks the sample lengths of at most one block and the tail through
// checked reads. Each read fails as Read does, and a trailer that does not fit the
// segment returns an error that wraps place.ErrCorrupt.
func (rd *Reader) Seek(r io.ReaderAt, s Committed, pos, samples uint32) (int64, error) {
	if pos > samples {
		return 0, errors.Newf("position %d is past the %d samples of the segment", pos,
			samples)
	}
	if pos == samples {
		return s.Length, nil
	}
	x, p, err := rd.seekBlock(r, s, pos, s.Length/BlockSize, samples)
	if err != nil {
		return 0, err
	}
	return rd.walk(r, s, x, p, pos)
}

// seekBlock returns the stream offset and position of a sample that starts at or before
// the sample at pos, in the last block of the n full blocks where such a sample starts.
func (rd *Reader) seekBlock(
	r io.ReaderAt,
	s Committed,
	pos uint32,
	n int64,
	samples uint32,
) (int64, uint32, error) {
	// first returns the first full block whose first_pos passes ok, or n.
	first := func(ok func(firstPos uint32) bool) (int64, error) {
		lo, hi := int64(0), n
		for lo < hi {
			mid := lo + (hi-lo)/2
			firstPos, _, err := rd.trailerAt(r, s, mid, samples)
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
	fp, off, err := rd.trailerAt(r, s, after-1, samples)
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
	fp, off, err = rd.trailerAt(r, s, run-1, samples)
	if err == nil && off == NoStart {
		err = errors.Wrapf(place.ErrCorrupt, "segment block %d starts no sample, but "+
			"the block after it counts more", run-1)
	}
	return (run-1)*BlockSize + int64(off), fp, err
}

// trailerAt reads and checks the trailer of full block b, and returns its first_pos and
// first_off. It reads once more when the first read fails.
func (rd *Reader) trailerAt(
	r io.ReaderAt,
	s Committed,
	b int64,
	samples uint32,
) (firstPos, firstOff uint32, err error) {
	at := s.FileOffset((b+1)*BlockSize) - VariableTrailer
	t := rd.block[:VariableTrailer]
	read := func() error {
		if _, err := r.ReadAt(t, at); err != nil {
			return err
		}
		if s.place(at).Sum(t[:12]) != binary.LittleEndian.Uint32(t[12:]) {
			return errors.Wrapf(
				place.ErrCorrupt,
				"trailer at file offset %d fails its "+
					"check",
				at,
			)
		}
		return nil
	}
	if err = read(); err != nil {
		if err = read(); err != nil {
			return 0, 0, errors.Wrapf(
				place.ErrCorrupt,
				"trailer of block %d of segment "+
					"%d of key %d fails two reads: %v",
				b,
				s.ID,
				s.Key,
				err,
			)
		}
	}
	firstPos = binary.LittleEndian.Uint32(t)
	firstOff = binary.LittleEndian.Uint32(t[4:])
	if firstPos > samples || (firstOff >= BlockSize && firstOff != NoStart) {
		return 0, 0, errors.Wrapf(place.ErrCorrupt, "trailer of segment block %d has "+
			"first_pos %d and first_off %d in a segment of %d samples", b, firstPos,
			firstOff, samples)
	}
	return firstPos, firstOff, nil
}

// walk returns the stream offset of the sample at pos, from the sample at position p
// that starts at stream offset x. It reads the lengths through checked reads of a
// window of one block.
func (rd *Reader) walk(
	r io.ReaderAt,
	s Committed,
	x int64,
	p, pos uint32,
) (int64, error) {
	if rd.window == nil {
		rd.window = make([]byte, BlockSize)
	}
	var lo, hi int64
	for ; ; p++ {
		if x+lengthSize > s.Length {
			return 0, errors.Wrapf(place.ErrCorrupt, "sample %d at stream offset %d "+
				"starts past the length %d", p, x, s.Length)
		}
		if p == pos {
			break
		}
		if x+lengthSize > hi {
			n := min(int64(len(rd.window)), s.Length-x)
			if err := rd.Read(r, s, rd.window[:n], x); err != nil {
				return 0, err
			}
			lo, hi = x, x+n
		}
		x += lengthSize + int64(binary.LittleEndian.Uint32(rd.window[x-lo:]))
	}
	return x, nil
}
