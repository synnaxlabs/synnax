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
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/ranger"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
	"io"
)

type Iterator struct {
	IteratorConfig
	Channel  core.Channel
	internal *ranger.Iterator
	view     telem.TimeRange
	frame    core.Frame
	idx      index.Index
	bounds   telem.TimeRange
	err      error
	logger   *zap.Logger
}

const AutoSpan telem.TimeSpan = -1

func (i *Iterator) SetBounds(tr telem.TimeRange) {
	i.bounds = tr
	i.internal.SetBounds(tr)
}

func (i *Iterator) Bounds() telem.TimeRange { return i.bounds }

func (i *Iterator) Value() core.Frame { return i.frame }

func (i *Iterator) View() telem.TimeRange { return i.view }

func (i *Iterator) SeekFirst() (ok bool) {
	i.log("unary seek first")
	defer func() {
		i.log("unary seek first done", zap.Bool("ok", ok))
	}()
	ok = i.internal.SeekFirst()
	i.seekReset(i.internal.Range().Start)
	return ok
}

func (i *Iterator) SeekLast() (ok bool) {
	i.log("unary seek first")
	defer func() {
		i.log("unary seek first done", zap.Bool("ok", ok))
	}()
	ok = i.internal.SeekLast()
	i.seekReset(i.internal.Range().End)
	return
}

func (i *Iterator) SeekLE(ts telem.TimeStamp) (ok bool) {
	i.log("unary seek le", zap.Stringer("ts", ts))
	defer func() {
		i.log("unary seek le done", zap.Bool("ok", ok))
	}()
	i.seekReset(ts)
	ok = i.internal.SeekLE(ts)
	return
}

func (i *Iterator) SeekGE(ts telem.TimeStamp) (ok bool) {
	i.log("unary seek ge", zap.Stringer("ts", ts))
	defer func() {
		i.log("unary seek ge done", zap.Stringer("ts", ts))
	}()
	i.seekReset(ts)
	ok = i.internal.SeekGE(ts)
	return
}

func (i *Iterator) Next(span telem.TimeSpan) (ok bool) {
	i.log("unary next", zap.Stringer("span", span))
	defer func() {
		ok = i.Valid()
		i.log("unary next done", zap.Bool("ok", ok))
	}()

	if i.atEnd() {
		i.reset(i.bounds.End.SpanRange(0))
		return
	}

	if span == AutoSpan {
		return i.autoNext()
	}

	i.reset(i.view.End.SpanRange(span).BoundBy(i.bounds))

	if i.view.IsZero() {
		return
	}

	i.accumulate()
	if i.satisfied() || i.err != nil {
		return
	}

	for i.internal.Next() && i.accumulate() {
	}
	return
}

func (i *Iterator) autoNext() bool {
	i.view.Start = i.view.End
	endApprox, err := i.idx.Stamp(i.view.Start, i.IteratorConfig.AutoChunkSize, false)
	if err != nil {
		i.err = err
		return false
	}
	if endApprox.Lower.After(i.bounds.End) {
		return i.Next(i.view.Start.Span(i.bounds.End))
	}
	i.view.End = endApprox.Lower
	i.reset(i.view.BoundBy(i.bounds))

	nRemaining := i.IteratorConfig.AutoChunkSize
	for {
		if !i.internal.Range().OverlapsWith(i.view) {
			if !i.internal.Next() {
				return false
			}
			continue
		}
		startApprox, err := i.approximateStart()
		if err != nil {
			i.err = err
			return false
		}
		startOffset := i.Channel.DataType.Density().Size(startApprox.Upper)
		arr, n, err := i.read(startOffset, i.Channel.DataType.Density().Size(nRemaining))
		nRead := i.Channel.DataType.Density().SampleCount(telem.Size(n))
		nRemaining -= arr.Len()
		if err != nil && !errors.Is(err, io.EOF) {
			i.err = err
			return false
		}

		i.insert(arr)

		if nRead >= nRemaining || !i.internal.Next() {
			break
		}
	}

	return i.partiallySatisfied()
}

func (i *Iterator) Prev(span telem.TimeSpan) (ok bool) {
	i.log("unary prev", zap.Stringer("span", span))
	defer func() {
		ok = i.Valid()
		i.log("unary prev done", zap.Stringer("span", span), zap.Bool("ok", ok))
	}()

	if i.atStart() {
		i.reset(i.bounds.Start.SpanRange(0))
		return
	}

	i.reset(i.view.Start.SpanRange(span).BoundBy(i.bounds))

	if i.view.IsZero() {
		return
	}

	i.accumulate()
	if i.satisfied() || i.err != nil {
		return
	}

	for i.internal.Prev() && i.accumulate() {
	}
	return
}

func (i *Iterator) Len() (l int64) {
	for _, arr := range i.frame.Arrays {
		l += arr.Len()
	}
	return
}

func (i *Iterator) Error() error { return i.err }

func (i *Iterator) Valid() bool { return i.partiallySatisfied() && i.err == nil }

func (i *Iterator) Close() error {
	return i.internal.Close()
}

func (i *Iterator) accumulate() bool {
	if !i.internal.Range().OverlapsWith(i.view) {
		return false
	}
	start, size, err := i.sliceRange()
	if err != nil {
		i.err = err
		return false
	}
	arr, _, err := i.read(start, size)
	if err != nil && !errors.Is(err, io.EOF) {
		i.err = err
		return false
	}
	i.insert(arr)
	return true
}

func (i *Iterator) insert(arr telem.Array) {
	if arr.Len() == 0 {
		return
	}
	if len(i.frame.Arrays) == 0 || i.frame.Arrays[len(i.frame.Arrays)-1].TimeRange.End.Before(arr.TimeRange.Start) {
		i.frame = i.frame.Append(i.Channel.Key, arr)
	} else {
		i.frame = i.frame.Prepend(i.Channel.Key, arr)
	}
}

func (i *Iterator) read(start telem.Offset, size telem.Size) (arr telem.Array, n int, err error) {
	arr.DataType = i.Channel.DataType
	arr.TimeRange = i.internal.Range().BoundBy(i.view)
	arr.Data = make([]byte, size)
	r, err := i.internal.NewReader()
	if err != nil {
		return
	}
	n, err = r.ReadAt(arr.Data, int64(start))
	if err != nil && !errors.Is(err, io.EOF) {
		return
	}
	if n < len(arr.Data) {
		arr.Data = arr.Data[:n]
	}
	return
}

func (i *Iterator) sliceRange() (telem.Offset, telem.Size, error) {
	startApprox, err := i.approximateStart()
	if err != nil {
		return 0, 0, err
	}
	endApprox, err := i.approximateEnd()
	if err != nil {
		return 0, 0, err
	}
	startOffset := i.Channel.DataType.Density().Size(startApprox.Upper)
	size := i.Channel.DataType.Density().Size(endApprox.Upper) - startOffset
	return startOffset, size, nil
}

// approximateStart approximates the number of samples between the start of the current
// range and the start of the current iterator view. If the start of the current view is
// before the start of the range, the returned value will be zero.
func (i *Iterator) approximateStart() (startApprox index.DistanceApproximation, err error) {
	if i.internal.Range().Start.Before(i.view.Start) {
		target := i.internal.Range().Start.Range(i.view.Start)
		startApprox, err = i.idx.Distance(target, true)
	}
	return
}

// approximateEnd approximates the number of samples between the start of the current
// range and the end of the current iterator view. If the end of the current view is
// after the end of the range, the returned value will be the number of samples in the
// range.
func (i *Iterator) approximateEnd() (endApprox index.DistanceApproximation, err error) {
	endApprox = index.Exactly(i.Channel.DataType.Density().SampleCount(telem.Size(i.internal.Len())))
	if i.internal.Range().End.After(i.view.End) {
		target := i.internal.Range().Start.Range(i.view.End)
		endApprox, err = i.idx.Distance(target, true)
	}
	return
}

func (i *Iterator) satisfied() bool {
	if !i.partiallySatisfied() {
		return false
	}
	start := i.frame.Arrays[0].TimeRange.Start
	end := i.frame.Arrays[len(i.frame.Arrays)-1].TimeRange.End
	return i.view == start.Range(end)
}

func (i *Iterator) partiallySatisfied() bool { return len(i.frame.Arrays) > 0 }

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

func (i *Iterator) log(msg string, fields ...zap.Field) {
	fields = append(
		fields,
		zap.String("channel", i.Channel.Key),
		zap.Stringer("view", i.view),
		zap.Error(i.err),
	)
	i.logger.Debug(msg, fields...)
}
