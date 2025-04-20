// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/change"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

type Streamer = confluence.Segment[Request, Response]

type streamer struct {
	confluence.AbstractLinear[Request, Response]
	addr    address.Address
	demands confluence.Inlet[demand]
	relay   *Relay
	cfg     StreamerConfig
}

type StreamerConfig struct {
	Keys        channel.Keys
	SendOpenAck bool
}

func (r *Relay) NewStreamer(ctx context.Context, cfg StreamerConfig) (Streamer, error) {
	err := r.cfg.ChannelReader.NewRetrieve().WhereKeys(cfg.Keys...).Exec(ctx, nil)
	return &streamer{
		cfg:     cfg,
		addr:    address.Rand(),
		demands: r.demands,
		relay:   r,
	}, err
}

// Flow implements confluence.Flow.
func (r *streamer) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(r.Out)
	ctx.Go(func(ctx context.Context) error {
		r.demands.Acquire(1)
		// We only set demands when we start the streamer, avoiding unnecessary overhead
		// when the streamer is not in use. We also need to make sure we send these
		// demands before we connect to the delta, otherwise, under extreme load we
		// may cause deadlock.
		r.demands.Inlet() <- demand{
			Variant: change.Set,
			Key:     r.addr,
			Value:   Request{Keys: r.cfg.Keys},
		}
		// NOTE: BEYOND THIS POINT THERE IS AN INHERENT RISK OF DEADLOCKING THE RELAY.
		// BE CAREFUL WHEN MAKING CHANGES TO THIS SECTION.
		responses, disconnect := r.relay.connectToDelta(defaultBuffer)
		defer func() {
			// Disconnect from the relay and drain the response channel. Important that
			// we do this before updating our demands, otherwise we may deadlock.
			disconnect()
			// Tell the tapper that we are no longer requesting any channels.
			r.demands.Inlet() <- demand{Variant: change.Delete, Key: r.addr}
			// If we add this in AttachClosables, it may not be closed at the end of
			// if the caller does not use the confluence.CloseOutputInletsOnExit option, so
			// we explicitly close it here.
			r.demands.Close()
		}()
		if r.cfg.SendOpenAck {
			if err := signal.SendUnderContext(ctx, r.Out.Inlet(), Response{}); err != nil {
				return err
			}
		}
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-r.In.Outlet():
				if !ok {
					return nil
				}
				req.Keys = lo.Uniq(req.Keys)
				r.cfg.Keys = req.Keys
				d := demand{Variant: change.Set, Key: r.addr, Value: req}
				if err := signal.SendUnderContext(ctx, r.demands.Inlet(), d); err != nil {
					return err
				}
			case f := <-responses.Outlet():
				filtered := f.Frame.FilterKeys(r.cfg.Keys)
				// Don't send if the frame is empty.
				if filtered.Empty() {
					continue
				}
				res := Response{Error: f.Error, Frame: f.Frame.FilterKeys(r.cfg.Keys)}
				if err := signal.SendUnderContext(ctx, r.Out.Inlet(), res); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}
