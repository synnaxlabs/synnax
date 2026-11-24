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
	"github.com/synnaxlabs/cesium/internal/place"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/cesium/internal/state"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
)

// logView is what one iterator of a key reads: the visible Channels of the key and of
// its clock, which the engine loads from one step of the state and pins together, so
// the samples and their stamps come from one state. The data cursor loads the view at
// each seek. The clock cursors that the index opens read the view and never load the
// live state. Each lookup of the index closes its clock cursor and reader before it
// returns, and the calls of an iterator never overlap, so a reload never unpins what a
// clock cursor still reads.
type logView struct {
	e          *engine
	key, clock ChannelKey
	ch, clocks *state.Channel
	loaded     bool
	// pinned holds the segments of the last load, which the view pins until the next
	// load or the Close of the data cursor. spare is the array that the next load
	// fills.
	pinned, spare []uint64
	// err holds the error of an unlink after an unpin of a load, which Close returns.
	err error
}

func (v *logView) load() {
	ch, clocks, pinned, err := v.e.loadView(v.key, v.clock, v.pinned, v.spare)
	v.ch, v.clocks, v.loaded = ch, clocks, true
	v.pinned, v.spare = pinned, v.pinned[:0]
	v.err = errors.Join(v.err, err)
}

func (v *logView) close() error {
	err := errors.Join(v.err, v.e.unpin(v.pinned))
	v.ch, v.clocks, v.err, v.loaded = nil, nil, nil, false
	v.pinned, v.spare = v.pinned[:0], v.spare[:0]
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

// load loads the visible pointers of the key in time order. The data cursor loads the
// view again, and a clock cursor reads the clock of the view, which it loads only when
// no seek loaded it yet.
func (c *logCursor) load() {
	c.ranges = c.ranges[:0]
	var ch *state.Channel
	if c.clock {
		if !c.view.loaded {
			c.view.load()
		}
		ch = c.view.clocks
	} else {
		c.view.load()
		ch = c.view.ch
	}
	if ch == nil {
		return
	}
	var last partRecords
	for _, seg := range ch.Segments {
		ptrs, err := c.pointers(seg, &last)
		if err != nil {
			c.err, c.broken = errors.Join(c.err, err), true
			c.ranges = c.ranges[:0]
			return
		}
		for _, ptr := range ptrs {
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

// partRecords are the records of the key of a cursor in one part, and the index of the
// record after the last one that a load took. A load takes segments in rising order of
// identity, as the part holds them.
type partRecords struct {
	part uint64
	next int
	checkpoint.Records
}

// pointers returns the pointers of seg: from the part that its tag names, through the
// part cache, or its own. last holds the records of the last part that a load read,
// which the next segment in that part reuses.
func (c *logCursor) pointers(
	seg state.Segment,
	last *partRecords,
) ([]state.Pointer, error) {
	if seg.Part == 0 {
		return seg.Pointers, nil
	}
	if last.part != seg.Part {
		r, err := c.records(seg)
		if err != nil {
			return nil, err
		}
		*last = partRecords{part: seg.Part, Records: r}
	}
	for last.next < len(last.Segments) && last.Segments[last.next].ID < seg.ID {
		last.next++
	}
	if last.next == len(last.Segments) || last.Segments[last.next].ID != seg.ID {
		return nil, errors.Wrapf(
			place.ErrCorrupt,
			"part %d holds no record of segment %d of key %d",
			seg.Part,
			seg.ID,
			c.key,
		)
	}
	last.next++
	return last.Segments[last.next-1].Pointers, nil
}

// records returns the records of the key of c in the part that the tag of seg names.
func (c *logCursor) records(seg state.Segment) (checkpoint.Records, error) {
	p, ok := c.e.part(seg.Part)
	if !ok {
		return checkpoint.Records{}, errors.Newf(
			"segment %d of key %d names part %d, which no head names",
			seg.ID,
			c.key,
			seg.Part,
		)
	}
	return c.e.cache.Records(p, c.key)
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
		id:     r.seg.ID,
		cfg:    c.e.segmentConfig(r.seg),
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
// Close.
type logReader struct {
	e      *engine
	id     uint64
	cfg    segment.Config
	h      *segment.Handle
	offset int64
	size   int64
}

var _ domain.RangeReader = (*logReader)(nil)

// ReadAt implements domain.RangeReader. A read past the end of the pointer returns the
// bytes before it with io.EOF.
func (r *logReader) ReadAt(p []byte, off int64) (int, error) {
	if off >= r.size {
		return 0, io.EOF
	}
	n := int(min(int64(len(p)), r.size-off))
	if err := r.cfg.ReadAt(r.h, p[:n], r.offset+off); err != nil {
		return 0, err
	}
	if n < len(p) {
		return n, io.EOF
	}
	return n, nil
}

// Size implements domain.RangeReader.
func (r *logReader) Size() telem.Size { return telem.Size(r.size) }

// Close implements domain.RangeReader.
func (r *logReader) Close() error {
	return errors.Join(r.h.Close(), r.e.unpin([]uint64{r.id}))
}
