// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type (
	StreamIterator = framer.StreamIterator
	Request        = framer.IteratorRequest
	Response       = framer.IteratorResponse
)

const (
	AutoSpan            = iterator.AutoSpan
	CommandSeekFirst    = iterator.CommandSeekFirst
	CommandSeekLast     = iterator.CommandSeekLast
	CommandSeekLE       = iterator.CommandSeekLE
	CommandSeekGE       = iterator.CommandSeekGE
	CommandNext         = iterator.CommandNext
	CommandPrev         = iterator.CommandPrev
	CommandSetBounds    = iterator.CommandSetBounds
	ResponseVariantAck  = iterator.ResponseVariantAck
	ResponseVariantData = iterator.ResponseVariantData
	CommandError        = iterator.CommandError
	CommandValid        = iterator.CommandValid
)

type responseSegment = confluence.Segment[Response, Response]

type Iterator struct {
	requests    confluence.Inlet[Request]
	responses   confluence.Outlet[Response]
	shutdown    context.CancelFunc
	wg          signal.WaitGroup
	value       []Response
	valueFrames []frame.Frame
}

// Next reads all channel data occupying the next span of time. Returns true
// if the current IteratorServer.View is pointing to any valid segments.
func (i *Iterator) Next(span telem.TimeSpan) bool {
	i.value = i.value[:0]
	return i.exec(Request{Command: CommandNext, Span: span})
}

// Prev reads all channel data occupying the previous span of time. Returns true
// if the current IteratorServer.View is pointing to any valid segments.
func (i *Iterator) Prev(span telem.TimeSpan) bool {
	i.value = i.value[:0]
	return i.exec(Request{Command: CommandPrev, Span: span})
}

// SeekFirst seeks the Iterator the start of the Iterator range.
// Returns true if the current IteratorServer.View is pointing to any valid segments.
func (i *Iterator) SeekFirst() bool {
	i.value = i.value[:0]
	return i.exec(Request{Command: CommandSeekFirst})
}

// SeekLast seeks the Iterator the end of the Iterator range.
// Returns true if the current IteratorServer.View is pointing to any valid segments.
func (i *Iterator) SeekLast() bool {
	i.value = i.value[:0]
	return i.exec(Request{Command: CommandSeekLast})
}

// SeekLE seeks the Iterator to the first whose timestamp is less than or equal
// to the given timestamp. Returns true if the current IteratorServer.View is pointing
// to any valid segments.
func (i *Iterator) SeekLE(stamp telem.TimeStamp) bool {
	i.value = i.value[:0]
	return i.exec(Request{Command: CommandSeekLE, Stamp: stamp})
}

// SeekGE seeks the Iterator to the first whose timestamp is greater than the
// given timestamp. Returns true if the current IteratorServer.View is pointing to
// any valid segments.
func (i *Iterator) SeekGE(stamp telem.TimeStamp) bool {
	i.value = i.value[:0]
	return i.exec(Request{Command: CommandSeekGE, Stamp: stamp})
}

// Valid returns true if the Iterator is pointing at valid data and is error free.
func (i *Iterator) Valid() bool {
	return i.exec(Request{Command: CommandValid})
}

// Error returns any errors accumulated during the iterators lifetime.
func (i *Iterator) Error() error {
	_, err := i.execErr(Request{Command: CommandError})
	return err
}

// Close closes the Iterator, ensuring that all in-progress reads complete
// before closing the Source outlet. All iterators must be Closed, or the
// distribution layer will panic.
func (i *Iterator) Close() error {
	defer i.shutdown()
	i.requests.Close()
	return i.wg.Wait()
}

// SetBounds sets the lower and upper bounds of the Iterator.
func (i *Iterator) SetBounds(bounds telem.TimeRange) bool {
	return i.exec(Request{Command: CommandSetBounds, Bounds: bounds})
}

func (i *Iterator) Value() frame.Frame {
	if cap(i.valueFrames) < len(i.value) {
		i.valueFrames = make([]frame.Frame, len(i.value))
	} else {
		i.valueFrames = i.valueFrames[:len(i.value)]
	}
	for idx, v := range i.value {
		i.valueFrames[idx] = v.Frame
	}
	return frame.Merge(i.valueFrames)
}

func (i *Iterator) exec(req Request) bool {
	ok, _ := i.execErr(req)
	return ok
}

func (i *Iterator) execErr(req Request) (bool, error) {
	i.requests.Inlet() <- req
	for res := range i.responses.Outlet() {
		if res.Variant == ResponseVariantAck {
			return res.Ack, res.Error
		}
		i.value = append(i.value, res)
	}
	return false, nil
}
