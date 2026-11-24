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
	"os"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
)

// Committed is the committed part of a segment that a read covers.
type Committed struct {
	Config
	// Length is the committed length in stream bytes.
	Length int64
	// TailCRC is the running checksum of the unfinished last block at Length.
	TailCRC uint32
}

// Reader reads checked stream bytes of segments. It keeps the last block that it
// checked on its own until Reset, so a later read inside that block reads no bytes
// again. It is not safe for concurrent use.
type Reader struct {
	block []byte
	// held names the checked bytes of block: their segment, the stream offset of the
	// block, and their size. size is zero when block holds none.
	held struct {
		db, id      uint64
		start, size int64
	}
}

// NewReader returns a reader.
func NewReader() *Reader {
	return &Reader{block: make([]byte, BlockSize+VariableTrailer)}
}

// Read reads len(p) stream bytes from stream offset off of the committed segment s in
// r, and checks every block that it reads: a full block against its trailer, and the
// unfinished last block against s.TailCRC. It reads each block with one ReadAt, and
// reads a block that fails its check once more before it decides. A block that fails
// twice, or a file that is missing or shorter than s.Length, returns an error that
// wraps place.ErrCorrupt and names the key, the segment, and the stream offset of the
// block.
func (rd *Reader) Read(r io.ReaderAt, s Committed, p []byte, off int64) error {
	if off < 0 || off+int64(len(p)) > s.Length {
		return errors.Newf(
			"read of %d bytes at stream offset %d is past the committed "+
				"length %d of segment %d",
			len(p),
			off,
			s.Length,
			s.ID,
		)
	}
	for len(p) > 0 {
		start := off - off%BlockSize
		end := min(start+BlockSize, s.Length)
		n := min(int(end-off), len(p))
		if off == start && n == BlockSize && len(p) >= BlockSize+int(s.trailer()) {
			// The block and its trailer land in p, and the next block overwrites the
			// trailer.
			if err := rd.checked(r, s, start, p[:BlockSize+s.trailer()]); err != nil {
				return err
			}
		} else if err := rd.hold(r, s, start, end); err != nil {
			return err
		} else {
			copy(p[:n], rd.block[off-start:])
		}
		p, off = p[n:], off+int64(n)
	}
	return nil
}

// hold makes rd hold the checked block of s from stream offset start to end.
func (rd *Reader) hold(r io.ReaderAt, s Committed, start, end int64) error {
	size := end - start
	if size == BlockSize {
		size += s.trailer()
	}
	if rd.holds(s, start, size) {
		return nil
	}
	rd.held.size = 0
	if err := rd.checked(r, s, start, rd.block[:size]); err != nil {
		return err
	}
	rd.held.db, rd.held.id, rd.held.start, rd.held.size = s.DB, s.ID, start, size
	return nil
}

// Reset drops the block that rd holds, so the next read of it reads and checks it
// again. A caller resets a reader before it reads for another caller.
func (rd *Reader) Reset() { rd.held.size = 0 }

// holds reports whether rd holds the size checked bytes at stream offset start of s.
// Blocks only grow, so a longer checked tail holds a shorter one.
func (rd *Reader) holds(s Committed, start, size int64) bool {
	h := rd.held
	return h.size >= size && h.db == s.DB && h.id == s.ID && h.start == start
}

// checked reads the block at stream offset start into b, which holds the block and its
// trailer, or the unfinished last block, and checks it. It reads once more when the
// first read fails.
func (rd *Reader) checked(r io.ReaderAt, s Committed, start int64, b []byte) error {
	err := rd.readBlock(r, s, start, b)
	if err == nil {
		return nil
	}
	if err = rd.readBlock(r, s, start, b); err == nil {
		return nil
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, os.ErrNotExist) {
		err = errors.Wrapf(place.ErrCorrupt, "%v", err)
	}
	if errors.Is(err, place.ErrCorrupt) {
		return errors.Wrapf(err, "block at stream offset %d of segment %d of key %d "+
			"fails two reads", start, s.ID, s.Key)
	}
	return err
}

func (rd *Reader) readBlock(r io.ReaderAt, s Committed, start int64, b []byte) error {
	fileOff := s.FileOffset(start)
	if _, err := r.ReadAt(b, fileOff); err != nil {
		return err
	}
	data, want := b, s.TailCRC
	if len(b) > BlockSize {
		data = b[:BlockSize]
		want = binary.LittleEndian.Uint32(b[BlockSize+crcOffset(s.Variable):])
	}
	if s.place(fileOff).Sum(data) != want {
		return errors.Wrapf(place.ErrCorrupt, "block at file offset %d fails its check",
			fileOff)
	}
	return nil
}

// crcOffset returns the offset of the crc field in a trailer.
func crcOffset(variable bool) int {
	if variable {
		return 8
	}
	return 0
}
