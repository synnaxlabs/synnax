// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/index"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

const AutoSpan = unary.AutoSpan

var ErrIteratorClosed = resource.NewClosedError("cesium.iterator")

// IteratorConfig is the configuration for opening an iterator :). See the fields for
// more information.
type IteratorConfig struct {
	// Channels is a list of channels to iterate over.
	Channels []channel.Key
	// Bounds sets the time range to iterator over. The lower bound is inclusive, while
	// the upper bound is exclusive.
	Bounds telem.TimeRange
	// AutoChunkSize sets the default chunk size to iterator over when sending a Next()
	// or Prev() request it IteratorAutoSpan as the span. It must not be negative.
	AutoChunkSize int64
	// Reduction reduces the samples of each channel the iterator reads. A stride keeps
	// the samples whose alignment is a multiple of its factor and never reads the rest.
	// A limit reduces each channel in Bounds to about PointLimit points. Either way,
	// every channel on one index keeps the same alignments.
	Reduction telem.Reduction
}

// Iterator reads the channels of a DB in time order. It is not safe for concurrent use.
type Iterator struct {
	internal []*unary.Iterator
	// cache is shared by every iterator in internal, so channels on one index resolve
	// each index lookup once. It is reset before each call and each chunk, because
	// writes can change the index between them.
	cache *index.Cache
	// limited is true when the read has a limit reduction. A limited read is never
	// chunked, because a chunk edge would cut its groups.
	limited   bool
	chunkSize int64
	frame     Frame
	ins       alamos.Instrumentation
	// stopped is true for each iterator in internal whose read error is logged.
	stopped []bool
	closed  bool
}

// Next reads all data occupying the next span of time, returning true if any channel
// read data. A channel whose read fails reads nothing more until a seek or SetBounds,
// and the iterator logs its error once. Note: If the internal iterators have different
// views, then they will each read the next span of time, ending at different times. For
// example, if the iterator on channel 1 has view [00:01, 00:02) while the iterator on
// channel 2 has view [00:03, 00:04), then they will read [00:02, 00:07) and [00:04,
// 00:09), respectively, after a call to Next(5).
func (i *Iterator) Next(span telem.TimeSpan) bool {
	ok, err := i.NextChunks(context.Background(), span, i.collect)
	return ok && err == nil
}

// Prev reads all data occupying the previous span of time, as Next does forward.
func (i *Iterator) Prev(span telem.TimeSpan) bool {
	ok, err := i.PrevChunks(context.Background(), span, i.collect)
	return ok && err == nil
}

// NextChunks reads the next span of time as Next does, and hands the data to emit in
// chunks, in time order, instead of keeping it as Value. WithChunkSize and
// AutoChunkSize set the size of a chunk. It returns whether any channel moved, and the
// error of emit or of an index lookup.
func (i *Iterator) NextChunks(
	ctx context.Context,
	span telem.TimeSpan,
	emit func(Frame) error,
) (bool, error) {
	return i.read(ctx, span, false, emit)
}

// PrevChunks reads the previous span of time as Prev does, and hands the data to emit
// as NextChunks does.
func (i *Iterator) PrevChunks(
	ctx context.Context,
	span telem.TimeSpan,
	emit func(Frame) error,
) (bool, error) {
	return i.read(ctx, span, true, emit)
}

// SeekFirst moves the iterator to the start of its bounds, returning true if any
// channel holds data there.
func (i *Iterator) SeekFirst() bool {
	ctx := context.Background()
	return i.seek(func(u *unary.Iterator) bool { return u.SeekFirst(ctx) })
}

// SeekLast moves the iterator to the end of its bounds, returning true if any channel
// holds data there.
func (i *Iterator) SeekLast() bool {
	ctx := context.Background()
	return i.seek(func(u *unary.Iterator) bool { return u.SeekLast(ctx) })
}

// SeekLE moves the iterator to the last sample at or before ts.
func (i *Iterator) SeekLE(ts telem.TimeStamp) bool {
	ctx := context.Background()
	return i.seek(func(u *unary.Iterator) bool { return u.SeekLE(ctx, ts) })
}

// SeekGE moves the iterator to the first sample at or after ts.
func (i *Iterator) SeekGE(ts telem.TimeStamp) bool {
	ctx := context.Background()
	return i.seek(func(u *unary.Iterator) bool { return u.SeekGE(ctx, ts) })
}

// Valid returns true if any channel's iterator points to data.
func (i *Iterator) Valid() bool {
	return i.each(func(u *unary.Iterator) bool { return u.Valid() })
}

// Error returns the first error of a channel's iterator, or ErrIteratorClosed once the
// iterator is closed.
func (i *Iterator) Error() error {
	if i.closed {
		return ErrIteratorClosed
	}
	i.frame = Frame{}
	for _, u := range i.internal {
		if err := u.Error(); err != nil {
			return err
		}
	}
	return nil
}

// SetBounds sets the iterator's bounds. The iterator is invalidated, and will not be
// valid until a seeking call is made.
func (i *Iterator) SetBounds(bounds telem.TimeRange) {
	i.seek(func(u *unary.Iterator) bool { u.SetBounds(bounds); return true })
}

// Value returns the data that the last Next or Prev read.
func (i *Iterator) Value() Frame { return i.frame }

// Close closes the iterator and returns the error that stopped it, unless a seek or
// SetBounds has cleared that error since. Closing a closed iterator returns nil.
func (i *Iterator) Close() error {
	if i.closed {
		return nil
	}
	i.closed = true
	var err error
	for _, u := range i.internal {
		err = errors.Join(err, u.Close())
	}
	return err
}

// each runs f on the iterator of each channel and returns true if f returned true for
// any of them. It returns false once the iterator is closed.
func (i *Iterator) each(f func(*unary.Iterator) bool) (ok bool) {
	if i.closed {
		return false
	}
	i.frame = Frame{}
	i.cache.Reset()
	for _, u := range i.internal {
		if f(u) {
			ok = true
		}
	}
	return ok
}

// seek runs each with f, which clears the error of each channel's iterator.
func (i *Iterator) seek(f func(*unary.Iterator) bool) bool {
	clear(i.stopped)
	return i.each(f)
}

// report logs the read error of the iterator at index j of internal, unless it is
// already logged.
func (i *Iterator) report(j int) {
	u := i.internal[j]
	if i.stopped[j] {
		return
	}
	if err := u.Error(); err != nil {
		i.stopped[j] = true
		i.ins.L.Error(
			"read of a channel failed",
			zap.Stringer("channel", u.Channel),
			zap.Stringer("time_range", u.View()),
			zap.Error(err),
		)
	}
}

func (i *Iterator) collect(fr Frame) error {
	i.frame = i.frame.Extend(fr)
	return nil
}

// read runs a Next, or a Prev when prev is true, and hands its data to emit. A limited
// read or a read of AutoSpan moves each channel's iterator once and emits one frame.
func (i *Iterator) read(
	ctx context.Context,
	span telem.TimeSpan,
	prev bool,
	emit func(Frame) error,
) (bool, error) {
	if i.closed {
		return false, ErrIteratorClosed
	}
	i.frame = Frame{}
	i.cache.Reset()
	if !i.limited && span != AutoSpan {
		return i.readInChunks(ctx, span, prev, emit)
	}
	var (
		fr Frame
		ok bool
	)
	for j, u := range i.internal {
		var moved bool
		if prev {
			moved = u.Prev(ctx, span)
		} else {
			moved = u.Next(ctx, span)
		}
		if !moved {
			i.report(j)
			continue
		}
		ok = true
		fr = fr.Extend(u.Value())
	}
	if err := emitFrame(emit, fr); err != nil {
		return false, err
	}
	return ok, nil
}

// readInChunks runs a Next or Prev with a fixed span. It reads the range of each
// iterator forward in chunks that each hold at most chunkSize samples summed over the
// channels, or more for a run of equal timestamps. Each chunk holds every channel's
// data up to the end of the chunk.
func (i *Iterator) readInChunks(
	ctx context.Context,
	span telem.TimeSpan,
	prev bool,
	emit func(Frame) error,
) (ok bool, err error) {
	var (
		ends = make([]telem.TimeStamp, len(i.internal))
		last telem.TimeStamp
		n    = max(1, i.chunkSize/int64(max(1, len(i.internal))))
	)
	for j, u := range i.internal {
		tr := u.View().End.SpanRange(span)
		if prev {
			tr = u.View().Start.SpanRange(-span)
		}
		tr = tr.BoundBy(u.Bounds())
		u.MoveTo(ctx, tr.Start)
		ends[j] = tr.End
		last = max(last, tr.End)
	}
	for {
		i.cache.Reset()
		end := last
		probed := make(set.Set[channel.Key], len(i.internal))
		for j, u := range i.internal {
			idx := u.Channel.Index
			if u.Channel.IsIndex {
				idx = u.Channel.Key
			}
			if u.View().End >= ends[j] || probed.Contains(idx) {
				continue
			}
			probed.Add(idx)
			chunkEnd, err := u.EndAfter(ctx, n)
			if err != nil {
				return ok, err
			}
			end = min(end, chunkEnd)
		}
		var chunk Frame
		for j, u := range i.internal {
			stop := min(end, ends[j])
			if stop <= u.View().End {
				continue
			}
			if !u.Continue(ctx, stop) {
				i.report(j)
				continue
			}
			ok = true
			chunk = chunk.Extend(u.Value())
		}
		if err = emitFrame(emit, chunk); err != nil {
			return false, err
		}
		if end >= last {
			return ok, nil
		}
	}
}

// emitFrame hands fr to emit unless it holds no series.
func emitFrame(emit func(Frame) error, fr Frame) error {
	if fr.Empty() {
		return nil
	}
	return emit(fr)
}
