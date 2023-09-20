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
	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/reflect"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type Streamer = confluence.Segment[StreamerRequest, StreamerResponse]

type streamer struct {
	confluence.AbstractUnarySink[StreamerRequest]
	confluence.AbstractUnarySource[StreamerResponse]
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

// Flow implements confluence.Flow.
func (l *streamer) Flow(sCtx signal.Context, opts ...confluence.Option) {
	hasIter := !reflect.IsNil(l.iter.flow)
	if hasIter {
		l.iter.flow.Flow(sCtx, opts...)
	}
	l.relay.flow.Flow(sCtx, opts...)
	o := confluence.NewOptions(opts)
	o.AttachClosables(l.Out)

	sCtx.Go(func(ctx context.Context) error {

		if hasIter {
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
					l.Out.Inlet() <- StreamerResponse{
						Frame: res.Frame,
						Error: res.Error,
					}
				}
			}
		}

		// Close the iterator and drain the response channel
		if hasIter {
			l.iter.requests.Close()
			confluence.Drain(l.iter.responses)
		}

		// Then we'll tap into the relay for stream updates
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case res, ok := <-l.relay.responses.Outlet():
				if !ok {
					return nil
				}
				l.Out.Inlet() <- StreamerResponse{
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

type StreamerConfig struct {
	Start telem.TimeStamp `json:"start" msgpack:"start"`
	Keys  channel.Keys    `json:"keys" msgpack:"keys"`
}

type StreamerRequest = StreamerConfig

func (s *Service) NewStreamer(ctx context.Context, cfg StreamerConfig) (Streamer, error) {
	l := &streamer{}

	if lo.SomeBy(cfg.Keys, func(k channel.Key) bool { return !k.Free() }) {
		iter, err := s.NewStreamIterator(ctx, IteratorConfig{
			Keys:   cfg.Keys,
			Bounds: cfg.Start.Range(telem.Now().Add(5 * telem.Second)),
		})
		if err != nil {
			return nil, err
		}
		iterReq, iterRes := confluence.Attach(iter, 1)
		l.iter.flow = iter
		l.iter.requests = iterReq
		l.iter.responses = iterRes
	}

	rel, err := s.relay.NewStreamer(ctx, relay.StreamerConfig{Keys: cfg.Keys})
	if err != nil {
		return nil, err
	}
	relayReq, relayRes := confluence.Attach(rel, 1)
	l.relay.flow = rel
	l.relay.requests = relayReq
	l.relay.responses = relayRes
	return l, err
}
