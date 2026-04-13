// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package variable

import (
	"context"
	"io"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/domain"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/telem"
)

type IteratorConfig struct {
	Bounds        telem.TimeRange
	AutoChunkSize int64
}

func (i IteratorConfig) domainIteratorConfig() domain.IteratorConfig {
	return domain.IteratorConfig{Bounds: i.Bounds}
}

func (i IteratorConfig) Override(other IteratorConfig) IteratorConfig {
	i.Bounds = override.Zero(i.Bounds, other.Bounds)
	i.AutoChunkSize = override.Numeric(i.AutoChunkSize, other.AutoChunkSize)
	return i
}

func (i IteratorConfig) Validate() error { return nil }

var (
	_                     config.Config[IteratorConfig] = IteratorConfig{}
	DefaultIteratorConfig                               = IteratorConfig{AutoChunkSize: 1e5}
)

func IterRange(tr telem.TimeRange) IteratorConfig {
	return IteratorConfig{Bounds: domain.IterRange(tr).Bounds, AutoChunkSize: 0}
}

var errIteratorClosed = resource.NewClosedError("variable.iterator")

type Iterator struct {
	alamos.Instrumentation
	err      error
	internal *domain.Iterator
	idx      *index.Domain
	Channel  channel.Channel
	frame    channel.Frame
	cache    *offsetCache
	IteratorConfig
	view   telem.TimeRange
	bounds telem.TimeRange
	closed bool
}

func (db *DB) OpenIterator(cfgs ...IteratorConfig) (*Iterator, error) {
	if db.closed.Load() {
		return nil, db.wrapError(ErrDBClosed)
	}
	cfg, err := config.New(DefaultIteratorConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	iter := db.domain.OpenIterator(cfg.domainIteratorConfig())
	i := &Iterator{
		idx:            db.index(),
		Channel:        db.cfg.Channel,
		internal:       iter,
		cache:          db.offsets,
		IteratorConfig: cfg,
	}
	i.SetBounds(cfg.Bounds)
	return i, nil
}

func (i *Iterator) SetBounds(tr telem.TimeRange) {
	i.bounds = tr
	i.internal.SetBounds(tr)
	i.seekReset(i.bounds.End)
}

func (i *Iterator) Bounds() telem.TimeRange { return i.bounds }

func (i *Iterator) Value() channel.Frame { return i.frame }

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
		i.seekReset(ts)
	} else {
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
		i.seekReset(ts)
	} else {
		i.seekReset(i.internal.TimeRange().BoundBy(i.bounds).Start)
	}
	return ok
}

func (i *Iterator) Next(ctx context.Context, span telem.TimeSpan) (ok bool) {
	if i.closed {
		i.err = errIteratorClosed
		return false
	}
	if i.atEnd() {
		i.reset(i.bounds.End.SpanRange(0))
		return
	}
	i.reset(i.view.End.SpanRange(span).BoundBy(i.bounds))
	if i.view.Span().IsZero() || i.view.End.BeforeEq(i.internal.TimeRange().Start) {
		return
	}
	i.accumulate(ctx)
	if i.satisfied() || i.err != nil {
		return i.Valid()
	}
	for i.internal.Next() && i.accumulate(ctx) && !i.satisfied() {
	}
	return i.Valid()
}

func (i *Iterator) Prev(ctx context.Context, span telem.TimeSpan) (ok bool) {
	if i.closed {
		i.err = errIteratorClosed
		return false
	}
	if i.atStart() {
		i.reset(i.bounds.Start.SpanRange(0))
		return
	}
	i.reset(i.view.Start.SpanRange(-1 * span).BoundBy(i.bounds))
	if i.view.Span().IsZero() || i.view.Start.AfterEq(i.internal.TimeRange().End) {
		return
	}
	i.accumulate(ctx)
	if i.satisfied() || i.err != nil {
		return i.Valid()
	}
	for i.internal.Prev() && i.accumulate(ctx) && !i.satisfied() {
	}
	return i.Valid()
}

func (i *Iterator) Len() int64 { return i.frame.Len() }

func (i *Iterator) Error() error {
	wrap := channel.NewErrorWrapper(i.Channel)
	return wrap(i.err)
}

func (i *Iterator) Valid() bool { return i.partiallySatisfied() && i.err == nil }

func (i *Iterator) Close() (err error) {
	if i.closed {
		return nil
	}
	i.closed = true
	wrap := channel.NewErrorWrapper(i.Channel)
	return wrap(i.internal.Close())
}

func (i *Iterator) getOrBuildOffsetTable(ctx context.Context) (*offsetTable, error) {
	domainIdx := i.internal.Position()
	if t, ok := i.cache.get(domainIdx); ok {
		return t, nil
	}
	r, err := i.internal.OpenReader(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { err = errors.Combine(err, r.Close()) }()
	t, err := buildOffsetTable(r, i.internal.Size())
	if err != nil {
		return nil, err
	}
	i.cache.set(domainIdx, t)
	return t, nil
}

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
	endApprox, err := i.approximateEnd(ctx)
	if err != nil {
		return 0, align, 0, err
	}

	table, err := i.getOrBuildOffsetTable(ctx)
	if err != nil {
		return 0, align, 0, err
	}

	startSample := startApprox.Upper
	if !startApprox.Exact() && !startApprox.StartExact {
		if startApprox.EndExact {
			startSample = startApprox.Lower
		} else {
			startSample = (startApprox.Lower + startApprox.Upper) / 2
		}
	}

	endSample := endApprox.Upper
	if !endApprox.Exact() && !endApprox.StartExact {
		if endApprox.EndExact {
			endSample = endApprox.Lower
		} else {
			endSample = (endApprox.Lower + endApprox.Upper) / 2
		}
	}

	var startOffset, endOffset telem.Size
	if startSample >= 0 && startSample < table.sampleCount {
		startOffset = table.byteOffsetAt(startSample)
	}
	if endSample >= table.sampleCount {
		endOffset = telem.Size(i.internal.Size())
	} else if endSample >= 0 {
		endOffset = table.byteOffsetAt(endSample)
	}

	size := endOffset - startOffset
	return startOffset, align, size, nil
}

func (i *Iterator) approximateStart(ctx context.Context) (
	index.DistanceApproximation,
	telem.Alignment,
	error,
) {
	target := i.internal.TimeRange().Start.SpanRange(0)
	if i.internal.TimeRange().Start.Before(i.view.Start) {
		target.End = i.view.Start
	}
	return i.idx.Distance(ctx, target, index.MustBeContinuous)
}

func (i *Iterator) approximateEnd(ctx context.Context) (endApprox index.DistanceApproximation, err error) {
	table, tableErr := i.getOrBuildOffsetTable(ctx)
	if tableErr != nil {
		return endApprox, tableErr
	}
	endApprox.Approximation = index.Exactly(table.sampleCount)
	if i.internal.TimeRange().End.After(i.view.End) {
		target := i.internal.TimeRange().Start.Range(i.view.End)
		endApprox, _, err = i.idx.Distance(ctx, target, index.MustBeContinuous)
	}
	return
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
	i.frame = channel.Frame{}
	i.view = nextView
}

func (i *Iterator) seekReset(ts telem.TimeStamp) {
	i.reset(ts.SpanRange(0))
	i.err = nil
}

func (i *Iterator) atStart() bool { return i.view.Start == i.bounds.Start }

func (i *Iterator) atEnd() bool { return i.view.End == i.bounds.End }
