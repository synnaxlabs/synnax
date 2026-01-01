// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/validate"
)

// StreamIterator provides a streaming interface for iterating over a DB's segments in
// time order. StreamIterator provides the underlying functionality for Iterator, and
// has almost exactly the same semantics. The streaming interface is exposed as a
// confluence segment that can accept one input stream and one output stream.
//
// To read frames, issue an IteratorRequest to the StreamIterator's inlet. The
// StreamIterator will respond by sending one or more IteratorResponse messages to the
// outlet. All responses containing frame data will have a type of IteratorDataResponse
// and will contain one or more frames. The last response for any request will have a
// type of IteratorAckResponse and will contain the name of the command that was
// acknowledged, an incremented sequence number, and ack boolean indicating whether the
// command was successfully processed.
//
// To close the StreamIterator, simply close the inlet. The StreamIterator will ensure
// that all in-progress requests have been served before closing the outlet. The
// StreamIterator will return any accumulated err through the signal context provided to
// Flow.
type StreamIterator = confluence.Segment[IteratorRequest, IteratorResponse]

// IteratorResponseVariant is the type of the response an Iterator will return.
type IteratorResponseVariant uint8

const (
	// IteratorAckResponse is a response that indicates that an iteration request has
	// completed successfully.
	IteratorAckResponse IteratorResponseVariant = iota + 1
	// IteratorDataResponse is a response that indicates that an iteration request
	// returned data.
	IteratorDataResponse
)

// IteratorCommand is an enumeration of commands that can be sent to an Iterator.
type IteratorCommand uint8

const (
	// IterNext represents a call to Iterator.Next.
	IterNext IteratorCommand = iota + 1
	// IterPrev represents a call to Iterator.Prev.
	IterPrev
	// IterSeekFirst represents a call to Iterator.SeekFirst.
	IterSeekFirst
	// IterSeekLast represents a call to Iterator.SeekLast.
	IterSeekLast
	// IterSeekLE represents a call to Iterator.SeekLE.
	IterSeekLE
	// IterSeekGE represents a call to Iterator.SeekGE.
	IterSeekGE
	// IterValid represents a call to Iterator.Valid.
	IterValid
	// IterError represents a call to Iterator.Error.
	IterError
	// IterSetBounds represents a call to Iterator.SetBounds.
	IterSetBounds
)

var validateIteratorCommand = validate.NewInclusiveBoundsChecker(IterNext, IterSetBounds)

// HasOps returns true if the IteratorCommand has any associated on disk operations.
func (i IteratorCommand) HasOps() bool { return i <= IterPrev }

// IteratorRequest is issued to an StreamIterator asking it to read data from a DB.
//
//go:generate stringer -type=IteratorCommand
type IteratorRequest struct {
	// Command is the command to execute.
	Command IteratorCommand
	// Stamp should be set during a request to IterSeekLE or IterSeekGE.
	Stamp telem.TimeStamp
	// Span should be set during a request to IterNext or IterPrev.
	Span telem.TimeSpan
	// Bounds should be set during a request to IterSetBounds.
	Bounds telem.TimeRange
	// SeqNum is the sequence number of the request. This is used to match the request
	// with the response. Each request should increment the sequence number by 1.
	SeqNum int
}

// IteratorResponse is a response containing segments satisfying an iterator request as
// well as any errors encountered during the retrieval.
type IteratorResponse struct {
	// Variant is the type of response being issued.
	Variant IteratorResponseVariant
	// SeqNum is the corresponding sequence number of the request. This is used to match
	// the request with the response. Each request should increment the sequence number
	// by 1.
	SeqNum int
	// Command defines the command that the response relates to.
	Command IteratorCommand
	// Ack is only valid when the response type is IteratorAckResponse. It indicates
	// whether the command was successfully processed.
	Ack bool
	// Err is only set an IterError command is being responded to.
	Err error
	// Frame is the telemetry frame that was read from the DB. It is only set when the
	// response type is IteratorDataResponse.
	Frame Frame
}

type streamIterator struct {
	confluence.UnarySink[IteratorRequest]
	confluence.AbstractUnarySource[IteratorResponse]
	internal []*unary.Iterator
}

// IteratorConfig is the configuration for opening an iterator :). See the fields for
// more information.
type IteratorConfig struct {
	// Bounds sets the time range to iterator over. The lower bound is inclusive, while
	// the upper bound is exclusive.
	Bounds telem.TimeRange
	// Channels is a list of channels to iterate over.
	Channels []channel.Key
	// AutoChunkSize sets the default chunk size to iterator over when sending a Next()
	// or Prev() request it IteratorAutoSpan as the span.
	AutoChunkSize int64
}

// Flow implements the confluence.Segment interface.
func (s *streamIterator) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(s.Out)
	sCtx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return errors.Combine(s.close(), ctx.Err())
			case req, ok := <-s.In.Outlet():
				if !ok {
					return s.close()
				}
				ok, err := s.exec(ctx, req)
				s.Out.Inlet() <- IteratorResponse{
					Variant: IteratorAckResponse,
					Command: req.Command,
					SeqNum:  req.SeqNum,
					Ack:     ok,
					Err:     err,
				}
			}
		}
	}, o.Signal...)
}

func (s *streamIterator) exec(ctx context.Context, req IteratorRequest) (ok bool, err error) {
	if err := validateIteratorCommand(req.Command); err != nil {
		return false, err
	}
	switch req.Command {
	case IterNext:
		ok = s.execWithResponse(req.SeqNum, func(i *unary.Iterator) bool { return i.Next(ctx, req.Span) })
	case IterPrev:
		ok = s.execWithResponse(req.SeqNum, func(i *unary.Iterator) bool { return i.Prev(ctx, req.Span) })
	case IterSeekFirst:
		ok = s.execWithoutResponse(func(i *unary.Iterator) bool { return i.SeekFirst(ctx) })
	case IterSeekLast:
		ok = s.execWithoutResponse(func(i *unary.Iterator) bool { return i.SeekLast(ctx) })
	case IterSeekLE:
		ok = s.execWithoutResponse(func(i *unary.Iterator) bool { return i.SeekLE(ctx, req.Stamp) })
	case IterSeekGE:
		ok = s.execWithoutResponse(func(i *unary.Iterator) bool { return i.SeekGE(ctx, req.Stamp) })
	case IterValid:
		ok = s.execWithoutResponse(func(i *unary.Iterator) bool { return i.Valid() })
	case IterError:
		err = s.error()
	case IterSetBounds:
		ok = s.execWithoutResponse(func(i *unary.Iterator) bool { i.SetBounds(req.Bounds); return true })
	}
	return
}

func (s *streamIterator) execWithResponse(seqNum int, f func(i *unary.Iterator) bool) (ok bool) {
	for _, i := range s.internal {
		if f(i) {
			ok = true
			s.Out.Inlet() <- IteratorResponse{
				Variant: IteratorDataResponse,
				Command: IterNext,
				SeqNum:  seqNum,
				Frame:   i.Value(),
			}
		}
	}
	return ok
}

func (s *streamIterator) execWithoutResponse(f func(i *unary.Iterator) bool) (ok bool) {
	for _, i := range s.internal {
		if f(i) {
			ok = true
		}
	}
	return
}

func (s *streamIterator) error() error {
	for _, i := range s.internal {
		if err := i.Error(); err != nil {
			return err
		}
	}
	return nil
}

func (s *streamIterator) close() error {
	for _, i := range s.internal {
		if err := i.Close(); err != nil {
			return err
		}
	}
	return nil
}
