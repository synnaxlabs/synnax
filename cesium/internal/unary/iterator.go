// Copyright 2025 Synnax Labs, Inc.
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
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
)

type IteratorConfig struct {
	Bounds telem.TimeRange
	// AutoChunkSize sets the maximum size of a chunk that will be returned by the
	// iterator when using AutoSpan in calls ot Next or Prev.
	AutoChunkSize int64
}

func (i IteratorConfig) domainIteratorConfig() domain.IteratorConfig {
	return domain.IteratorConfig{Bounds: i.Bounds}
}

// Override implements config.Config.
func (i IteratorConfig) Override(other IteratorConfig) IteratorConfig {
	i.Bounds = override.Zero(i.Bounds, other.Bounds)
	i.AutoChunkSize = override.Numeric(i.AutoChunkSize, other.AutoChunkSize)
	return i
}

// Validate implements config.Config.
func (i IteratorConfig) Validate() error { return nil }

var (
	_                     config.Config[IteratorConfig] = IteratorConfig{}
	DefaultIteratorConfig                               = IteratorConfig{AutoChunkSize: 1e5}
)

func IterRange(tr telem.TimeRange) IteratorConfig {
	return IteratorConfig{Bounds: domain.IterRange(tr).Bounds, AutoChunkSize: 0}
}

var errIteratorClosed = core.NewErrResourceClosed("unary.iterator")

type Iterator struct {
	alamos.Instrumentation
	IteratorConfig
	Channel  core.Channel
	internal *domain.Iterator
	view     telem.TimeRange
	frame    core.Frame
	idx      *index.Domain
	bounds   telem.TimeRange
	err      error
	closed   bool
}

func (db *DB) OpenIterator(cfgs ...IteratorConfig) (*Iterator, error) {
	if db.closed.Load() {
		return nil, db.wrapError(ErrDBClosed)
	}
	// Safe to ignore error here as Validate will always return nil
	cfg, err := config.New(DefaultIteratorConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	iter := db.domain.OpenIterator(cfg.domainIteratorConfig())
	i := &Iterator{
		idx:            db.index(),
		Channel:        db.cfg.Channel,
		internal:       iter,
		IteratorConfig: cfg,
	}
	i.SetBounds(cfg.Bounds)
	return i, nil
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
// [start, end), after Next(span) is called, the view becomes [end, end + span). After
// the view changes, the internal iterator moves forward and accumulates data until the
// entire view is contained in the iterator's frame.
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

	if i.view.Span().IsZero() || i.view.End.BeforeEq(i.internal.TimeRange().Start) {
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
	endApprox, err := i.idx.Stamp(
		ctx,
		i.view.Start,
		i.AutoChunkSize,
		index.AllowDiscontinuous,
	)
	if err != nil {
		i.err = err
		return false
	}
	if endApprox.Lower.After(i.bounds.End) {
		return i.Next(ctx, i.view.Start.Span(i.bounds.End))
	}
	i.view.End = endApprox.Lower
	i.reset(i.view.BoundBy(i.bounds))

	nRemaining := i.AutoChunkSize
	for {
		if !i.internal.TimeRange().OverlapsWith(i.view) {
			if !i.internal.Next() {
				return false
			}
			continue
		}
		startApprox, dmn, err := i.approximateStart(ctx)
		if err != nil {
			i.err = err
			return false
		}
		startOffset := i.Channel.DataType.Density().Size(startApprox.Upper)
		if !startApprox.Exact() && !startApprox.StartExact {
			// If we are starting from a cutoff dmn, use the lower offset.
			startOffset = i.Channel.DataType.Density().Size(startApprox.Lower)
		}
		series, err := i.read(
			ctx,
			dmn,
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

func (i *Iterator) autoPrev(ctx context.Context) bool {
	i.view.End = i.view.Start
	startApprox, err := i.idx.Stamp(
		ctx,
		i.view.Start,
		-i.AutoChunkSize,
		index.AllowDiscontinuous,
	)
	if err != nil {
		i.err = err
		return false
	}
	if startApprox.Lower.Before(i.bounds.Start) {
		return i.Prev(ctx, i.bounds.Start.Span(i.view.End))
	}
	i.view.Start = startApprox.Lower + 1
	i.reset(i.view.BoundBy(i.bounds))
	nRemaining := i.AutoChunkSize
	for {
		if !i.internal.TimeRange().OverlapsWith(i.view) {
			if !i.internal.Prev() {
				return false
			}
			continue
		}
		endApprox, err := i.approximateEnd(ctx)
		if err != nil {
			i.err = err
			return false
		}
		endOffset := i.Channel.DataType.Density().Size(endApprox.Upper)
		if !startApprox.Exact() && !endApprox.StartExact {
			endOffset = i.Channel.DataType.Density().Size(endApprox.Lower)
		}
		bytesToRead := i.Channel.DataType.Density().Size(nRemaining)
		if (endOffset - bytesToRead) < 0 {
			bytesToRead = endOffset
		}
		series, err := i.read(
			ctx,
			0,
			endOffset-bytesToRead,
			bytesToRead,
		)
		if err != nil && !errors.Is(err, io.EOF) {
			i.err = err
			return false
		}
		nRemaining -= series.Len()
		i.insert(series)
		if nRemaining <= 0 || !i.internal.Prev() {
			break
		}
	}
	return i.partiallySatisfied()
}

// Prev moves the iterator backward by span. More specifically, if the current view is
// [start, end), after Next(span) is called, the view becomes [start - span, start).
// After the view changes, the internal iterator moves backward and accumulates data
// until the entire view is contained in the iterator's frame.
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

	if span == AutoSpan {
		return i.autoPrev(ctx)
	}

	i.reset(i.view.Start.SpanRange(-1 * span).BoundBy(i.bounds))

	if i.view.Span().IsZero() || i.view.Start.AfterEq(i.internal.TimeRange().End) {
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

// Len returns the number of samples in the iterator's frame.
func (i *Iterator) Len() int64 { return i.frame.Len() }

// Error returns the error that caused the iterator to stop moving. If the iterator is
// still moving, Error returns nil.
func (i *Iterator) Error() error {
	wrap := core.NewChannelErrWrapper(i.Channel)
	return wrap(i.err)
}

// Valid checks if an iterator has accumulated no errors and has at least one series in
// its current frame.
func (i *Iterator) Valid() bool { return i.partiallySatisfied() && i.err == nil }

// Close closes the iterator and releases any resources it holds. As with all other
// iterator methods, Close is not safe to call concurrently with any other database
// method.
//
// After close is called, the iterator should no longer be used.
func (i *Iterator) Close() (err error) {
	if i.closed {
		return nil
	}
	i.closed = true
	wrap := core.NewChannelErrWrapper(i.Channel)
	return wrap(i.internal.Close())
}

// accumulate reads the underlying data contained in the view from OS and appends them
// to the frame. accumulate returns false if iterator must stop moving.
func (i *Iterator) accumulate(ctx context.Context) bool {
	if !i.internal.TimeRange().OverlapsWith(i.view) {
		return false
	}
	offset, alignment, size, err := i.sliceDomain(ctx)
	if err != nil {
		i.err = err
		return false
	}
	series, err := i.read(ctx, alignment, offset, size)
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
	if i.frame.Empty() || i.frame.SeriesAt(-1).TimeRange.End.BeforeEq(series.TimeRange.Start) {
		i.frame = i.frame.Append(i.Channel.Key, series)
	} else {
		i.frame = i.frame.Prepend(i.Channel.Key, series)
	}
}

func (i *Iterator) read(
	ctx context.Context,
	alignment telem.Alignment,
	offset telem.Size,
	size telem.Size,
) (series telem.Series, err error) {
	series.DataType = i.Channel.DataType
	series.TimeRange = i.internal.TimeRange().BoundBy(i.view)
	series.Data = make([]byte, size)
	// set the first 32 bits to the domain index, and the last 32 bits to the alignment
	series.Alignment = alignment
	r, err := i.internal.OpenReader(ctx)
	if err != nil {
		return series, err
	}
	n, err := r.ReadAt(series.Data, int64(offset))
	if err != nil && !errors.Is(err, io.EOF) {
		return series, err
	}
	if err = r.Close(); err != nil {
		return series, err
	}
	if n < len(series.Data) {
		series.Data = series.Data[:n]
	}
	return series, err
}

func (i *Iterator) sliceDomain(ctx context.Context) (
	telem.Size,
	telem.Alignment,
	telem.Size,
	error,
) {
	startApprox, align, err := i.approximateStart(ctx)
	if err != nil {
		return 0, align, 0, err
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
		return 0, align, 0, err
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
	return startOffset, align, size, nil
}

// approximateStart approximates the number of samples between the start of the current
// domain and the start of the current iterator view. If the start of the current view
// is before the start of the range, the returned value will be zero.
func (i *Iterator) approximateStart(ctx context.Context) (
	index.DistanceApproximation,
	telem.Alignment,
	error,
) {
	target := i.internal.TimeRange().Start.SpanRange(0)
	if i.internal.TimeRange().Start.Before(i.view.Start) {
		target.End = i.view.Start
	}
	startApprox, alignment, err := i.idx.Distance(ctx, target, index.MustBeContinuous)
	return startApprox, alignment, err
}

// approximateEnd approximates the number of samples between the start of the current
// range and the end of the current iterator view. If the end of the current view is
// after the end of the range, the returned value will be the number of samples in the
// range.
func (i *Iterator) approximateEnd(ctx context.Context) (endApprox index.DistanceApproximation, err error) {
	endApprox.Approximation = index.Exactly(i.Channel.DataType.Density().SampleCount(telem.Size(i.internal.Size())))
	if i.internal.TimeRange().End.After(i.view.End) {
		target := i.internal.TimeRange().Start.Range(i.view.End)
		endApprox, _, err = i.idx.Distance(ctx, target, index.MustBeContinuous)
	}
	return
}

// satisfied returns whether an iterator collected all telemetry in its view. An
// iterator is said to be satisfied when its frame's start and end time range is
// congruent to its view.
func (i *Iterator) satisfied() bool {
	if !i.partiallySatisfied() {
		return false
	}
	start := i.frame.SeriesAt(0).TimeRange.Start
	end := i.frame.SeriesAt(-1).TimeRange.End
	return i.view == start.Range(end)
}

func (i *Iterator) partiallySatisfied() bool { return i.frame.HasData() }

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
