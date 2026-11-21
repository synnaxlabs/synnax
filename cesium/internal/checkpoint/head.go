// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Package checkpoint writes and reads the heads of a Cesium database. A head holds the
// state after a log sequence, so recovery replays only the logs from its generation.
package checkpoint

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/invariants"
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
	recordVersion uint16 = 1
	tableEntry           = 24
	sectorCRC            = 508
	// noPart is the part index of a channel that no part holds.
	noPart        uint32 = 0xFFFFFFFF
	flagSealed    uint8  = 1
	sectionParts         = 1
	sectionDead          = 2
	sectionKills         = 3
	sectionChans         = 4
	sectionSegs          = 5
	sectionPtrs          = 6
	sectionTombs         = 7
	sectionClocks        = 8
	sectionIntent        = 9
	sectionCount         = 9
)

var (
	magic = []byte("CSCHKPT\x00")
	// recordSizes holds the record size of each section kind of a head, from 1.
	recordSizes = [sectionCount + 1]uint32{0, 48, 1, 24, 20, 32, 36, 96, 4, 4}
)

// Name returns the file name of the head of generation.
func Name(generation uint64) string { return fmt.Sprintf("%016x.head", generation) }

// Encode returns the bytes of the head of generation in database db that holds s. The
// bytes depend on db, generation, and s alone.
func Encode(db, generation uint64, s *state.Snapshot) []byte {
	b := encode(db, generation, s)
	if invariants.Enabled {
		checkRoundTrip(b)
	}
	return b
}

func encode(db, generation uint64, s *state.Snapshot) []byte {
	clocks := clocksOf(s)
	var l layout
	l.counts[sectionChans] = uint64(len(s.Channels))
	for _, c := range s.Channels {
		l.counts[sectionSegs] += uint64(len(c.Segments))
		for _, seg := range c.Segments {
			l.counts[sectionPtrs] += uint64(len(seg.Pointers))
		}
	}
	l.counts[sectionClocks] = uint64(len(clocks))
	l.counts[sectionIntent] = uint64(len(s.Intents))
	l.place()
	b := make([]byte, l.length)
	putChannels(b, &l, s)
	at := l.offsets[sectionClocks]
	for _, c := range clocks {
		binary.LittleEndian.PutUint32(b[at:], c)
		at += 4
	}
	at = l.offsets[sectionIntent]
	for _, k := range s.Intents {
		binary.LittleEndian.PutUint32(b[at:], k)
		at += 4
	}
	for kind := 1; kind <= sectionCount; kind++ {
		e := b[l.table+uint64(kind-1)*tableEntry:]
		binary.LittleEndian.PutUint16(e, uint16(kind))
		binary.LittleEndian.PutUint16(e[2:], recordVersion)
		binary.LittleEndian.PutUint32(e[4:], recordSizes[kind])
		putLE64(e[8:], l.offsets[kind], l.counts[kind])
	}
	for i, off := 0, uint64(BodyStart); off < l.chunks; i, off = i+1, off+ChunkSize {
		end := min(off+ChunkSize, l.chunks)
		sum := place.Head(db, generation, off).Sum(b[off:end])
		binary.LittleEndian.PutUint32(b[l.chunks+4*uint64(i):], sum)
	}
	putSector(b, &l, db, generation, s)
	return b
}

// layout holds where each part of a head goes.
type layout struct {
	offsets, counts       [sectionCount + 1]uint64
	table, chunks, length uint64
}

// place sets the offsets of l from its counts.
func (l *layout) place() {
	end := uint64(BodyStart)
	for kind := 1; kind <= sectionCount; kind++ {
		l.offsets[kind] = (end + 7) &^ 7
		end = l.offsets[kind] + l.counts[kind]*uint64(recordSizes[kind])
	}
	l.table = (end + 7) &^ 7
	l.chunks = l.table + sectionCount*tableEntry
	l.length = l.chunks + 4*((l.chunks-BodyStart+ChunkSize-1)/ChunkSize)
}

func putChannels(b []byte, l *layout, s *state.Snapshot) {
	cAt, sAt := l.offsets[sectionChans], l.offsets[sectionSegs]
	pAt := l.offsets[sectionPtrs]
	var segs, ptrs uint32
	for _, c := range s.Channels {
		putLE32(b[cAt:], c.Key, segs, 0, noPart, noPart)
		cAt += uint64(recordSizes[sectionChans])
		for _, seg := range c.Segments {
			binary.LittleEndian.PutUint64(b[sAt:], seg.ID)
			putLE32(b[sAt+8:], seg.Clock, seg.Length, seg.TailCRC, ptrs)
			binary.LittleEndian.PutUint16(b[sAt+24:], seg.Version)
			b[sAt+26] = seg.DataType
			if seg.Sealed {
				b[sAt+27] = flagSealed
			}
			sAt += uint64(recordSizes[sectionSegs])
			for _, p := range seg.Pointers {
				putLE64(b[pAt:], uint64(p.First), uint64(p.Last), uint64(p.Alignment))
				putLE32(b[pAt+24:], p.Offset, p.Length, p.Samples)
				pAt += uint64(recordSizes[sectionPtrs])
			}
			ptrs += uint32(len(seg.Pointers))
			segs++
		}
	}
}

func putSector(b []byte, l *layout, db, generation uint64, s *state.Snapshot) {
	h := b[:SectorSize]
	copy(h, magic)
	binary.LittleEndian.PutUint16(h[8:], formatVersion)
	h[10] = kindHead
	putLE64(h[16:], db, generation, s.Sequence)
	putLE64(h[40:], s.Limits.Segments, s.Limits.Alignments, s.Limits.Parts)
	putLE64(h[64:], l.length, l.table)
	binary.LittleEndian.PutUint32(h[80:], sectionCount)
	binary.LittleEndian.PutUint64(h[88:], l.chunks)
	tableCRC := place.Head(db, generation, l.chunks).Sum(b[l.chunks:])
	binary.LittleEndian.PutUint32(h[96:], tableCRC)
	sum := place.Head(db, generation, 0).Sum(h[:sectorCRC])
	binary.LittleEndian.PutUint32(h[sectorCRC:], sum)
}

// clocksOf returns the distinct clock keys that the data segments of s name, in rising
// order.
func clocksOf(s *state.Snapshot) []uint32 {
	var clocks []uint32
	for _, c := range s.Channels {
		for _, seg := range c.Segments {
			if seg.Clock != c.Key {
				clocks = append(clocks, seg.Clock)
			}
		}
	}
	slices.Sort(clocks)
	return slices.Compact(clocks)
}

func putLE32(b []byte, vs ...uint32) {
	for i, v := range vs {
		binary.LittleEndian.PutUint32(b[4*i:], v)
	}
}

func putLE64(b []byte, vs ...uint64) {
	for i, v := range vs {
		binary.LittleEndian.PutUint64(b[8*i:], v)
	}
}

// checkRoundTrip panics when the head b does not decode, or decodes to a state that
// encodes to other bytes.
func checkRoundTrip(b []byte) {
	h, err := decodeSector(b)
	if err == nil {
		err = checkChunkTable(b, h)
	}
	for i := uint64(0); err == nil && i < chunkCount(h); i++ {
		err = checkChunk(b, h, i)
	}
	var s *state.Snapshot
	if err == nil {
		s, err = parse(b, h)
	}
	if err != nil {
		panic(errors.Wrap(err, "checkpoint: an encoded head does not decode"))
	}
	if !bytes.Equal(encode(h.DB, h.Generation, s), b) {
		panic(errors.New("checkpoint: a decoded head encodes to other bytes"))
	}
}
