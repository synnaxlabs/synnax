// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package log

import (
	"bytes"
	"encoding/binary"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
)

const (
	// FileSize is the size of a log file, written as zeros before use.
	FileSize = 64 << 20
	// HeaderSize is the size of one copy of the file header.
	HeaderSize = 64
	// HeaderCopy is the offset of the second copy of the file header.
	HeaderCopy = 4096
	// FramesStart is the offset of the first frame of a log file.
	FramesStart = 8192
	// BlockSize is the size of a log block. No block that holds a confirmed frame is
	// written again.
	BlockSize = 4096
	// FrameHeaderSize is the size of a frame header.
	FrameHeaderSize = 32
	// MaxFrame is the largest frame, its header included.
	MaxFrame = 16 << 20
	// DeleteReserve is the space at the end of a log file that only deletes may use.
	DeleteReserve = 1 << 20

	headerVersion uint16 = 1
	frameMagic    uint32 = 0x464C5043
	flagClean     uint32 = 1
)

var fileMagic = []byte("CSPTRLOG")

// Header is the header of a log file.
type Header struct {
	// DB is the identity of the database.
	DB uint64
	// Generation is the generation of the log.
	Generation uint64
	// Sequence is the sequence of the first entry in the file.
	Sequence uint64
	// Size is the size of the file.
	Size uint64
	// PrevCRC is the crc field of the last frame of the generation before, or 0.
	PrevCRC uint32
}

// Encode returns the copy of h at offset, which is 0 or HeaderCopy.
func (h Header) Encode(offset uint64) []byte {
	b := make([]byte, HeaderSize)
	copy(b, fileMagic)
	binary.LittleEndian.PutUint16(b[8:], headerVersion)
	binary.LittleEndian.PutUint64(b[16:], h.DB)
	binary.LittleEndian.PutUint64(b[24:], h.Generation)
	binary.LittleEndian.PutUint64(b[32:], h.Sequence)
	binary.LittleEndian.PutUint64(b[40:], h.Size)
	binary.LittleEndian.PutUint32(b[48:], h.PrevCRC)
	crc := place.Log(h.DB, h.Generation, offset).Sum(b[:60])
	binary.LittleEndian.PutUint32(b[60:], crc)
	return b
}

// DecodeHeader decodes the copy of the header at offset from the start of b, for the
// log of generation in database db. It returns ErrUnknown for a version that this
// binary does not know.
func DecodeHeader(b []byte, db, generation, offset uint64) (Header, error) {
	if len(b) < HeaderSize {
		return Header{}, errors.Wrapf(
			place.ErrCorrupt,
			"log header at %d needs 64 bytes, but %d remain",
			offset,
			len(b),
		)
	}
	b = b[:HeaderSize]
	want := place.Log(db, generation, offset).Sum(b[:60])
	if !bytes.Equal(b[:8], fileMagic) || binary.LittleEndian.Uint32(b[60:]) != want {
		return Header{}, errors.Wrapf(
			place.ErrCorrupt,
			"log header at %d fails its check for generation %d",
			offset,
			generation,
		)
	}
	h := Header{
		DB:         binary.LittleEndian.Uint64(b[16:]),
		Generation: binary.LittleEndian.Uint64(b[24:]),
		Sequence:   binary.LittleEndian.Uint64(b[32:]),
		Size:       binary.LittleEndian.Uint64(b[40:]),
		PrevCRC:    binary.LittleEndian.Uint32(b[48:]),
	}
	if v := binary.LittleEndian.Uint16(b[8:]); v != headerVersion {
		return Header{}, errors.Wrapf(ErrUnknown, "log format version %d", v)
	}
	if !allZero(b[10:16]) || !allZero(b[52:60]) {
		return Header{}, errors.Wrapf(
			place.ErrCorrupt,
			"log header at %d has a set reserved byte",
			offset,
		)
	}
	return h, nil
}

// Frame is one group commit batch of entries.
type Frame struct {
	// Count is the number of entries. A frame with no entry is a mark.
	Count uint32
	// Sequence is the sequence of the first entry.
	Sequence uint64
	// PrevCRC is the crc field of the frame before, or the CRC of the header at 0 for
	// the first frame of a file.
	PrevCRC uint32
	// Clean is true for the mark of a clean close.
	Clean bool
	// Entries holds the encoded entries. A decoded frame's Entries alias the bytes it
	// was decoded from.
	Entries []byte
	// CRC is the frame's crc field. Encode sets it.
	CRC uint32
}

// Size returns the size of the encoded frame.
func (f *Frame) Size() int { return FrameHeaderSize + len(f.Entries) }

// Append appends the frame at offset of the log of generation in database db to dst,
// and sets f.CRC.
func (f *Frame) Append(dst []byte, db, generation, offset uint64) []byte {
	start := len(dst)
	dst = binary.LittleEndian.AppendUint32(dst, frameMagic)
	dst = binary.LittleEndian.AppendUint32(dst, 0)
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(f.Entries)))
	dst = binary.LittleEndian.AppendUint32(dst, f.Count)
	dst = binary.LittleEndian.AppendUint64(dst, f.Sequence)
	dst = binary.LittleEndian.AppendUint32(dst, f.PrevCRC)
	var flags uint32
	if f.Clean {
		flags = flagClean
	}
	dst = binary.LittleEndian.AppendUint32(dst, flags)
	dst = append(dst, f.Entries...)
	f.CRC = place.Log(db, generation, offset).Sum(dst[start+8:])
	binary.LittleEndian.PutUint32(dst[start+4:], f.CRC)
	return dst
}

// DecodeFrame decodes the frame at offset of the log of generation in database db from
// the start of b. It checks the magic, the bounds, and the checksum. The caller checks
// the sequence and the link.
func DecodeFrame(b []byte, db, generation, offset uint64) (Frame, error) {
	if len(b) < FrameHeaderSize {
		return Frame{}, errors.Wrapf(
			place.ErrCorrupt,
			"frame at %d needs 32 bytes, but %d remain",
			offset,
			len(b),
		)
	}
	if binary.LittleEndian.Uint32(b) != frameMagic {
		return Frame{}, errors.Wrapf(place.ErrCorrupt, "no frame magic at %d", offset)
	}
	size := uint64(binary.LittleEndian.Uint32(b[8:]))
	if size > MaxFrame-FrameHeaderSize || size > uint64(len(b)-FrameHeaderSize) {
		return Frame{}, errors.Wrapf(
			place.ErrCorrupt,
			"frame at %d holds %d bytes of entries, past its bounds",
			offset,
			size,
		)
	}
	end := FrameHeaderSize + int(size)
	crc := binary.LittleEndian.Uint32(b[4:])
	if place.Log(db, generation, offset).Sum(b[8:end]) != crc {
		return Frame{}, errors.Wrapf(
			place.ErrCorrupt,
			"frame at %d fails its check",
			offset,
		)
	}
	f := Frame{
		Count:    binary.LittleEndian.Uint32(b[12:]),
		Sequence: binary.LittleEndian.Uint64(b[16:]),
		PrevCRC:  binary.LittleEndian.Uint32(b[24:]),
		Entries:  b[FrameHeaderSize:end],
		CRC:      crc,
	}
	flags := binary.LittleEndian.Uint32(b[28:])
	if flags&^flagClean != 0 {
		return Frame{}, errors.Wrapf(
			place.ErrCorrupt,
			"frame at %d has unknown flags %#x",
			offset,
			flags,
		)
	}
	f.Clean = flags&flagClean != 0
	if uint64(f.Count)*headerSize > size {
		return Frame{}, errors.Wrapf(
			place.ErrCorrupt,
			"frame at %d of %d bytes cannot hold %d entries",
			offset,
			size,
			f.Count,
		)
	}
	return f, nil
}

func allZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}
