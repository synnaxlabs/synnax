package core

import (
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/telem"
)

type MDIterator interface {
	Value() []segment.MD
}

type BaseIterator interface {
	Close() error
	Error() error
	Valid() bool
}

type PositionIterator interface {
	SeekFirst() bool
	SeekLast() bool
	SeekLE(pos position.Position) bool
	SeekGE(pos position.Position) bool
	Next(span position.Span) bool
	Prev(span position.Span) bool
	SetBounds(rng position.Range)
	Bounds() position.Range
	View() position.Range
	BaseIterator
}

type MDPositionIterator interface {
	PositionIterator
	MDIterator
}

type StampIterator interface {
	SeekFirst() bool
	SeekLast() bool
	SeekGE(telem.TimeStamp) bool
	SeekLE(telem.TimeStamp) bool
	Next(telem.TimeSpan) bool
	Prev(telem.TimeSpan) bool
	SetBounds(telem.TimeRange) bool
	Bounds() telem.TimeRange
	View() telem.TimeRange
	BaseIterator
	MDIterator
}

type MDStampIterator interface {
	StampIterator
	MDIterator
}
