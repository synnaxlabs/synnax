package kv

import (
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/kv"
)

// kvPositionIterator iterates sequentially over a channel's segments in position space.
type kvPositionIterator struct {
	ch       channel.Channel
	internal *UnaryMDIterator
	view     position.Range
	bounds   position.Range
	value    *segment.Accumulator
	_key     []byte
}

var _ core.MDPositionIterator = (*kvPositionIterator)(nil)

func newPositionIterator(db kv.DB, ch channel.Channel) *kvPositionIterator {
	return &kvPositionIterator{
		ch:       ch,
		internal: NewMDIterator(db, kv.IteratorOptions{}),
		value:    &segment.Accumulator{Density: ch.Density, Compact: false, Slice: true},
		bounds:   position.RangeMax,
		_key:     make([]byte, 11),
	}
}

func (i *kvPositionIterator) SetBounds(rng position.Range) {
	// AcquireSearcher an iterator over the entire range.
	internal := i.internal

	prefix := make([]byte, 3)
	segment.WriteKeyPrefix(i.ch.Key, prefix)
	opts := kv.PrefixIter(prefix)
	internal.SetBounds(opts.LowerBound, opts.UpperBound)

	start, end := make([]byte, 11), make([]byte, 11)
	segment.WriteKey(i.ch.Key, rng.Start, start)
	segment.WriteKey(i.ch.Key, rng.End, end)

	// SeekLE to the first segment that starts BEFORE the start of the range. If the
	// segment range overlaps with our desired range, we'll use internal as the starting
	// point for the iterator. Otherwise, we'll seek to the first segment that starts
	// after the start of the range. If this Bounds overlaps with our desired range,
	// we'll use internal as the starting point for the iterator. Otherwise, return an
	// err that the range has no data.
	if (internal.SeekLT(start) && internal.Value().Range(i.ch.Density).OverlapsWith(rng)) ||
		(internal.SeekGE(start) && internal.Value().Range(i.ch.Density).OverlapsWith(rng)) {
		rng.Start = internal.Value().Alignment
	}

	// SeekLE to the first segment that ends AFTER then end of the range If internal
	// overlaps with our desired range, we'll use internal as the ending point for the
	// iterator. Otherwise, we'll seek to the first segment that ends before the
	// end of the range. If this Bounds overlaps with our desired range, we'll use internal
	// as the ending point for the iterator. Otherwise, return an err that the
	// range has no data.
	if (internal.SeekGE(end) && internal.Value().Range(i.ch.Density).OverlapsWith(rng)) ||
		(internal.SeekLT(end) && internal.Value().Range(i.ch.Density).OverlapsWith(rng)) {
		rng.End = internal.Value().End(i.ch.Density)
	}

	segment.WriteKey(i.ch.Key, rng.Start, start)
	segment.WriteKey(i.ch.Key, rng.End, end)
	internal.SetBounds(start, end)
}

func (i *kvPositionIterator) SeekLE(pos position.Position) bool {
	if !i.internal.SeekLT(i.key(pos)) {
		return false
	}

	rng := i.internal.Value().Range(i.ch.Density)
	if !rng.OverlapsWith(i.bounds) {
		return false
	}

	// If the value range doesn't contain the position, set our view to the
	// closest position possible.
	if !rng.ContainsPos(pos) {
		i.view = rng.End.SpanRange(0)
	} else {
		i.view = pos.SpanRange(0)
	}
	return true
}

func (i *kvPositionIterator) SeekGE(pos position.Position) bool {
	if i.SeekLE(pos) && i.view.Start == pos {
		return true
	}
	if !i.internal.SeekGE(i.key(pos)) {
		return false
	}
	rng := i.internal.Value().Range(i.ch.Density)
	if !rng.OverlapsWith(i.bounds) {
		return false
	}
	i.view = rng.Start.SpanRange(0)
	return true
}

func (i *kvPositionIterator) SeekFirst() bool {
	if !i.internal.First() {
		return false
	}
	i.reset(i.internal.Value().Alignment.SpanRange(0).BoundBy(i.bounds))
	return true
}

func (i *kvPositionIterator) SeekLast() bool {
	if !i.internal.Last() {
		return false
	}
	i.reset(i.internal.Value().End(i.ch.Density).SpanRange(0).BoundBy(i.bounds))
	return true
}

func (i *kvPositionIterator) Next(span position.Span) bool {
	// If the current view is already at the end of the bounds, we can't go any further.
	if i.view.End == i.bounds.End {
		return false
	}

	i.reset(i.view.End.SpanRange(span).BoundBy(i.bounds))

	// Check the current iterator value. If it overlaps with the view, we'll use it.
	i.value.Accumulate(i.internal.Value())

	// If the current value satisfies the view, we're done.
	if i.value.Satisfied() {
		return true
	}

	// Otherwise, iterate until we stop finding values that overlap with the view.
	for i.internal.Next() && i.value.Accumulate(i.internal.Value()) {
	}

	return i.value.PartiallySatisfied()
}

func (i *kvPositionIterator) Prev(span position.Span) bool {
	// If the current view is already at the beginning of the bounds, we can't go
	// any further.
	if i.view.Start == i.bounds.Start {
		return false
	}

	i.view = i.view.Start.SpanRange(-span).BoundBy(i.bounds)
	i.value.Reset(i.view)

	// Check the current iterator value. If it overlaps with the view, we'll use it.
	i.value.Accumulate(i.internal.Value())

	// If the current value satisfies the view, we're done.
	if i.value.Satisfied() {
		return true
	}

	// Otherwise, iterate until we stop finding values that overlap with the view.
	for i.internal.Prev() && i.value.Accumulate(i.internal.Value()) {
	}

	return i.value.PartiallySatisfied()
}

func (i *kvPositionIterator) Value() []segment.MD { return i.value.Segments }

func (i *kvPositionIterator) Valid() bool {
	return i.internal.Error() == nil && i.value.PartiallySatisfied()
}

func (i *kvPositionIterator) Close() error {
	i.reset(position.Range{})
	i._key = nil
	return i.internal.Close()
}

func (i *kvPositionIterator) Error() error { return i.internal.Error() }

func (i *kvPositionIterator) View() position.Range { return i.view }

func (i *kvPositionIterator) Bounds() position.Range { return i.bounds }

func (i *kvPositionIterator) reset(view position.Range) { i.view = view; i.value.Reset(view) }

func (i *kvPositionIterator) key(pos position.Position) []byte {
	segment.WriteKey(i.ch.Key, pos, i._key)
	return i._key
}
