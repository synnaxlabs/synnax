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
	"cmp"
	"encoding/binary"
	"io"
	"os"
	"slices"
	"sort"
	"sync"

	"github.com/synnaxlabs/cesium/internal/lru"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	xfs "github.com/synnaxlabs/x/io/fs"
	"github.com/synnaxlabs/x/telem"
)

// defaultCacheSize is the budget of a part cache that a Config leaves out.
const defaultCacheSize = 64 << 20

// Records are the segment records of one key in a part.
type Records struct {
	// First is the index in the part of the first record, from which a dead bitmap
	// counts.
	First int
	// Segments are the records, in rising order of identity.
	Segments []state.Segment
	// order holds the indexes of the records in rising order of their lowest stamp, and
	// reach the highest stamp of the records up to each place in order.
	order []int
	reach []telem.TimeStamp
}

// Reaching appends to buf, in rising order, the index of each record whose stamps may
// reach [first, last], and returns buf. A record it leaves out has no pointer in the
// range.
func (r Records) Reaching(first, last telem.TimeStamp, buf []int) []int {
	n := len(buf)
	k := sort.Search(len(r.order), func(i int) bool {
		return lowest(r.Segments[r.order[i]]) > last
	})
	for i := k - 1; i >= 0 && r.reach[i] >= first; i-- {
		buf = append(buf, r.order[i])
	}
	slices.Sort(buf[n:])
	return buf
}

// indexed returns r with its order and reach set.
func (r Records) indexed() Records {
	r.order = make([]int, 0, len(r.Segments))
	for i, seg := range r.Segments {
		if len(seg.Pointers) > 0 {
			r.order = append(r.order, i)
		}
	}
	slices.SortFunc(r.order, func(a, b int) int {
		return cmp.Compare(lowest(r.Segments[a]), lowest(r.Segments[b]))
	})
	r.reach = make([]telem.TimeStamp, len(r.order))
	for i, j := range r.order {
		r.reach[i] = highest(r.Segments[j])
		if i > 0 {
			r.reach[i] = max(r.reach[i], r.reach[i-1])
		}
	}
	return r
}

// lowest returns the lowest first stamp of the pointers of seg, which has one or more.
func lowest(seg state.Segment) telem.TimeStamp {
	lo := seg.Pointers[0].First
	for _, p := range seg.Pointers[1:] {
		lo = min(lo, p.First)
	}
	return lo
}

// highest returns the highest last stamp of the pointers of seg, which has one or more.
func highest(seg state.Segment) telem.TimeStamp {
	hi := seg.Pointers[0].Last
	for _, p := range seg.Pointers[1:] {
		hi = max(hi, p.Last)
	}
	return hi
}

// Cache holds what readers load of parts, up to a byte budget: the layout of each part
// and the records of each key in it. It drops the least recently used first. A Cache is
// safe for concurrent use.
type Cache struct {
	fs xfs.FS
	db uint64
	mu struct {
		sync.Mutex
		entries *lru.Cache[cacheKey, *cacheEntry]
	}
}

// cacheKey names the records of key in part, or the layout of part when layout is true.
type cacheKey struct {
	part   uint64
	key    uint32
	layout bool
}

type cacheEntry struct {
	layout  *partLayout
	records Records
}

// partLayout is what a read of the records of a part needs: its header, its sections,
// the checksum of each chunk, and its channel records.
type partLayout struct {
	h      Header
	secs   [sectionCount + 1]section
	sums   []uint32
	keys   []uint32
	firsts []uint32
	// clocks are the clock keys that data segments of the part name, in rising order.
	clocks []uint32
}

// Timed is a segment record of a part that names a clock other than its key.
type Timed struct {
	Key uint32
	// Index is the index of the record in the part, from which a dead bitmap counts.
	Index   int
	Segment state.Segment
}

// Timed returns the segment records of part p, of keys other than clock, that name
// clock as their clock. It reads no record when p names no data segment of clock.
func (c *Cache) Timed(p Part, clock uint32) ([]Timed, error) {
	l, err := c.layout(p)
	if err != nil {
		return nil, err
	}
	if _, ok := slices.BinarySearch(l.clocks, clock); !ok {
		return nil, nil
	}
	var timed []Timed
	for _, key := range l.keys {
		if key == clock {
			continue
		}
		r, err := c.Records(p, key)
		if err != nil {
			return nil, err
		}
		for i, seg := range r.Segments {
			if seg.Clock == clock {
				timed = append(timed, Timed{Key: key, Index: r.First + i, Segment: seg})
			}
		}
	}
	return timed, nil
}

// NewCache returns a cache of the parts of database db in the root fs that holds at
// most budget bytes.
func NewCache(fs xfs.FS, db uint64, budget int) *Cache {
	c := &Cache{fs: fs, db: db}
	c.mu.entries = lru.New[cacheKey, *cacheEntry](budget)
	return c
}

// Records returns the segment records of key in the part p that a head names, which are
// empty when the part holds no record of key. It reads only the chunks that hold them,
// and a chunk that fails its check is read a second time. It returns an error that
// wraps place.ErrCorrupt when a chunk fails twice, or when the part is not the one that
// p names.
func (c *Cache) Records(p Part, key uint32) (Records, error) {
	k := cacheKey{part: p.ID, key: key}
	if e, ok := c.get(k); ok {
		return e.records, nil
	}
	l, err := c.layout(p)
	if err != nil {
		return Records{}, err
	}
	r, err := c.loadRecords(p, l, key)
	if err != nil {
		return Records{}, err
	}
	size := 64
	for _, seg := range r.Segments {
		size += 80 + 48*len(seg.Pointers)
	}
	c.put(k, &cacheEntry{records: r}, size)
	return r, nil
}

// Cached returns the records of key in the part p when the cache holds them, with no
// read of the part.
func (c *Cache) Cached(p Part, key uint32) (Records, bool) {
	e, ok := c.get(cacheKey{part: p.ID, key: key})
	if !ok {
		return Records{}, false
	}
	return e.records, true
}

func (c *Cache) get(k cacheKey) (*cacheEntry, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.mu.entries.Get(k)
}

// put adds e of size bytes under k, unless another load added k first, and drops the
// least recently used entries past the budget.
func (c *Cache) put(k cacheKey, e *cacheEntry, size int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.mu.entries.Put(k, e, size) {
		return
	}
	for {
		if _, ok := c.mu.entries.Evict(nil); !ok {
			return
		}
	}
}

// layout returns the layout of p.
func (c *Cache) layout(p Part) (*partLayout, error) {
	k := cacheKey{part: p.ID, layout: true}
	if e, ok := c.get(k); ok {
		return e.layout, nil
	}
	l, err := c.loadLayout(p)
	if err != nil {
		return nil, err
	}
	c.put(k, &cacheEntry{layout: l}, 256+4*len(l.sums)+8*len(l.keys))
	return l, nil
}

func (c *Cache) loadLayout(p Part) (l *partLayout, err error) {
	f, err := c.fs.Open(partName(p.ID), os.O_RDONLY)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	info, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := uint64(info.Size())
	sector := make([]byte, min(size, SectorSize))
	l = &partLayout{}
	if err = twice(f, sector, 0, func() error {
		l.h, err = partFormat.decodeSectorOf(sector, size)
		if errors.Is(err, ErrIncomplete) {
			err = partFormat.corrupt("%s", err.Error())
		}
		return err
	}); err != nil {
		return nil, err
	}
	if l.h.DB != c.db || l.h.Generation != p.ID || l.h.Sequence != p.Generation ||
		l.h.Length != p.Length || l.h.CRC != p.CRC {
		return nil, partFormat.corrupt("part %d is not the part that its head names",
			p.ID)
	}
	if err = c.loadSums(f, l, sector); err != nil {
		return nil, err
	}
	table, err := c.read(f, l, l.h.table, l.h.chunks)
	if err != nil {
		return nil, err
	}
	if l.secs, err = partFormat.sectionsOf(table, l.h); err != nil {
		return nil, err
	}
	if err = c.loadChannels(f, l); err != nil {
		return nil, err
	}
	return l, c.loadClocks(f, l)
}

// loadClocks reads the clock records of l.
func (c *Cache) loadClocks(f xfs.File, l *partLayout) error {
	cs := l.secs[sectionClocks]
	b, err := c.read(f, l, cs.off, cs.off+cs.count*4)
	if err != nil {
		return err
	}
	l.clocks = make([]uint32, cs.count)
	for i := range l.clocks {
		l.clocks[i] = binary.LittleEndian.Uint32(b[4*i:])
	}
	return nil
}

// loadSums reads and checks the chunk table of l, whose header sector is sector.
func (c *Cache) loadSums(f xfs.File, l *partLayout, sector []byte) error {
	b := make([]byte, l.h.Length-l.h.chunks)
	want := binary.LittleEndian.Uint32(sector[96:])
	if err := twice(f, b, l.h.chunks, func() error {
		if partFormat.place(c.db, l.h.Generation, l.h.chunks).Sum(b) != want {
			return partFormat.corrupt("chunk table fails its check")
		}
		return nil
	}); err != nil {
		return err
	}
	l.sums = make([]uint32, len(b)/4)
	for i := range l.sums {
		l.sums[i] = binary.LittleEndian.Uint32(b[4*i:])
	}
	return nil
}

// loadChannels reads the channel records of l.
func (c *Cache) loadChannels(f xfs.File, l *partLayout) error {
	cs, ss := l.secs[sectionChans], l.secs[sectionSegs]
	b, err := c.read(f, l, cs.off, cs.off+cs.count*8)
	if err != nil {
		return err
	}
	l.keys, l.firsts = make([]uint32, cs.count), make([]uint32, cs.count)
	for i := range l.keys {
		l.keys[i] = binary.LittleEndian.Uint32(b[8*i:])
		l.firsts[i] = binary.LittleEndian.Uint32(b[8*i+4:])
		bad := uint64(l.firsts[i]) >= ss.count || i == 0 && l.firsts[i] != 0 ||
			i > 0 && (l.keys[i] <= l.keys[i-1] || l.firsts[i] <= l.firsts[i-1])
		if bad {
			return partFormat.corrupt("channel record %d of part %d is out of order",
				i, l.h.Generation)
		}
	}
	return nil
}

func (c *Cache) loadRecords(p Part, l *partLayout, key uint32) (r Records, err error) {
	i, ok := slices.BinarySearch(l.keys, key)
	if !ok {
		return Records{}, nil
	}
	ss, ps := l.secs[sectionSegs], l.secs[sectionPtrs]
	first, end := uint64(l.firsts[i]), ss.count
	if i+1 < len(l.keys) {
		end = uint64(l.firsts[i+1])
	}
	f, err := c.fs.Open(partName(p.ID), os.O_RDONLY)
	if err != nil {
		return Records{}, err
	}
	defer func() { err = errors.Join(err, f.Close()) }()
	// The record after the last one of key gives the end of its pointer records.
	next := min(end+1, ss.count)
	sb, err := c.read(f, l, ss.off+first*32, ss.off+next*32)
	if err != nil {
		return Records{}, err
	}
	r = Records{First: int(first), Segments: make([]state.Segment, end-first)}
	starts := make([]uint64, next-first+1)
	starts[len(starts)-1] = ps.count
	for j := range next - first {
		seg, start, err := partFormat.decodeSegment(sb[j*32:])
		if err != nil {
			return Records{}, err
		}
		if j < end-first {
			r.Segments[j] = seg
		}
		starts[j] = start
	}
	if !slices.IsSorted(starts) || starts[len(starts)-1] > ps.count {
		return Records{}, partFormat.corrupt("pointer records of key %d in part %d "+
			"are out of order", key, p.ID)
	}
	lo, hi := starts[0], starts[end-first]
	pb, err := c.read(f, l, ps.off+lo*36, ps.off+hi*36)
	if err != nil {
		return Records{}, err
	}
	ptrs := make([]state.Pointer, hi-lo)
	for j := range ptrs {
		ptrs[j] = decodePointer(pb[j*36:])
	}
	for j := range r.Segments {
		a, b := starts[j]-lo, starts[j+1]-lo
		if b > a {
			r.Segments[j].Pointers = ptrs[a:b:b]
		}
	}
	return r.indexed(), nil
}

// read returns bytes lo to hi of the body of the part of l, from the whole chunks that
// hold them, each checked, and each that fails read a second time.
func (c *Cache) read(f xfs.File, l *partLayout, lo, hi uint64) ([]byte, error) {
	if lo >= hi {
		return nil, nil
	}
	if lo < BodyStart || hi > l.h.chunks {
		return nil, partFormat.corrupt("records at %d to %d are outside the body of "+
			"part %d", lo, hi, l.h.Generation)
	}
	first, last := (lo-BodyStart)/ChunkSize, (hi-1-BodyStart)/ChunkSize
	start, _ := chunkRange(l.h, first)
	_, stop := chunkRange(l.h, last)
	b := make([]byte, stop-start)
	if err := readAt(f, b, start); err != nil {
		return nil, err
	}
	for i := first; i <= last; i++ {
		clo, chi := chunkRange(l.h, i)
		chunk := b[clo-start : chi-start]
		check := func() error {
			if partFormat.place(c.db, l.h.Generation, clo).Sum(chunk) != l.sums[i] {
				return partFormat.corrupt("chunk at %d of part %d fails its check",
					clo, l.h.Generation)
			}
			return nil
		}
		if err := again(check(), func() error {
			if err := readAt(f, chunk, clo); err != nil {
				return err
			}
			return check()
		}); err != nil {
			return nil, err
		}
	}
	return b[lo-start : hi-start], nil
}

// twice reads b at off and runs check, and reads and checks once more when the check
// fails.
func twice(f xfs.File, b []byte, off uint64, check func() error) error {
	if err := readAt(f, b, off); err != nil {
		return err
	}
	return again(check(), func() error {
		if err := readAt(f, b, off); err != nil {
			return err
		}
		return check()
	})
}

func readAt(f xfs.File, b []byte, off uint64) error {
	_, err := f.ReadAt(b, int64(off))
	if errors.Is(err, io.EOF) {
		err = io.ErrUnexpectedEOF
	}
	return err
}
