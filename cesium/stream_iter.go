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
// in time order. StreamIterator provides the underlying functionality for kvPositionIterator,
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
	mdIter  core.MDStampIterator
	counter int
	readRes confluence.Outlet[storage.ReadResponse[Segment]]
	readReq confluence.Inlet[Segment]
}

func (s *streamIterator) Flow(ctx signal.Context, opts ...confluence.Option) {
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
	})
}

func (s *streamIterator) exec(ctx context.Context, req IteratorRequest) {
	ok, err := s.runCmd(req)
	if !ok || !req.Command.HasOps() {
		s.sendAck(req, ok, err)
	}
	mds := s.mdIter.Value()
	for _, md := range mds {
		s.readReq.Inlet() <- Segment{ChannelKey: md.ChannelKey, MD: md}
	}
	for range mds {
		res := <-s.readRes.Outlet()
		s.sendData(req, res)
	}
	s.sendAck(req, ok, err)
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
	s.counter += 1
	s.Out.Inlet() <- IteratorResponse{
		Counter: s.counter,
		Variant: IteratorResponseTypeAck,
		Ack:     ok,
		Err:     err,
		Command: req.Command,
	}
}

func (s *streamIterator) sendData(req IteratorRequest, readRes storage.ReadResponse[Segment]) {
	readRes.Request.Data = readRes.Data
	s.Out.Inlet() <- IteratorResponse{
		Counter:  s.counter,
		Variant:  IteratorResponseTypeData,
		Segments: []Segment{readRes.Request},
		Err:      readRes.Err,
		Command:  req.Command,
	}
}
