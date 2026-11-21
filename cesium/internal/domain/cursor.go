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

// Cursor moves over the committed ranges of one channel in time order. An Iterator is
// the Cursor of the old layout, and the pointer log layout has its own.
type Cursor interface {
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
	// Size returns the bytes of the range.
	Size() telem.Size
	// Close closes the reader.
	Close() error
}

// Source opens the cursors of one channel.
type Source interface {
	OpenCursor(cfg IteratorConfig) Cursor
}

var (
	_ Cursor      = (*Iterator)(nil)
	_ RangeReader = (*Reader)(nil)
	_ Source      = (*DB)(nil)
)

// OpenCursor implements Source.
func (db *DB) OpenCursor(cfg IteratorConfig) Cursor { return db.OpenIterator(cfg) }

// Alignment implements Cursor. On the old layout, the position of a domain in the index
// is its alignment number.
func (i *Iterator) Alignment() telem.Alignment {
	return telem.NewAlignment(uint32(i.position), 0)
}
