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

// logSource opens the cursors of one key on the pointer log layout.
type logSource struct {
	e   *engine
	key ChannelKey
}

var _ domain.Source = logSource{}

// OpenCursor implements domain.Source.
func (s logSource) OpenCursor(cfg domain.IteratorConfig) domain.Cursor {
	return &logCursor{e: s.e, key: s.key, bounds: cfg.Bounds}
}

// logRange is one committed pointer of a key, with the segment that holds it.
type logRange struct {
	seg state.Segment
	ptr state.Pointer
}

func (r logRange) timeRange() telem.TimeRange {
	return telem.TimeRange{
		Start: r.ptr.First,
		End:   r.ptr.Last + 1,
	}
}

// logCursor moves over the pointers of one key in time order. Each seek loads the
// visible pointers of the key, which never change once the state publishes them.
type logCursor struct {
	e      *engine
	key    ChannelKey
	bounds telem.TimeRange
	ranges []logRange
	pos    int
	valid  bool
	// cur is the range of the last valid move, which a move that fails keeps, as on the
	// old layout.
	cur logRange
}

var _ domain.Cursor = (*logCursor)(nil)

// load loads the visible pointers of the key in time order.
func (c *logCursor) load() {
	c.ranges = c.ranges[:0]
	ch := c.e.state.Channel(c.key)
	if ch == nil {
		return
	}
	for _, seg := range ch.Segments {
		for _, ptr := range seg.Pointers {
			c.ranges = append(c.ranges, logRange{seg: seg, ptr: ptr})
		}
	}
	slices.SortFunc(c.ranges, func(a, b logRange) int {
		return cmp.Compare(a.ptr.First, b.ptr.First)
	})
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

// SetBounds implements domain.Cursor.
func (c *logCursor) SetBounds(bounds telem.TimeRange) {
	c.bounds, c.valid = bounds, false
}

// SeekFirst implements domain.Cursor.
func (c *logCursor) SeekFirst(ctx context.Context) bool {
	return c.SeekGE(ctx, c.bounds.Start)
}

// SeekLast implements domain.Cursor.
func (c *logCursor) SeekLast(ctx context.Context) bool {
	return c.SeekLE(ctx, c.bounds.End-1)
}

// SeekLE implements domain.Cursor.
func (c *logCursor) SeekLE(_ context.Context, stamp telem.TimeStamp) bool {
	c.load()
	pos, _ := slices.BinarySearchFunc(
		c.ranges,
		stamp,
		func(r logRange, t telem.TimeStamp) int {
			return cmp.Compare(r.ptr.First, t+1)
		},
	)
	return c.at(pos - 1)
}

// SeekGE implements domain.Cursor.
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

// Next implements domain.Cursor.
func (c *logCursor) Next() bool { return c.valid && c.at(c.pos+1) }

// Prev implements domain.Cursor.
func (c *logCursor) Prev() bool { return c.valid && c.at(c.pos-1) }

// Valid implements domain.Cursor.
func (c *logCursor) Valid() bool { return c.valid }

// TimeRange implements domain.Cursor.
func (c *logCursor) TimeRange() telem.TimeRange { return c.cur.timeRange() }

// Alignment implements domain.Cursor.
func (c *logCursor) Alignment() telem.Alignment { return c.cur.ptr.Alignment }

// Size implements domain.Cursor.
func (c *logCursor) Size() telem.Size { return telem.Size(c.cur.ptr.Length) }

// OpenReader implements domain.Cursor.
func (c *logCursor) OpenReader(context.Context) (domain.RangeReader, error) {
	r := c.cur
	return &logReader{
		cfg:    c.e.segmentConfig(r.seg),
		h:      c.e.budget.Open(c.e.fs, segmentName(c.key, r.seg.ID)),
		offset: int64(r.ptr.Offset),
		size:   int64(r.ptr.Length),
	}, nil
}

// Close implements domain.Cursor.
func (c *logCursor) Close() error {
	c.valid = false
	return nil
}

// logReader reads the bytes of one pointer from its segment file.
type logReader struct {
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
func (r *logReader) Close() error { return r.h.Close() }
