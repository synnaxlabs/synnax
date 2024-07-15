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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/reflect"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/telem"
)

type Streamer = confluence.Segment[StreamerRequest, StreamerResponse]

type streamer struct {
	ts                 *ts.DB
	sendControlDigests bool
	controlStateKey    channel.Key
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

		l.relay.flow.Flow(sCtx, append(opts, confluence.WithAddress("relay-reader"))...)

		if l.sendControlDigests {
			u := l.ts.ControlUpdateToFrame(ctx, l.ts.ControlStates())
			l.Out.Inlet() <- StreamerResponse{Frame: core.NewFrameFromStorage(u)}
		}

		// Then we'll tap into the Relay for stream updates
		for {
			select {
			case <-ctx.Done():
				l.relay.requests.Close()
				confluence.Drain(l.relay.responses)
				return ctx.Err()
			case res, ok := <-l.relay.responses.Outlet():
				if !ok {
					return nil
				}
				if err := signal.SendUnderContext(ctx, l.Out.Inlet(), StreamerResponse{Frame: res.Frame}); err != nil {
					l.relay.requests.Close()
					confluence.Drain(l.relay.responses)
					return err
				}
			case req, ok := <-l.In.Outlet():
				if !ok {
					l.relay.requests.Close()
					confluence.Drain(l.relay.responses)
					return nil
				}
				if !l.sendControlDigests && lo.Contains(req.Keys, l.controlStateKey) {
					l.sendControlDigests = true
					u := l.ts.ControlUpdateToFrame(ctx, l.ts.ControlStates())
					l.Out.Inlet() <- StreamerResponse{Frame: core.NewFrameFromStorage(u)}
				}
				if err := signal.SendUnderContext(ctx, l.relay.requests.Inlet(), relay.Request{Keys: req.Keys}); err != nil {
					l.relay.requests.Close()
					confluence.Drain(l.relay.responses)
					return err
				}
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
	l := &streamer{
		ts:                 s.iterator.TS,
		controlStateKey:    s.controlStateKey,
		sendControlDigests: lo.Contains(cfg.Keys, s.controlStateKey),
	}
	rel, err := s.Relay.NewStreamer(ctx, relay.StreamerConfig{Keys: cfg.Keys})
	if err != nil {
		return nil, err
	}
	relayReq, relayRes := confluence.Attach(rel, 30)
	relayReq.SetInletAddress("relay-reader")
	relayRes.SetOutletAddress("relay-reader")
	l.relay.flow = rel
	l.relay.requests = relayReq
	l.relay.responses = relayRes
	return l, err
}
