// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package unary

import (
	"context"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"io"
)

type IteratorConfig struct {
	Bounds telem.TimeRange
	// AutoChunkSize sets the maximum size of a chunk that will be returned by the
	// iterator when using AutoSpan in calls ot Next or Prev.
	AutoChunkSize int64
}

func IterRange(tr telem.TimeRange) IteratorConfig {
	return IteratorConfig{Bounds: domain.IterRange(tr).Bounds, AutoChunkSize: 0}
}

var (
	errIteratorClosed     = core.EntityClosed("unary.iterator")
	DefaultIteratorConfig = IteratorConfig{AutoChunkSize: 1e5}
)

type Iterator struct {
	alamos.Instrumentation
	IteratorConfig
	Channel  core.Channel
	onClose  func()
	internal *domain.Iterator
	view     telem.TimeRange
	frame    core.Frame
	idx      index.Index
	bounds   telem.TimeRange
	err      error
	closed   bool
}

const AutoSpan telem.TimeSpan = -1

// SetBounds sets the iterator's bounds. The iterator is invalidated, and will not be
// valid until a seeking call is made.
func (i *Iterator) SetBounds(tr telem.TimeRange) {
	i.bounds = tr
	i.internal.SetBounds(tr)
	i.seekReset(i.bounds.End)
}

func (i *Iterator) Bounds() telem.TimeRange { return i.bounds }

func (i *Iterator) Value() core.Frame { return i.frame }

func (i *Iterator) View() telem.TimeRange { return i.view }

func (i *Iterator) SeekFirst(ctx context.Context) bool {
	if i.closed {
		i.err = errIteratorClosed
		return false
	}
	ok := i.internal.SeekFirst(ctx)
	i.seekReset(i.internal.TimeRange().BoundBy(i.bounds).Start)
	return ok
}

// SeekLast moves the iterator to the end of the last domain in its bounds.
func (i *Iterator) SeekLast(ctx context.Context) bool {
	if i.closed {
		i.err = errIteratorClosed
		return false
	}
	ok := i.internal.SeekLast(ctx)
	i.seekReset(i.internal.TimeRange().BoundBy(i.bounds).End)
	return ok
}

func (i *Iterator) SeekLE(ctx context.Context, ts telem.TimeStamp) bool {
	if i.closed {
		i.err = errIteratorClosed
		return false
	}

	ok := i.internal.SeekLE(ctx, ts)

	if i.internal.TimeRange().OverlapsWith(ts.SpanRange(0)) {
		// If the provided ts is in the seeked domain, set the view to be ts.
		i.seekReset(ts)
	} else {
		// Otherwise, set the view to the end of the seeked domain or bounds, whichever
		// one is earlier.
		i.seekReset(i.internal.TimeRange().BoundBy(i.bounds).End)
	}
	return ok
}

func (i *Iterator) SeekGE(ctx context.Context, ts telem.TimeStamp) bool {
	if i.closed {
		i.err = errIteratorClosed
		return false
	}

	ok := i.internal.SeekGE(ctx, ts)

	if i.internal.TimeRange().OverlapsWith(ts.SpanRange(0)) {
		// If the provided ts is in the seeked domain, set the view to be ts.
		i.seekReset(ts)
	} else {
		// Otherwise, set the view to the start of the seeked domain or bounds, whichever
		// one is later.
		i.seekReset(i.internal.TimeRange().BoundBy(i.bounds).Start)
	}
	return ok
}

// Next moves the iterator forward by span. More specifically, if the current view is
// [start, end), after Next(span) is called, the view becomes [end, end + span).
// After the view changes, the internal iterator moves forward and accumulates data until
// the entire view is contained in the iterator's frame.
func (i *Iterator) Next(ctx context.Context, span telem.TimeSpan) (ok bool) {
	if i.closed {
		i.err = errIteratorClosed
		return false
	}
	ctx, span_ := i.T.Bench(ctx, "Next")
	defer func() {
		ok = i.Valid()
		span_.End()
	}()
	if i.atEnd() {
		i.reset(i.bounds.End.SpanRange(0))
		return
	}

	if span == AutoSpan {
		return i.autoNext(ctx)
	}

	i.reset(i.view.End.SpanRange(span).BoundBy(i.bounds))

	if i.view.IsZero() || i.view.End.BeforeEq(i.internal.TimeRange().Start) {
		return
	}

	i.accumulate(ctx)
	if i.satisfied() || i.err != nil {
		return
	}

	for i.internal.Next() &&
		i.accumulate(ctx) &&
		!i.satisfied() {
	}
	return
}

func (i *Iterator) autoNext(ctx context.Context) bool {
	i.view.Start = i.view.End
	endApprox, err := i.idx.Stamp(ctx, i.view.Start, i.IteratorConfig.AutoChunkSize, false)
	if err != nil {
		i.err = err
		return false
	}
	if endApprox.Lower.After(i.bounds.End) {
		return i.Next(ctx, i.view.Start.Span(i.bounds.End))
	}
	i.view.End = endApprox.Lower
	i.reset(i.view.BoundBy(i.bounds))

	nRemaining := i.IteratorConfig.AutoChunkSize
	for {
		if !i.internal.TimeRange().OverlapsWith(i.view) {
			if !i.internal.Next() {
				return false
			}
			continue
		}
		startApprox, domain, err := i.approximateStart(ctx)
		if err != nil {
			i.err = err
			return false
		}
		startOffset := i.Channel.DataType.Density().Size(startApprox.Upper)
		if !startApprox.Exact() && !startApprox.StartExact {
			// If we are starting from a cutoff domain, use the lower offset.
			startOffset = i.Channel.DataType.Density().Size(startApprox.Lower)
		}
		series, err := i.read(
			ctx,
			domain,
			startOffset,
			i.Channel.DataType.Density().Size(nRemaining),
		)
		if err != nil && !errors.Is(err, io.EOF) {
			i.err = err
			return false
		}
		nRemaining -= series.Len()
		i.insert(series)
		if nRemaining <= 0 || !i.internal.Next() {
			break
		}
	}

	return i.partiallySatisfied()
}

// Prev moves the iterator backward by span. More specifically, if the current view is
// [start, end), after Next(span) is called, the view becomes [start - span, start).
// After the view changes, the internal iterator moves backward and accumulates data until
// the entire view is contained in the iterator's frame.
func (i *Iterator) Prev(ctx context.Context, span telem.TimeSpan) (ok bool) {
	if i.closed {
		i.err = errIteratorClosed
		return false
	}
	ctx, span_ := i.T.Bench(ctx, "Prev")
	defer func() {
		ok = i.Valid()
		span_.End()
	}()

	if i.atStart() {
		i.reset(i.bounds.Start.SpanRange(0))
		return
	}

	i.reset(i.view.Start.SpanRange(-1 * span).BoundBy(i.bounds))

	if i.view.IsZero() || i.view.Start.AfterEq(i.internal.TimeRange().End) {
		return
	}

	i.accumulate(ctx)
	if i.satisfied() || i.err != nil {
		return
	}

	for i.internal.Prev() &&
		i.accumulate(ctx) &&
		!i.satisfied() {
	}
	return
}

func (i *Iterator) Len() (l int64) {
	for _, series := range i.frame.Series {
		l += series.Len()
	}
	return
}

func (i *Iterator) Error() error {
	wrap := core.NewErrorWrapper(i.Channel)
	return wrap(i.err)
}

// Valid checks if an iterator has accumulated no errors and has at least one series
// in its current frame.
func (i *Iterator) Valid() bool { return i.partiallySatisfied() && i.err == nil }

func (i *Iterator) Close() (err error) {
	if i.closed {
		return nil
	}
	i.onClose()
	i.closed = true
	wrap := core.NewErrorWrapper(i.Channel)
	return wrap(i.internal.Close())
}

// accumulate reads the underlying data contained in the view from OS and
// appends them to the frame.
// accumulate returns false if iterator must stop moving.
func (i *Iterator) accumulate(ctx context.Context) bool {
	if !i.internal.TimeRange().OverlapsWith(i.view) {
		return false
	}
	offset, domain, size, err := i.sliceDomain(ctx)
	if err != nil {
		i.err = err
		return false
	}
	series, err := i.read(ctx, domain, offset, size)
	if err != nil && !errors.Is(err, io.EOF) {
		i.err = err
		return false
	}
	i.insert(series)
	return true
}

func (i *Iterator) insert(series telem.Series) {
	if series.Len() == 0 {
		return
	}
	if len(i.frame.Series) == 0 || i.frame.Series[len(i.frame.Series)-1].TimeRange.End.BeforeEq(series.TimeRange.Start) {
		i.frame = i.frame.Append(i.Channel.Key, series)
	} else {
		i.frame = i.frame.Prepend(i.Channel.Key, series)
	}
}

func (i *Iterator) read(
	ctx context.Context,
	idxDomain uint32,
	offset telem.Offset,
	size telem.Size,
) (series telem.Series, err error) {
	series.DataType = i.Channel.DataType
	series.TimeRange = i.internal.TimeRange().BoundBy(i.view)
	series.Data = make([]byte, size)
	inDomainAlignment := uint32(i.Channel.DataType.Density().SampleCount(offset))
	// set the first 32 bits to the domain index, and the last 32 bits to the alignment
	series.Alignment = telem.AlignmentPair(idxDomain)<<32 | telem.AlignmentPair(inDomainAlignment)
	r, err := i.internal.NewReader(ctx)
	if err != nil {
		return
	}
	n, err := r.ReadAt(series.Data, int64(offset))
	if err != nil && !errors.Is(err, io.EOF) {
		return
	}
	err = r.Close()
	if err != nil {
		return
	}
	if n < len(series.Data) {
		series.Data = series.Data[:n]
	}
	return
}

func (i *Iterator) sliceDomain(ctx context.Context) (
	telem.Offset,
	uint32,
	telem.Size,
	error,
) {
	startApprox, domain, err := i.approximateStart(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	startOffset := i.Channel.DataType.Density().Size(startApprox.Upper)
	// Split into cases to determine which offsets to use. See unary/delete.go's
	// calculateStartOffset function for more detail.
	if !startApprox.Exact() && !startApprox.StartExact {
		if startApprox.EndExact {
			// If the start of the domain is inexact due to cutoff, but the end
			// approximation is exact, we want to use the lower approximation.
			startOffset = i.Channel.DataType.Density().Size(startApprox.Lower)
		} else {
			off := (startApprox.Lower + startApprox.Upper) / 2
			startOffset = i.Channel.DataType.Density().Size(off)
		}
	}
	endApprox, err := i.approximateEnd(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	endOffset := i.Channel.DataType.Density().Size(endApprox.Upper)
	// Split into cases to determine which offsets to use. See unary/delete.go's
	// calculateEndOffset function for more detail.
	if !endApprox.Exact() && !endApprox.StartExact {
		if endApprox.EndExact {
			// If the start of the domain is inexact due to cutoff, but the end
			// approximation is exact, we want to use the lower approximation.
			endOffset = i.Channel.DataType.Density().Size(endApprox.Lower)
		} else {
			off := (endApprox.Lower + endApprox.Upper) / 2
			endOffset = i.Channel.DataType.Density().Size(off)
		}
	}

	size := endOffset - startOffset
	return startOffset, domain, size, nil
}

// approximateStart approximates the number of samples between the start of the current
// range and the start of the current iterator view. If the start of the current view is
// before the start of the range, the returned value will be zero.
func (i *Iterator) approximateStart(ctx context.Context) (
	index.DistanceApproximation,
	uint32,
	error,
) {
	target := i.internal.TimeRange().Start.SpanRange(0)
	if i.internal.TimeRange().Start.Before(i.view.Start) {
		target.End = i.view.Start
	}
	startApprox, domainApprox, err := i.idx.Distance(ctx, target, true)
	return startApprox, domainApprox.Lower, err
}

// approximateEnd approximates the number of samples between the start of the current
// range and the end of the current iterator view. If the end of the current view is
// after the end of the range, the returned value will be the number of samples in the
// range.
func (i *Iterator) approximateEnd(ctx context.Context) (endApprox index.DistanceApproximation, err error) {
	endApprox.Approximation = index.Exactly(i.Channel.DataType.Density().SampleCount(telem.Size(i.internal.Len())))
	if i.internal.TimeRange().End.After(i.view.End) {
		target := i.internal.TimeRange().Start.Range(i.view.End)
		endApprox, _, err = i.idx.Distance(ctx, target, true)
	}
	return
}

// satisfied returns whether an iterator collected all telemetry in its view.
// An iterator is said to be satisfied when its frame's start and end timerange is
// congruent to its view.
func (i *Iterator) satisfied() bool {
	if !i.partiallySatisfied() {
		return false
	}
	start := i.frame.Series[0].TimeRange.Start
	end := i.frame.Series[len(i.frame.Series)-1].TimeRange.End
	return i.view == start.Range(end)
}

func (i *Iterator) partiallySatisfied() bool { return len(i.frame.Series) > 0 }

func (i *Iterator) reset(nextView telem.TimeRange) {
	i.frame = core.Frame{}
	i.view = nextView
}

func (i *Iterator) seekReset(ts telem.TimeStamp) {
	i.reset(ts.SpanRange(0))
	i.err = nil
}

func (i *Iterator) atStart() bool { return i.view.Start == i.bounds.Start }

func (i *Iterator) atEnd() bool { return i.view.End == i.bounds.End }
