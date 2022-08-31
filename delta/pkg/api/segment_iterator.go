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

	go func() {
		for {
			req, err := stream.Receive()
			if err != nil {
				return
			}
			executeIteratorRequest(iter, req)
			if req.Command == iterator.Close {
				return
			}
		}
	}()

	c := 0

	for {
		select {
		case <-ctx.Done():
			return errors.Canceled
		case res, ok := <-iter.Responses():
			if !ok {
				return errors.Nil
			}
			logrus.Info(res.Variant)
			if res.Variant == iterator.DataResponse {
				c++
			}
			segments := make([]Segment, len(res.Segments))
			for i, seg := range res.Segments {
				segments[i] = Segment{
					ChannelKey: seg.ChannelKey.String(),
					Start:      seg.Segment.Start,
					Data:       seg.Segment.Data,
				}
			}
			tRes := IteratorResponse{
				Variant:  res.Variant,
				Command:  res.Command,
				Ack:      res.Ack,
				Segments: segments,
			}
			if res.Error != nil {
				tRes.Err = ferrors.Encode(res.Error)
			}
			if err := stream.Send(tRes); err != nil {
				return errors.Unexpected(err)
			}
		}
	}
}

func executeIteratorRequest(iter segment.Iterator, req IteratorRequest) {
	switch req.Command {
	case iterator.Next:
		iter.Next()
	case iterator.Prev:
		iter.Prev()
	case iterator.First:
		iter.First()
	case iterator.Last:
		iter.Last()
	case iterator.NextSpan:
		iter.NextSpan(req.Span)
	case iterator.PrevSpan:
		iter.PrevSpan(req.Span)
	case iterator.NextRange:
		iter.NextRange(req.Range)
	case iterator.SeekFirst:
		iter.SeekFirst()
	case iterator.SeekLast:
		iter.SeekLast()
	case iterator.SeekLT:
		iter.SeekLT(req.Stamp)
	case iterator.SeekGE:
		iter.SeekGE(req.Stamp)
	case iterator.Valid:
		iter.Valid()
	case iterator.Error:
		_ = iter.Error()
	case iterator.Exhaust:
		iter.Exhaust()
	case iterator.Close:
		_ = iter.Close()
	}
}

func (s *SegmentService) openIterator(ctx context.Context, srv IteratorStream) (segment.Iterator, errors.Typed) {
	keys, rng, sync, _err := receiveIteratorOpenArgs(srv)
	if _err.Occurred() {
		return nil, _err
	}
	q := s.Internal.NewRetrieve().WhereChannels(keys...).WhereTimeRange(rng)
	if sync {
		q = q.Sync().SendAcknowledgements()
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
