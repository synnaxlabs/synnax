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
	"os"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
)

const (
	// BlockSize is the number of stream bytes in a block.
	BlockSize = 65_536
	// FixedTrailer is the size of the trailer of a block of a fixed-density type.
	FixedTrailer = 4
	// Version is the segment format version that this package writes.
	Version uint16 = 1
)

// Config is the identity of a segment, which every checksum of its blocks binds.
type Config struct {
	// DB is the identity of the database.
	DB uint64
	// ID is the identity of the segment.
	ID uint64
	// DataType is the data type code of the samples.
	DataType uint8
	// Clock is the key of the clock that times the samples.
	Clock uint32
}

func (c Config) place(offset int64) place.Place {
	return place.Segment(c.DB, c.ID, uint64(offset), Version, c.DataType, c.Clock)
}

// FileOffset returns the file offset of stream offset x of a fixed-density segment.
func FileOffset(x int64) int64 { return x + FixedTrailer*(x/BlockSize) }

// Writer appends the samples of a fixed-density type to one open segment. It is not
// safe for concurrent use.
type Writer struct {
	cfg  Config
	file xfs.File
	// length and crc are the stream length and the running checksum of the last block.
	length int64
	crc    uint32
	// committed holds length and crc at the last commit.
	committed struct {
		length int64
		crc    uint32
	}
	buf []byte
}

// Create creates the file name in fs for the segment of cfg, and returns its writer.
func Create(fs xfs.FS, name string, cfg Config) (*Writer, error) {
	file, err := fs.Open(name, os.O_CREATE|os.O_EXCL|os.O_RDWR)
	if err != nil {
		return nil, err
	}
	w := &Writer{cfg: cfg, file: file}
	w.crc = cfg.place(0).Start()
	w.committed.crc = w.crc
	return w, nil
}

// Write appends p to the stream, with the trailer of each block that p fills, in one
// write and with no sync. A failed write puts the writer back to its last commit.
func (w *Writer) Write(p []byte) error {
	start := FileOffset(w.length)
	w.buf = w.buf[:0]
	for len(p) > 0 {
		room := BlockSize - int(w.length%BlockSize)
		n := min(room, len(p))
		w.crc = place.Update(w.crc, p[:n])
		w.buf = append(w.buf, p[:n]...)
		w.length += int64(n)
		p = p[n:]
		if n == room {
			w.buf = binary.LittleEndian.AppendUint32(w.buf, w.crc)
			w.crc = w.cfg.place(FileOffset(w.length)).Start()
		}
	}
	if _, err := w.file.WriteAt(w.buf, start); err != nil {
		w.PutBack()
		return err
	}
	return nil
}

// Commit marks the written bytes as committed, and returns the committed length and the
// running checksum of the unfinished last block there.
func (w *Writer) Commit() (length, tailCRC uint32) {
	w.committed.length, w.committed.crc = w.length, w.crc
	return uint32(w.length), w.crc
}

// PutBack puts the length and the running checksum back to the last commit. Later
// writes overwrite the bytes past it.
func (w *Writer) PutBack() { w.length, w.crc = w.committed.length, w.committed.crc }

// Len returns the stream length of the written bytes.
func (w *Writer) Len() int64 { return w.length }

// Sync makes the written bytes durable.
func (w *Writer) Sync() error { return w.file.Sync() }

// Close closes the file of the segment.
func (w *Writer) Close() error { return w.file.Close() }

// ReadAt reads len(p) stream bytes from stream offset off of the fixed-density segment
// in file, with no check, and leaves out the trailers.
func ReadAt(file xfs.File, p []byte, off int64) error {
	for len(p) > 0 {
		n := min(BlockSize-int(off%BlockSize), len(p))
		if _, err := file.ReadAt(p[:n], FileOffset(off)); err != nil {
			return errors.Wrapf(err, "read of %d bytes at stream offset %d", n, off)
		}
		p, off = p[n:], off+int64(n)
	}
	return nil
}
