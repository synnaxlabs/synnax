// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package checkpoint writes and reads the heads and parts of a Cesium database. A head
// holds the state after a log sequence, so recovery replays only the logs from its
// generation. A part holds sealed segments that a head names.
package checkpoint

import (
	"bytes"
	"cmp"
	"encoding/binary"
	"fmt"
	"slices"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/invariants"
	"github.com/synnaxlabs/x/telem"
)

const (
	// SectorSize is the size of the header sector, which is zero until a head is
	// complete.
	SectorSize = 512
	// BodyStart is the offset of the first section.
	BodyStart = 4096
	// ChunkSize is the number of bytes that each checksum of the chunk table covers.
	ChunkSize = 65_536

	formatVersion uint16 = 1
	kindHead      uint8  = 4
	kindPart      uint8  = 5
	recordVersion uint16 = 1
	tableEntry           = 24
	sectorCRC            = 508
	// noPart is the part index of a channel that no part holds, and the bitmap offset
	// of a part with no dead segment.
	noPart        uint32 = 0xFFFFFFFF
	flagSealed    uint8  = 1
	flagKept      uint8  = 4
	sectionParts         = 1
	sectionDead          = 2
	sectionKills         = 3
	sectionChans         = 4
	sectionSegs          = 5
	sectionPtrs          = 6
	sectionClocks        = 8
	sectionIntent        = 9
	sectionCount         = 9
)

var magic = []byte("CSCHKPT\x00")

// format is the layout of one kind of checkpoint file.
type format struct {
	name string
	kind uint8
	// sections are the section kinds of the file, in file order.
	sections []int
	// sizes holds the record size of each section kind, from 1.
	sizes [sectionCount + 1]uint32
	place func(db, file, offset uint64) place.Place
}

var (
	headFormat = format{
		name:     "head",
		kind:     kindHead,
		sections: []int{1, 2, 3, 4, 5, 6, 8, 9},
		sizes:    [sectionCount + 1]uint32{0, 48, 1, 24, 16, 32, 36, 0, 4, 4},
		place:    place.Head,
	}
	partFormat = format{
		name:     "part",
		kind:     kindPart,
		sections: []int{sectionChans, sectionSegs, sectionPtrs, sectionClocks},
		sizes:    [sectionCount + 1]uint32{0, 0, 0, 0, 8, 32, 36, 0, 4, 0},
		place:    place.Part,
	}
)

// Part is a part that a head names.
type Part struct {
	// ID is the identity of the part, and Generation the generation of the switch that
	// wrote it.
	ID, Generation uint64
	// Length is the length of the part file.
	Length uint64
	// First and Last are the lowest and highest pointer stamps in the part.
	First, Last telem.TimeStamp
	// CRC is the header CRC of the part.
	CRC uint32
	// Dead is the bitmap of the dead segment records of the part, or nil when none is.
	Dead []byte
}

// Live reports whether segment record i of p is not dead.
func (p Part) Live(i int) bool {
	return i/8 >= len(p.Dead) || p.Dead[i/8]&(1<<(i%8)) == 0
}

// Name returns the file name of the head of generation.
func Name(generation uint64) string { return fmt.Sprintf("%016x.head", generation) }

// Encode returns the bytes of the head of generation in database db that holds s. The
// bytes depend on db, generation, and s alone.
func Encode(db, generation uint64, s *state.Snapshot) []byte {
	return encodeHead(db, generation, s, nil, nil)
}

// encodeHead is Encode for a head that names parts.
func encodeHead(
	db, generation uint64,
	s *state.Snapshot,
	parts []Part,
	ranges []state.Range,
) []byte {
	return headFormat.encode(db, body{
		file:     generation,
		second:   s.Sequence,
		limits:   s.Limits,
		channels: s.Channels,
		intents:  s.Intents,
		parts:    parts,
		ranges:   ranges,
		ranged:   true,
	})
}

// body is what a checkpoint file holds.
type body struct {
	// file is the generation of a head or the identity of a part, and second the log
	// sequence of a head or the generation of the switch that wrote a part.
	file, second uint64
	limits       state.Limits
	channels     []*state.Channel
	intents      []uint32
	parts        []Part
	// ranges are the part ranges of a head in rising order of key. A key of ranges
	// that channels lack gets a channel record with no segment.
	ranges []state.Range
	// ranged is true when ranges alone name the part ranges, as for the head of a
	// switch, whose channels may carry the ranges of the head before.
	ranged bool
}

func (f *format) encode(db uint64, bd body) []byte {
	b := f.encodeBody(db, bd)
	if invariants.Enabled {
		f.checkRoundTrip(b)
	}
	return b
}

func (f *format) encodeBody(db uint64, bd body) []byte {
	clocks := clocksOf(bd.channels)
	l := layout{f: f}
	l.counts[sectionParts] = uint64(len(bd.parts))
	for _, p := range bd.parts {
		l.counts[sectionDead] += uint64(len(p.Dead))
	}
	eachChannel(bd, func(uint32, []state.Segment, state.Range) {
		l.counts[sectionChans]++
	})
	for _, c := range bd.channels {
		l.counts[sectionSegs] += uint64(len(c.Segments))
		for _, seg := range c.Segments {
			l.counts[sectionPtrs] += uint64(len(seg.Pointers))
		}
	}
	l.counts[sectionClocks] = uint64(len(clocks))
	l.counts[sectionIntent] = uint64(len(bd.intents))
	l.place()
	b := make([]byte, l.length)
	putParts(b, &l, bd.parts)
	putChannels(b, &l, bd)
	putUint32s(b[l.offsets[sectionClocks]:], clocks)
	putUint32s(b[l.offsets[sectionIntent]:], bd.intents)
	for i, kind := range f.sections {
		e := b[l.table+uint64(i)*tableEntry:]
		binary.LittleEndian.PutUint16(e, uint16(kind))
		binary.LittleEndian.PutUint16(e[2:], recordVersion)
		binary.LittleEndian.PutUint32(e[4:], f.sizes[kind])
		putLE64(e[8:], l.offsets[kind], l.counts[kind])
	}
	for i, off := 0, uint64(BodyStart); off < l.chunks; i, off = i+1, off+ChunkSize {
		end := min(off+ChunkSize, l.chunks)
		sum := f.place(db, bd.file, off).Sum(b[off:end])
		binary.LittleEndian.PutUint32(b[l.chunks+4*uint64(i):], sum)
	}
	f.putSector(b, &l, db, bd)
	return b
}

// layout holds where each part of a checkpoint file goes.
type layout struct {
	f                     *format
	offsets, counts       [sectionCount + 1]uint64
	table, chunks, length uint64
}

// place sets the offsets of l from its counts.
func (l *layout) place() {
	end := uint64(BodyStart)
	for _, kind := range l.f.sections {
		l.offsets[kind] = (end + 7) &^ 7
		end = l.offsets[kind] + l.counts[kind]*uint64(l.f.sizes[kind])
	}
	l.table = (end + 7) &^ 7
	l.chunks = l.table + uint64(len(l.f.sections))*tableEntry
	l.length = l.chunks + 4*((l.chunks-BodyStart+ChunkSize-1)/ChunkSize)
}

// putParts puts the part records of a head and their dead bitmaps.
func putParts(b []byte, l *layout, parts []Part) {
	at, dead := l.offsets[sectionParts], l.offsets[sectionDead]
	var bitmaps uint32
	for _, p := range parts {
		putLE64(b[at:], p.ID, p.Generation, p.Length, uint64(p.First), uint64(p.Last))
		offset := noPart
		if len(p.Dead) > 0 {
			offset = bitmaps
			copy(b[dead+uint64(bitmaps):], p.Dead)
			bitmaps += uint32(len(p.Dead))
		}
		putLE32(b[at+40:], offset, p.CRC)
		at += uint64(l.f.sizes[sectionParts])
	}
}

// eachChannel calls f with the key, segments, and part range of each channel of bd in
// rising order of key: a channel of bd.channels with the range of its key in bd.ranges,
// or a channel of no segment for a key of bd.ranges alone. Unless bd.ranged, a key with
// no range in bd.ranges keeps the range of its channel.
func eachChannel(bd body, f func(key uint32, segs []state.Segment, r state.Range)) {
	ranges := bd.ranges
	i := 0
	for _, c := range bd.channels {
		for ; i < len(ranges) && ranges[i].Key < c.Key; i++ {
			f(ranges[i].Key, nil, ranges[i])
		}
		r := state.Range{Key: c.Key}
		if c.Parts != nil && !bd.ranged {
			r.Oldest, r.Newest = c.Parts.Oldest, c.Parts.Newest
		}
		if i < len(ranges) && ranges[i].Key == c.Key {
			r = ranges[i]
			i++
		}
		f(c.Key, c.Segments, r)
	}
	for ; i < len(ranges); i++ {
		f(ranges[i].Key, nil, ranges[i])
	}
}

func putChannels(b []byte, l *layout, bd body) {
	cAt, sAt := l.offsets[sectionChans], l.offsets[sectionSegs]
	pAt := l.offsets[sectionPtrs]
	var segs, ptrs uint32
	eachChannel(bd, func(
		key uint32,
		channelSegs []state.Segment,
		r state.Range,
	) {
		if l.f.kind == kindHead {
			putLE32(b[cAt:], key, segs, partIndex(bd.parts, r.Oldest),
				partIndex(bd.parts, r.Newest))
		} else {
			putLE32(b[cAt:], key, segs)
		}
		cAt += uint64(l.f.sizes[sectionChans])
		for _, seg := range channelSegs {
			binary.LittleEndian.PutUint64(b[sAt:], seg.ID)
			putLE32(b[sAt+8:], seg.Clock, seg.Length, seg.TailCRC, ptrs)
			binary.LittleEndian.PutUint16(b[sAt+24:], seg.Version)
			b[sAt+26] = seg.DataType
			if seg.Sealed {
				b[sAt+27] = flagSealed
			}
			if seg.Kept {
				b[sAt+27] |= flagKept
			}
			sAt += uint64(l.f.sizes[sectionSegs])
			for _, p := range seg.Pointers {
				putLE64(b[pAt:], uint64(p.First), uint64(p.Last), uint64(p.Alignment))
				putLE32(b[pAt+24:], p.Offset, p.Length, p.Samples)
				pAt += uint64(l.f.sizes[sectionPtrs])
			}
			ptrs += uint32(len(seg.Pointers))
			segs++
		}
	})
}

// partIndex returns the index in parts of the part of identity id, or noPart for 0.
// parts are in rising order of identity.
func partIndex(parts []Part, id uint64) uint32 {
	if id == 0 {
		return noPart
	}
	i, found := slices.BinarySearchFunc(parts, id, func(p Part, id uint64) int {
		return cmp.Compare(p.ID, id)
	})
	if !found {
		panic(errors.Newf("checkpoint: a channel names part %d, which the head does "+
			"not", id))
	}
	return uint32(i)
}

func (f *format) putSector(b []byte, l *layout, db uint64, bd body) {
	h := b[:SectorSize]
	copy(h, magic)
	binary.LittleEndian.PutUint16(h[8:], formatVersion)
	h[10] = f.kind
	putLE64(h[16:], db, bd.file, bd.second)
	putLE64(h[40:], bd.limits.Segments, bd.limits.Alignments, bd.limits.Parts)
	putLE64(h[64:], l.length, l.table)
	binary.LittleEndian.PutUint32(h[80:], uint32(len(f.sections)))
	binary.LittleEndian.PutUint64(h[88:], l.chunks)
	tableCRC := f.place(db, bd.file, l.chunks).Sum(b[l.chunks:])
	binary.LittleEndian.PutUint32(h[96:], tableCRC)
	sum := f.place(db, bd.file, 0).Sum(h[:sectorCRC])
	binary.LittleEndian.PutUint32(h[sectorCRC:], sum)
}

// clocksOf returns the distinct clock keys that the data segments of channels name, in
// rising order.
func clocksOf(channels []*state.Channel) []uint32 {
	var clocks []uint32
	for _, c := range channels {
		for _, seg := range c.Segments {
			if seg.Clock != c.Key {
				clocks = append(clocks, seg.Clock)
			}
		}
	}
	slices.Sort(clocks)
	return slices.Compact(clocks)
}

func putUint32s(b []byte, vs []uint32) {
	for i, v := range vs {
		binary.LittleEndian.PutUint32(b[4*i:], v)
	}
}

func putLE32(b []byte, vs ...uint32) { putUint32s(b, vs) }

func putLE64(b []byte, vs ...uint64) {
	for i, v := range vs {
		binary.LittleEndian.PutUint64(b[8*i:], v)
	}
}

// checkRoundTrip panics when the file b does not decode, or decodes to a body that
// encodes to other bytes.
func (f *format) checkRoundTrip(b []byte) {
	h, err := f.decodeSector(b)
	if err == nil {
		err = f.checkChunkTable(b, h)
	}
	for i := uint64(0); err == nil && i < chunkCount(h); i++ {
		err = f.checkChunk(b, h, i)
	}
	var bd body
	if err == nil {
		bd, err = f.parse(b, h)
	}
	if err != nil {
		panic(errors.Wrapf(err, "checkpoint: an encoded %s does not decode", f.name))
	}
	if !bytes.Equal(f.encodeBody(h.DB, bd), b) {
		panic(errors.Newf("checkpoint: a decoded %s encodes to other bytes", f.name))
	}
}
