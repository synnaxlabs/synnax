package cesium

import (
	"context"
	"github.com/cockroachdb/errors"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/internal/storage"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

// StreamIterator provides a streaming interface for iterating over a DB's segments
// in time order. StreamIterator provides the underlying functionality for Iterator,
// and has almost exactly the same semantics. The streaming interface is exposed
// as a confluence segment that can accept one input stream and one output stream.
//
// To read segments issue an IteratorRequest to the StreamIterator's inlet. The
// StreamIterator will respond by sending one or more IteratorResponse messages to
// the outlet. All responses containing segment data will have a type of
// IteratorResponseTypeData and will contain one or more segments. The last response
// for any request will have a type of IteratorResponseTypeAck and will contain
// the name of the command that was acknowledged, and incremented sequence number,
// and ack boolean indicating whether the command was successfully processed.
//
// To close the StreamIterator, simply close the inlet. The StreamIterator will ensure
// that all in-progress requests have been served before closing the outlet. The
// StreamIterator will return any accumulated err through the signal context
// provided to Flow.
type StreamIterator = confluence.Segment[IteratorRequest, IteratorResponse]

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
		return s.mdIter.SeekGE(req.Target), nil
	case IterSeekLE:
		return s.mdIter.SeekLE(req.Target), nil
	case IterValid:
		return s.mdIter.Valid(), nil
	case IterError:
		return false, s.mdIter.Error()
	default:
		return false, errors.Errorf("unknown iterator command: %v", req.Command)
	}
}

func (s *streamIterator) sendAck(req IteratorRequest, ok bool, err error) {
	s.seqNum += 1
	s.Out.Inlet() <- IteratorResponse{
		SeqNum:  s.seqNum,
		Variant: IteratorResponseTypeAck,
		Ack:     ok,
		Err:     err,
		Command: req.Command,
	}
}

func (s *streamIterator) sendData(req IteratorRequest, ss []core.SugaredSegment) {
	segs := make([]Segment, len(ss))
	for i, seg := range ss {
		segs[i] = Segment{ChannelKey: seg.ChannelKey, Start: seg.Start, Data: seg.Data}
	}
	s.Out.Inlet() <- IteratorResponse{
		SeqNum:   s.seqNum,
		Variant:  IteratorResponseTypeData,
		Segments: segs,
		Command:  req.Command,
	}
}
