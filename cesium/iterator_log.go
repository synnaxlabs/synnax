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

	"github.com/synnaxlabs/cesium/internal/checkpoint"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// logView is what one iterator of a key reads: the visible Channels of the key and of
// its clock, with the parts of their ranges, which the engine loads from one step of
// the state and pins together, so the samples and their stamps come from one state. The
// data cursor loads the view at each seek. The clock cursors that the index opens read
// the view and never load the live state. Each lookup of the index closes its clock
// cursor and reader before it returns, and the calls of an iterator never overlap, so a
// reload never unpins what a clock cursor still reads.
type logView struct {
	e          *engine
	key, clock ChannelKey
	// chs and pinned hold the Channels of the key and the clock of the last load, and
	// what it pins until the next load or the Close of the data cursor. spare holds the
	// segment arrays that the next load fills.
	chs    [2]*state.Channel
	pinned [2]held
	spare  [2][]uint64
	loaded bool
	// failed is true when the last load failed to read the parts of a range.
	failed bool
	// err holds the errors of the loads, a part read or an unlink after an unpin, which
	// Close returns.
	err error
}

func (v *logView) load() {
	chs, pinned, err, unlinkErr := v.e.loadView(v.key, v.clock, v.pinned, v.spare)
	v.spare = [2][]uint64{v.pinned[0].segments[:0], v.pinned[1].segments[:0]}
	v.chs, v.pinned, v.loaded, v.failed = chs, pinned, true, err != nil
	v.err = errors.Join(v.err, err, unlinkErr)
}

// half returns the Channel of the key, or of the clock when clock is true, and the
// parts of its range that the view pins.
func (v *logView) half(clock bool) (*state.Channel, []checkpoint.Part) {
	if clock && v.clock != v.key {
		return v.chs[1], v.pinned[1].parts
	}
	return v.chs[0], v.pinned[0].parts
}

func (v *logView) close() error {
	err := errors.Join(v.err, v.e.unpin(v.pinned[0]), v.e.unpin(v.pinned[1]))
	*v = logView{e: v.e, key: v.key, clock: v.clock}
	return err
}

// logSource opens the cursors of the key of view, or of its clock when clock is true,
// on the pointer log layout.
type logSource struct {
	view  *logView
	clock bool
}

var _ domain.Source = logSource{}

// OpenCursor implements domain.Source.
func (s logSource) OpenCursor(cfg domain.IteratorConfig) domain.Cursor {
	key := s.view.key
	if s.clock {
		key = s.view.clock
	}
	return domain.NewCursor(&logCursor{
		e:      s.view.e,
		key:    key,
		view:   s.view,
		clock:  s.clock,
		bounds: cfg.Bounds,
	})
}

// logRange is one committed pointer of a key, with the segment that holds it. Its time
// range starts at start, which is the end of the pointer before it when the pointer
// goes on with that pointer's stretch in a new segment.
type logRange struct {
	seg   state.Segment
	ptr   state.Pointer
	start telem.TimeStamp
}

func (r logRange) timeRange() telem.TimeRange {
	return telem.TimeRange{Start: r.start, End: r.ptr.Last + 1}
}

// logCursor moves over the pointers of one key in time order. Each seek loads the
// visible pointers of the key, which never change once the state publishes them.
type logCursor struct {
	e      *engine
	key    ChannelKey
	view   *logView
	clock  bool
	bounds telem.TimeRange
	ranges []logRange
	pos    int
	valid  bool
	// cur is the range of the last valid move, which a move that fails keeps, as on the
	// old layout.
	cur logRange
	// err holds the errors of the part record reads, which Close returns. broken is
	// true once a load failed to read a part, which ends the cursor.
	err    error
	broken bool
}

var _ domain.Ranges = (*logCursor)(nil)

// load loads the visible pointers of the key in time order: the live records of the
// parts of its range, through the part cache, then the segments of the state that no
// part holds. The data cursor loads the view again, and a clock cursor reads the clock
// of the view, which it loads only when no seek loaded it yet.
func (c *logCursor) load() {
	c.ranges = c.ranges[:0]
	if !c.clock || !c.view.loaded {
		c.view.load()
	}
	if c.view.failed {
		c.broken = true
		return
	}
	ch, parts := c.view.half(c.clock)
	if ch == nil {
		return
	}
	for _, p := range parts {
		r, err := c.e.cache.Records(p, c.key)
		if err != nil {
			c.err, c.broken = errors.Join(c.err, err), true
			c.ranges = c.ranges[:0]
			return
		}
		c.ranges = slices.Grow(c.ranges, len(r.Segments))
		for i := range r.Segments {
			if seg := &r.Segments[i]; p.Live(r.First+i) &&
				!killed(ch.Parts.Kills, p, seg.ID) {
				c.add(seg)
			}
		}
	}
	c.ranges = slices.Grow(c.ranges, len(ch.Segments))
	for i := range ch.Segments {
		if seg := &ch.Segments[i]; seg.Part == 0 {
			c.add(seg)
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

// add adds the pointers of seg to the ranges of c.
func (c *logCursor) add(seg *state.Segment) {
	for _, ptr := range seg.Pointers {
		c.ranges = append(c.ranges, logRange{seg: *seg, ptr: ptr, start: ptr.First})
	}
}

// killed reports whether a kill in kills marks the record of segment id in p dead.
func killed(kills []state.Kill, p checkpoint.Part, id uint64) bool {
	return slices.ContainsFunc(kills, func(k state.Kill) bool {
		return k.Segment == id && k.Generation >= p.Generation
	})
}

// at moves the cursor to position pos, which is valid when it holds a range that
// overlaps the bounds. A cursor that failed to read a part is never valid.
func (c *logCursor) at(pos int) bool {
	c.valid = !c.broken && pos >= 0 && pos < len(c.ranges) &&
		c.ranges[pos].timeRange().OverlapsWith(c.bounds)
	if c.valid {
		c.pos, c.cur = pos, c.ranges[pos]
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
	return c.at(pos - 1)
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
	return c.at(pos)
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
	c.e.pin(r.seg.ID)
	return &logReader{
		e:      c.e,
		rd:     c.e.reader(),
		seg:    c.e.committed(c.key, r.seg),
		h:      c.e.budget.Open(c.e.fs, segmentName(c.key, r.seg.ID)),
		offset: int64(r.ptr.Offset),
		size:   int64(r.ptr.Length),
	}, nil
}

// Close implements domain.Ranges. The Close of the data cursor unpins the view.
func (c *logCursor) Close() error {
	c.valid = false
	err := c.err
	c.err = nil
	if c.clock {
		return err
	}
	return errors.Join(err, c.view.close())
}

// logReader reads the bytes of one pointer from its segment file, which it pins until
// Close. It checks each block that it reads with a segment reader from the pool of the
// engine.
type logReader struct {
	e      *engine
	rd     *segment.Reader
	seg    segment.Committed
	h      *segment.Handle
	offset int64
	size   int64
}

var _ domain.RangeReader = (*logReader)(nil)

// ReadAt implements domain.RangeReader. A read past the end of the pointer returns the
// bytes before it with io.EOF. A block that fails its check on two reads returns an
// error that wraps place.ErrCorrupt. A read of a version 2 segment decodes its stamps,
// so it must cover whole stamps.
func (r *logReader) ReadAt(p []byte, off int64) (int, error) {
	if off >= r.size {
		return 0, io.EOF
	}
	n := int(min(int64(len(p)), r.size-off))
	var err error
	if r.seg.Version == segment.ProgressionVersion {
		if (r.offset+off)%stampSize != 0 || n%stampSize != 0 {
			return 0, errors.Newf("read of %d bytes at offset %d of a version 2 "+
				"segment holds no whole stamps", n, r.offset+off)
		}
		err = r.rd.Stamps(r.h, r.seg, p[:n], (r.offset+off)/stampSize)
	} else {
		err = r.rd.Read(r.h, r.seg, p[:n], r.offset+off)
	}
	if err != nil {
		return 0, err
	}
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// Search implements domain.RangeReader. A version 2 segment answers from one block of
// its table.
func (r *logReader) Search(stamp telem.TimeStamp) (domain.Approximation[int64], error) {
	if r.seg.Version != segment.ProgressionVersion {
		return domain.SearchStamps(r, telem.Size(r.size), stamp)
	}
	lower, upper, err := r.rd.Search(
		r.h,
		r.seg,
		int64(stamp),
		r.offset/stampSize,
		r.size/stampSize,
	)
	return domain.Between(lower, upper), err
}

// Size implements domain.RangeReader.
func (r *logReader) Size() telem.Size { return telem.Size(r.size) }

// Close implements domain.RangeReader.
func (r *logReader) Close() error {
	r.rd.Reset()
	r.e.readers.Put(r.rd)
	return errors.Join(r.h.Close(), r.e.unpin(held{segments: []uint64{r.seg.ID}}))
}
