// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package cesium

import (
	"context"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

// StreamerRequest can be used to update the channel set a Streamer subscribes
// to.
type StreamerRequest struct {
	// Channels sets the channels the Streamer subscribes to.
	Channels []core.ChannelKey
}

// StreamerConfig sets the configuration parameters used when opening the Streamer.
type StreamerConfig struct {
	// Channels sets the channels the Streamer subscribes to.
	Channels []core.ChannelKey
}

// StreamerResponse contains a frame representing the series of all subscribed channels.
// This Frame is guaranteed to only contain data for the channels that are currently
// subscribed to.
type StreamerResponse struct {
	// Frame is the frame containing the channel data.
	Frame Frame
}

// Streamer allows the caller to tap into the DB's write pipeline using a confluence
// Segment based interface. To use a Streamer, call DB.NewStreamer with a list
// of channels whose series you'd like to receive. Then, call Streamer.Flow to start
// receiving frames.
//
// Streamer must be used carefully, as it can clog the write pipeline if the caller
// does not receive the incoming frames fast enough. It's recommended that you use a
// buffered channel for the readers output.
//
// Issuing a new StreamerRequest updates the set of channels the stream reader
// subscribes to.
//
// To stop receiving values, simply close the inlet of the streamer. The streamer will then
// gracefully exit and close its output channel.
type Streamer = confluence.Segment[StreamerRequest, StreamerResponse]

// NewStreamer opens a new Streamer using the given configuration. To start
// receiving frames, call Streamer.Flow. The provided context is only used for
// opening the streamer, and cancelling it has no implications after NewStreamer
// returns.
func (db *DB) NewStreamer(_ context.Context, cfg StreamerConfig) (Streamer, error) {
	if db.closed.Load() {
		return nil, errDBClosed
	}
	return &streamer{StreamerConfig: cfg, relay: db.relay}, nil
}

type streamer struct {
	StreamerConfig
	confluence.AbstractLinear[StreamerRequest, StreamerResponse]
	relay *relay
}

var _ Streamer = (*streamer)(nil)

// relayBufferSize is set to 100 to give ample room for the streamer to not block
// writes to the DB.
const relayBufferSize = 100

// Flow implements confluence.Flow.
func (s *streamer) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(s.Out)
	frames, disconnect := s.relay.connect(relayBufferSize)
	// HERE
	sCtx.Go(func(ctx context.Context) error {
		defer disconnect()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-s.In.Outlet():
				if !ok {
					return nil
				}
				s.Channels = req.Channels
			case f := <-frames.Outlet():
				filtered := f.FilterKeys(s.Channels)
				if len(filtered.Keys) == 0 {
					continue
				}
				if err := signal.SendUnderContext(
					ctx,
					s.Out.Inlet(),
					StreamerResponse{Frame: filtered},
				); err != nil {
					return err
				}
			}
		}
	}, o.Signal...)
}
