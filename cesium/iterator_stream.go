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

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/cesium/internal/unary"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/set"
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
	// IteratorResponseVariantAck is a response that indicates that an iteration request
	// has completed successfully.
	IteratorResponseVariantAck IteratorResponseVariant = iota + 1
	// IteratorResponseVariantData is a response that indicates that an iteration
	// request returned data.
	IteratorResponseVariantData
)

// IteratorCommand is an enumeration of commands that can be sent to an Iterator.
type IteratorCommand uint8

const (
	// IteratorCommandNext represents a call to Iterator.Next.
	IteratorCommandNext IteratorCommand = iota + 1
	// IteratorCommandPrev represents a call to Iterator.Prev.
	IteratorCommandPrev
	// IteratorCommandSeekFirst represents a call to Iterator.SeekFirst.
	IteratorCommandSeekFirst
	// IteratorCommandSeekLast represents a call to Iterator.SeekLast.
	IteratorCommandSeekLast
	// IterCommandSeekLE represents a call to Iterator.SeekLE.
	IterCommandSeekLE
	// IteratorCommandSeekGE represents a call to Iterator.SeekGE.
	IteratorCommandSeekGE
	// IteratorCommandValid represents a call to Iterator.Valid.
	IteratorCommandValid
	// IteratorCommandError represents a call to Iterator.Error.
	IteratorCommandError
	// IteratorCommandSetBounds represents a call to Iterator.SetBounds.
	IteratorCommandSetBounds
)

var validateIteratorCommand = validate.NewInclusiveBoundsChecker(
	IteratorCommandNext,
	IteratorCommandSetBounds,
)

// HasOps returns true if the IteratorCommand has any associated on disk operations.
func (i IteratorCommand) HasOps() bool { return i <= IteratorCommandPrev }

// IteratorRequest is issued to an StreamIterator asking it to read data from a DB.
//
//go:generate stringer -type=IteratorCommand
type IteratorRequest struct {
	// Command is the command to execute.
	Command IteratorCommand
	// Stamp should be set during a request to IteratorCommandSeekLE or
	// IteratorCommandSeekGE.
	Stamp telem.TimeStamp
	// Span should be set during a request to IteratorCommandNext or
	// IteratorCommandPrev.
	Span telem.TimeSpan
	// Bounds should be set during a request to IteratorCommandSetBounds.
	Bounds telem.TimeRange
	// SeqNum is the sequence number of the request. This is used to match the request
	// with the response. Each request should increment the sequence number by 1.
	SeqNum int
}

// IteratorResponse is a response containing segments satisfying an iterator request as
// well as any errors encountered during the retrieval.
type IteratorResponse struct {
	// Err is only set an IterError command is being responded to.
	Err error
	// Frame is the telemetry frame that was read from the DB. It is only set when the
	// response type is IteratorDataResponse.
	Frame Frame
	// SeqNum is the corresponding sequence number of the request. This is used to match
	// the request with the response. Each request should increment the sequence number
	// by 1.
	SeqNum int
	// Variant is the type of response being issued.
	Variant IteratorResponseVariant
	// Command defines the command that the response relates to.
	Command IteratorCommand
	// Ack is only valid when the response type is IteratorAckResponse. It indicates
	// whether the command was successfully processed.
	Ack bool
}

type streamIterator struct {
	confluence.UnarySink[IteratorRequest]
	confluence.AbstractUnarySource[IteratorResponse]
	internal []*unary.Iterator
	// reduced is true when the iterator reduces its reads, which bounds their size.
	reduced bool
}

// IteratorConfig is the configuration for opening an iterator :). See the fields for
// more information.
type IteratorConfig struct {
	// Channels is a list of channels to iterate over.
	Channels []channel.Key
	// Bounds sets the time range to iterator over. The lower bound is inclusive, while
	// the upper bound is exclusive.
	Bounds telem.TimeRange
	// AutoChunkSize sets the default chunk size to iterator over when sending a Next()
	// or Prev() request it IteratorAutoSpan as the span.
	AutoChunkSize int64
	// Reduction reduces the samples of each channel the iterator reads. A stride skips
	// samples without reading them. A limit reduces each channel in Bounds to about
	// PointLimit points, and every channel on one index reduces over the same groups,
	// so reduced index and data series share alignments.
	Reduction telem.Reduction
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
					Variant: IteratorResponseVariantAck,
					Command: req.Command,
					SeqNum:  req.SeqNum,
					Ack:     ok,
					Err:     err,
				}
			}
		}
	}, o.Signal...)
}

func (s *streamIterator) exec(
	ctx context.Context,
	req IteratorRequest,
) (ok bool, err error) {
	if err := validateIteratorCommand(req.Command); err != nil {
		return false, err
	}
	switch req.Command {
	case IteratorCommandNext:
		if s.reduced || req.Span == AutoSpan {
			ok = s.execWithResponse(
				req.SeqNum,
				func(i *unary.Iterator) bool { return i.Next(ctx, req.Span) },
			)
		} else {
			ok, err = s.nextInPieces(ctx, req.SeqNum, req.Span)
		}
	case IteratorCommandPrev:
		ok = s.execWithResponse(
			req.SeqNum,
			func(i *unary.Iterator) bool { return i.Prev(ctx, req.Span) },
		)
	case IteratorCommandSeekFirst:
		ok = s.execWithoutResponse(
			func(i *unary.Iterator) bool { return i.SeekFirst(ctx) },
		)
	case IteratorCommandSeekLast:
		ok = s.execWithoutResponse(
			func(i *unary.Iterator) bool { return i.SeekLast(ctx) },
		)
	case IterCommandSeekLE:
		ok = s.execWithoutResponse(
			func(i *unary.Iterator) bool { return i.SeekLE(ctx, req.Stamp) },
		)
	case IteratorCommandSeekGE:
		ok = s.execWithoutResponse(
			func(i *unary.Iterator) bool { return i.SeekGE(ctx, req.Stamp) },
		)
	case IteratorCommandValid:
		ok = s.execWithoutResponse(func(i *unary.Iterator) bool { return i.Valid() })
	case IteratorCommandError:
		err = s.error()
	case IteratorCommandSetBounds:
		ok = s.execWithoutResponse(
			func(i *unary.Iterator) bool { i.SetBounds(req.Bounds); return true },
		)
	}
	return ok, err
}

// execWithResponse runs f on each iterator and sends the data of every iterator that f
// moves as one response.
func (s *streamIterator) execWithResponse(
	seqNum int,
	f func(i *unary.Iterator) bool,
) (ok bool) {
	var fr Frame
	for _, i := range s.internal {
		if f(i) {
			ok = true
			fr = fr.Extend(i.Value())
		}
	}
	s.send(seqNum, fr)
	return ok
}

// nextInPieces moves each iterator forward by span in pieces that each hold at most
// AutoChunkSize samples of every index. It sends one data response per piece, holding
// every channel's data up to the end of that piece.
func (s *streamIterator) nextInPieces(
	ctx context.Context,
	seqNum int,
	span telem.TimeSpan,
) (ok bool, err error) {
	var (
		targets = make([]telem.TimeStamp, len(s.internal))
		started = make([]bool, len(s.internal))
		last    telem.TimeStamp
	)
	for j, i := range s.internal {
		targets[j] = i.View().End.SpanRange(span).BoundBy(i.Bounds()).End
		last = max(last, targets[j])
	}
	for {
		end := last
		probed := make(set.Set[channel.Key], len(s.internal))
		for j, i := range s.internal {
			idx := i.Channel.Index
			if i.Channel.IsIndex {
				idx = i.Channel.Key
			}
			if i.View().End >= targets[j] || probed.Contains(idx) {
				continue
			}
			probed.Add(idx)
			pieceEnd, err := i.EndAfter(ctx, i.AutoChunkSize)
			if err != nil {
				return ok, err
			}
			end = min(end, pieceEnd)
		}
		var piece Frame
		for j, i := range s.internal {
			stop := min(end, targets[j])
			if stop <= i.View().End {
				continue
			}
			var read bool
			if started[j] {
				read = i.Continue(ctx, stop)
			} else {
				read = i.Next(ctx, i.View().End.Span(stop))
				started[j] = true
			}
			if read {
				ok = true
				piece = piece.Extend(i.Value())
			}
		}
		s.send(seqNum, piece)
		if end >= last {
			break
		}
	}
	var rest Frame
	for j, i := range s.internal {
		if !started[j] && i.Next(ctx, span) {
			ok = true
			rest = rest.Extend(i.Value())
		}
	}
	s.send(seqNum, rest)
	return ok, nil
}

// send sends fr as a data response unless it holds no series.
func (s *streamIterator) send(seqNum int, fr Frame) {
	if fr.Empty() {
		return
	}
	s.Out.Inlet() <- IteratorResponse{
		Variant: IteratorResponseVariantData,
		Command: IteratorCommandNext,
		SeqNum:  seqNum,
		Frame:   fr,
	}
}

func (s *streamIterator) execWithoutResponse(f func(i *unary.Iterator) bool) (ok bool) {
	for _, i := range s.internal {
		if f(i) {
			ok = true
		}
	}
	return ok
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
	var err error
	for _, i := range s.internal {
		err = errors.Join(err, i.Close())
	}
	return err
}
