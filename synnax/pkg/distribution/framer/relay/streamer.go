// Copyright 2023 Synnax Labs, Inc.
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
	keys    channel.Keys
	relay   *Relay
}

type StreamerConfig struct {
	Keys channel.Keys
}

func (r *Relay) NewReader(_ context.Context, cfg StreamerConfig) (Streamer, error) {
	rd := &streamer{
		keys:    cfg.Keys,
		addr:    address.Rand(),
		demands: r.peerDemands,
		relay:   r,
	}
	r.peerDemands.Inlet() <- demand{
		Variant: change.Set,
		Key:     rd.addr,
		Value:   Request{Keys: cfg.Keys},
	}
	return rd, nil
}

// Flow implements confluence.Flow.
func (r *streamer) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(r.Out)
	responses, disconnect := r.relay.connect(1)
	ctx.Go(func(ctx context.Context) error {
		defer disconnect()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-r.In.Outlet():
				if !ok {
					r.demands.Inlet() <- demand{Variant: change.Delete, Key: r.addr}
					return nil
				}
				r.keys = req.Keys
				r.demands.Inlet() <- demand{Variant: change.Set, Key: r.addr, Value: req}
			case f := <-responses.Outlet():
				filtered := f.Frame.FilterKeys(r.keys)
				if len(filtered.Keys) != 0 {
					r.Out.Inlet() <- Response{
						Error: f.Error,
						Frame: f.Frame.FilterKeys(r.keys),
					}
				}
			}
		}
	}, o.Signal...)
}
