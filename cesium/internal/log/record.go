// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package log holds the pointer log of Cesium: the log file, its frames, the entries
// of a frame, and the records of an entry. A byte of a segment is part of the database
// only when a durable record covers it.
package log

import (
	"encoding/binary"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
)

// ErrUnknown is the error of a record type that this binary does not know and cannot
// skip. A newer binary wrote it.
var ErrUnknown = errors.New("unknown log record")

const (
	typeReserve uint16 = 1
	typeCreate  uint16 = 2
	typeCommit  uint16 = 3
	typeSeal    uint16 = 4
	typeDrop    uint16 = 7
	typeRestart uint16 = 8
	typeIntent  uint16 = 10
	// typeSkippable marks a record type that an older binary skips.
	typeSkippable uint16 = 1 << 15

	headerSize     = 8
	commitBaseSize = 32
	slotSize       = 16
	// MaxAlignment is the limit of alignment numbers that the log reserves. Numbers
	// from it up are for data that is not stored.
	MaxAlignment = 1 << 31
	// SegmentVersion is the only segment format version that this binary writes and
	// opens.
	SegmentVersion  uint16 = 1
	flagIntentClear uint32 = 1
)

// Record is a record of a log entry: a *Reserve, *Create, *Commit, *Seal, *Drop,
// *Restart, or *Intent.
type Record interface {
	append(dst []byte) []byte
	decode(body []byte) error
}

// Reserve raises the limits below which the log gives out segment identities and
// alignment numbers.
type Reserve struct {
	// Segments is the segment identity limit.
	Segments uint64
	// Alignments is the alignment number limit, at most MaxAlignment.
	Alignments uint64
}

// Create creates a segment.
type Create struct {
	// Segment is the identity of the segment, from 1.
	Segment uint64
	// Key is the key whose samples the segment holds.
	Key uint32
	// Clock is the key of the clock that times the samples. It equals Key for a clock.
	Clock uint32
	// Version is the format version of the segment.
	Version uint16
	// DataType is the data type code of the samples.
	DataType uint8
}

// Slot is the range that one segment gains in a Commit.
type Slot struct {
	// Segment is the identity of the segment.
	Segment uint64
	// Length is the committed length of the segment after the commit, in stream bytes.
	// The range starts at the length before.
	Length uint32
	// TailCRC is the running checksum of the unfinished last block at Length.
	TailCRC uint32
}

// Commit is one clock write: every slot gains the same samples, which take the time of
// the clock's samples.
type Commit struct {
	// Samples is the number of samples that each slot gains, at least 1.
	Samples uint32
	// Alignment is the alignment of the first sample.
	Alignment uint64
	// First and Last are the stamps of the first and last samples. First is at most
	// Last.
	First, Last int64
	// Slots hold at least one slot, in rising order of segment identity. A decoded
	// Commit's Slots stay valid only until the next call of Decoder.Next.
	Slots []Slot
}

// Seal seals a segment at a committed length.
type Seal struct {
	Segment uint64
	Length  uint32
	TailCRC uint32
}

// Drop drops every segment of a key and clears the key's delete intent.
type Drop struct{ Key uint32 }

// Restart seals every open segment at its committed length.
type Restart struct{}

// Intent sets or clears the delete intent of a key.
type Intent struct {
	Key uint32
	// Clear is true when the record clears the intent.
	Clear bool
}

func appendHeader(dst []byte, t uint16, size int) []byte {
	dst = binary.LittleEndian.AppendUint16(dst, t)
	dst = binary.LittleEndian.AppendUint16(dst, 0)
	return binary.LittleEndian.AppendUint32(dst, uint32(size))
}

func (r *Reserve) append(dst []byte) []byte {
	dst = appendHeader(dst, typeReserve, 16)
	dst = binary.LittleEndian.AppendUint64(dst, r.Segments)
	return binary.LittleEndian.AppendUint64(dst, r.Alignments)
}

func (r *Reserve) decode(b []byte) error {
	if len(b) != 16 {
		return sizeError("reserve", len(b), 16)
	}
	r.Segments = binary.LittleEndian.Uint64(b)
	r.Alignments = binary.LittleEndian.Uint64(b[8:])
	if r.Alignments > MaxAlignment {
		return errors.Wrapf(
			place.ErrCorrupt,
			"reserve record has alignment limit %d, above %d",
			r.Alignments,
			uint64(MaxAlignment),
		)
	}
	return nil
}

func (r *Create) append(dst []byte) []byte {
	dst = appendHeader(dst, typeCreate, 24)
	dst = binary.LittleEndian.AppendUint64(dst, r.Segment)
	dst = binary.LittleEndian.AppendUint32(dst, r.Key)
	dst = binary.LittleEndian.AppendUint32(dst, r.Clock)
	dst = binary.LittleEndian.AppendUint16(dst, r.Version)
	dst = append(dst, r.DataType)
	return append(dst, 0, 0, 0, 0, 0)
}

func (r *Create) decode(b []byte) error {
	if len(b) != 24 {
		return sizeError("create", len(b), 24)
	}
	r.Segment = binary.LittleEndian.Uint64(b)
	r.Key = binary.LittleEndian.Uint32(b[8:])
	r.Clock = binary.LittleEndian.Uint32(b[12:])
	r.Version = binary.LittleEndian.Uint16(b[16:])
	r.DataType = b[18]
	if r.Segment == 0 {
		return errors.Wrap(place.ErrCorrupt, "create record has segment 0")
	}
	if r.Version != SegmentVersion {
		return errors.Wrapf(
			ErrUnknown,
			"create record of segment %d has format version %d",
			r.Segment,
			r.Version,
		)
	}
	return zeros("create", b[19:])
}

func (r *Commit) append(dst []byte) []byte {
	dst = appendHeader(dst, typeCommit, commitBaseSize+slotSize*len(r.Slots))
	dst = binary.LittleEndian.AppendUint32(dst, uint32(len(r.Slots)))
	dst = binary.LittleEndian.AppendUint32(dst, r.Samples)
	dst = binary.LittleEndian.AppendUint64(dst, r.Alignment)
	dst = binary.LittleEndian.AppendUint64(dst, uint64(r.First))
	dst = binary.LittleEndian.AppendUint64(dst, uint64(r.Last))
	for _, s := range r.Slots {
		dst = binary.LittleEndian.AppendUint64(dst, s.Segment)
		dst = binary.LittleEndian.AppendUint32(dst, s.Length)
		dst = binary.LittleEndian.AppendUint32(dst, s.TailCRC)
	}
	return dst
}

func (r *Commit) decode(b []byte) error {
	if len(b) < commitBaseSize+slotSize || (len(b)-commitBaseSize)%slotSize != 0 {
		return errors.Wrapf(
			place.ErrCorrupt,
			"commit record has %d bytes, not 32 plus 16 per slot",
			len(b),
		)
	}
	k := int(binary.LittleEndian.Uint32(b))
	if k != (len(b)-commitBaseSize)/slotSize {
		return errors.Wrapf(
			place.ErrCorrupt,
			"commit record of %d bytes names %d slots",
			len(b),
			k,
		)
	}
	r.Samples = binary.LittleEndian.Uint32(b[4:])
	r.Alignment = binary.LittleEndian.Uint64(b[8:])
	r.First = int64(binary.LittleEndian.Uint64(b[16:]))
	r.Last = int64(binary.LittleEndian.Uint64(b[24:]))
	if r.Samples == 0 {
		return errors.Wrap(place.ErrCorrupt, "commit record gains no samples")
	}
	if r.First > r.Last {
		return errors.Wrapf(
			place.ErrCorrupt,
			"commit record starts at %d, after its end %d",
			r.First,
			r.Last,
		)
	}
	r.Slots = r.Slots[:0]
	for i := range k {
		at := commitBaseSize + slotSize*i
		s := Slot{
			Segment: binary.LittleEndian.Uint64(b[at:]),
			Length:  binary.LittleEndian.Uint32(b[at+8:]),
			TailCRC: binary.LittleEndian.Uint32(b[at+12:]),
		}
		if i > 0 && s.Segment <= r.Slots[i-1].Segment {
			return errors.Wrapf(
				place.ErrCorrupt,
				"commit record slot %d names segment %d, not above segment %d",
				i,
				s.Segment,
				r.Slots[i-1].Segment,
			)
		}
		if s.Segment == 0 {
			return errors.Wrap(place.ErrCorrupt, "commit record names segment 0")
		}
		r.Slots = append(r.Slots, s)
	}
	return nil
}

func (r *Seal) append(dst []byte) []byte {
	dst = appendHeader(dst, typeSeal, 16)
	dst = binary.LittleEndian.AppendUint64(dst, r.Segment)
	dst = binary.LittleEndian.AppendUint32(dst, r.Length)
	return binary.LittleEndian.AppendUint32(dst, r.TailCRC)
}

func (r *Seal) decode(b []byte) error {
	if len(b) != 16 {
		return sizeError("seal", len(b), 16)
	}
	r.Segment = binary.LittleEndian.Uint64(b)
	r.Length = binary.LittleEndian.Uint32(b[8:])
	r.TailCRC = binary.LittleEndian.Uint32(b[12:])
	if r.Segment == 0 {
		return errors.Wrap(place.ErrCorrupt, "seal record has segment 0")
	}
	return nil
}

func (r *Drop) append(dst []byte) []byte {
	dst = appendHeader(dst, typeDrop, 8)
	dst = binary.LittleEndian.AppendUint32(dst, r.Key)
	return binary.LittleEndian.AppendUint32(dst, 0)
}

func (r *Drop) decode(b []byte) error {
	if len(b) != 8 {
		return sizeError("drop", len(b), 8)
	}
	r.Key = binary.LittleEndian.Uint32(b)
	return zeros("drop", b[4:])
}

func (*Restart) append(dst []byte) []byte { return appendHeader(dst, typeRestart, 0) }

func (*Restart) decode(b []byte) error {
	if len(b) != 0 {
		return sizeError("restart", len(b), 0)
	}
	return nil
}

func (r *Intent) append(dst []byte) []byte {
	dst = appendHeader(dst, typeIntent, 8)
	dst = binary.LittleEndian.AppendUint32(dst, r.Key)
	var flags uint32
	if r.Clear {
		flags = flagIntentClear
	}
	return binary.LittleEndian.AppendUint32(dst, flags)
}

func (r *Intent) decode(b []byte) error {
	if len(b) != 8 {
		return sizeError("intent", len(b), 8)
	}
	r.Key = binary.LittleEndian.Uint32(b)
	flags := binary.LittleEndian.Uint32(b[4:])
	if flags&^flagIntentClear != 0 {
		return errors.Wrapf(
			place.ErrCorrupt,
			"intent record of key %d has unknown flags %#x",
			r.Key,
			flags,
		)
	}
	r.Clear = flags&flagIntentClear != 0
	return nil
}

func sizeError(name string, got, want int) error {
	return errors.Wrapf(
		place.ErrCorrupt,
		"%s record has %d bytes, not %d",
		name,
		got,
		want,
	)
}

func zeros(name string, b []byte) error {
	for _, c := range b {
		if c != 0 {
			return errors.Wrapf(
				place.ErrCorrupt,
				"%s record has a set reserved byte",
				name,
			)
		}
	}
	return nil
}

// Entry builds the bytes of one entry, which applies all or nothing. The zero value is
// an empty entry.
type Entry struct {
	b     []byte
	count uint32
}

// Add appends r to the entry.
func (e *Entry) Add(r Record) {
	if len(e.b) == 0 {
		e.b = append(e.b, 0, 0, 0, 0, 0, 0, 0, 0)
	}
	e.b = r.append(e.b)
	e.count++
}

// Len returns the number of records in the entry.
func (e *Entry) Len() int { return int(e.count) }

// Bytes returns the encoded entry. The bytes stay valid until the next Add or Reset.
func (e *Entry) Bytes() []byte {
	if len(e.b) == 0 {
		e.b = append(e.b, 0, 0, 0, 0, 0, 0, 0, 0)
	}
	binary.LittleEndian.PutUint32(e.b, uint32(len(e.b)-headerSize))
	binary.LittleEndian.PutUint32(e.b[4:], e.count)
	return e.b
}

// Reset empties the entry and keeps its buffer.
func (e *Entry) Reset() {
	e.b = e.b[:0]
	e.count = 0
}

// Decoder reads the records of entries. It owns one record of each type, so it
// allocates nothing per record. The zero value is ready for Reset.
type Decoder struct {
	b       []byte
	left    uint32
	reserve Reserve
	create  Create
	commit  Commit
	seal    Seal
	drop    Drop
	restart Restart
	intent  Intent
}

// Reset starts reading the entry at the start of b, and returns the length of the
// entry in b.
func (d *Decoder) Reset(b []byte) (int, error) {
	if len(b) < headerSize {
		return 0, errors.Wrapf(
			place.ErrCorrupt,
			"entry header needs 8 bytes, but %d remain",
			len(b),
		)
	}
	size := binary.LittleEndian.Uint32(b)
	count := binary.LittleEndian.Uint32(b[4:])
	if uint64(size) > uint64(len(b)-headerSize) {
		return 0, errors.Wrapf(
			place.ErrCorrupt,
			"entry holds %d bytes of records, but %d remain",
			size,
			len(b)-headerSize,
		)
	}
	if uint64(count)*headerSize > uint64(size) {
		return 0, errors.Wrapf(
			place.ErrCorrupt,
			"entry of %d bytes cannot hold %d records",
			size,
			count,
		)
	}
	d.b = b[headerSize : headerSize+int(size)]
	d.left = count
	return headerSize + int(size), nil
}

// Next returns the next record of the entry, or false at its end. The record stays
// valid until the next call. Next skips a record of an unknown type that an older
// binary may skip, and returns ErrUnknown for one that it may not.
func (d *Decoder) Next() (Record, bool, error) {
	for d.left > 0 {
		if len(d.b) < headerSize {
			return nil, false, errors.Wrapf(
				place.ErrCorrupt,
				"entry ends with %d records left",
				d.left,
			)
		}
		t := binary.LittleEndian.Uint16(d.b)
		zero := binary.LittleEndian.Uint16(d.b[2:])
		size := binary.LittleEndian.Uint32(d.b[4:])
		if zero != 0 {
			return nil, false, errors.Wrapf(
				place.ErrCorrupt,
				"record of type %d has a set reserved field",
				t,
			)
		}
		if uint64(size) > uint64(len(d.b)-headerSize) {
			return nil, false, errors.Wrapf(
				place.ErrCorrupt,
				"record of type %d holds %d bytes, but %d remain in its entry",
				t,
				size,
				len(d.b)-headerSize,
			)
		}
		body := d.b[headerSize : headerSize+int(size)]
		d.b = d.b[headerSize+int(size):]
		d.left--
		r := d.record(t)
		if r == nil {
			if t&typeSkippable != 0 {
				continue
			}
			return nil, false, errors.Wrapf(ErrUnknown, "record type %d", t)
		}
		if err := r.decode(body); err != nil {
			return nil, false, err
		}
		return r, true, nil
	}
	if len(d.b) != 0 {
		return nil, false, errors.Wrapf(
			place.ErrCorrupt,
			"entry has %d bytes after its last record",
			len(d.b),
		)
	}
	return nil, false, nil
}

func (d *Decoder) record(t uint16) Record {
	switch t {
	case typeReserve:
		return &d.reserve
	case typeCreate:
		return &d.create
	case typeCommit:
		return &d.commit
	case typeSeal:
		return &d.seal
	case typeDrop:
		return &d.drop
	case typeRestart:
		return &d.restart
	case typeIntent:
		return &d.intent
	default:
		return nil
	}
}
