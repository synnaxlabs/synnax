// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package checkpoint

import (
	"bytes"
	"encoding/binary"

	"github.com/synnaxlabs/cesium/internal/log"
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// ErrIncomplete is the error of a head whose header sector is zero: its switch never
// reached its commit point.
var ErrIncomplete = errors.New("head is not complete")

// Header is the header sector of a head.
type Header struct {
	// DB is the identity of the database.
	DB uint64
	// Generation is the generation of the head.
	Generation uint64
	// Sequence is the sequence of the last log entry that the head covers.
	Sequence uint64
	Limits   state.Limits
	// Length is the length of the file.
	Length uint64
	table  uint64
	chunks uint64
}

// Decode decodes and checks the head in b. It reads no byte between the header sector
// and the body, which no checksum covers. It returns ErrIncomplete for a zero header
// sector, an error that wraps log.ErrUnknown for a format this binary does not know,
// and an error that wraps place.ErrCorrupt for any other check that fails.
func Decode(b []byte) (Header, *state.Snapshot, error) {
	h, err := decodeSector(b)
	if err != nil {
		return h, nil, err
	}
	if err = checkChunkTable(b, h); err != nil {
		return h, nil, err
	}
	for i := range chunkCount(h) {
		if err = checkChunk(b, h, i); err != nil {
			return h, nil, err
		}
	}
	s, err := parse(b, h)
	return h, s, err
}

func corrupt(format string, args ...any) error {
	return errors.Wrapf(place.ErrCorrupt, "head: "+format, args...)
}

func decodeSector(b []byte) (Header, error) {
	if len(b) < SectorSize {
		return Header{}, corrupt("file of %d bytes has no header sector", len(b))
	}
	s := b[:SectorSize]
	if allZero(s) {
		return Header{}, ErrIncomplete
	}
	h := Header{
		DB:         binary.LittleEndian.Uint64(s[16:]),
		Generation: binary.LittleEndian.Uint64(s[24:]),
		Sequence:   binary.LittleEndian.Uint64(s[32:]),
		Limits: state.Limits{
			Segments:   binary.LittleEndian.Uint64(s[40:]),
			Alignments: binary.LittleEndian.Uint64(s[48:]),
			Parts:      binary.LittleEndian.Uint64(s[56:]),
		},
		Length: binary.LittleEndian.Uint64(s[64:]),
		table:  binary.LittleEndian.Uint64(s[72:]),
		chunks: binary.LittleEndian.Uint64(s[88:]),
	}
	want := place.Head(h.DB, h.Generation, 0).Sum(s[:sectorCRC])
	if !bytes.Equal(s[:8], magic) || binary.LittleEndian.Uint32(s[sectorCRC:]) != want {
		return h, corrupt("header sector fails its check")
	}
	if v := binary.LittleEndian.Uint16(s[8:]); v != formatVersion {
		return h, errors.Wrapf(log.ErrUnknown, "head format version %d", v)
	}
	if s[10] != kindHead {
		return h, corrupt("header sector has kind %d, not a head", s[10])
	}
	if n := binary.LittleEndian.Uint32(s[80:]); n != sectionCount {
		return h, errors.Wrapf(log.ErrUnknown, "head with %d sections", n)
	}
	if !allZero(s[11:16]) || !allZero(s[84:88]) || !allZero(s[100:sectorCRC]) {
		return h, corrupt("header sector has a set reserved byte")
	}
	if h.Length != uint64(len(b)) {
		return h, corrupt("file is %d bytes, but its header says %d", len(b), h.Length)
	}
	if h.table < BodyStart ||
		h.chunks != h.table+sectionCount*tableEntry ||
		h.chunks+4*chunkCount(h) != h.Length {
		return h, corrupt("section table at %d and chunk table at %d do not fit a "+
			"file of %d bytes", h.table, h.chunks, h.Length)
	}
	return h, nil
}

// chunkCount returns the number of chunks of the body of h. It is bounded by the
// length that decodeSector checked.
func chunkCount(h Header) uint64 {
	if h.chunks < BodyStart {
		return 0
	}
	return (h.chunks - BodyStart + ChunkSize - 1) / ChunkSize
}

func checkChunkTable(b []byte, h Header) error {
	want := binary.LittleEndian.Uint32(b[96:])
	if place.Head(h.DB, h.Generation, h.chunks).Sum(b[h.chunks:h.Length]) != want {
		return corrupt("chunk table fails its check")
	}
	return nil
}

// chunkRange returns the file range of chunk i of h.
func chunkRange(h Header, i uint64) (lo, hi uint64) {
	lo = BodyStart + i*ChunkSize
	return lo, min(lo+ChunkSize, h.chunks)
}

func checkChunk(b []byte, h Header, i uint64) error {
	lo, hi := chunkRange(h, i)
	want := binary.LittleEndian.Uint32(b[h.chunks+4*i:])
	if place.Head(h.DB, h.Generation, lo).Sum(b[lo:hi]) != want {
		return corrupt("chunk at %d fails its check", lo)
	}
	return nil
}

type section struct {
	off   uint64
	count uint64
}

func sections(b []byte, h Header) ([sectionCount + 1]section, error) {
	var out [sectionCount + 1]section
	end := uint64(BodyStart)
	for kind := 1; kind <= sectionCount; kind++ {
		e := b[h.table+uint64(kind-1)*tableEntry:]
		if k := binary.LittleEndian.Uint16(e); k != uint16(kind) {
			return out, corrupt("section %d of the table has kind %d", kind, k)
		}
		if v := binary.LittleEndian.Uint16(e[2:]); v != recordVersion {
			return out, errors.Wrapf(
				log.ErrUnknown,
				"head section %d version %d",
				kind,
				v,
			)
		}
		size := binary.LittleEndian.Uint32(e[4:])
		s := section{
			off:   binary.LittleEndian.Uint64(e[8:]),
			count: binary.LittleEndian.Uint64(e[16:]),
		}
		if size != recordSizes[kind] {
			return out, corrupt(
				"section %d has records of %d bytes, not %d",
				kind,
				size,
				recordSizes[kind],
			)
		}
		if s.off != (end+7)&^7 || s.count > (h.table-s.off)/uint64(size) {
			return out, corrupt(
				"section %d at %d with %d records is out of place",
				kind,
				s.off,
				s.count,
			)
		}
		end = s.off + s.count*uint64(size)
		out[kind] = s
	}
	if (end+7)&^7 != h.table {
		return out, corrupt("section table at %d does not follow the last section",
			h.table)
	}
	for _, kind := range []int{sectionParts, sectionDead, sectionKills, sectionTombs} {
		if out[kind].count != 0 {
			return out, errors.Wrapf(log.ErrUnknown, "head with parts or tombstones "+
				"(section %d)", kind)
		}
	}
	return out, nil
}

func parse(b []byte, h Header) (*state.Snapshot, error) {
	secs, err := sections(b, h)
	if err != nil {
		return nil, err
	}
	s := &state.Snapshot{Sequence: h.Sequence, Limits: h.Limits}
	if s.Channels, err = parseChannels(b, secs); err != nil {
		return nil, err
	}
	clocks := Clocks(s)
	cs := secs[sectionClocks]
	if cs.count != uint64(len(clocks)) {
		return nil, corrupt(
			"%d clocks, but the segments name %d",
			cs.count,
			len(clocks),
		)
	}
	for i, c := range clocks {
		if got := binary.LittleEndian.Uint32(b[cs.off+4*uint64(i):]); got != c {
			return nil, corrupt("clock %d is %d, but the segments name %d", i, got, c)
		}
	}
	is := secs[sectionIntent]
	if is.count > 0 {
		s.Intents = make([]uint32, is.count)
	}
	for i := range s.Intents {
		s.Intents[i] = binary.LittleEndian.Uint32(b[is.off+4*uint64(i):])
		if i > 0 && s.Intents[i] <= s.Intents[i-1] {
			return nil, corrupt("intent of key %d is not above key %d", s.Intents[i],
				s.Intents[i-1])
		}
	}
	return s, nil
}

func parseChannels(
	b []byte,
	secs [sectionCount + 1]section,
) ([]*state.Channel, error) {
	cs, ss := secs[sectionChans], secs[sectionSegs]
	var channels []*state.Channel
	if cs.count > 0 {
		channels = make([]*state.Channel, cs.count)
	}
	var nextSeg, nextPtr uint64
	for i := range channels {
		r := b[cs.off+uint64(i)*20:]
		c := &state.Channel{Key: binary.LittleEndian.Uint32(r)}
		first := uint64(binary.LittleEndian.Uint32(r[4:]))
		end := ss.count
		if uint64(i)+1 < cs.count {
			end = uint64(binary.LittleEndian.Uint32(b[cs.off+uint64(i+1)*20+4:]))
		}
		switch {
		case i > 0 && c.Key <= channels[i-1].Key:
			return nil, corrupt("key %d is not above key %d", c.Key, channels[i-1].Key)
		case first != nextSeg || end <= first || end > ss.count:
			return nil, corrupt(
				"key %d has segment records %d to %d",
				c.Key,
				first,
				end,
			)
		case binary.LittleEndian.Uint32(r[8:]) != 0 ||
			binary.LittleEndian.Uint32(r[12:]) != noPart ||
			binary.LittleEndian.Uint32(r[16:]) != noPart:
			return nil, corrupt("key %d names tombstones or parts", c.Key)
		}
		if err := parseSegments(b, secs, c, first, end, &nextPtr); err != nil {
			return nil, err
		}
		nextSeg = end
		channels[i] = c
	}
	if nextSeg != ss.count || nextPtr != secs[sectionPtrs].count {
		return nil, corrupt("%d segment and %d pointer records, but the channels name "+
			"%d and %d", ss.count, secs[sectionPtrs].count, nextSeg, nextPtr)
	}
	return channels, nil
}

// parseSegments parses segment records first to end into c.
func parseSegments(
	b []byte,
	secs [sectionCount + 1]section,
	c *state.Channel,
	first, end uint64,
	nextPtr *uint64,
) error {
	c.Segments = make([]state.Segment, end-first)
	for j := range c.Segments {
		seg, err := parseSegment(b, secs, first+uint64(j), nextPtr)
		if err != nil {
			return err
		}
		if j > 0 && seg.ID <= c.Segments[j-1].ID {
			return corrupt(
				"segment %d of key %d is not above segment %d",
				seg.ID,
				c.Key,
				c.Segments[j-1].ID,
			)
		}
		c.Segments[j] = seg
	}
	return nil
}

func parseSegment(
	b []byte,
	secs [sectionCount + 1]section,
	index uint64,
	nextPtr *uint64,
) (state.Segment, error) {
	ss, ps := secs[sectionSegs], secs[sectionPtrs]
	r := b[ss.off+index*32:]
	seg := state.Segment{
		ID:       binary.LittleEndian.Uint64(r),
		Clock:    binary.LittleEndian.Uint32(r[8:]),
		Length:   binary.LittleEndian.Uint32(r[12:]),
		TailCRC:  binary.LittleEndian.Uint32(r[16:]),
		Version:  binary.LittleEndian.Uint16(r[24:]),
		DataType: r[26],
		Sealed:   r[27]&flagSealed != 0,
	}
	first := uint64(binary.LittleEndian.Uint32(r[20:]))
	end := ps.count
	if index+1 < ss.count {
		end = uint64(binary.LittleEndian.Uint32(b[ss.off+(index+1)*32+20:]))
	}
	switch {
	case r[27]&^flagSealed != 0 || !allZero(r[28:32]):
		return seg, corrupt("segment %d has unknown flags or a set reserved byte",
			seg.ID)
	case first != *nextPtr || end < first || end > ps.count:
		return seg, corrupt("segment %d has pointer records %d to %d", seg.ID, first,
			end)
	}
	parsePointers(b, ps, &seg, first, end)
	*nextPtr = end
	return seg, nil
}

func parsePointers(b []byte, ps section, seg *state.Segment, first, end uint64) {
	if end > first {
		seg.Pointers = make([]state.Pointer, end-first)
	}
	for i := range seg.Pointers {
		r := b[ps.off+(first+uint64(i))*36:]
		seg.Pointers[i] = state.Pointer{
			First:     telem.TimeStamp(binary.LittleEndian.Uint64(r)),
			Last:      telem.TimeStamp(binary.LittleEndian.Uint64(r[8:])),
			Alignment: telem.Alignment(binary.LittleEndian.Uint64(r[16:])),
			Offset:    binary.LittleEndian.Uint32(r[24:]),
			Length:    binary.LittleEndian.Uint32(r[28:]),
			Samples:   binary.LittleEndian.Uint32(r[32:]),
		}
	}
}

func allZero(b []byte) bool {
	for _, c := range b {
		if c != 0 {
			return false
		}
	}
	return true
}
