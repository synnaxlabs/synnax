// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package segment writes and reads Cesium segment files. A version 1 segment is a
// stream of sample bytes cut into blocks, each full block followed by a trailer that
// holds its checksum. Lengths and offsets count stream bytes, with no trailers.
package segment

import (
	"encoding/binary"
	"io"
	"math"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
)

const (
	// BlockSize is the number of stream bytes in a block.
	BlockSize = 65_536
	// FixedTrailer is the size of the trailer of a block of a fixed-density type.
	FixedTrailer = 4
	// VariableTrailer is the size of the trailer of a block of a variable-density type:
	// first_pos, first_off, crc, and tcrc.
	VariableTrailer = 16
	// NoStart is the first_off of a block in which no sample starts.
	NoStart uint32 = 0xFFFFFFFF
	// Version is the segment format version that this package writes.
	Version uint16 = 1
	// lengthSize is the size of the length that starts each variable-density sample.
	lengthSize = 4
)

// Config is the identity of a segment, which every checksum of its blocks binds, and
// the layout of its blocks.
type Config struct {
	// DB is the identity of the database.
	DB uint64
	// ID is the identity of the segment.
	ID uint64
	// DataType is the data type code of the samples.
	DataType uint8
	// Clock is the key of the clock that times the samples.
	Clock uint32
	// Variable is true for a variable-density type. Each of its samples is a uint32
	// length and then that many bytes, and a sample can cross blocks.
	Variable bool
}

func (c Config) place(offset int64) place.Place {
	return place.Segment(c.DB, c.ID, uint64(offset), Version, c.DataType, c.Clock)
}

func (c Config) trailer() int64 {
	if c.Variable {
		return VariableTrailer
	}
	return FixedTrailer
}

// FileOffset returns the file offset of stream offset x of the segment.
func (c Config) FileOffset(x int64) int64 { return x + c.trailer()*(x/BlockSize) }

// ReadAt reads len(p) stream bytes from stream offset off of the segment in r, with no
// check, and leaves out the trailers.
func (c Config) ReadAt(r io.ReaderAt, p []byte, off int64) error {
	for len(p) > 0 {
		n := min(BlockSize-int(off%BlockSize), len(p))
		if _, err := r.ReadAt(p[:n], c.FileOffset(off)); err != nil {
			return errors.Wrapf(err, "read of %d bytes at stream offset %d", n, off)
		}
		p, off = p[n:], off+int64(n)
	}
	return nil
}

// cursor is the position of a writer in its stream.
type cursor struct {
	// length is the stream length, and crc the running checksum of the unfinished
	// block.
	length int64
	crc    uint32
	// The fields below count only variable-density samples. samples is the number of
	// samples that start before length, blockPos the number that start before the
	// unfinished block, and blockOff where the first sample in it starts, or NoStart.
	samples  uint32
	blockPos uint32
	blockOff uint32
}

// Writer appends samples to one open segment. It is not safe for concurrent use.
type Writer struct {
	cfg       Config
	handle    *Handle
	at        cursor
	committed cursor
	buf       []byte
}

// NewWriter returns the writer of the empty segment of cfg in h.
func NewWriter(h *Handle, cfg Config) *Writer {
	w := &Writer{cfg: cfg, handle: h}
	w.at = cursor{crc: cfg.place(0).Start(), blockOff: NoStart}
	w.committed = w.at
	return w
}

// Resume returns the writer of the open segment of cfg in h, at its committed stream
// length, whose unfinished last block has the running checksum tailCRC. It reads that
// block's committed bytes back, and returns an error that wraps place.ErrCorrupt when
// their checksum is not tailCRC. For a variable-density type, it then counts the sample
// starts back from the last block trailer that names one.
func Resume(h *Handle, cfg Config, length int64, tailCRC uint32) (*Writer, error) {
	block := length - length%BlockSize
	tail := make([]byte, length-block)
	if err := cfg.ReadAt(h, tail, block); err != nil {
		return nil, err
	}
	crc := place.Update(cfg.place(cfg.FileOffset(block)).Start(), tail)
	if crc != tailCRC {
		return nil, errors.Wrapf(
			place.ErrCorrupt,
			"tail of segment %d at stream offset %d has checksum %d, not %d",
			cfg.ID,
			block,
			crc,
			tailCRC,
		)
	}
	at := cursor{length: length, crc: tailCRC, blockOff: NoStart}
	if cfg.Variable {
		if err := resumeStarts(h, cfg, &at); err != nil {
			return nil, err
		}
	}
	return &Writer{cfg: cfg, handle: h, at: at, committed: at}, nil
}

// resumeStarts sets the sample counts of at, whose length lies on a sample boundary. It
// walks the samples from the first start in the last full block that has one, or from
// the stream start when none has one. It checks that block and its trailer, since the
// walk reads sample lengths there, and returns an error that wraps place.ErrCorrupt
// when either fails.
func resumeStarts(h *Handle, cfg Config, at *cursor) error {
	tail := at.length - at.length%BlockSize
	var base uint32
	next := int64(0)
	for k := tail/BlockSize - 1; k >= 0; k-- {
		start := k * BlockSize
		trailer := make([]byte, VariableTrailer)
		end := cfg.FileOffset(start + BlockSize)
		if _, err := h.ReadAt(trailer, end-VariableTrailer); err != nil {
			return err
		}
		off := binary.LittleEndian.Uint32(trailer[4:])
		if off == NoStart {
			continue
		}
		if err := checkBlock(h, cfg, start, trailer, end); err != nil {
			return err
		}
		base, next = binary.LittleEndian.Uint32(trailer), start+int64(off)
		break
	}
	var before, total uint32
	size := make([]byte, lengthSize)
	for next < at.length {
		if next >= tail && at.blockOff == NoStart {
			at.blockOff = uint32(next - tail)
		}
		if next < tail {
			before++
		}
		total++
		if err := cfg.ReadAt(h, size, next); err != nil {
			return err
		}
		next += lengthSize + int64(binary.LittleEndian.Uint32(size))
	}
	at.blockPos, at.samples = base+before, base+total
	return nil
}

// checkBlock checks the full block of cfg in h at stream offset start against its
// trailer, which ends at file offset end, and the trailer against its own checksum.
func checkBlock(h *Handle, cfg Config, start int64, trailer []byte, end int64) error {
	body := VariableTrailer - 4
	tcrc := cfg.place(end - VariableTrailer).Sum(trailer[:body])
	data := make([]byte, BlockSize)
	if err := cfg.ReadAt(h, data, start); err != nil {
		return err
	}
	crc := place.Update(cfg.place(cfg.FileOffset(start)).Start(), data)
	if tcrc != binary.LittleEndian.Uint32(trailer[body:]) ||
		crc != binary.LittleEndian.Uint32(trailer[8:]) {
		return errors.Wrapf(
			place.ErrCorrupt,
			"block of segment %d at stream offset %d fails its trailer",
			cfg.ID,
			start,
		)
	}
	return nil
}

// Write appends p to the stream, with the trailer of each block that p fills, and no
// sync. p holds whole samples. Write stages at most one block and its trailer, so a
// large write takes one write call per block. A failed write puts the writer back to
// its last commit. A write that ends inside a variable-density sample, or that would
// grow the stream past 4 GiB, fails and changes nothing.
func (w *Writer) Write(p []byte) error {
	if w.at.length+int64(len(p)) > math.MaxUint32 {
		return errors.Newf(
			"segment of %d stream bytes cannot grow by %d past 4 GiB",
			w.at.length,
			len(p),
		)
	}
	if w.cfg.Variable {
		if err := wholeSamples(p); err != nil {
			return err
		}
	}
	at := w.cfg.FileOffset(w.at.length)
	w.buf = w.buf[:0]
	flush := func() error {
		if len(w.buf) == 0 {
			return nil
		}
		if _, err := w.handle.WriteAt(w.buf, at); err != nil {
			w.PutBack()
			return err
		}
		at += int64(len(w.buf))
		w.buf = w.buf[:0]
		return nil
	}
	// next is the offset in p of the next variable-density sample.
	next := 0
	for i := 0; i < len(p); {
		inBlock := int(w.at.length % BlockSize)
		n := min(BlockSize-inBlock, len(p)-i)
		if len(w.buf)+n+int(w.cfg.trailer()) > BlockSize+VariableTrailer {
			if err := flush(); err != nil {
				return err
			}
		}
		if w.cfg.Variable {
			next = w.countStarts(p, next, i, n)
		}
		w.at.crc = place.Update(w.at.crc, p[i:i+n])
		w.at.length += int64(n)
		if inBlock+n < BlockSize && len(w.buf) == 0 {
			// The last piece, with nothing staged before it, needs no copy.
			if _, err := w.handle.WriteAt(p[i:i+n], at); err != nil {
				w.PutBack()
				return err
			}
			return nil
		}
		w.buf = append(w.buf, p[i:i+n]...)
		i += n
		if inBlock+n == BlockSize {
			w.buf = w.appendTrailer(w.buf)
		}
	}
	return flush()
}

// countStarts counts the variable-density samples that start in the piece p[i:i+n] of
// one block, from the sample at next, and returns the offset of the sample after them.
func (w *Writer) countStarts(p []byte, next, i, n int) int {
	inBlock := int(w.at.length % BlockSize)
	for ; next < i+n; w.at.samples++ {
		if w.at.blockOff == NoStart {
			w.at.blockOff = uint32(inBlock + next - i)
		}
		next += lengthSize + int(binary.LittleEndian.Uint32(p[next:]))
	}
	return next
}

// appendTrailer appends the trailer of the block that just filled to dst, and starts
// the next block.
func (w *Writer) appendTrailer(dst []byte) []byte {
	next := w.cfg.FileOffset(w.at.length)
	if w.cfg.Variable {
		t := len(dst)
		dst = binary.LittleEndian.AppendUint32(dst, w.at.blockPos)
		dst = binary.LittleEndian.AppendUint32(dst, w.at.blockOff)
		dst = binary.LittleEndian.AppendUint32(dst, w.at.crc)
		tcrc := w.cfg.place(next - VariableTrailer).Sum(dst[t:])
		dst = binary.LittleEndian.AppendUint32(dst, tcrc)
		w.at.blockPos, w.at.blockOff = w.at.samples, NoStart
	} else {
		dst = binary.LittleEndian.AppendUint32(dst, w.at.crc)
	}
	w.at.crc = w.cfg.place(next).Start()
	return dst
}

func wholeSamples(p []byte) error {
	for i := 0; i < len(p); {
		if len(p)-i < lengthSize {
			return errors.Newf(
				"variable-density write of %d bytes ends inside the length at %d",
				len(p),
				i,
			)
		}
		i += lengthSize + int(binary.LittleEndian.Uint32(p[i:]))
		if i > len(p) {
			return errors.Newf(
				"variable-density write of %d bytes ends inside a sample",
				len(p),
			)
		}
	}
	return nil
}

// Pending returns the stream length of the written bytes and the running checksum of
// the unfinished last block there, which Commit commits. A caller logs them, and calls
// Commit once the commit is accepted, or PutBack when it is refused.
func (w *Writer) Pending() (length, tailCRC uint32) {
	return uint32(w.at.length), w.at.crc
}

// Commit marks the written bytes as committed, and returns what Pending returns.
func (w *Writer) Commit() (length, tailCRC uint32) {
	w.committed = w.at
	return w.Pending()
}

// PutBack puts the writer back to its last commit. Later writes overwrite the bytes
// past it.
func (w *Writer) PutBack() { w.at = w.committed }

// Len returns the stream length of the written bytes.
func (w *Writer) Len() int64 { return w.at.length }
