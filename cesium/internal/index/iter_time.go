package index

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

type timeIterator struct {
	internal core.PositionIterator
	idx      Searcher
	bounds   telem.TimeRange
	view     telem.TimeRange
	err      error
	logger   *zap.Logger
}

// WrapPositionIter wraps a core.PositionIterator and a searchable index to implement
// a core.TimeIterator.
func WrapPositionIter(
	wrap core.PositionIterator,
	idx Searcher,
	logger *zap.Logger,
) core.TimeIterator {
	return &timeIterator{internal: wrap, idx: idx, logger: logger}
}

// SetBounds implements core.TimeIterator.
func (i *timeIterator) SetBounds(bounds telem.TimeRange) bool {
	startApprox, startOk := i.searchP(bounds.Start, position.Uncertain)
	endApprox, endOk := i.searchP(bounds.End, position.Uncertain)
	i.logger.Debug(
		"setting bounds",
		zap.Stringer("bounds", bounds),
		zap.Stringer("idx", i.idx.Key()),
		zap.Stringer("startApprox", startApprox),
		zap.Stringer("endApprox", endApprox),
	)
	if !startOk || !endOk {
		return false
	}

	i.internal.SetBounds(position.Range{
		Start: startApprox.End,
		End:   endApprox.End,
	})

	i.bounds = bounds

	return true
}

// SeekLE implements core.TimeIterator.
func (i *timeIterator) SeekLE(stamp telem.TimeStamp) bool {
	pos, ok := i.searchPInBounds(stamp)
	i.logger.Debug(
		"seeking LE",
		zap.Stringer("idx", i.idx.Key()),
		zap.Stringer("stamp", stamp),
		zap.Stringer("posApprox", pos),
	)
	if !ok {
		return false
	}
	i.updateView()
	return i.internal.SeekLE(pos.Start)
}

// SeekGE implements core.TimeIterator.
func (i *timeIterator) SeekGE(stamp telem.TimeStamp) bool {
	pos, ok := i.searchPInBounds(stamp)
	i.logger.Debug(
		"seeking GE",
		zap.Stringer("idx", i.idx.Key()),
		zap.Stringer("stamp", stamp),
		zap.Stringer("posApprox", pos),
	)
	if !ok {
		return false
	}
	if !i.internal.SeekLE(pos.End) {
		return false
	}
	return i.updateView()
}

// SeekFirst implements core.TimeIterator.
func (i *timeIterator) SeekFirst() bool {
	i.logger.Debug("seeking first",
		zap.Stringer("idx", i.idx.Key()),
	)
	if !i.internal.SeekFirst() {
		return false
	}
	return i.updateView()
}

// SeekLast implements core.TimeIterator.
func (i *timeIterator) SeekLast() bool {
	i.logger.Debug("seeking last",
		zap.Stringer("idx", i.idx.Key()),
	)
	if !i.internal.SeekLast() {
		return false
	}
	return i.updateView()
}

// Next implements core.TimeIterator.
func (i *timeIterator) Next(span telem.TimeSpan) bool {
	i.logger.Debug("next",
		zap.Stringer("idx", i.idx.Key()),
		zap.Stringer("span", span),
		zap.Bool("auto", span == core.AutoTimeSpan),
	)
	if span == core.AutoTimeSpan {
		if !i.internal.Next(core.AutoPosSpan) {
			return false
		}
		return i.updateView()
	}

	start := i.internal.View().End
	endApprox, ok := i.searchPInBounds(i.view.End.Add(span))
	i.logger.Debug("next: approximations",
		zap.Stringer("idx", i.idx.Key()),
		zap.Stringer("startApprox", start),
		zap.Stringer("endApprox", endApprox),
	)
	if !ok {
		return false
	}
	posSpan := start.Span(endApprox.Start)
	// When we approach the end of the bounds, we end up
	// approximating a zero span, and end up in an infinite loop.
	// This is a workaround for that.
	if endApprox.End == i.internal.Bounds().End {
		posSpan = start.Span(endApprox.End)
	}
	if !i.internal.Next(posSpan) {
		return false
	}
	return i.updateView()
}

// Prev implements core.TimeIterator.
func (i *timeIterator) Prev(span telem.TimeSpan) bool {
	i.logger.Debug("prev",
		zap.Stringer("idx", i.idx.Key()),
		zap.Stringer("span", span),
		zap.Bool("auto", span == core.AutoTimeSpan),
	)
	if span == core.AutoTimeSpan {
		if !i.internal.Prev(core.AutoPosSpan) {
			return false
		}
		return i.updateView()
	}
	end := i.internal.View().Start
	startApprox, ok := i.searchPInBounds(i.view.Start.Sub(span))
	i.logger.Debug("prev: startApprox",
		zap.Stringer("idx", i.idx.Key()),
		zap.Stringer("endApprox", end),
		zap.Stringer("startApprox", startApprox),
	)
	if !ok {
		return false
	}
	posSpan := startApprox.End.Span(end)
	// When we approach the ned of the bounds, we end up
	// approximating a zero span, and end up in an infinite loop.
	// This is a workaround for that.
	if startApprox.Start == i.internal.Bounds().Start {
		posSpan = startApprox.Start.Span(end)
	}
	if !i.internal.Prev(posSpan) {
		return false
	}
	return i.updateView()
}

// Close implements core.TimeIterator.
func (i *timeIterator) Close() error {
	i.idx.Release()
	return i.internal.Close()
}

// Valid implements core.TimeIterator.
func (i *timeIterator) Valid() bool { return i.err == nil && i.internal.Valid() }

// View implements core.TimeIterator.
func (i *timeIterator) View() telem.TimeRange { return i.view }

// Value implements core.TimeIterator.
func (i *timeIterator) Value() []core.SegmentMD {
	segs := i.internal.Value()
	for j, seg := range segs {
		start, _ := i.searchTSInBounds(seg.Alignment)
		start.WarnIfInexact()
		seg.Start = start.Start
		segs[j] = seg
	}
	return segs
}

// Error implements core.TimeIterator.
func (i *timeIterator) Error() error {
	return errors.CombineErrors(i.err, i.internal.Error())
}

// Bounds implements core.TimeIterator.
func (i *timeIterator) Bounds() telem.TimeRange { return i.bounds }

func (i *timeIterator) updateView() bool {
	startApprox, ok := i.searchTSInBounds(i.internal.View().Start)
	if !ok {
		return false
	}
	endApprox, ok := i.searchTSInBounds(i.internal.View().End)
	if !ok {
		return false
	}
	startApprox.WarnIfInexact()
	startApprox.WarnIfInexact()
	i.view = telem.TimeRange{Start: startApprox.Start, End: endApprox.End}
	return true
}

func (i *timeIterator) searchSpan(span telem.TimeSpan) (position.Span, bool) {
	rng := i.view.Start.SpanRange(span)
	startPos, ok := i.searchPInBounds(rng.Start)
	if !ok {
		return 0, false
	}
	endPos, ok := i.searchPInBounds(rng.End)
	if !ok {
		return 0, false
	}
	return position.Span(endPos.Start - startPos.End), true
}

func (i *timeIterator) searchP(stamp telem.TimeStamp, guess position.Approximation) (position.Approximation, bool) {
	rng, err := i.idx.SearchP(stamp, guess)
	if err != nil {
		i.err = err
	}
	return rng, err == nil
}

func (i *timeIterator) searchPInBounds(stamp telem.TimeStamp) (position.Approximation, bool) {
	return i.searchP(
		stamp,
		position.Between(i.internal.Bounds().Start, i.internal.Bounds().End),
	)
}

func (i *timeIterator) searchTS(pos position.Position, guess telem.Approximation) (telem.Approximation, bool) {
	rng, err := i.idx.SearchTS(pos, guess)
	if err != nil {
		i.err = err
	}
	return rng, err == nil
}

func (i *timeIterator) searchTSInBounds(pos position.Position) (telem.Approximation, bool) {
	return i.searchTS(
		pos,
		telem.Between(i.bounds.Start, i.bounds.End),
	)
}

func (i *timeIterator) internalAtStart() bool {
	return i.internal.View().Start == i.internal.Bounds().Start
}

func (i *timeIterator) internalAtEnd() bool {
	return i.internal.View().End == i.internal.Bounds().End
}
