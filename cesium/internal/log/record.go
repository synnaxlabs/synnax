// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package log holds the pointer log of Cesium: the log file, its frames, the entries of
// a frame, and the records of an entry. A byte of a segment is part of the database
// only when a durable record covers it. The codec checks the structure of its bytes:
// lengths, counts against lengths, zero fields, and types. The state checks values.
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
	typeReserve   uint16 = 1
	typeCreate    uint16 = 2
	typeCommit    uint16 = 3
	typeSeal      uint16 = 4
	typeTombstone uint16 = 5
	typeReplace   uint16 = 6
	typeDrop      uint16 = 7
	typeRestart   uint16 = 8
	typeCheck     uint16 = 9
	typeIntent    uint16 = 10
	// typeSkippable marks a record type that an older binary skips.
	typeSkippable uint16 = 1 << 15

	headerSize      = 8
	commitBaseSize  = 32
	slotSize        = 16
	tombstoneSize   = 72
	cutSize         = 24
	replaceBaseSize = 32
	pointerSize     = 40

	flagIntentCleared uint32 = 1
	flagLeftCut       uint32 = 1
	flagRightCut      uint32 = 2
)

// Record is a record of a log entry: a *Reserve, *Create, *Commit, *Seal, *Tombstone,
// *Replace, *Drop, *Restart, *Check, or *Intent.
type Record interface {
	append(dst []byte) []byte
	decode(body []byte) error
}

// Reserve raises the limits below which the log gives out segment identities and
// alignment numbers.
type Reserve struct {
	// Segments is the segment identity limit.
	Segments uint64
	// Alignments is the alignment number limit.
	Alignments uint64
}

// Create creates a segment.
type Create struct {
	// Segment is the identity of the segment.
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
	// Samples is the number of samples that each slot gains.
	Samples uint32
	// Alignment is the alignment of the first sample.
	Alignment uint64
	// First and Last are the stamps of the first and last samples.
	First, Last int64
	// Slots hold one range per segment. A decoded Commit's Slots stay valid only
	// until the next call of Decoder.Next.
	Slots []Slot
}

// Seal seals a segment at a committed length.
type Seal struct {
	// Segment is the identity of the segment.
	Segment uint64
	// Length is the committed length at which the segment seals, in stream bytes.
	Length uint32
	// TailCRC is the running checksum of the unfinished last block at Length.
	TailCRC uint32
}

// Cut is where a Tombstone cuts a pointer.
type Cut struct {
	// Segment is the identity of the segment of the pointer.
	Segment uint64
	// Offset is the stream offset of the cut.
	Offset uint32
	// Alignment is the alignment of the sample at the cut.
	Alignment uint64
}

// Tombstone deletes a time range of a key.
type Tombstone struct {
	// Key is the key whose samples the range deletes.
	Key uint32
	// Start and End bound the deleted time range. End is not in it.
	Start, End int64
	// Left cuts the pointer that keeps the bytes below the cut, at the first dead
	// sample. Nil when no pointer has a left cut.
	Left *Cut
	// Right cuts the pointer that keeps the bytes from the cut on, at the first live
	// sample. Nil when no pointer has a right cut.
	Right *Cut
	// left and right hold the cuts of a decoded Tombstone.
	left, right Cut
}

// Pointer is a committed byte range that a Replace gives its new segment.
type Pointer struct {
	// Offset and Length are the stream offset and the byte length of the range.
	Offset, Length uint32
	// Samples is the number of samples in the range.
	Samples uint32
	// Alignment is the alignment of the first sample.
	Alignment uint64
	// First and Last are the stamps of the first and last samples.
	First, Last int64
}

// Replace replaces an old sealed segment with a new one that a Create earlier in the
// entry made, and seals the new one.
type Replace struct {
	// Old and New are the identities of the old and the new segment.
	Old, New uint64
	// Length is the final length of the new segment, and TailCRC the running checksum
	// of its unfinished last block there.
	Length, TailCRC uint32
	// Pointers are the ranges of the new segment. A decoded Replace's Pointers stay
	// valid only until the next call of Decoder.Next.
	Pointers []Pointer
}

// Drop drops every segment of a key and clears the key's delete intent.
type Drop struct {
	// Key is the key that the record drops.
	Key uint32
}

// Restart seals every open segment at its committed length.
type Restart struct{}

// Check turns on the checks of a version 0 segment that now has a check file.
type Check struct {
	// Segment is the identity of the segment.
	Segment uint64
}

// Intent sets or clears the delete intent of a key.
type Intent struct {
	// Key is the key of the intent.
	Key uint32
	// Cleared is true when the record clears the intent.
	Cleared bool
}

func appendHeader(dst []byte, t uint16, size int) []byte {
	dst = binary.LittleEndian.AppendUint16(dst, t)
	dst = binary.LittleEndian.AppendUint16(dst, 0)
	return binary.LittleEndian.AppendUint32(dst, uint32(size))
}

func le32(dst []byte, vs ...uint32) []byte {
	for _, v := range vs {
		dst = binary.LittleEndian.AppendUint32(dst, v)
	}
	return dst
}

func le64(dst []byte, vs ...uint64) []byte {
	for _, v := range vs {
		dst = binary.LittleEndian.AppendUint64(dst, v)
	}
	return dst
}

func (r *Reserve) append(dst []byte) []byte {
	return le64(appendHeader(dst, typeReserve, 16), r.Segments, r.Alignments)
}

func (r *Reserve) decode(b []byte) error {
	if len(b) != 16 {
		return sizeError("reserve", len(b), 16)
	}
	r.Segments = binary.LittleEndian.Uint64(b)
	r.Alignments = binary.LittleEndian.Uint64(b[8:])
	return nil
}

func (r *Create) append(dst []byte) []byte {
	dst = le64(appendHeader(dst, typeCreate, 24), r.Segment)
	dst = le32(dst, r.Key, r.Clock)
	dst = binary.LittleEndian.AppendUint16(dst, r.Version)
	return append(dst, r.DataType, 0, 0, 0, 0, 0)
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
	return zeros("create", b[19:])
}

func (r *Commit) append(dst []byte) []byte {
	dst = appendHeader(dst, typeCommit, commitBaseSize+slotSize*len(r.Slots))
	dst = le32(dst, uint32(len(r.Slots)), r.Samples)
	dst = le64(dst, r.Alignment, uint64(r.First), uint64(r.Last))
	for _, s := range r.Slots {
		dst = le32(le64(dst, s.Segment), s.Length, s.TailCRC)
	}
	return dst
}

func (r *Commit) decode(b []byte) error {
	k := uint64(0)
	if len(b) >= 4 {
		k = uint64(binary.LittleEndian.Uint32(b))
	}
	if len(b) < commitBaseSize || uint64(len(b)) != commitBaseSize+slotSize*k {
		return errors.Wrapf(
			place.ErrCorrupt,
			"commit record of %d bytes does not hold 32 bytes and its %d slots",
			len(b),
			k,
		)
	}
	r.Samples = binary.LittleEndian.Uint32(b[4:])
	r.Alignment = binary.LittleEndian.Uint64(b[8:])
	r.First = int64(binary.LittleEndian.Uint64(b[16:]))
	r.Last = int64(binary.LittleEndian.Uint64(b[24:]))
	r.Slots = r.Slots[:0]
	for at := commitBaseSize; at < len(b); at += slotSize {
		r.Slots = append(r.Slots, Slot{
			Segment: binary.LittleEndian.Uint64(b[at:]),
			Length:  binary.LittleEndian.Uint32(b[at+8:]),
			TailCRC: binary.LittleEndian.Uint32(b[at+12:]),
		})
	}
	return nil
}

func (r *Seal) append(dst []byte) []byte {
	return le32(le64(appendHeader(dst, typeSeal, 16), r.Segment), r.Length, r.TailCRC)
}

func (r *Seal) decode(b []byte) error {
	if len(b) != 16 {
		return sizeError("seal", len(b), 16)
	}
	r.Segment = binary.LittleEndian.Uint64(b)
	r.Length = binary.LittleEndian.Uint32(b[8:])
	r.TailCRC = binary.LittleEndian.Uint32(b[12:])
	return nil
}

func appendCut(dst []byte, c *Cut) []byte {
	if c == nil {
		return append(dst, make([]byte, cutSize)...)
	}
	return le64(le32(le64(dst, c.Segment), c.Offset, 0), c.Alignment)
}

func (r *Tombstone) append(dst []byte) []byte {
	var flags uint32
	if r.Left != nil {
		flags |= flagLeftCut
	}
	if r.Right != nil {
		flags |= flagRightCut
	}
	dst = le32(appendHeader(dst, typeTombstone, tombstoneSize), r.Key, flags)
	dst = le64(dst, uint64(r.Start), uint64(r.End))
	return appendCut(appendCut(dst, r.Left), r.Right)
}

// decodeCut decodes the cut in b into c when set, and checks that an absent cut is
// zero.
func decodeCut(b []byte, set bool, c *Cut) (*Cut, error) {
	if !set {
		return nil, zeros("tombstone", b[:cutSize])
	}
	*c = Cut{
		Segment:   binary.LittleEndian.Uint64(b),
		Offset:    binary.LittleEndian.Uint32(b[8:]),
		Alignment: binary.LittleEndian.Uint64(b[16:]),
	}
	return c, zeros("tombstone", b[12:16])
}

func (r *Tombstone) decode(b []byte) error {
	if len(b) != tombstoneSize {
		return sizeError("tombstone", len(b), tombstoneSize)
	}
	r.Key = binary.LittleEndian.Uint32(b)
	flags := binary.LittleEndian.Uint32(b[4:])
	if flags&^(flagLeftCut|flagRightCut) != 0 {
		return errors.Wrapf(
			place.ErrCorrupt,
			"tombstone record of key %d has unknown flags %#x",
			r.Key,
			flags,
		)
	}
	r.Start = int64(binary.LittleEndian.Uint64(b[8:]))
	r.End = int64(binary.LittleEndian.Uint64(b[16:]))
	var err error
	if r.Left, err = decodeCut(b[24:], flags&flagLeftCut != 0, &r.left); err != nil {
		return err
	}
	r.Right, err = decodeCut(b[48:], flags&flagRightCut != 0, &r.right)
	return err
}

func (r *Replace) append(dst []byte) []byte {
	dst = appendHeader(dst, typeReplace, replaceBaseSize+pointerSize*len(r.Pointers))
	dst = le32(le64(dst, r.Old, r.New), r.Length, r.TailCRC, uint32(len(r.Pointers)), 0)
	for _, p := range r.Pointers {
		dst = le32(dst, p.Offset, p.Length, p.Samples, 0)
		dst = le64(dst, p.Alignment, uint64(p.First), uint64(p.Last))
	}
	return dst
}

func (r *Replace) decode(b []byte) error {
	n := uint64(0)
	if len(b) >= 28 {
		n = uint64(binary.LittleEndian.Uint32(b[24:]))
	}
	if len(b) < replaceBaseSize || uint64(len(b)) != replaceBaseSize+pointerSize*n {
		return errors.Wrapf(
			place.ErrCorrupt,
			"replace record of %d bytes does not hold 32 bytes and its %d pointers",
			len(b),
			n,
		)
	}
	r.Old = binary.LittleEndian.Uint64(b)
	r.New = binary.LittleEndian.Uint64(b[8:])
	r.Length = binary.LittleEndian.Uint32(b[16:])
	r.TailCRC = binary.LittleEndian.Uint32(b[20:])
	if err := zeros("replace", b[28:32]); err != nil {
		return err
	}
	r.Pointers = r.Pointers[:0]
	for at := replaceBaseSize; at < len(b); at += pointerSize {
		if err := zeros("replace", b[at+12:at+16]); err != nil {
			return err
		}
		r.Pointers = append(r.Pointers, Pointer{
			Offset:    binary.LittleEndian.Uint32(b[at:]),
			Length:    binary.LittleEndian.Uint32(b[at+4:]),
			Samples:   binary.LittleEndian.Uint32(b[at+8:]),
			Alignment: binary.LittleEndian.Uint64(b[at+16:]),
			First:     int64(binary.LittleEndian.Uint64(b[at+24:])),
			Last:      int64(binary.LittleEndian.Uint64(b[at+32:])),
		})
	}
	return nil
}

func (r *Drop) append(dst []byte) []byte {
	return le32(appendHeader(dst, typeDrop, 8), r.Key, 0)
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

func (r *Check) append(dst []byte) []byte {
	return le64(appendHeader(dst, typeCheck, 8), r.Segment)
}

func (r *Check) decode(b []byte) error {
	if len(b) != 8 {
		return sizeError("check", len(b), 8)
	}
	r.Segment = binary.LittleEndian.Uint64(b)
	return nil
}

func (r *Intent) append(dst []byte) []byte {
	var flags uint32
	if r.Cleared {
		flags = flagIntentCleared
	}
	return le32(appendHeader(dst, typeIntent, 8), r.Key, flags)
}

func (r *Intent) decode(b []byte) error {
	if len(b) != 8 {
		return sizeError("intent", len(b), 8)
	}
	r.Key = binary.LittleEndian.Uint32(b)
	flags := binary.LittleEndian.Uint32(b[4:])
	if flags&^flagIntentCleared != 0 {
		return errors.Wrapf(
			place.ErrCorrupt,
			"intent record of key %d has unknown flags %#x",
			r.Key,
			flags,
		)
	}
	r.Cleared = flags&flagIntentCleared != 0
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
	if !allZero(b) {
		return errors.Wrapf(place.ErrCorrupt, "%s record has a set reserved byte", name)
	}
	return nil
}
