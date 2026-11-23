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
	"slices"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

var (
	// ErrIncomplete is the error of a head whose header sector is zero: its switch
	// never reached its commit point.
	ErrIncomplete = errors.New("head is not complete")
	// errUnknown is the error of a head format that this binary does not know. A newer
	// binary wrote it.
	errUnknown = errors.New("unknown head format")
)

// Header is the header sector of a head or a part.
type Header struct {
	// DB is the identity of the database.
	DB uint64
	// Generation is the generation of a head, or the identity of a part.
	Generation uint64
	// Sequence is the sequence of the last log entry that a head covers, or the
	// generation of the switch that wrote a part.
	Sequence uint64
	Limits   state.Limits
	// Length is the length of the file.
	Length uint64
	// CRC is the CRC of the header sector.
	CRC uint32
	// Parts are the parts that a head names.
	Parts  []Part
	table  uint64
	chunks uint64
}

func (f *format) corrupt(format string, args ...any) error {
	return errors.Wrapf(place.ErrCorrupt, f.name+": "+format, args...)
}

func (f *format) decodeSector(b []byte) (Header, error) {
	return f.decodeSectorOf(b, uint64(len(b)))
}

// decodeSectorOf decodes the header sector at the start of b, of a file of size bytes.
func (f *format) decodeSectorOf(b []byte, size uint64) (Header, error) {
	s := b[:min(uint64(len(b)), size, SectorSize)]
	if allZero(s) {
		return Header{}, ErrIncomplete
	}
	if len(s) < SectorSize {
		return Header{}, f.corrupt("file of %d bytes has no header sector", size)
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
		CRC:    binary.LittleEndian.Uint32(s[sectorCRC:]),
		table:  binary.LittleEndian.Uint64(s[72:]),
		chunks: binary.LittleEndian.Uint64(s[88:]),
	}
	want := f.place(h.DB, h.Generation, 0).Sum(s[:sectorCRC])
	if !bytes.Equal(s[:8], magic) || h.CRC != want {
		return h, f.corrupt("header sector fails its check")
	}
	if v := binary.LittleEndian.Uint16(s[8:]); v != formatVersion {
		return h, errors.Wrapf(errUnknown, "head format version %d", v)
	}
	if s[10] != f.kind {
		return h, f.corrupt("header sector has kind %d, not a %s", s[10], f.name)
	}
	if n := binary.LittleEndian.Uint32(s[80:]); n != uint32(len(f.sections)) {
		return h, errors.Wrapf(errUnknown, "%s with %d sections", f.name, n)
	}
	if !allZero(s[11:16]) || !allZero(s[84:88]) || !allZero(s[100:sectorCRC]) {
		return h, f.corrupt("header sector has a set reserved byte")
	}
	if h.Length != size {
		return h, f.corrupt(
			"file is %d bytes, but its header says %d",
			size,
			h.Length,
		)
	}
	if h.table < BodyStart || h.table > h.Length || h.chunks > h.Length ||
		h.chunks != h.table+uint64(len(f.sections))*tableEntry ||
		h.Length-h.chunks != 4*chunkCount(h) {
		return h, f.corrupt("section table at %d and chunk table at %d do not fit a "+
			"file of %d bytes", h.table, h.chunks, h.Length)
	}
	return h, nil
}

// chunkCount returns the number of chunks of the body of h. It is bounded by the length
// that decodeSector checked.
func chunkCount(h Header) uint64 {
	if h.chunks < BodyStart {
		return 0
	}
	return (h.chunks - BodyStart + ChunkSize - 1) / ChunkSize
}

func (f *format) checkChunkTable(b []byte, h Header) error {
	want := binary.LittleEndian.Uint32(b[96:])
	if f.place(h.DB, h.Generation, h.chunks).Sum(b[h.chunks:h.Length]) != want {
		return f.corrupt("chunk table fails its check")
	}
	return nil
}

// chunkRange returns the file range of chunk i of h.
func chunkRange(h Header, i uint64) (lo, hi uint64) {
	lo = BodyStart + i*ChunkSize
	return lo, min(lo+ChunkSize, h.chunks)
}

func (f *format) checkChunk(b []byte, h Header, i uint64) error {
	lo, hi := chunkRange(h, i)
	want := binary.LittleEndian.Uint32(b[h.chunks+4*i:])
	if f.place(h.DB, h.Generation, lo).Sum(b[lo:hi]) != want {
		return f.corrupt("chunk at %d fails its check", lo)
	}
	return nil
}

type section struct {
	off   uint64
	count uint64
}

// sectionsOf parses the section table of h, whose bytes start table.
func (f *format) sectionsOf(table []byte, h Header) ([sectionCount + 1]section, error) {
	var out [sectionCount + 1]section
	end := uint64(BodyStart)
	for i, kind := range f.sections {
		e := table[uint64(i)*tableEntry:]
		if k := binary.LittleEndian.Uint16(e); k != uint16(kind) {
			return out, f.corrupt("section %d of the table has kind %d", kind, k)
		}
		if v := binary.LittleEndian.Uint16(e[2:]); v != recordVersion {
			return out, errors.Wrapf(
				errUnknown,
				"%s section %d version %d",
				f.name,
				kind,
				v,
			)
		}
		size := binary.LittleEndian.Uint32(e[4:])
		s := section{
			off:   binary.LittleEndian.Uint64(e[8:]),
			count: binary.LittleEndian.Uint64(e[16:]),
		}
		if size != f.sizes[kind] {
			return out, f.corrupt(
				"section %d has records of %d bytes, not %d",
				kind,
				size,
				f.sizes[kind],
			)
		}
		if s.off != (end+7)&^7 || s.count > (h.table-s.off)/uint64(size) {
			return out, f.corrupt(
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
		return out, f.corrupt("section table at %d does not follow the last section",
			h.table)
	}
	for _, kind := range []int{sectionKills, sectionTombs} {
		if out[kind].count != 0 {
			return out, errors.Wrapf(errUnknown, "head with kills or tombstones "+
				"(section %d)", kind)
		}
	}
	return out, nil
}

func (f *format) parse(b []byte, h Header) (body, error) {
	bd := body{file: h.Generation, second: h.Sequence, limits: h.Limits}
	secs, err := f.sectionsOf(b[h.table:h.chunks], h)
	if err != nil {
		return bd, err
	}
	if bd.parts, err = f.parseParts(b, secs); err != nil {
		return bd, err
	}
	if bd.channels, err = f.parseChannels(b, secs); err != nil {
		return bd, err
	}
	clocks := clocksOf(bd.channels)
	cs := secs[sectionClocks]
	if cs.count != uint64(len(clocks)) {
		return bd, f.corrupt(
			"%d clocks, but the segments name %d",
			cs.count,
			len(clocks),
		)
	}
	for i, c := range clocks {
		if got := binary.LittleEndian.Uint32(b[cs.off+4*uint64(i):]); got != c {
			return bd, f.corrupt("clock %d is %d, but the segments name %d", i, got, c)
		}
	}
	is := secs[sectionIntent]
	if is.count > 0 {
		bd.intents = make([]uint32, is.count)
	}
	for i := range bd.intents {
		bd.intents[i] = binary.LittleEndian.Uint32(b[is.off+4*uint64(i):])
		if i > 0 && bd.intents[i] <= bd.intents[i-1] {
			return bd, f.corrupt("intent of key %d is not above key %d",
				bd.intents[i], bd.intents[i-1])
		}
	}
	return bd, nil
}

// parseParts parses the part records of a head and their dead bitmaps. A bitmap runs to
// the next one, or to the end of the dead section.
func (f *format) parseParts(b []byte, secs [sectionCount + 1]section) ([]Part, error) {
	ps, ds := secs[sectionParts], secs[sectionDead]
	if ps.count == 0 {
		if ds.count != 0 {
			return nil, f.corrupt("%d dead bitmap bytes, but no part", ds.count)
		}
		return nil, nil
	}
	parts := make([]Part, ps.count)
	offsets := make([]uint32, ps.count)
	for i := range parts {
		r := b[ps.off+uint64(i)*uint64(f.sizes[sectionParts]):]
		parts[i] = Part{
			ID:         binary.LittleEndian.Uint64(r),
			Generation: binary.LittleEndian.Uint64(r[8:]),
			Length:     binary.LittleEndian.Uint64(r[16:]),
			First:      telem.TimeStamp(binary.LittleEndian.Uint64(r[24:])),
			Last:       telem.TimeStamp(binary.LittleEndian.Uint64(r[32:])),
			CRC:        binary.LittleEndian.Uint32(r[44:]),
		}
		offsets[i] = binary.LittleEndian.Uint32(r[40:])
		if i > 0 && parts[i].ID <= parts[i-1].ID {
			return nil, f.corrupt("part %d is not above part %d", parts[i].ID,
				parts[i-1].ID)
		}
	}
	return parts, f.parseBitmaps(b, ds, parts, offsets)
}

// parseBitmaps gives each part with a bitmap offset the bytes from that offset to the
// next one, or to the end of the dead section. The offsets must start at 0 and rise.
func (f *format) parseBitmaps(
	b []byte,
	ds section,
	parts []Part,
	offsets []uint32,
) error {
	var at []int
	for i, o := range offsets {
		if o != noPart {
			at = append(at, i)
		}
	}
	if len(at) == 0 && ds.count != 0 {
		return f.corrupt("%d dead bitmap bytes, but no part names one", ds.count)
	}
	for k, i := range at {
		lo, hi := uint64(offsets[i]), ds.count
		if k+1 < len(at) {
			hi = uint64(offsets[at[k+1]])
		}
		if (k == 0 && lo != 0) || hi <= lo || hi > ds.count {
			return f.corrupt("part %d has its dead bitmap at %d", parts[i].ID, lo)
		}
		parts[i].Dead = slices.Clone(b[ds.off+lo : ds.off+hi])
	}
	return nil
}

func (f *format) parseChannels(
	b []byte,
	secs [sectionCount + 1]section,
) ([]*state.Channel, error) {
	cs, ss := secs[sectionChans], secs[sectionSegs]
	var channels []*state.Channel
	if cs.count > 0 {
		channels = make([]*state.Channel, cs.count)
	}
	sl := slabs{
		channels: make([]state.Channel, cs.count),
		segs:     make([]state.Segment, ss.count),
		ptrs:     make([]state.Pointer, secs[sectionPtrs].count),
	}
	var nextSeg, nextPtr uint64
	size := uint64(f.sizes[sectionChans])
	for i := range channels {
		r := b[cs.off+uint64(i)*size:]
		c := &sl.channels[i]
		c.Key = binary.LittleEndian.Uint32(r)
		first := uint64(binary.LittleEndian.Uint32(r[4:]))
		end := ss.count
		if uint64(i)+1 < cs.count {
			end = uint64(binary.LittleEndian.Uint32(b[cs.off+uint64(i+1)*size+4:]))
		}
		switch {
		case i > 0 && c.Key <= channels[i-1].Key:
			return nil, f.corrupt(
				"key %d is not above key %d",
				c.Key,
				channels[i-1].Key,
			)
		case first != nextSeg || end <= first || end > ss.count:
			return nil, f.corrupt(
				"key %d has segment records %d to %d",
				c.Key,
				first,
				end,
			)
		case f.kind == kindHead && (binary.LittleEndian.Uint32(r[8:]) != 0 ||
			binary.LittleEndian.Uint32(r[12:]) != noPart ||
			binary.LittleEndian.Uint32(r[16:]) != noPart):
			return nil, f.corrupt("key %d names tombstones or parts", c.Key)
		}
		if err := f.parseSegments(b, secs, &sl, c, first, end, &nextPtr); err != nil {
			return nil, err
		}
		nextSeg = end
		channels[i] = c
	}
	if nextSeg != ss.count || nextPtr != secs[sectionPtrs].count {
		return nil, f.corrupt(
			"%d segment and %d pointer records, but the channels name %d and %d",
			ss.count,
			secs[sectionPtrs].count,
			nextSeg,
			nextPtr,
		)
	}
	return channels, nil
}

// slabs hold every channel, segment, and pointer of a file, so a parse allocates once
// per kind. Each slice cut from them has no spare capacity, so an append copies.
type slabs struct {
	channels []state.Channel
	segs     []state.Segment
	ptrs     []state.Pointer
}

// parseSegments parses segment records first to end into c.
func (f *format) parseSegments(
	b []byte,
	secs [sectionCount + 1]section,
	sl *slabs,
	c *state.Channel,
	first, end uint64,
	nextPtr *uint64,
) error {
	c.Segments = sl.segs[first:end:end]
	for j := range c.Segments {
		seg, err := f.parseSegment(b, secs, sl, first+uint64(j), nextPtr)
		if err != nil {
			return err
		}
		if j > 0 && seg.ID <= c.Segments[j-1].ID {
			return f.corrupt(
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

func (f *format) parseSegment(
	b []byte,
	secs [sectionCount + 1]section,
	sl *slabs,
	index uint64,
	nextPtr *uint64,
) (state.Segment, error) {
	ss, ps := secs[sectionSegs], secs[sectionPtrs]
	r := b[ss.off+index*32:]
	seg, first, err := f.decodeSegment(r)
	if err != nil {
		return seg, err
	}
	end := ps.count
	if index+1 < ss.count {
		end = uint64(binary.LittleEndian.Uint32(b[ss.off+(index+1)*32+20:]))
	}
	if first != *nextPtr || end < first || end > ps.count {
		return seg, f.corrupt("segment %d has pointer records %d to %d", seg.ID, first,
			end)
	}
	parsePointers(b, ps, sl, &seg, first, end)
	*nextPtr = end
	return seg, nil
}

func parsePointers(
	b []byte,
	ps section,
	sl *slabs,
	seg *state.Segment,
	first, end uint64,
) {
	if end > first {
		seg.Pointers = sl.ptrs[first:end:end]
	}
	for i := range seg.Pointers {
		seg.Pointers[i] = decodePointer(b[ps.off+(first+uint64(i))*36:])
	}
}

// decodeSegment decodes the segment record r, and returns the index of its first
// pointer record.
func (f *format) decodeSegment(r []byte) (state.Segment, uint64, error) {
	seg := state.Segment{
		ID:       binary.LittleEndian.Uint64(r),
		Clock:    binary.LittleEndian.Uint32(r[8:]),
		Length:   binary.LittleEndian.Uint32(r[12:]),
		TailCRC:  binary.LittleEndian.Uint32(r[16:]),
		Version:  binary.LittleEndian.Uint16(r[24:]),
		DataType: r[26],
		Sealed:   r[27]&flagSealed != 0,
	}
	if r[27]&^flagSealed != 0 || !allZero(r[28:32]) {
		return seg, 0, f.corrupt(
			"segment %d has unknown flags or a set reserved byte",
			seg.ID,
		)
	}
	return seg, uint64(binary.LittleEndian.Uint32(r[20:])), nil
}

// decodePointer decodes the pointer record r.
func decodePointer(r []byte) state.Pointer {
	return state.Pointer{
		First:     telem.TimeStamp(binary.LittleEndian.Uint64(r)),
		Last:      telem.TimeStamp(binary.LittleEndian.Uint64(r[8:])),
		Alignment: telem.Alignment(binary.LittleEndian.Uint64(r[16:])),
		Offset:    binary.LittleEndian.Uint32(r[24:]),
		Length:    binary.LittleEndian.Uint32(r[28:]),
		Samples:   binary.LittleEndian.Uint32(r[32:]),
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
