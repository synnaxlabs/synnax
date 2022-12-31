// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type StreamIterator = confluence.Segment[Request, Response]

type Iterator interface {
	// Next reads all channelClient data occupying the next span of time. Returns true
	// if the current IteratorServer.View is pointing to any valid segments.
	Next(span telem.TimeSpan) bool
	// Prev reads all channelClient data occupying the previous span of time. Returns true
	// if the current IteratorServer.View is pointing to any valid segments.
	Prev(span telem.TimeSpan) bool
	// SeekFirst seeks the iterator the start of the iterator range.
	// Returns true if the current IteratorServer.View is pointing to any valid segments.
	SeekFirst() bool
	// SeekLast seeks the iterator the end of the iterator range.
	// Returns true if the current IteratorServer.View is pointing to any valid segments.
	SeekLast() bool
	// SeekLE seeks the iterator to the first whose timestamp is less than or equal
	// to the given timestamp. Returns true if the current IteratorServer.View is pointing
	// to any valid segments.
	SeekLE(t telem.TimeStamp) bool
	// SeekGE seeks the iterator to the first whose timestamp is greater than the
	// given timestamp. Returns true if the current IteratorServer.View is pointing to
	// any valid segments.
	SeekGE(t telem.TimeStamp) bool
	// Close closes the Iterator, ensuring that all in-progress reads complete
	// before closing the Source outlet. All iterators must be Closed, or the
	// distribution layer will panic.
	Close() error
	// Valid returns true if the iterator is pointing at valid data and is error free.
	Valid() bool
	// Error returns any errors accumulated during the iterators lifetime.
	Error() error
	// SetBounds sets the lower and upper bounds of the iterator.
	SetBounds(bounds telem.TimeRange) bool
	Value() core.Frame
}

type iterator struct {
	requests  confluence.Inlet[Request]
	responses confluence.Outlet[Response]
	shutdown  context.CancelFunc
	wg        signal.WaitGroup
	value     []Response
}

// Next implements Iterator.
func (i *iterator) Next(span telem.TimeSpan) bool {
	i.value = nil
	return i.exec(Request{Command: Next, Span: span})
}

// Prev implements Iterator.
func (i *iterator) Prev(span telem.TimeSpan) bool {
	i.value = nil
	return i.exec(Request{Command: Prev, Span: span})
}

// SeekFirst implements Iterator.
func (i *iterator) SeekFirst() bool {
	i.value = nil
	return i.exec(Request{Command: SeekFirst})
}

// SeekLast implements Iterator.
func (i *iterator) SeekLast() bool {
	i.value = nil
	return i.exec(Request{Command: SeekLast})
}

// SeekLE implements Iterator.
func (i *iterator) SeekLE(stamp telem.TimeStamp) bool {
	i.value = nil
	return i.exec(Request{Command: SeekLE, Stamp: stamp})
}

// SeekGE implements Iterator.
func (i *iterator) SeekGE(stamp telem.TimeStamp) bool {
	i.value = nil
	return i.exec(Request{Command: SeekGE, Stamp: stamp})
}

// Valid implements Iterator.
func (i *iterator) Valid() bool {
	return i.exec(Request{Command: Valid})
}

// Error implements Iterator.
func (i *iterator) Error() error {
	_, err := i.execErr(Request{Command: Error})
	return err
}

// Close implements Iterator.
func (i *iterator) Close() error {
	defer i.shutdown()
	i.requests.Close()
	return i.wg.Wait()
}

func (i *iterator) SetBounds(bounds telem.TimeRange) bool {
	return i.exec(Request{Command: SetBounds, Bounds: bounds})
}

func (i *iterator) Value() core.Frame {
	frames := make([]core.Frame, len(i.value))
	for i, v := range i.value {
		frames[i] = v.Frame
	}
	return core.MergeFrames(frames)
}

func (i *iterator) exec(req Request) bool {
	ok, _ := i.execErr(req)
	return ok
}

func (i *iterator) execErr(req Request) (bool, error) {
	i.requests.Inlet() <- req
	for res := range i.responses.Outlet() {
		if res.Variant == AckResponse {
			return res.Ack, res.Err
		}
		i.value = append(i.value, res)
	}
	return false, nil
}
