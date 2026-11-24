// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package domain

import (
	"context"

	"github.com/synnaxlabs/x/telem"
)

// Ranges moves over the committed ranges of one channel in time order. The pointer log
// layout implements it, and a Cursor holds it.
type Ranges interface {
	// SetBounds sets the bounds of the cursor and invalidates it.
	SetBounds(bounds telem.TimeRange)
	// SeekFirst moves to the first range in the bounds, and returns false when none is.
	SeekFirst(ctx context.Context) bool
	// SeekLast moves to the last range in the bounds, and returns false when none is.
	SeekLast(ctx context.Context) bool
	// SeekLE moves to the range that holds stamp, or else to the last range that ends
	// before it.
	SeekLE(ctx context.Context, stamp telem.TimeStamp) bool
	// SeekGE moves to the range that holds stamp, or else to the first range that
	// starts after it.
	SeekGE(ctx context.Context, stamp telem.TimeStamp) bool
	// Next moves to the next range, and returns false past the last.
	Next() bool
	// Prev moves to the previous range, and returns false before the first.
	Prev() bool
	// Valid returns true when the cursor is at a range in its bounds.
	Valid() bool
	// TimeRange returns the time range of the current range.
	TimeRange() telem.TimeRange
	// Alignment returns the alignment of the first sample of the current range.
	Alignment() telem.Alignment
	// Size returns the bytes of the current range.
	Size() telem.Size
	// OpenReader returns a reader of the bytes of the current range, which the caller
	// closes.
	OpenReader(ctx context.Context) (RangeReader, error)
	// Close closes the cursor.
	Close() error
}

// RangeReader reads the bytes of one committed range.
type RangeReader interface {
	// ReadAt reads len(p) bytes at off of the range.
	ReadAt(p []byte, off int64) (int, error)
	// Search returns the position of stamp in a range of rising index stamps: exactly
	// the first position that holds stamp, or the positions before and after it.
	Search(stamp telem.TimeStamp) (Approximation[int64], error)
	// Size returns the bytes of the range.
	Size() telem.Size
	// Close closes the reader.
	Close() error
}

// Source opens the cursors of one channel.
type Source interface {
	OpenCursor(cfg IteratorConfig) Cursor
}

// Cursor is the cursor of one channel on either layout: an Iterator of the old layout,
// held by value so that a cursor of a read stays off the heap, or the Ranges of another
// layout. A Cursor is not safe to copy once it is in use.
type Cursor struct {
	it     Iterator
	ranges Ranges
}

// NewCursor returns the Cursor of r.
func NewCursor(r Ranges) Cursor { return Cursor{ranges: r} }

var (
	_ Ranges      = (*Iterator)(nil)
	_ RangeReader = (*reader)(nil)
	_ Source      = (*DB)(nil)
)

// OpenCursor implements Source.
func (db *DB) OpenCursor(cfg IteratorConfig) Cursor {
	db.resourceCount.Add(1)
	c := Cursor{it: Iterator{db: db}}
	c.it.SetBounds(cfg.Bounds)
	return c
}

// SetBounds sets the bounds of the cursor and invalidates it.
func (c *Cursor) SetBounds(bounds telem.TimeRange) {
	if c.ranges == nil {
		c.it.SetBounds(bounds)
		return
	}
	c.ranges.SetBounds(bounds)
}

// SeekFirst moves to the first range in the bounds.
func (c *Cursor) SeekFirst(ctx context.Context) bool {
	if c.ranges == nil {
		return c.it.SeekFirst(ctx)
	}
	return c.ranges.SeekFirst(ctx)
}

// SeekLast moves to the last range in the bounds.
func (c *Cursor) SeekLast(ctx context.Context) bool {
	if c.ranges == nil {
		return c.it.SeekLast(ctx)
	}
	return c.ranges.SeekLast(ctx)
}

// SeekLE moves to the range that holds stamp, or else to the last range before it.
func (c *Cursor) SeekLE(ctx context.Context, stamp telem.TimeStamp) bool {
	if c.ranges == nil {
		return c.it.SeekLE(ctx, stamp)
	}
	return c.ranges.SeekLE(ctx, stamp)
}

// SeekGE moves to the range that holds stamp, or else to the first range after it.
func (c *Cursor) SeekGE(ctx context.Context, stamp telem.TimeStamp) bool {
	if c.ranges == nil {
		return c.it.SeekGE(ctx, stamp)
	}
	return c.ranges.SeekGE(ctx, stamp)
}

// Next moves to the next range.
func (c *Cursor) Next() bool {
	if c.ranges == nil {
		return c.it.Next()
	}
	return c.ranges.Next()
}

// Prev moves to the previous range.
func (c *Cursor) Prev() bool {
	if c.ranges == nil {
		return c.it.Prev()
	}
	return c.ranges.Prev()
}

// Valid returns true when the cursor is at a range in its bounds.
func (c *Cursor) Valid() bool {
	if c.ranges == nil {
		return c.it.Valid()
	}
	return c.ranges.Valid()
}

// TimeRange returns the time range of the current range.
func (c *Cursor) TimeRange() telem.TimeRange {
	if c.ranges == nil {
		return c.it.TimeRange()
	}
	return c.ranges.TimeRange()
}

// Alignment returns the alignment of the first sample of the current range.
func (c *Cursor) Alignment() telem.Alignment {
	if c.ranges == nil {
		return c.it.Alignment()
	}
	return c.ranges.Alignment()
}

// Size returns the bytes of the current range.
func (c *Cursor) Size() telem.Size {
	if c.ranges == nil {
		return c.it.Size()
	}
	return c.ranges.Size()
}

// OpenReader returns a reader of the bytes of the current range.
func (c *Cursor) OpenReader(ctx context.Context) (RangeReader, error) {
	if c.ranges == nil {
		return c.it.OpenReader(ctx)
	}
	return c.ranges.OpenReader(ctx)
}

// Close closes the cursor.
func (c *Cursor) Close() error {
	if c.ranges == nil {
		return c.it.Close()
	}
	return c.ranges.Close()
}

// Alignment implements Ranges. On the old layout, the position of a domain in the index
// is its alignment number.
func (i *Iterator) Alignment() telem.Alignment {
	return telem.NewAlignment(uint32(i.position), 0)
}
