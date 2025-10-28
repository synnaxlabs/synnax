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
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/override"
	"github.com/synnaxlabs/x/signal"
	"github.com/synnaxlabs/x/validate"
)

type Streamer = confluence.Segment[Request, Response]

type streamer struct {
	confluence.AbstractLinear[Request, Response]
	addr    address.Address
	demands confluence.Inlet[demand]
	relay   *Relay
	cfg     StreamerConfig
}

// StreamerConfig is the configuration for creating a new streamer.
type StreamerConfig struct {
	// Keys are the list of channels to read from. This slice may be empty, in
	// which case no data will be streamed until a new configuration is provided
	// as a request to the streamer.
	// [OPTIONAL]
	Keys channel.Keys `json:"keys" msgpack:"keys"`
	// SendOpenAck sets whether to send an acknowledgement when the streamer has
	// successfully connected to the relay and is ready to start streaming data.
	// [OPTIONAL] - defaults to false
	SendOpenAck *bool `json:"send_open_ack" msgpack:"send_open_ack"`
}

var (
	_ config.Config[StreamerConfig] = StreamerConfig{}
	// DefaultStreamerConfig is the default configuration for opening a new streamer.
	// This configuration is valid and will create a streamer that does
	// not stream from any channels.
	DefaultStreamerConfig = StreamerConfig{SendOpenAck: config.False()}
)

// Override implements config.Config.
func (c StreamerConfig) Override(other StreamerConfig) StreamerConfig {
	c.Keys = override.Slice(c.Keys, other.Keys)
	c.SendOpenAck = override.Nil(c.SendOpenAck, other.SendOpenAck)
	return c
}

// Validate implements config.Config.
func (c StreamerConfig) Validate() error {
	v := validate.New("streamer_config")
	validate.NotNil(v, "send_open_ack", c.SendOpenAck)
	return v.Error()
}

// NewStreamer opens a new Streamer for consuming real-time telemetry frames from the
// relay. Each subsequent StreamerConfig overrides the parameters specified in the
// previous config. See the StreamerConfig struct for information on required fields.
func (r *Relay) NewStreamer(ctx context.Context, cfgs ...StreamerConfig) (Streamer, error) {
	cfg, err := config.New(DefaultStreamerConfig, cfgs...)
	if err != nil {
		return nil, err
	}
	if err = r.cfg.ChannelReader.NewRetrieve().WhereKeys(cfg.Keys...).Exec(ctx, nil); err != nil {
		return nil, err
	}
	return &streamer{
		cfg:     cfg,
		addr:    address.Rand(),
		demands: r.demands,
		relay:   r,
	}, nil
}

// Flow implements confluence.Flow.
func (s *streamer) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(s.Out)
	ctx.Go(func(ctx context.Context) error {
		s.demands.Acquire(1)
		// We only set demands when we start the streamer, avoiding unnecessary overhead
		// when the streamer is not in use. We also need to make sure we send these
		// demands before we connect to the delta, otherwise, under extreme load we
		// may cause deadlock.
		s.demands.Inlet() <- demand{
			Variant: change.Set,
			Key:     s.addr,
			Value:   Request{Keys: s.cfg.Keys},
		}
		// NOTE: BEYOND THIS POINT THERE IS AN INHERENT RISK OF DEADLOCKING THE RELAY.
		// BE CAREFUL WHEN MAKING CHANGES TO THIS SECTION.
		responses, disconnect := s.relay.connectToDelta(1)
		defer func() {
			// Disconnect from the relay and drain the response channel. Important that
			// we do this before updating our demands, otherwise we may deadlock.
			disconnect()
			// Tell the tapper that we are no longer requesting any channels.
			s.demands.Inlet() <- demand{Variant: change.Delete, Key: s.addr}
			// If we add this in AttachClosables, it may not be closed at the end of
			// if the caller does not use the confluence.CloseOutputInletsOnExit option, so
			// we explicitly close it here.
			s.demands.Close()
		}()
		if *s.cfg.SendOpenAck {
			if err := signal.SendUnderContext(ctx, s.Out.Inlet(), Response{}); err != nil {
				return err
			}
		}
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-s.In.Outlet():
				if !ok {
					return nil
				}
				req.Keys = lo.Uniq(req.Keys)
				s.cfg.Keys = req.Keys
				d := demand{Variant: change.Set, Key: s.addr, Value: req}
				if err := signal.SendUnderContext(ctx, s.demands.Inlet(), d); err != nil {
					return err
				}
			case r := <-responses.Outlet():
				if filtered := r.Frame.KeepKeys(s.cfg.Keys); !filtered.Empty() {
					res := Response{Frame: filtered}
					if err := signal.SendUnderContext(ctx, s.Out.Inlet(), res); err != nil {
						return err
					}
				}
			}
		}
	}, o.Signal...)
}
