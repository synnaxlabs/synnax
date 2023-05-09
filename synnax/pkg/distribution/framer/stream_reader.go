// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type StreamReader = confluence.Segment[StreamReaderRequest, StreamReaderResponse]

type liveReader struct {
	confluence.AbstractUnarySink[StreamReaderRequest]
	confluence.AbstractUnarySource[StreamReaderResponse]
	iter struct {
		flow      confluence.Flow
		requests  confluence.Inlet[IteratorRequest]
		responses confluence.Outlet[IteratorResponse]
	}
	relay struct {
		flow      confluence.Flow
		requests  confluence.Inlet[relay.Request]
		responses confluence.Outlet[relay.Response]
	}
}

func (l *liveReader) Flow(sCtx signal.Context, opts ...confluence.Option) {
	l.iter.flow.Flow(sCtx, opts...)
	l.relay.flow.Flow(sCtx, opts...)
	o := confluence.NewOptions(opts)
	o.AttachClosables(l.Out)
	sCtx.Go(func(ctx context.Context) error {
		// start off by exhausting the iterator
	o:
		for {
			l.iter.requests.Inlet() <- IteratorRequest{
				Command: iterator.Next,
				Span:    iterator.AutoSpan,
			}

			for res := range l.iter.responses.Outlet() {
				if res.Variant == iterator.AckResponse {
					if res.Ack == false {
						break o
					}
					break
				}
				l.Out.Inlet() <- StreamReaderResponse{
					Frame: res.Frame,
					Error: res.Error,
				}
			}
		}

		// Close the iterator and drain the response channel
		l.iter.requests.Close()
		for range l.iter.responses.Outlet() {
		}

		// then we'll tap into the relay for live updates
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case res, ok := <-l.relay.responses.Outlet():
				if !ok {
					return nil
				}
				l.Out.Inlet() <- StreamReaderResponse{
					Frame: res.Frame,
				}
			case req, ok := <-l.In.Outlet():
				if !ok {
					l.relay.requests.Close()
					return nil
				}
				l.relay.requests.Inlet() <- relay.Request{Keys: req.Keys}
			}
		}
	}, o.Signal...)
}

type StreamReaderConfig struct {
	Start telem.TimeStamp `json:"start" msgpack:"start"`
	Keys  channel.Keys    `json:"keys" msgpack:"keys"`
}

type StreamReaderRequest = StreamReaderConfig

func (s *Service) NewStreamReader(ctx context.Context, cfg StreamReaderConfig) (StreamReader, error) {
	l := &liveReader{}
	iter, err := s.NewStreamIterator(ctx, IteratorConfig{
		Keys:   cfg.Keys,
		Bounds: cfg.Start.Range(telem.Now().Add(5 * telem.Second)),
	})
	if err != nil {
		return nil, err
	}
	iterRequests := confluence.NewStream[IteratorRequest](1)
	iterResponses := confluence.NewStream[IteratorResponse](1)
	iter.InFrom(iterRequests)
	iter.OutTo(iterResponses)
	l.iter.flow = iter
	l.iter.requests = iterRequests
	l.iter.responses = iterResponses

	rel, err := s.relay.NewReader(ctx, relay.ReaderConfig{Keys: cfg.Keys})
	if err != nil {
		return nil, err
	}
	relayRequests := confluence.NewStream[relay.Request](1)
	relayResponses := confluence.NewStream[relay.Response](1)
	rel.InFrom(relayRequests)
	rel.OutTo(relayResponses)
	l.relay.flow = rel
	l.relay.requests = relayRequests
	l.relay.responses = relayResponses
	return l, err
}
