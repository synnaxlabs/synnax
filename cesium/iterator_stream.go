package cesium

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

// StreamIterator provides a streaming interface for iterating over a DB's segments
// in time order. StreamIterator provides the underlying functionality for Iterator,
// and has almost exactly the same semantics. The streaming interface is exposed
// as a confluence segment that can accept one input stream and one output stream.
//
// To read segments, issue an IteratorRequest to the StreamIterator's inlet. The
// StreamIterator will respond by sending one or more IteratorResponse messages to
// the outlet. All responses containing segment data will have a type of
// IteratorDataResponse and will contain one or more segments. The last response
// for any request will have a type of IteratorAckResponse and will contain
// the name of the command that was acknowledged, and incremented sequence number,
// and ack boolean indicating whether the command was successfully processed.
//
// To close the StreamIterator, simply close the inlet. The StreamIterator will ensure
// that all in-progress requests have been served before closing the outlet. The
// StreamIterator will return any accumulated err through the signal context
// provided to Flow.
type StreamIterator = confluence.Segment[IteratorRequest, IteratorResponse]

// IteratorResponseVariant is the type of the response an iterator will return.
type IteratorResponseVariant uint8

const (
	// IteratorAckResponse is a response that indicates that an iteration request
	// has completed successfully.
	IteratorAckResponse IteratorResponseVariant = iota + 1
	// IteratorDataResponse is a response that indicates that an iteration request
	// returned data.
	IteratorDataResponse
)

// IteratorCommand is an enumeration of commands that can be sent to an iterator.
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
	// IterError represents a call to Iterator.Close.
	IterError
	// IterSetBounds represents a call to Iterator.SetBounds.
	IterSetBounds
)

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
}

// IteratorResponse is a response containing segments satisfying a RetrieveP Query as
// well as any errors encountered during the retrieval.
type IteratorResponse struct {
	// Variant is the type of response being issued.
	Variant IteratorResponseVariant
	// SeqNum is incremented for each request issued to the StreamIterator. The
	// first request will have a sequence number of 1.
	SeqNum int
	// Command defines the command that the response relates to.
	Command IteratorCommand
	// Ack is only valid when the response type is IteratorAckResponse. It
	// indicates whether the command was successfully processed.
	Ack bool
	// Err is only set an IterError command is being responded to.
	Err error
	// Segments is only set when the response type is IteratorDataResponse. It
	// contains the segments that were read.
	Segments []Segment
}

type streamIterator struct {
	confluence.UnarySink[IteratorRequest]
	confluence.AbstractUnarySource[IteratorResponse]
	mdIter core.TimeIterator
	reader storage.Reader
	seqNum int
}

// Flow implements the confluence.Segment interface.
func (s *streamIterator) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(s.Out)
	ctx.Go(func(ctx context.Context) error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-s.In.Outlet():
				if !ok {
					return s.mdIter.Close()
				}
				s.exec(ctx, req)
			}
		}
	}, o.Signal...)
}

func (s *streamIterator) exec(ctx context.Context, req IteratorRequest) {
	ok, err := s.runCmd(req)
	if !ok || !req.Command.HasOps() {
		s.sendAck(req, ok, err)
		return
	}
	segments, err := s.reader.Read(s.mdIter.Value())

	if err != nil {
		s.sendAck(req, false, err)
		return
	}
	s.sendData(req, segments)
	s.sendAck(req, true, err)
}

func (s *streamIterator) runCmd(req IteratorRequest) (bool, error) {
	switch req.Command {
	case IterNext:
		return s.mdIter.Next(req.Span), nil
	case IterPrev:
		return s.mdIter.Prev(req.Span), nil
	case IterSeekFirst:
		return s.mdIter.SeekFirst(), nil
	case IterSeekLast:
		return s.mdIter.SeekLast(), nil
	case IterSeekGE:
		return s.mdIter.SeekGE(req.Stamp), nil
	case IterSeekLE:
		return s.mdIter.SeekLE(req.Stamp), nil
	case IterValid:
		return s.mdIter.Valid(), nil
	case IterError:
		return false, s.mdIter.Error()
	case IterSetBounds:
		return s.mdIter.SetBounds(req.Bounds), nil
	default:
		return false, errors.Errorf("unknown iterator command: %v", req.Command)
	}
}

func (s *streamIterator) sendAck(req IteratorRequest, ok bool, err error) {
	s.seqNum += 1
	s.Out.Inlet() <- IteratorResponse{
		SeqNum:  s.seqNum,
		Variant: IteratorAckResponse,
		Ack:     ok,
		Err:     err,
		Command: req.Command,
	}
}

func (s *streamIterator) sendData(req IteratorRequest, ss []core.SugaredSegment) {
	segments := make([]Segment, len(ss))
	for i, seg := range ss {
		segments[i] = Segment{ChannelKey: seg.ChannelKey, Start: seg.Start, Data: seg.Data}
	}
	s.Out.Inlet() <- IteratorResponse{
		SeqNum:   s.seqNum,
		Variant:  IteratorDataResponse,
		Segments: segments,
		Command:  req.Command,
	}
}
