// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"cmp"
	"context"
	"io"
	"slices"

	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// logView holds the Channels of a clock and of the keys it times, for one read. It
// loads them from one step of the state at the first seek of each step of the read in
// which the state published. A key that a load drops keeps its last Channel for the
// rest of the read, and so does every key once a load drops the clock or cuts its
// stamps, since the keys take their stamps from it.
type logView struct {
	e *engine
	// keys holds the clock and then each key of the read on it, and chs their Channels.
	// next is the array that the next load fills.
	keys      []ChannelKey
	chs, next []*state.Channel
	// frozen is true for each key that keeps its last Channel, which came from the load
	// whose registered count from holds. live counts the other keys, whose Channels
	// came from the load of cur.
	frozen []bool
	from   []uint64
	live   int
	cur    uint64
	loaded bool
	fresh  bool
	// count is the publish count that the last load returned.
	count uint64
	// loads holds the registered count of each load whose Channels a key or a data
	// cursor uses, with its uses. The view unregisters a count after its last use.
	loads []viewLoad
	// readers counts the data cursors of the view. The Close of the last drops every
	// use.
	readers int
	// err holds the error of an unlink after an unregister, which close returns.
	err error
}

// viewLoad is the registered count of a load of a view, and the uses of its Channels.
type viewLoad struct {
	count uint64
	uses  int
}

func newLogView(e *engine, clock ChannelKey) *logView {
	v := &logView{e: e}
	v.add(clock)
	return v
}

// add adds key to the keys of v, and returns its index. It must come before the first
// load.
func (v *logView) add(key ChannelKey) int {
	v.keys = append(v.keys, key)
	v.chs = append(v.chs, nil)
	v.frozen = append(v.frozen, false)
	v.from = append(v.from, 0)
	v.live++
	return len(v.keys) - 1
}

// channel returns the Channel of the key at index i in the load of the step, and the
// registered count of the load that it came from.
func (v *logView) channel(i int) (*state.Channel, uint64) {
	v.load()
	if v.frozen[i] {
		return v.chs[i], v.from[i]
	}
	return v.chs[i], v.cur
}

// load loads the live keys of v again when the step is new and the state published
// since the last load. It registers the count of the load first, as register says.
func (v *logView) load() {
	if v.fresh || (v.loaded && v.live == 0) {
		return
	}
	v.fresh = true
	if v.loaded && v.e.state.Count() == v.count {
		return
	}
	count := v.e.register()
	v.next = slices.Grow(v.next[:0], len(v.keys))[:len(v.keys)]
	v.count = v.e.state.Load(v.keys, v.next)
	if v.loaded {
		v.freeze()
	}
	if v.live == 0 {
		v.err = errors.Join(v.err, v.e.unregister(count))
		return
	}
	for i, ch := range v.next {
		if !v.frozen[i] {
			v.chs[i] = ch
		}
	}
	v.use(count, v.live)
	if v.loaded {
		v.drop(v.cur, v.live)
	}
	v.cur, v.loaded = count, true
}

// freeze freezes each live key that the load in next drops, and every live key when the
// load drops the clock or cuts its stamps. A frozen key keeps its use of the last load.
func (v *logView) freeze() {
	last, next := v.chs[0], v.next[0]
	all := last != nil && (next == nil || next.Cuts != last.Cuts)
	for i, ch := range v.chs {
		if !v.frozen[i] && (all || (ch != nil && v.next[i] == nil)) {
			v.frozen[i], v.from[i] = true, v.cur
			v.live--
		}
	}
}

// use adds n uses of the load of count.
func (v *logView) use(count uint64, n int) {
	for i := range v.loads {
		if v.loads[i].count == count {
			v.loads[i].uses += n
			return
		}
	}
	v.loads = append(v.loads, viewLoad{count: count, uses: n})
}

// drop drops n uses of the load of count, and unregisters the count after its last use.
func (v *logView) drop(count uint64, n int) {
	for i := range v.loads {
		if v.loads[i].count != count {
			continue
		}
		if v.loads[i].uses -= n; v.loads[i].uses == 0 {
			v.loads = slices.Delete(v.loads, i, i+1)
			v.err = errors.Join(v.err, v.e.unregister(count))
		}
		return
	}
}

// stale makes the next seek load v again.
func (v *logView) stale() { v.fresh = false }

// close drops a data cursor of v. The close of the last drops every use of v, and makes
// the next seek load v anew.
func (v *logView) close() error {
	if v.readers--; v.readers > 0 {
		return nil
	}
	if v.loaded && v.live > 0 {
		v.drop(v.cur, v.live)
	}
	for i, frozen := range v.frozen {
		if frozen {
			v.drop(v.from[i], 1)
		}
	}
	err := v.err
	v.err, v.fresh, v.loaded, v.live = nil, false, false, len(v.keys)
	clear(v.chs)
	clear(v.frozen)
	return err
}

// logSource opens the cursors of the key at index slot of view on the pointer log
// layout. Slot 0 is the clock, whose source is that of the index. The calls of a read
// never overlap, so the source gives the cursor and range reader that a call closed to
// the next call.
type logSource struct {
	view *logView
	slot int
	// closed is the cursor that the last Close gave back, or nil.
	closed *logCursor
	// spare is the range reader that the last reader Close gave back, or nil.
	spare *logReader
}

var _ domain.Source = (*logSource)(nil)

// OpenCursor implements domain.Source.
func (s *logSource) OpenCursor(cfg domain.IteratorConfig) domain.Cursor {
	if s.slot != 0 {
		s.view.readers++
	}
	c := s.closed
	s.closed = nil
	if c == nil {
		c = &logCursor{}
	}
	*c = logCursor{
		e:      s.view.e,
		key:    s.view.keys[s.slot],
		src:    s,
		view:   s.view,
		slot:   s.slot,
		bounds: cfg.Bounds,
		ranges: c.ranges[:0],
	}
	return domain.NewCursor(c)
}

// logRange is one committed pointer of a key, with the record of the segment that holds
// it in a published Channel, which never changes. Its time range starts at start, which
// is the end of the pointer before it when the pointer goes on with that pointer's
// stretch in a new segment.
type logRange struct {
	seg   *state.Segment
	ptr   state.Pointer
	start telem.TimeStamp
}

// timeRange returns the time range of r, or the zero range when r holds no pointer, as
// for a cursor that never held a range.
func (r logRange) timeRange() telem.TimeRange {
	if r.ptr.Samples == 0 {
		return telem.TimeRange{}
	}
	return telem.TimeRange{Start: r.start, End: r.ptr.Last + 1}
}

// logCursor moves over the pointers of one key in time order. Each seek loads the
// visible pointers of the key, which never change once the state publishes them.
type logCursor struct {
	e    *engine
	key  ChannelKey
	view *logView
	slot int
	src  *logSource
	// from is the registered count of the load that the ranges of a data cursor came
	// from, once sought is true. The cursor uses that load until its next seek or
	// Close, since another cursor can load the view again before then.
	from   uint64
	sought bool
	bounds telem.TimeRange
	ranges []logRange
	pos    int
	valid  bool
	// cur is the range of the last valid move, which a failed Next or Prev keeps, as on
	// the old layout. A seek loads again, so one that finds no range leaves none.
	cur logRange
}

var _ domain.Ranges = (*logCursor)(nil)

// load loads the visible pointers of the key in time order, from the view. A data
// cursor moves its use to the load that they came from.
func (c *logCursor) load() {
	c.ranges = c.ranges[:0]
	ch, from := c.view.channel(c.slot)
	if c.slot != 0 && (!c.sought || from != c.from) {
		c.view.use(from, 1)
		if c.sought {
			c.view.drop(c.from, 1)
		}
		c.from, c.sought = from, true
	}
	if ch == nil {
		return
	}
	for i := range ch.Segments {
		seg := &ch.Segments[i]
		for _, ptr := range seg.Pointers {
			c.ranges = append(c.ranges, logRange{seg: seg, ptr: ptr, start: ptr.First})
		}
	}
	slices.SortFunc(c.ranges, func(a, b logRange) int {
		return cmp.Compare(a.ptr.First, b.ptr.First)
	})
	for i := 1; i < len(c.ranges); i++ {
		prev, r := c.ranges[i-1].ptr, &c.ranges[i]
		if prev.Alignment.AddSamples(prev.Samples) == r.ptr.Alignment {
			r.start = prev.Last + 1
		}
	}
}

// at moves the cursor to position pos, which is valid when it holds a range that
// overlaps the bounds.
func (c *logCursor) at(pos int) bool {
	c.valid = pos >= 0 && pos < len(c.ranges) &&
		c.ranges[pos].timeRange().OverlapsWith(c.bounds)
	if c.valid {
		c.pos, c.cur = pos, c.ranges[pos]
	}
	return c.valid
}

// seek moves the cursor to position pos of the load that a seek just made. A seek that
// finds no range leaves the cursor at pos with no range, so it never offers a range
// outside its load.
func (c *logCursor) seek(pos int) bool {
	if !c.at(pos) {
		c.pos, c.cur = pos, logRange{}
	}
	return c.valid
}

// SetBounds implements domain.Ranges.
func (c *logCursor) SetBounds(bounds telem.TimeRange) {
	c.bounds, c.valid = bounds, false
}

// SeekFirst implements domain.Ranges.
func (c *logCursor) SeekFirst(ctx context.Context) bool {
	return c.SeekGE(ctx, c.bounds.Start)
}

// SeekLast implements domain.Ranges.
func (c *logCursor) SeekLast(ctx context.Context) bool {
	return c.SeekLE(ctx, c.bounds.End-1)
}

// SeekLE implements domain.Ranges.
func (c *logCursor) SeekLE(_ context.Context, stamp telem.TimeStamp) bool {
	c.load()
	pos, _ := slices.BinarySearchFunc(
		c.ranges,
		stamp,
		func(r logRange, t telem.TimeStamp) int {
			if r.start <= t {
				return -1
			}
			return 1
		},
	)
	return c.seek(pos - 1)
}

// SeekGE implements domain.Ranges.
func (c *logCursor) SeekGE(_ context.Context, stamp telem.TimeStamp) bool {
	c.load()
	pos, _ := slices.BinarySearchFunc(
		c.ranges,
		stamp,
		func(r logRange, t telem.TimeStamp) int {
			return cmp.Compare(r.ptr.Last, t)
		},
	)
	return c.seek(pos)
}

// Next implements domain.Ranges.
func (c *logCursor) Next() bool { return c.valid && c.at(c.pos+1) }

// Prev implements domain.Ranges.
func (c *logCursor) Prev() bool { return c.valid && c.at(c.pos-1) }

// Valid implements domain.Ranges.
func (c *logCursor) Valid() bool { return c.valid }

// TimeRange implements domain.Ranges.
func (c *logCursor) TimeRange() telem.TimeRange { return c.cur.timeRange() }

// Alignment implements domain.Ranges.
func (c *logCursor) Alignment() telem.Alignment { return c.cur.ptr.Alignment }

// Size implements domain.Ranges.
func (c *logCursor) Size() telem.Size { return telem.Size(c.cur.ptr.Length) }

// OpenReader implements domain.Ranges.
func (c *logCursor) OpenReader(context.Context) (domain.RangeReader, error) {
	r := c.cur
	hs := c.e.readHandles(c.key, r.seg.ID)
	h := hs.take(c.e.budget, c.e.fs)
	file, err := h.Hold()
	if err != nil {
		hs.give(h)
		return nil, err
	}
	var rd *segment.Reader
	if c.e.readsChecked {
		rd = c.e.reader()
	}
	lr := c.src.spare
	c.src.spare = nil
	if lr == nil {
		lr = &logReader{}
	}
	*lr = logReader{
		src:    c.src,
		rd:     rd,
		seg:    c.e.committed(c.key, *r.seg),
		hs:     hs,
		h:      h,
		file:   file,
		offset: r.ptr.Offset,
		size:   r.ptr.Length,
	}
	return lr, nil
}

// Close implements domain.Ranges. The Close of a data cursor drops its use of the view,
// and every use of the view after the last data cursor of the view. It gives c back to
// its source, so c must not be used after it.
func (c *logCursor) Close() error {
	c.valid = false
	c.src.closed = c
	if c.slot == 0 {
		return nil
	}
	if c.sought {
		c.view.drop(c.from, 1)
		c.sought = false
	}
	return c.view.close()
}

// logReader reads the bytes of one pointer from its segment file, through a handle that
// no other read holds, which it holds open until Close. It registers nothing: the use
// of its cursor or its view keeps the segment for the call that opens and closes it.
// With read checks on, it checks each block that it reads.
type logReader struct {
	src    *logSource
	rd     *segment.Reader
	seg    segment.Committed
	hs     *segmentHandles
	h      *segment.Handle
	file   io.ReaderAt
	offset uint32
	size   uint32
}

var _ domain.RangeReader = (*logReader)(nil)

// ReadAt implements domain.RangeReader. A read past the end of the pointer returns the
// bytes before it with io.EOF. A block that fails its check on two reads returns an
// error that wraps place.ErrCorrupt.
func (r *logReader) ReadAt(p []byte, off int64) (int, error) {
	if off >= int64(r.size) {
		return 0, io.EOF
	}
	n := int(min(int64(len(p)), int64(r.size)-off))
	at := int64(r.offset) + off
	var err error
	if r.rd == nil {
		err = r.seg.ReadStored(r.file, p[:n], at)
	} else {
		err = r.rd.Read(r.file, r.seg, p[:n], at)
	}
	if err != nil {
		return 0, err
	}
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// Size implements domain.RangeReader.
func (r *logReader) Size() telem.Size { return telem.Size(r.size) }

// Close implements domain.RangeReader. It gives r back to its source, so r must not be
// used after it.
func (r *logReader) Close() error {
	if r.rd != nil {
		r.src.view.e.readers.Put(r.rd)
	}
	r.h.Release()
	r.hs.give(r.h)
	r.src.spare = r
	return nil
}
