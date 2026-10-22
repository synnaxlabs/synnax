// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package virtual

import (
	"context"
	"sync/atomic"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/x/telem"
)

// IteratorConfig configures an Iterator opened on a DB.
type IteratorConfig struct {
	// Bounds is the time range the iterator reads within.
	Bounds telem.TimeRange
}

var ErrIteratorClosed = resource.NewClosedError("virtual.iterator")

// Iterator reads the DB's retained sample with the same movement contract as a unary
// iterator. A seek snapshots the sample, so a write during iteration is seen by the
// next seek. Not safe for concurrent use.
type Iterator struct {
	lastSample *atomic.Pointer[telem.Series]
	wrapError  func(error) error
	Channel    channel.Channel
	sample     *telem.Series
	frame      channel.Frame
	view       telem.TimeRange
	bounds     telem.TimeRange
	err        error
	closed     bool
}

// OpenIterator opens an Iterator over the DB's retained sample.
func (db *DB) OpenIterator(cfg IteratorConfig) (*Iterator, error) {
	if db.closed.Load() {
		return nil, db.wrapError(ErrDBClosed)
	}
	i := &Iterator{
		lastSample: db.lastSample,
		wrapError:  db.wrapError,
		Channel:    db.cfg.Channel,
	}
	i.SetBounds(cfg.Bounds)
	return i, nil
}

// SetBounds sets the iterator's bounds. The iterator is invalidated until a seek.
func (i *Iterator) SetBounds(tr telem.TimeRange) {
	i.bounds = tr
	i.seekReset(tr.End)
}

func (i *Iterator) Value() channel.Frame { return i.frame }

func (i *Iterator) SeekFirst(_ context.Context) bool {
	found, ok := i.seek()
	if !ok {
		return false
	}
	i.seekReset(found.BoundBy(i.bounds).Start)
	return true
}

func (i *Iterator) SeekLast(_ context.Context) bool {
	found, ok := i.seek()
	if !ok {
		return false
	}
	i.seekReset(found.BoundBy(i.bounds).End)
	return true
}

func (i *Iterator) SeekLE(_ context.Context, ts telem.TimeStamp) bool {
	found, ok := i.seek()
	if !ok || found.Start.After(ts) {
		return false
	}
	if found.ContainsStamp(ts) {
		i.seekReset(ts)
	} else {
		i.seekReset(found.BoundBy(i.bounds).End)
	}
	return true
}

func (i *Iterator) SeekGE(_ context.Context, ts telem.TimeStamp) bool {
	found, ok := i.seek()
	if !ok || found.Start.Before(ts) {
		return false
	}
	if found.ContainsStamp(ts) {
		i.seekReset(ts)
	} else {
		i.seekReset(found.BoundBy(i.bounds).Start)
	}
	return true
}

// Next moves the view forward by span and fills the frame when the retained sample
// falls inside it. AutoSpan moves to the end of the bounds.
func (i *Iterator) Next(_ context.Context, span telem.TimeSpan) bool {
	if i.closed {
		i.err = ErrIteratorClosed
		return false
	}
	if i.view.End == i.bounds.End {
		i.move(i.bounds.End.SpanRange(0))
		return false
	}
	if span == channel.AutoSpan {
		i.move(i.view.End.Range(i.bounds.End))
	} else {
		i.move(i.view.End.SpanRange(span).BoundBy(i.bounds))
	}
	return i.Valid()
}

// Prev moves the view backward by span and fills the frame when the retained sample
// falls inside it. AutoSpan moves to the start of the bounds.
func (i *Iterator) Prev(_ context.Context, span telem.TimeSpan) bool {
	if i.closed {
		i.err = ErrIteratorClosed
		return false
	}
	if i.view.Start == i.bounds.Start {
		i.move(i.bounds.Start.SpanRange(0))
		return false
	}
	if span == channel.AutoSpan {
		i.move(i.bounds.Start.Range(i.view.Start))
	} else {
		i.move(i.view.Start.SpanRange(-1 * span).BoundBy(i.bounds))
	}
	return i.Valid()
}

func (i *Iterator) Valid() bool { return i.frame.HasData() && i.err == nil }

func (i *Iterator) Error() error { return i.wrapError(i.err) }

// Close closes the iterator. Closing a closed iterator returns nil.
func (i *Iterator) Close() error {
	if i.closed {
		return nil
	}
	i.closed = true
	return i.Error()
}

// seek snapshots the retained sample and reports its range when the bounds hold it.
// A seek on a closed iterator records the error, as the unary iterator does.
func (i *Iterator) seek() (telem.TimeRange, bool) {
	if i.closed {
		i.err = ErrIteratorClosed
		return telem.TimeRangeZero, false
	}
	i.sample = i.lastSample.Load()
	if i.sample == nil || !i.bounds.ContainsStamp(i.sample.TimeRange.Start) {
		i.seekReset(i.bounds.Start)
		return telem.TimeRangeZero, false
	}
	return i.sample.TimeRange, true
}

func (i *Iterator) move(view telem.TimeRange) {
	i.view = view
	i.frame = channel.Frame{}
	if i.sample != nil && view.ContainsStamp(i.sample.TimeRange.Start) {
		i.frame = telem.UnaryFrame(i.Channel.Key, *i.sample)
	}
}

func (i *Iterator) seekReset(ts telem.TimeStamp) {
	i.move(ts.SpanRange(0))
	i.err = nil
}
