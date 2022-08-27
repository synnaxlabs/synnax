package api

import (
	"context"
	"github.com/arya-analytics/delta/pkg/api/errors"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/segment"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/x/telem"
	roacherrors "github.com/cockroachdb/errors"
)

type IteratorRequest struct {
	Command IteratorCommand `json:"command" msgpack:"command"`
	Span    telem.TimeSpan  `json:"span" msgpack:"span"`
	Range   telem.TimeRange `json:"range" msgpack:"range"`
	Stamp   telem.TimeStamp `json:"stamp" msgpack:"stamp"`
	Keys    []string        `json:"keys" msgpack:"keys"`
}

type IteratorResponse struct {
	Variant  IteratorResponseVariant `json:"variant"`
	Ack      bool                    `json:"ack"`
	Command  IteratorCommand         `json:"command"`
	Err      errors.Typed            `json:"error"`
	Segments []Segment               `json:"segments"`
}

type IteratorStream = freighter.ServerStream[IteratorRequest, IteratorResponse]

func (s *SegmentService) Iterate(_ctx context.Context, stream IteratorStream) errors.Typed {
	ctx, cancel := context.WithCancel(_ctx)
	// cancellation here would occur for one of two reasons. Either we encounter
	// a fatal error (transport or iterator internal) and we need to free all
	// resources, OR the client executed the close command on the iterator (in
	// which case resources have already been freed and cancel does nothing).
	defer cancel()

	keys, tr, _err := receiveIteratorOpenArgs(stream)
	if _err.Occurred() {
		return _err
	}

	iter, err := s.Internal.NewRetrieve().WhereChannels(keys...).WhereTimeRange(tr).Iterate(ctx)
	if err != nil {
		return errors.General(err)
	}

	go func() {
		for {
			req, err := stream.Receive()
			if err != nil {
				return
			}
			ok, err := executeIteratorRequest(iter, req)
			if err := stream.Send(IteratorResponse{
				Variant: iterator.AckResponse,
				Command: req.Command,
				Ack:     ok,
				Err:     errors.General(err),
			}); err != nil {
				return
			}
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return errors.Canceled
		case res, ok := <-iter.Responses():
			if !ok {
				return errors.Nil
			}
			segments := make([]Segment, len(res.Segments))
			for i, seg := range res.Segments {
				segments[i] = Segment{
					ChannelKey: seg.ChannelKey.String(),
					Start:      seg.Segment.Start,
					Data:       seg.Segment.Data,
				}
			}
			if err := stream.Send(IteratorResponse{
				Variant:  iterator.DataResponse,
				Segments: segments,
			}); err != nil {
				return errors.Unexpected(err)
			}
		}
	}
}

func executeIteratorRequest(iter segment.Iterator, req IteratorRequest) (bool, error) {
	switch req.Command {
	case iterator.Next:
		return iter.Next(), nil
	case iterator.Prev:
		return iter.Prev(), nil
	case iterator.First:
		return iter.First(), nil
	case iterator.Last:
		return iter.Last(), nil
	case iterator.NextSpan:
		return iter.NextSpan(req.Span), nil
	case iterator.PrevSpan:
		return iter.PrevSpan(req.Span), nil
	case iterator.NextRange:
		return iter.NextRange(req.Range), nil
	case iterator.SeekFirst:
		return iter.SeekFirst(), nil
	case iterator.SeekLast:
		return iter.SeekLast(), nil
	case iterator.SeekLT:
		return iter.SeekLT(req.Stamp), nil
	case iterator.SeekGE:
		return iter.SeekGE(req.Stamp), nil
	case iterator.Valid:
		return iter.Valid(), nil
	case iterator.Error:
		err := iter.Error()
		return err == nil, err
	default:
		return false, errors.Parse(roacherrors.New("unexpected command"))
	}
}

func receiveIteratorOpenArgs(srv IteratorStream) (channel.Keys, telem.TimeRange, errors.Typed) {
	req, err := srv.Receive()
	if err != nil {
		return nil, telem.TimeRangeZero, errors.Unexpected(err)
	}
	if req.Command != iterator.Open {
		return nil, telem.TimeRangeZero, errors.Parse(roacherrors.New("expected open command"))
	}
	keys, err := channel.ParseKeys(req.Keys)

	if req.Range.IsZero() {
		req.Range = telem.TimeRangeMax
	}

	return keys, req.Range, errors.MaybeParse(err)
}
