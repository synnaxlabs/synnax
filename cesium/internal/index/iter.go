package index

import (
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/position"
	"github.com/synnaxlabs/cesium/internal/segment"
	"github.com/synnaxlabs/x/telem"
)

type mdIndexIterator struct {
	internal  core.MDPositionIterator
	positionS PositionSearcher
	stampS    StampSearcher
	bounds    telem.TimeRange
	view      telem.TimeRange
	err       error
}

func WrapMDIter(
	wrapped core.MDPositionIterator,
	positionSeeker PositionSearcher,
	stampSeeker StampSearcher,
) core.MDStampIterator {
	return &mdIndexIterator{
		internal:  wrapped,
		positionS: positionSeeker,
		stampS:    stampSeeker,
	}
}

func (i *mdIndexIterator) SetBounds(bounds telem.TimeRange) bool {
	start, ok := i.searchP(bounds.Start, position.Uncertain)
	if !ok {
		return false
	}
	end, ok := i.searchP(bounds.End, position.Uncertain)
	if !ok {
		return false
	}
	i.internal.SetBounds(position.Range{Start: start.Start, End: end.End})
	i.bounds = bounds
	return true
}

func (i *mdIndexIterator) SeekLE(stamp telem.TimeStamp) bool {
	pos, ok := i.searchPInBounds(stamp)
	if !ok {
		return false
	}
	i.updateView()
	return i.internal.SeekLE(pos.Start)
}

func (i *mdIndexIterator) SeekGE(stamp telem.TimeStamp) bool {
	pos, ok := i.searchPInBounds(stamp)
	if !ok {
		return false
	}
	if !i.internal.SeekLE(pos.End) {
		return false
	}
	return i.updateView()
}

func (i *mdIndexIterator) SeekFirst() bool {
	if !i.internal.SeekFirst() {
		return false
	}
	return i.updateView()
}

func (i *mdIndexIterator) SeekLast() bool {
	if !i.internal.SeekLast() {
		return false
	}
	return i.updateView()
}

func (i *mdIndexIterator) Next(span telem.TimeSpan) bool {
	posSpan, ok := i.seekSpan(span)
	if !ok {
		return false
	}
	if !i.internal.Next(posSpan) {
		return false
	}
	return i.updateView()
}

func (i *mdIndexIterator) Prev(span telem.TimeSpan) bool {
	posSpan, ok := i.seekSpan(span)
	if !ok {
		return false
	}
	if !i.internal.Prev(posSpan) {
		return false
	}
	return i.updateView()
}

func (i *mdIndexIterator) Close() error {
	return i.internal.Close()
}

func (i *mdIndexIterator) Valid() bool {
	return i.err == nil && i.internal.Valid()
}

func (i *mdIndexIterator) View() telem.TimeRange {
	return i.view
}

func (i *mdIndexIterator) Value() []segment.MD {
	return i.internal.Value()
}

func (i *mdIndexIterator) Error() error {
	return errors.CombineErrors(i.err, i.internal.Error())
}

func (i *mdIndexIterator) Bounds() telem.TimeRange {
	return i.bounds
}

func (i *mdIndexIterator) updateView() bool {
	start, ok := i.searchTSInBounds(i.internal.View().Start)
	if !ok {
		return false
	}
	end, ok := i.searchTSInBounds(i.internal.View().End)
	if !ok {
		return false
	}
	i.view = telem.TimeRange{Start: start.Value(), End: end.Value()}
	return true
}

func (i *mdIndexIterator) seekSpan(span telem.TimeSpan) (position.Span, bool) {
	rng := i.view.Start.SpanRange(span)
	startPos, ok := i.searchPInBounds(rng.Start)
	if !ok {
		return 0, false
	}
	endPos, ok := i.searchPInBounds(rng.End)
	if !ok {
		return 0, false
	}
	return position.Span(startPos.Start - endPos.End), true
}

func (i *mdIndexIterator) searchP(stamp telem.TimeStamp, guess position.Approximation) (position.Approximation, bool) {
	rng, err := i.positionS.SearchP(stamp, guess)
	if err != nil {
		i.err = err
	}
	return rng, err == nil
}

func (i *mdIndexIterator) searchPInBounds(stamp telem.TimeStamp) (position.Approximation, bool) {
	return i.searchP(
		stamp,
		position.Between(i.internal.Bounds().Start, i.internal.Bounds().End),
	)
}

func (i *mdIndexIterator) searchTS(pos position.Position, guess telem.Approximation) (telem.Approximation, bool) {
	rng, err := i.stampS.SearchTS(pos, guess)
	if err != nil {
		i.err = err
	}
	return rng, err == nil
}

func (i *mdIndexIterator) searchTSInBounds(pos position.Position) (telem.Approximation, bool) {
	return i.searchTS(
		pos,
		telem.Between(i.bounds.Start, i.bounds.End),
	)
}
