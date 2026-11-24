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
	"slices"
	"sort"

	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/x/errors"
)

// table is the footer and the fences of a version 2 segment.
type table struct {
	db, id uint64
	// last is the last stamp of the segment.
	last                  int64
	entries, samples, raw uint32
	// at is the stream offset of the first entry.
	at     int64
	fences []fence
	bytes  []byte
}

// fence is the stamp and the position of the first entry that starts in a block of the
// table.
type fence struct {
	stamp int64
	pos   uint32
}

func (rd *Reader) corrupt(s Committed, format string, args ...any) error {
	return errors.Wrapf(place.ErrCorrupt, "table of version 2 segment %d of key %d: "+
		format, append([]any{s.ID, s.Key}, args...)...)
}

// loadTable makes rd hold the table of the version 2 segment s.
func (rd *Reader) loadTable(r io.ReaderAt, s Committed) error {
	t := &rd.tab
	if t.id == s.ID && t.db == s.DB && t.id != 0 {
		return nil
	}
	t.id = 0
	if s.Length < BlockSize || s.Length%BlockSize != 0 {
		return rd.corrupt(s, "length %d is no whole number of blocks", s.Length)
	}
	f := rd.word[:footerSize]
	if err := rd.Read(r, s, f, s.Length-footerSize); err != nil {
		return err
	}
	t.last = int64(binary.LittleEndian.Uint64(f))
	t.entries = binary.LittleEndian.Uint32(f[8:])
	t.samples = binary.LittleEndian.Uint32(f[12:])
	t.raw = binary.LittleEndian.Uint32(f[16:])
	if t.entries == 0 || t.entries > t.samples || t.raw > t.samples ||
		encodedSize(t.raw, t.entries) != s.Length {
		return rd.corrupt(s, "footer of %d entries, %d samples, and %d raw stamps "+
			"does not fit length %d", t.entries, t.samples, t.raw, s.Length)
	}
	t.at = tableAt(t.raw)
	n := fenceCount(t.at, t.entries)
	t.bytes = slices.Grow(t.bytes[:0], int(n*fenceSize))[:n*fenceSize]
	if err := rd.Read(r, s, t.bytes, t.at+entrySize*int64(t.entries)); err != nil {
		return err
	}
	t.fences = t.fences[:0]
	for b := t.bytes; len(b) > 0; b = b[fenceSize:] {
		f := fence{
			stamp: int64(binary.LittleEndian.Uint64(b)),
			pos:   binary.LittleEndian.Uint32(b[8:]),
		}
		if k := len(t.fences); (k == 0 && f.pos != 0) || f.pos >= t.samples ||
			(k > 0 && (f.pos <= t.fences[k-1].pos || f.stamp <= t.fences[k-1].stamp)) {
			return rd.corrupt(s, "fence %d at position %d and stamp %d is out of "+
				"order", k, f.pos, f.stamp)
		}
		t.fences = append(t.fences, f)
	}
	t.db, t.id = s.DB, s.ID
	return nil
}

// span returns the indexes of the entries that start in the block of fence k.
func (t *table) span(k int) (lo, hi uint32) {
	first := func(k int) uint32 {
		if k == 0 {
			return 0
		}
		return uint32(((t.at/BlockSize+int64(k))*BlockSize - t.at) / entrySize)
	}
	return first(k), min(first(k+1), t.entries)
}

func (rd *Reader) entryAt(r io.ReaderAt, s Committed, i uint32) (entry, error) {
	b := rd.word[:entrySize]
	if err := rd.Read(r, s, b, rd.tab.at+entrySize*int64(i)); err != nil {
		return entry{}, err
	}
	return entry{
		stamp: int64(binary.LittleEndian.Uint64(b)),
		pos:   binary.LittleEndian.Uint32(b[8:]),
		word:  binary.LittleEndian.Uint32(b[12:]),
	}, nil
}

// entryEnd returns the position where entry i ends: the position of the entry after it,
// or the sample count after the last entry. A fence gives the position of an entry that
// starts a block, so entryEnd reads no other block of the table.
func (rd *Reader) entryEnd(r io.ReaderAt, s Committed, i uint32) (uint32, error) {
	t := &rd.tab
	if i+1 == t.entries {
		return t.samples, nil
	}
	if at := t.at + entrySize*int64(i+1); at%BlockSize == 0 {
		return t.fences[at/BlockSize-t.at/BlockSize].pos, nil
	}
	e, err := rd.entryAt(r, s, i+1)
	return e.pos, err
}

// lastIn returns the index and the entry of the last entry in the block of fence k for
// which after is false. after must be false for the first entry of the block.
func (rd *Reader) lastIn(
	r io.ReaderAt,
	s Committed,
	k int,
	after func(entry) bool,
) (uint32, entry, error) {
	lo, hi := rd.tab.span(k)
	for hi-lo > 1 {
		mid := lo + (hi-lo)/2
		e, err := rd.entryAt(r, s, mid)
		if err != nil {
			return 0, entry{}, err
		}
		if after(e) {
			hi = mid
		} else {
			lo = mid
		}
	}
	e, err := rd.entryAt(r, s, lo)
	return lo, e, err
}

// checkEntry checks that entry e, which ends at end, fits the table.
func (rd *Reader) checkEntry(s Committed, e entry, end uint32) error {
	if end <= e.pos || end > rd.tab.samples {
		return rd.corrupt(
			s,
			"entry at position %d ends at %d of %d samples",
			e.pos,
			end,
			rd.tab.samples,
		)
	}
	if e.word&rawEntry != 0 {
		if int64(e.word&^rawEntry)+int64(end-e.pos) > int64(rd.tab.raw) {
			return rd.corrupt(
				s,
				"raw entry at position %d reads past the raw area",
				e.pos,
			)
		}
	} else if e.word == 0 {
		return rd.corrupt(s, "progression at position %d has no step", e.pos)
	}
	return nil
}

// Stamps reads the stamps from position pos of the version 2 segment s in r into dst,
// each an int64, little-endian. It reads and checks the footer and the fences of s
// first when rd does not hold them, then one block of the table, and the raw stamps of
// the range.
func (rd *Reader) Stamps(r io.ReaderAt, s Committed, dst []byte, pos int64) error {
	if err := rd.loadTable(r, s); err != nil {
		return err
	}
	n := int64(len(dst)) / stampSize
	if len(dst)%stampSize != 0 || pos < 0 || pos+n > int64(rd.tab.samples) {
		return errors.Newf("read of %d bytes at position %d of version 2 segment %d "+
			"of %d samples", len(dst), pos, s.ID, rd.tab.samples)
	}
	if n == 0 {
		return nil
	}
	k := sort.Search(len(rd.tab.fences), func(i int) bool {
		return int64(rd.tab.fences[i].pos) > pos
	}) - 1
	i, e, err := rd.lastIn(r, s, k, func(e entry) bool { return int64(e.pos) > pos })
	if err == nil && int64(e.pos) > pos {
		err = rd.corrupt(
			s,
			"entry at position %d starts its block after %d",
			e.pos,
			pos,
		)
	}
	for err == nil {
		var end uint32
		if end, err = rd.entryEnd(r, s, i); err != nil {
			break
		}
		if err = rd.checkEntry(s, e, end); err != nil {
			break
		}
		m := min(int64(end)-pos, n)
		if e.word&rawEntry != 0 {
			idx := int64(e.word&^rawEntry) + pos - int64(e.pos)
			if err = rd.Read(r, s, dst[:m*stampSize], idx*stampSize); err != nil {
				break
			}
		} else {
			x := e.stamp + (pos-int64(e.pos))*int64(e.word)
			for j := range m {
				binary.LittleEndian.PutUint64(dst[j*stampSize:], uint64(x))
				x += int64(e.word)
			}
		}
		dst, pos, n = dst[m*stampSize:], pos+m, n-m
		if n == 0 {
			return nil
		}
		i++
		e, err = rd.entryAt(r, s, i)
	}
	return err
}

// Search returns where ts falls among the n stamps from position first of the version 2
// segment s in r, as a search of those stamps alone does: the position, from first, of
// the stamp that is ts as lower and upper, or the positions before and after ts. It
// reads and checks the footer and the fences of s first when rd does not hold them,
// then one block of the table, and for a raw entry the block of its stamps that holds
// ts.
func (rd *Reader) Search(
	r io.ReaderAt,
	s Committed,
	ts, first, n int64,
) (lower, upper int64, err error) {
	pos, exact, err := rd.position(r, s, ts)
	if err != nil {
		return 0, 0, err
	}
	at := min(max(pos-first, 0), n)
	if exact && at == pos-first && at < n {
		return at, at, nil
	}
	return at - 1, at, nil
}

// position returns the position of the first stamp of the version 2 segment s in r that
// is at or after ts, or the sample count of s when none is, and whether that stamp is
// ts.
func (rd *Reader) position(r io.ReaderAt, s Committed, ts int64) (int64, bool, error) {
	if err := rd.loadTable(r, s); err != nil {
		return 0, false, err
	}
	t := &rd.tab
	if ts > t.last {
		return int64(t.samples), false, nil
	}
	k := sort.Search(
		len(t.fences),
		func(i int) bool { return t.fences[i].stamp > ts },
	) - 1
	if k < 0 {
		return 0, false, nil
	}
	i, e, err := rd.lastIn(r, s, k, func(e entry) bool { return e.stamp > ts })
	if err != nil {
		return 0, false, err
	}
	end, err := rd.entryEnd(r, s, i)
	if err == nil {
		err = rd.checkEntry(s, e, end)
	}
	if err == nil && e.stamp > ts {
		err = rd.corrupt(s, "entry at stamp %d starts its block after %d", e.stamp, ts)
	}
	if err != nil {
		return 0, false, err
	}
	if e.word&rawEntry == 0 {
		d := ts - e.stamp
		pos, exact := int64(e.pos)+d/int64(e.word), d%int64(e.word) == 0
		if !exact {
			pos++
		}
		if pos >= int64(end) {
			return int64(end), false, nil
		}
		return pos, exact, nil
	}
	return rd.searchRaw(r, s, e, end, ts)
}

// searchRaw is Search inside the raw entry e, which ends at position end.
func (rd *Reader) searchRaw(
	r io.ReaderAt,
	s Committed,
	e entry,
	end uint32,
	ts int64,
) (int64, bool, error) {
	idx, b := int64(e.word&^rawEntry), rd.word[:stampSize]
	lo, hi := int64(0), int64(end-e.pos)
	exact := false
	for lo < hi {
		mid := lo + (hi-lo)/2
		if err := rd.Read(r, s, b, (idx+mid)*stampSize); err != nil {
			return 0, false, err
		}
		if x := int64(binary.LittleEndian.Uint64(b)); x < ts {
			lo = mid + 1
		} else {
			hi, exact = mid, x == ts
		}
	}
	return int64(e.pos) + lo, exact && lo < int64(end-e.pos), nil
}
