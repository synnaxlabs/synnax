package api

import (
	"context"
	"github.com/arya-analytics/delta/pkg/api/errors"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/segment"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/freighter"
	"github.com/arya-analytics/freighter/ferrors"
	"github.com/arya-analytics/x/telem"
	roacherrors "github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"
)

type (
	IteratorCommand         = iterator.Command
	IteratorResponseVariant = iterator.ResponseVariant
)

type IteratorRequest struct {
	Command IteratorCommand `json:"command" msgpack:"command"`
	Span    telem.TimeSpan  `json:"span" msgpack:"span"`
	Range   telem.TimeRange `json:"range" msgpack:"range"`
	Stamp   telem.TimeStamp `json:"stamp" msgpack:"stamp"`
	Keys    []string        `json:"keys" msgpack:"keys"`
	Sync    bool            `json:"sync" msgpack:"sync"`
}

type IteratorResponse struct {
	Variant  IteratorResponseVariant `json:"variant" msgpack:"variant"`
	Command  IteratorCommand         `json:"command" msgpack:"command"`
	Ack      bool                    `json:"ack" msgpack:"ack"`
	Err      ferrors.Payload         `json:"error" msgpack:"error"`
	Segments []Segment               `json:"segments" msgpack:"segments"`
}

type IteratorStream = freighter.ServerStream[IteratorRequest, IteratorResponse]

func (s *SegmentService) Iterate(_ctx context.Context, stream IteratorStream) errors.Typed {
	ctx, cancel := context.WithCancel(_ctx)
	// cancellation here would occur for one of two reasons. Either we encounter
	// a fatal error (transport or iterator internal) and we need to free all
	// resources, OR the client executed the close command on the iterator (in
	// which case resources have already been freed and cancel does nothing).
	defer cancel()

	iter, err := s.openIterator(ctx, stream)
	if err.Occurred() {
		return errors.Unexpected(err)
	}

	acks := make(chan IteratorResponse)

	go func() {
		for {
			req, err := stream.Receive()
			if err != nil {
				return
			}
			ok, err := executeIteratorRequest(iter, req)
			res := IteratorResponse{
				Variant: iterator.AckResponse,
				Command: req.Command,
				Ack:     ok,
			}
			if err != nil {
				res.Err = ferrors.Encode(errors.General(err))
			}
			acks <- res
			if req.Command == iterator.Close {
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
				logrus.Info("returning here")
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
		case ack := <-acks:
			if err := stream.Send(ack); err != nil {
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
	case iterator.Close:
		err := iter.Close()
		return err == nil, err
	default:
		return false, errors.Parse(roacherrors.New("unexpected command"))
	}
}

func (s *SegmentService) openIterator(ctx context.Context, srv IteratorStream) (segment.Iterator, errors.Typed) {
	keys, rng, sync, _err := receiveIteratorOpenArgs(srv)
	if _err.Occurred() {
		return nil, _err
	}
	q := s.Internal.NewRetrieve().WhereChannels(keys...).WhereTimeRange(rng)
	if sync {
		q = q.Sync()
	}
	i, err := q.Iterate(ctx)
	if err != nil {
		return nil, errors.Query(err)
	}
	return i, errors.MaybeUnexpected(srv.Send(IteratorResponse{Variant: iterator.AckResponse, Ack: true}))
}

func receiveIteratorOpenArgs(srv IteratorStream) (channel.Keys, telem.TimeRange, bool, errors.Typed) {
	req, err := srv.Receive()
	if err != nil {
		return nil, telem.TimeRangeZero, req.Sync, errors.Unexpected(err)
	}
	if req.Command != iterator.Open {
		return nil, telem.TimeRangeZero, req.Sync, errors.Parse(roacherrors.New("expected open command"))
	}
	keys, err := channel.ParseKeys(req.Keys)

	if req.Range.IsZero() {
		req.Range = telem.TimeRangeMax
	}

	return keys, req.Range, req.Sync, errors.MaybeParse(err)
}
