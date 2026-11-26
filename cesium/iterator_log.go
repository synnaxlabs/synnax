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
	"github.com/synnaxlabs/x/telem"
)

// logSource opens the cursors of one key on the pointer log layout for one read. The
// calls of a read never overlap, so the source gives the cursor that a call closed to
// the next call.
type logSource struct {
	e   *engine
	key ChannelKey
	// closed is the cursor that the last Close gave back, or nil.
	closed *logCursor
}

var _ domain.Source = (*logSource)(nil)

// OpenCursor implements domain.Source.
func (s *logSource) OpenCursor(cfg domain.IteratorConfig) domain.Cursor {
	c := s.closed
	s.closed = nil
	if c == nil {
		c = &logCursor{}
	}
	*c = logCursor{e: s.e, key: s.key, src: s, bounds: cfg.Bounds, ranges: c.ranges[:0]}
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
	e      *engine
	key    ChannelKey
	src    *logSource
	bounds telem.TimeRange
	ranges []logRange
	pos    int
	valid  bool
	// cur is the range of the last valid move, which a move that fails keeps, as on the
	// old layout.
	cur logRange
}

var _ domain.Ranges = (*logCursor)(nil)

// load loads the visible pointers of the key in time order.
func (c *logCursor) load() {
	c.ranges = c.ranges[:0]
	ch := c.e.state.Channel(c.key)
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
	cfg := c.e.segmentConfig(*r.seg)
	cfg.Key = c.key
	h := c.e.handle(c.key, r.seg.ID)
	file, err := h.Hold()
	if err != nil {
		return nil, err
	}
	var rd *segment.Reader
	if c.e.readsChecked {
		rd = c.e.reader()
	}
	return &logReader{
		e:  c.e,
		rd: rd,
		seg: segment.Committed{
			Config:  cfg,
			Length:  int64(r.seg.Length),
			TailCRC: r.seg.TailCRC,
		},
		h:      h,
		file:   file,
		offset: int64(r.ptr.Offset),
		size:   int64(r.ptr.Length),
	}, nil
}

// Close implements domain.Ranges. It gives c back to its source, so c must not be used
// after it.
func (c *logCursor) Close() error {
	c.valid = false
	c.src.closed = c
	return nil
}

// logReader reads the bytes of one pointer from its segment file, which it holds open
// until Close. When data reads are checked, it checks each block that it reads with a
// segment reader from the pool of the engine. Otherwise it reads the bytes as stored.
type logReader struct {
	e      *engine
	rd     *segment.Reader
	seg    segment.Committed
	h      *segment.Handle
	file   io.ReaderAt
	offset int64
	size   int64
}

var _ domain.RangeReader = (*logReader)(nil)

// ReadAt implements domain.RangeReader. A read past the end of the pointer returns the
// bytes before it with io.EOF. A block that fails its check on two reads returns an
// error that wraps place.ErrCorrupt.
func (r *logReader) ReadAt(p []byte, off int64) (int, error) {
	if off >= r.size {
		return 0, io.EOF
	}
	n := int(min(int64(len(p)), r.size-off))
	var err error
	if r.rd == nil {
		err = r.seg.ReadStored(r.file, p[:n], r.offset+off)
	} else {
		err = r.rd.Read(r.file, r.seg, p[:n], r.offset+off)
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

// Close implements domain.RangeReader.
func (r *logReader) Close() error {
	if r.rd != nil {
		r.e.readers.Put(r.rd)
	}
	r.h.Release()
	return nil
}
