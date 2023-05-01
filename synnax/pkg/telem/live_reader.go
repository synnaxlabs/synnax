// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem

import (
	"context"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/relay"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type LiveReadRequest struct {
	Keys channel.Keys
}

type LiveReadResponse struct {
	Frame framer.Frame
	Err   error
}

type LiveReader = confluence.Segment[LiveReadRequest, LiveReadResponse]

type liveReader struct {
	confluence.AbstractUnarySink[LiveReadRequest]
	confluence.AbstractUnarySource[LiveReadResponse]
	iter struct {
		flow      confluence.Flow
		requests  confluence.Inlet[framer.IteratorRequest]
		responses confluence.Outlet[framer.IteratorResponse]
	}
	relay struct {
		flow      confluence.Flow
		requests  confluence.Inlet[relay.ReadRequest]
		responses confluence.Outlet[relay.Data]
	}
}

func (l *liveReader) Flow(sCtx signal.Context, opts ...confluence.Option) {
	l.iter.flow.Flow(sCtx, opts...)
	o := confluence.NewOptions(opts)
	o.AttachClosables(l.Out)
	sCtx.Go(func(ctx context.Context) error {
		// start off by exhausting the iterator
	o:
		for {
			l.iter.requests.Inlet() <- framer.IteratorRequest{
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
				l.Out.Inlet() <- LiveReadResponse{
					Frame: res.Frame,
					Err:   res.Err,
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
				l.Out.Inlet() <- LiveReadResponse{
					Frame: res.Frame,
				}
			case req, ok := <-l.In.Outlet():
				if !ok {
					l.relay.requests.Close()
				}
				l.relay.requests.Inlet() <- relay.ReadRequest{
					Keys: req.Keys,
				}
			}
		}
	}, o.Signal...)
}

type LiveReaderConfig struct {
	From telem.TimeStamp `json:"from" msgpack:"from"`
	Keys channel.Keys    `json:"keys" msgpack:"keys"`
}

func (s *Service) NewLiveReader(ctx context.Context, cfg LiveReaderConfig) (LiveReader, error) {
	var (
		err error
		l   = &liveReader{}
	)
	// Open up our iterator
	iter, err := s.framer.NewStreamIterator(ctx, framer.IteratorConfig{
		Keys:   cfg.Keys,
		Bounds: cfg.From.Range(telem.Now().Add(5 * telem.Second)),
	})
	iterRequests := confluence.NewStream[framer.IteratorRequest](1)
	iterResponses := confluence.NewStream[framer.IteratorResponse](1)
	iter.InFrom(iterRequests)
	iter.OutTo(iterResponses)
	l.iter.flow = iter
	l.iter.requests = iterRequests
	l.iter.responses = iterResponses

	// Open up our relay
	rel := s.relay.NewReader(cfg.Keys...)
	relayRequests := confluence.NewStream[relay.ReadRequest](1)
	relayResponses := confluence.NewStream[relay.Data](1)
	rel.InFrom(relayRequests)
	rel.OutTo(relayResponses)
	l.relay.flow = rel
	l.relay.requests = relayRequests
	l.relay.responses = relayResponses

	return l, err
}
