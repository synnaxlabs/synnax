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
	"bytes"
	"encoding/binary"
	"io"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
)

// Checks are the checksums of a version 0 segment of Length bytes, one for each block,
// which its check file holds.
type Checks struct {
	// Length is the length of the old file in bytes.
	Length int64
	// Sums holds the checksum of each block, in file order.
	Sums []uint32
}

// barePlace returns the place of the version 0 block at file offset offset.
func (c Config) barePlace(offset int64) place.Place {
	return place.Segment(c.DB, c.ID, uint64(offset), 0, c.DataType, c.Clock)
}

// BlockSum returns the checksum of p, the bytes of block b of the version 0 segment of
// c.
func (c Config) BlockSum(b int64, p []byte) uint32 {
	return c.barePlace(b * BlockSize).Sum(p)
}

// EncodeChecks returns the bytes of the check file of the version 0 segment of c: each
// sum of sums, then the checksum of those sums.
func (c Config) EncodeChecks(sums []uint32) []byte {
	b := make([]byte, 4*len(sums)+4)
	for i, s := range sums {
		binary.LittleEndian.PutUint32(b[4*i:], s)
	}
	n := 4 * len(sums)
	binary.LittleEndian.PutUint32(b[n:], place.Check(c.DB, c.ID, 0, 1).Sum(b[:n]))
	return b
}

// DecodeChecks returns the checks of the version 0 segment of c, of length bytes, from
// b,
// the bytes of its check file. It returns an error that wraps place.ErrCorrupt when b
// does not hold one sum for each block and the checksum of those sums.
func (c Config) DecodeChecks(b []byte, length int64) (*Checks, error) {
	blocks := (length + BlockSize - 1) / BlockSize
	n := 4 * blocks
	if int64(len(b)) != n+4 {
		return nil, errors.Wrapf(place.ErrCorrupt, "check file of segment %d holds %d "+
			"bytes, not %d", c.ID, len(b), n+4)
	}
	if place.Check(c.DB, c.ID, 0, 1).Sum(b[:n]) != binary.LittleEndian.Uint32(b[n:]) {
		return nil, errors.Wrapf(
			place.ErrCorrupt,
			"check file of segment %d fails its "+
				"checksum",
			c.ID,
		)
	}
	sums := make([]uint32, blocks)
	for i := range sums {
		sums[i] = binary.LittleEndian.Uint32(b[4*i:])
	}
	return &Checks{Length: length, Sums: sums}, nil
}

// readBare reads len(p) bytes from offset off of the version 0 segment of c in r, a
// whole block at a time, and checks each block against its sum. It reads a block that
// fails once more, then returns an error that wraps place.ErrCorrupt.
func (c Config) readBare(r io.ReaderAt, p []byte, off int64) error {
	var block, again []byte
	for len(p) > 0 {
		if off >= c.Checks.Length {
			return errors.Wrapf(io.ErrUnexpectedEOF, "read at %d past the length %d "+
				"of segment %d", off, c.Checks.Length, c.ID)
		}
		b := off / BlockSize
		start := b * BlockSize
		size := int(min(BlockSize, c.Checks.Length-start))
		block = append(block[:0], make([]byte, size)...)
		// A whole read can end at the end of the file with io.EOF.
		if n, err := r.ReadAt(block, start); n < size {
			return errors.Wrapf(err, "read of block %d of segment %d", b, c.ID)
		}
		if c.BlockSum(b, block) != c.Checks.Sums[b] {
			again = append(again[:0], make([]byte, size)...)
			if n, err := r.ReadAt(again, start); n < size {
				return errors.Wrapf(err, "read of block %d of segment %d", b, c.ID)
			}
			if bytes.Equal(again, block) || c.BlockSum(b, again) != c.Checks.Sums[b] {
				return errors.Wrapf(place.ErrCorrupt, "block %d of segment %d fails "+
					"its check", b, c.ID)
			}
			block, again = again, block
		}
		n := copy(p, block[off-start:])
		p, off = p[n:], off+int64(n)
	}
	return nil
}
