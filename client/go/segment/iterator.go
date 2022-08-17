package segment

import "github.com/arya-analytics/x/telem"

type Iterator interface {
	First() bool
	Last() bool
	Next() bool
	Prev() bool
	NextSpan(span telem.TimeSpan) bool
	PrevSpan(span telem.TimeSpan) bool
	NextRange(tr telem.TimeRange) bool
	SeekFirst() bool
	SeekLast() bool
	SeekLT(t telem.TimeStamp) bool
	SeekGE(t telem.TimeStamp) bool
	Close() error
	Valid() bool
	Error() error
	Exhaust() bool
}

type Sample[V any] struct {
	TimeStamp telem.TimeStamp
	Value     V
}

type Range[V any] struct {
	TimeRange telem.TimeRange
	Values    []V
}
