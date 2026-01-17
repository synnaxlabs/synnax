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

	"github.com/synnaxlabs/cesium/internal/resource"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

const AutoSpan = unary.AutoSpan

var errIteratorClosed = resource.NewClosedError("cesium.iterator")

type Iterator struct {
	inlet    confluence.Inlet[IteratorRequest]
	outlet   confluence.Outlet[IteratorResponse]
	frame    Frame
	shutdown context.CancelFunc
	wg       signal.WaitGroup
	logger   *zap.Logger
	closed   bool
}

func wrapStreamIterator(internal *streamIterator) *Iterator {
	ctx, cancel := signal.Isolated()
	req, res := confluence.Attach(internal, 1)
	internal.Flow(ctx, confluence.RecoverWithErrOnPanic())
	return &Iterator{
		inlet:    req,
		outlet:   res,
		shutdown: cancel,
		wg:       ctx,
	}
}

// Next reads all data occupying the next span of time, returning true if the iterator
// has not been exhausted and has not accumulated an error. Note: If the internal
// iterators have different views, then they will each read the next span of time,
// ending at different times. For example, if the iterator on channel 1 has view [00:01,
// 00:02) while the iterator on channel 2 has view [00:03, 00:04), then they will read
// [00:02, 00:07) and [00:04, 00:09), respectively, after a call to Next(5).
func (i *Iterator) Next(span telem.TimeSpan) bool {
	return i.exec(IteratorRequest{Command: IteratorCommandNext, Span: span})
}

// Prev implements Iterator.
func (i *Iterator) Prev(span telem.TimeSpan) bool {
	return i.exec(IteratorRequest{Command: IteratorCommandPrev, Span: span})
}

// SeekFirst implements Iterator.
func (i *Iterator) SeekFirst() bool {
	return i.exec(IteratorRequest{Command: IteratorCommandSeekFirst})
}

// SeekLast implements Iterator.
func (i *Iterator) SeekLast() bool {
	return i.exec(IteratorRequest{Command: IteratorCommandSeekLast})
}

// SeekLE implements Iterator.
func (i *Iterator) SeekLE(ts telem.TimeStamp) bool {
	return i.exec(IteratorRequest{Command: IterCommandSeekLE, Stamp: ts})
}

// SeekGE implements Iterator.
func (i *Iterator) SeekGE(ts telem.TimeStamp) bool {
	return i.exec(IteratorRequest{Command: IteratorCommandSeekGE, Stamp: ts})
}

// Error implements Iterator.
func (i *Iterator) Error() error {
	_, err := i.execErr(IteratorRequest{Command: IteratorCommandError})
	return err
}

// Valid implements Iterator.
func (i *Iterator) Valid() bool {
	ok, _ := i.execErr(IteratorRequest{Command: IteratorCommandValid})
	return ok
}

// SetBounds sets the iterator's bounds. The iterator is invalidated, and will not be
// valid until a seeking call is made.
func (i *Iterator) SetBounds(bounds telem.TimeRange) {
	i.exec(IteratorRequest{Command: IteratorCommandSetBounds, Bounds: bounds})
}

// Value implements Iterator.
func (i *Iterator) Value() Frame { return i.frame }

// Close implements Iterator.
func (i *Iterator) Close() error {
	if i.closed {
		return nil
	}
	i.closed = true
	i.inlet.Close()
	err := i.wg.Wait()
	i.shutdown()
	return err
}

func (i *Iterator) exec(req IteratorRequest) bool {
	ok, _ := i.execErr(req)
	return ok
}

func (i *Iterator) execErr(req IteratorRequest) (bool, error) {
	if i.closed {
		return false, errIteratorClosed
	}
	i.frame = Frame{}
	i.inlet.Inlet() <- req
	for res := range i.outlet.Outlet() {
		if res.Variant == IteratorResponseVariantAck {
			return res.Ack, res.Err
		}
		i.frame = i.frame.Extend(res.Frame)
	}
	i.logger.DPanic("unexpected early closure of response stream")
	return false, nil
}
