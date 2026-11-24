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

// Reader reads checked stream bytes of segments. It holds the last block that it read
// for part of a block until Reset, so a later read inside that block reads no bytes
// again. With a cache, a probe takes its full block from the cache when the cache holds
// it, and adds the block once it checks it. A probe is a read inside one block that
// does not go on from the last read of the reader. Other reads, such as scans, read
// past the cache. A Reader is not safe for concurrent use.
type Reader struct {
	cache *Cache
	// end is where the last read ended: its segment, and the stream offset after it.
	end struct {
		id  uint64
		off int64
	}
	// buf is what the next read from the disk fills, or nil until the reader needs one.
	buf []byte
	// block holds the checked bytes that held names: buf, or the bytes of cached.
	block []byte
	// cached is the block of the cache that the reader holds, or nil.
	cached *block
	// held names the checked bytes of block: their segment, the stream offset of the
	// block, and their size. size is zero when block holds none.
	held struct {
		db, id      uint64
		start, size int64
	}
	// tab is the table of the version 2 segment that the reader read last.
	tab table
	// word holds a footer, an entry, or a stamp of a version 2 table.
	word [footerSize]byte
}

// NewReader returns a reader that shares the blocks of cache, or that reads each block
// from the disk when cache is nil.
func NewReader(cache *Cache) *Reader { return &Reader{cache: cache} }

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
	probe := rd.cache != nil && (s.ID != rd.end.id || off != rd.end.off) &&
		off/BlockSize == (off+int64(len(p))-1)/BlockSize
	rd.end.id, rd.end.off = s.ID, off+int64(len(p))
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
		} else if err := rd.hold(r, s, start, end, probe); err != nil {
			return err
		} else {
			copy(p[:n], rd.block[off-start:])
		}
		p, off = p[n:], off+int64(n)
	}
	return nil
}

// hold makes rd hold the checked block of s from stream offset start to end, through
// the cache for a probe.
func (rd *Reader) hold(r io.ReaderAt, s Committed, start, end int64, probe bool) error {
	size := end - start
	if size == BlockSize {
		size += s.trailer()
	}
	if rd.holds(s, start, size) {
		return nil
	}
	rd.held.size = 0
	if err := rd.fill(r, s, start, size, probe && size > BlockSize); err != nil {
		return err
	}
	rd.held.db, rd.held.id, rd.held.start, rd.held.size = s.DB, s.ID, start, size
	return nil
}

// fill makes the block of rd the checked size bytes at stream offset start of s: from
// the cache for a shared full block that it holds, or from r.
func (rd *Reader) fill(
	r io.ReaderAt,
	s Committed,
	start, size int64,
	shared bool,
) error {
	k := blockKey{id: s.ID, start: start}
	if !shared {
		rd.release()
	} else if b, ok := rd.cache.get(k, rd.cached); ok {
		rd.cached, rd.block = b, b.data
		return nil
	} else {
		rd.cached = nil
	}
	if rd.buf == nil {
		rd.buf = make([]byte, BlockSize+VariableTrailer)
	}
	if err := rd.checked(r, s, start, rd.buf[:size]); err != nil {
		return err
	}
	rd.block = rd.buf
	if shared {
		rd.cached, rd.buf = rd.cache.put(k, rd.buf)
		rd.block = rd.cached.data
	}
	return nil
}

// Reset drops the block that rd holds, so the next read of it reads it from the cache
// or checks it again, and forgets where the last read ended. A caller resets a reader
// when it is done with it, which lets the cache drop the block.
func (rd *Reader) Reset() {
	rd.release()
	rd.held.size, rd.end.id = 0, 0
}

// release drops the hold of rd on its cached block.
func (rd *Reader) release() {
	if rd.cached != nil {
		rd.cache.release(rd.cached)
		rd.cached = nil
	}
}

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
