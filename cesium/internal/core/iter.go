package core

import (
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
)

const (
	AutoTimeSpan = telem.TimeSpan(-1)
	AutoPosSpan  = position.Span(-1)
)

// BaseIterator is the base iterator all iterators must implement.
type BaseIterator interface {
	// Close closes the iterator, frees all resources, and returns any accumulated
	// error.
	Close() error
	// Error returns the accumulated error.
	Error() error
	// Valid returns true if the iterator is pointing to a valid entry.
	Valid() bool
	// Value represents the segment metadata values occupying the current iterator
	// view. The values are guaranteed to be within the iterator's view, sorted in ascending
	// order, non-overlapping. It's important to not that these values are not guaranteed
	// to be contiguous i.e. there may be gaps between them.
	Value() []SegmentMD
}

// PositionIterator iterates over segment metadata in position space.
type PositionIterator interface {
	// SeekFirst moves the iterator to the first position in its bounds. Returns false
	// if no such value exists or the iterator has accumulated an error.
	SeekFirst() bool
	// SeekLast moves the iterator to the last position in its bounds. Returns false
	// if no such value exists or the iterator has accumulated an error.
	SeekLast() bool
	// SeekLE moves the iterator to the last position less than or equal to the given
	// position. Returns false if no such value exists or the iterator has accumulated
	// an error.
	SeekLE(pos position.Position) bool
	// SeekGE moves the iterator to the first position greater than or equal to the
	// given position. Returns false if no such value exists or the iterator has accumulated
	// an error.
	SeekGE(pos position.Position) bool
	// Next moves the iterator across the provided span, accumulating any values it
	// encounters. Returns false if no values were accumulated or the iterator has
	// encountered an error.
	Next(span position.Span) bool
	// Prev moves the iterator across the provided span, accumulating any entries it
	// encounters. Returns false if no values were accumulated or the iterator has
	// encountered an error.
	Prev(span position.Span) bool
	// SetBounds sets the bounds of the iterator.
	SetBounds(rng position.Range)
	// Bounds returns the bounds of the iterator.
	Bounds() position.Range
	// View returns the iterator's current 'view' i.e. the range of positions its current
	// value represents.
	View() position.Range
	BaseIterator
}

// TimeIterator iterates over segment metadata in time space.
type TimeIterator interface {
	// SeekFirst moves the iterator to the first time in its bounds. Returns false
	// if no such value exists or the iterator has accumulated an error.
	SeekFirst() bool
	// SeekLast moves the iterator to the last time in its bounds. Returns false
	// if no such value exists or the iterator has accumulated an error.
	SeekLast() bool
	// SeekLE moves the iterator to the last time less than or equal to the given
	// time. Returns false if no such value exists or the iterator has accumulated
	// an error.
	SeekLE(telem.TimeStamp) bool
	// SeekGE moves the iterator to the first time greater than or equal to the
	// given time. Returns false if no such value exists or the iterator has accumulated
	// an error.
	SeekGE(telem.TimeStamp) bool
	// Next moves the iterator across the provided span, accumulating any values it
	// encounters. Returns false if no values were accumulated or the iterator has
	// encountered an error.
	Next(telem.TimeSpan) bool
	// Prev moves the iterator across the provided span, accumulating any entries it
	// encounters. Returns false if no values were accumulated or the iterator has
	// encountered an error.
	Prev(telem.TimeSpan) bool
	// SetBounds sets the bounds of the iterator.
	SetBounds(telem.TimeRange) bool
	// Bounds returns the bounds of the iterator.
	Bounds() telem.TimeRange
	// View returns the iterator's current 'view' i.e. the range of times its current
	// value represents.
	View() telem.TimeRange
	BaseIterator
}
