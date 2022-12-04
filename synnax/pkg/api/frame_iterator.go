package api

import (
	"context"
	roacherrors "github.com/cockroachdb/errors"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type (
	IteratorCommand         = iterator.Command
	IteratorResponseVariant = iterator.ResponseVariant
)

type FrameIteratorRequest struct {
	Command IteratorCommand `json:"command" msgpack:"command"`
	Span    telem.TimeSpan  `json:"span" msgpack:"span"`
	Range   telem.TimeRange `json:"range" msgpack:"range"`
	Stamp   telem.TimeStamp `json:"stamp" msgpack:"stamp"`
	Keys    []string        `json:"keys" msgpack:"keys"`
}

type FrameIteratorResponse struct {
	Variant IteratorResponseVariant `json:"variant" msgpack:"variant"`
	Command IteratorCommand         `json:"command" msgpack:"command"`
	Ack     bool                    `json:"ack" msgpack:"ack"`
	Err     ferrors.Payload         `json:"error" msgpack:"error"`
	Frame   Frame                   `json:"frame" msgpack:"frame"`
}

type FrameIteratorStream = freighter.ServerStream[FrameIteratorRequest, FrameIteratorResponse]

func (s *FrameService) Iterate(_ctx context.Context, stream FrameIteratorStream) errors.Typed {
	ctx, cancel := signal.WithCancel(_ctx, signal.WithLogger(s.logger.Desugar()))
	// Cancellation here would occur for one of two reasons. Either we encounter
	// a fatal error (transport or iterator internal) and we need to free all
	// resources, OR the client executed the close command on the iterator (in
	// which case resources have already been freed and cancel does nothing).
	defer cancel()

	iter, err := s.openIterator(ctx, stream)
	if err.Occurred() {
		return err
	}
	requests := confluence.NewStream[iterator.Request]()
	iter.InFrom(requests)
	responses := confluence.NewStream[iterator.Response](1)
	iter.OutTo(responses)
	iter.Flow(ctx, confluence.CloseInletsOnExit(), confluence.CancelOnExitErr())

	go func() {
		for {
			req, err := stream.Receive()
			if roacherrors.Is(err, freighter.EOF) {
				requests.Close()
				return
			}
			if err != nil {
				return
			}
			requests.Inlet() <- iterator.Request{
				Command: req.Command,
				Span:    req.Span,
				Bounds:  req.Range,
				Stamp:   req.Stamp,
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return errors.Canceled
		case res, ok := <-responses.Outlet():
			if !ok {
				return errors.Nil
			}
			tRes := FrameIteratorResponse{
				Variant: res.Variant,
				Command: res.Command,
				Ack:     res.Ack,
				Frame:   newFrameFromDistribution(res.Frame),
			}
			if res.Err != nil {
				tRes.Err = ferrors.Encode(res.Err)
			}
			if err := stream.Send(tRes); err != nil {
				return errors.Unexpected(err)
			}
		}
	}
}

func (s *FrameService) openIterator(ctx context.Context, srv FrameIteratorStream) (framer.StreamIterator, errors.Typed) {
	keys, bounds, _err := receiveIteratorOpenArgs(srv)
	if _err.Occurred() {
		return nil, _err
	}
	iter, err := s.Internal.NewStreamIterator(ctx, framer.IteratorConfig{Bounds: bounds, Keys: keys})
	if err != nil {
		return nil, errors.Query(err)
	}
	return iter, errors.MaybeUnexpected(srv.Send(FrameIteratorResponse{Variant: iterator.AckResponse, Ack: true}))
}

func receiveIteratorOpenArgs(srv FrameIteratorStream) (channel.Keys, telem.TimeRange, errors.Typed) {
	req, err := srv.Receive()
	if err != nil {
		return nil, telem.TimeRangeZero, errors.Unexpected(err)
	}
	keys, err := channel.ParseKeys(req.Keys)
	if req.Range.IsZero() {
		req.Range = telem.TimeRangeMax
	}

	return keys, req.Range, errors.MaybeParse(err)
}
