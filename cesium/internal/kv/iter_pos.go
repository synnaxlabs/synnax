package kv

import (
	"github.com/synnaxlabs/cesium/internal/accumulate"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/kv"
	"go.uber.org/zap"
)

// positionIterator is a key-value backed implementation of core.PositionIterator.
type positionIterator struct {
	// ch is the channel that this iterator is iterating over.
	ch core.Channel
	// internal is the underlying naive segment iterator.
	internal *coreMDIterator
	// view stores the iterators current view.
	view position.Range
	// bounds stores the iterators bounds.
	bounds position.Range
	// value stores the  current accumulated value satisfying the iterator's view.
	value *accumulate.Accumulator
	// logger is the witness of it all.
	logger *zap.Logger
	// keyBuf is a reusable buffer for writing keys.
	keyBuf []byte
}

func newPositionIterator(
	db kv.Reader,
	ch core.Channel,
	logger *zap.Logger,
) core.PositionIterator {
	return &positionIterator{
		ch:       ch,
		internal: newCoreMDIterator(db, kv.IteratorOptions{}),
		value:    &accumulate.Accumulator{Density: ch.Density, Merge: false, Slice: true},
		bounds:   position.RangeMax,
		logger:   logger,
		keyBuf:   make([]byte, 11),
	}
}

// SetBounds implements core.PositionIterator.
func (i *positionIterator) SetBounds(rng position.Range) {
	i.logger.Info("setting bounds",
		zap.Stringer("channel", i.ch.Key),
		zap.Stringer("bounds", rng),
	)
	// acquireSearcher an iterator over the entire range.
	internal := i.internal
	i.bounds = rng
	i.reset(rng)

	prefix := make([]byte, 3)
	core.WriteSegmentKeyPrefix(i.ch.Key, prefix)
	// Open an iterator over the entire channel's data, so we can
	// establish an accurate lower and upper kv bound.
	opts := kv.PrefixIter(prefix)
	internal.SetBounds(opts.LowerBound, opts.UpperBound)

	start, end := make([]byte, 11), make([]byte, 11)
	core.WriteSegmentKey(i.ch.Key, rng.Start, start)
	core.WriteSegmentKey(i.ch.Key, rng.End, end)
	i.logger.Debug("setting bounds",
		zap.Stringer("channel", i.ch.Key),
		zap.Binary("kvPrefix", prefix),
		zap.Stringer("bounds", rng),
		zap.Binary("kvStart", start),
		zap.Binary("kvEnd", end),
	)

	// SeekLE to the first segment that starts BEFORE the start of the range. If the
	// segment range overlaps with our desired range, we'll use it as the starting
	// point for the iterator. Otherwise, we'll seek to the first segment that starts
	// after the start of the range. If this overlaps with our desired range,
	// we'll use internal as the starting point for the iterator. Otherwise,
	// the iterator has no data.
	if (internal.SeekLT(start) && internal.Value().Range(i.ch.Density).OverlapsWith(rng)) ||
		(internal.SeekGE(start) && internal.Value().Range(i.ch.Density).OverlapsWith(rng)) {
		rng.Start = internal.Value().Alignment
	}

	// Same process as above, but ofr hte end of the range.
	if (internal.SeekGE(end) && internal.Value().Range(i.ch.Density).OverlapsWith(rng)) ||
		(internal.SeekLT(end) && internal.Value().Range(i.ch.Density).OverlapsWith(rng)) {
		rng.End = internal.Value().End(i.ch.Density)
	}

	core.WriteSegmentKey(i.ch.Key, rng.Start, start)
	core.WriteSegmentKey(i.ch.Key, rng.End, end)

	i.logger.Debug("adjusted kv bounds",
		zap.Stringer("channel", i.ch.Key),
		zap.Binary("prefix", prefix),
		zap.Stringer("bounds", rng),
		zap.Binary("kvStart", start),
		zap.Binary("kvEnd", end),
	)

	internal.SetBounds(start, end)
}

// SeekLE implements core.PositionIterator.
func (i *positionIterator) SeekLE(pos position.Position) bool {
	i.logger.Debug("seeking le",
		zap.Stringer("channel", i.ch.Key),
		zap.Stringer("position", pos),
	)

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

// SeekGE implements core.PositionIterator.
func (i *positionIterator) SeekGE(pos position.Position) bool {
	i.logger.Debug("seeking ge",
		zap.Stringer("channel", i.ch.Key),
		zap.Stringer("position", pos),
	)
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

// SeekFirst implements core.PositionIterator.
func (i *positionIterator) SeekFirst() bool {
	i.logger.Debug("seeking first",
		zap.Stringer("channel", i.ch.Key),
	)
	if !i.internal.First() {
		return false
	}
	i.reset(i.internal.Value().Alignment.SpanRange(0).BoundBy(i.bounds))
	return true
}

// SeekLast implements core.PositionIterator.
func (i *positionIterator) SeekLast() bool {
	i.logger.Debug("seeking last",
		zap.Stringer("channel", i.ch.Key),
	)
	if !i.internal.Last() {
		return false
	}
	i.reset(i.internal.Value().End(i.ch.Density).SpanRange(0).BoundBy(i.bounds))
	return true
}

// Next implements core.PositionIterator.
func (i *positionIterator) Next(span position.Span) bool {
	i.clearValue()
	i.logger.Debug("next",
		zap.Stringer("channel", i.ch.Key),
		zap.Stringer("view", i.view),
		zap.Stringer("span", span),
		zap.Bool("auto", span == core.AutoPosSpan),
		zap.Stringer("bounds", i.bounds),
	)
	defer func() {
		i.logger.Debug("next done",
			zap.Stringer("channel", i.ch.Key),
			zap.Stringer("view", i.view),
			zap.Stringer("accumulatorBounds", i.value.Bounds),
		)
	}()

	if i.atEnd() {
		i.view.Start = i.view.End
		return false
	}

	if span == core.AutoPosSpan {
		return i.autoNext()
	}

	i.reset(i.view.End.SpanRange(span).BoundBy(i.bounds))

	// Check the current iterator value. If it overlaps with the view, we'll use it.
	i.value.Accumulate(i.internal.Value())

	// If the current value satisfies the view, we're done.
	if i.value.Satisfied() {
		return true
	}

	// Otherwise, iterate until we stop finding values that overlap with the view.
	for i.internal.Next() {
		if !i.value.Accumulate(i.internal.Value()) {
			break
		}
	}

	return i.value.PartiallySatisfied()
}

func (i *positionIterator) autoNext() bool {
	i.value.Reset(i.view.End.SpanRange(position.SpanMax).BoundBy(i.Bounds()))
	if !i.View().IsZero() && !i.internal.Next() {
		return false
	}
	i.value.Accumulate(i.internal.Value())
	i.view = i.internal.Value().Range(i.ch.Density).BoundBy(i.bounds)
	return i.value.PartiallySatisfied()
}

// Prev implements core.PositionIterator.
func (i *positionIterator) Prev(span position.Span) bool {
	i.clearValue()
	i.logger.Debug("prev",
		zap.Stringer("channel", i.ch.Key),
		zap.Stringer("view", i.view),
		zap.Stringer("span", span),
		zap.Bool("auto", span == core.AutoPosSpan),
	)
	// If the current view is already at the beginning of the bounds, we can't go
	// any further.
	if i.atStart() {
		i.view.End = i.view.Start
		return false
	}

	if span == core.AutoPosSpan {
		return i.autoPrev()
	}

	i.reset(i.view.Start.SpanRange(-span).BoundBy(i.bounds))

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

func (i *positionIterator) atStart() bool {
	return i.view.Start == i.bounds.Start
}

func (i *positionIterator) atEnd() bool {
	return i.view.End == i.bounds.End
}

func (i *positionIterator) autoPrev() bool {
	i.value.Reset(i.view.Start.SpanRange(position.SpanMax).BoundBy(i.Bounds()))
	if !i.View().IsZero() && !i.internal.Prev() {
		return false
	}
	i.value.Accumulate(i.internal.Value())
	i.view = i.internal.Value().Range(i.ch.Density).BoundBy(i.bounds)
	return i.value.PartiallySatisfied()
}

// Value implements core.PositionIterator.
func (i *positionIterator) Value() []core.SegmentMD { return i.value.Segments }

// Valid implements core.PositionIterator.
func (i *positionIterator) Valid() bool {
	return i.internal.Error() == nil && i.value.PartiallySatisfied()
}

// Close implements core.PositionIterator.
func (i *positionIterator) Close() error {
	i.reset(position.Range{})
	i.keyBuf = nil
	return i.internal.Close()
}

// Error implements core.PositionIterator.
func (i *positionIterator) Error() error { return i.internal.Error() }

// View implements core.PositionIterator.
func (i *positionIterator) View() position.Range { return i.view }

// Bounds implements core.PositionIterator.
func (i *positionIterator) Bounds() position.Range { return i.bounds }

func (i *positionIterator) reset(view position.Range) { i.view = view; i.value.Reset(view) }

func (i *positionIterator) clearValue() { i.value.Reset(position.RangeZero) }

func (i *positionIterator) key(pos position.Position) []byte {
	core.WriteSegmentKey(i.ch.Key, pos, i.keyBuf)
	return i.keyBuf
}
