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

// StreamReaderRequest can be used to update the channel set a StreamReader subscribes
// to.
type StreamReaderRequest struct {
	// Channels sets the channels the StreamReader subscribes to.
	Channels []core.ChannelKey
}

// StreamReaderConfig sets the configuration parameters used when opening the StreamReader.
type StreamReaderConfig struct {
	// Channels sets the channels the StreamReader subscribes to.
	Channels []core.ChannelKey
}

// StreamReaderResponse contains a frame representing the arrays of all subscribed channels.
// This Frame is guaranteed to only contain data for the channels that are currently
// subscribed to.
type StreamReaderResponse struct {
	// Frame is the frame containing the channel data.
	Frame Frame
}

// StreamReader allows the caller to tap into the DB's write pipeline using a confluence
// Segment based interface. To use a StreamReader, call DB.NewStreamReader with a list
// of channels whose arrays you'd like to receive. Then, call StreamReader.Flow to start
// receiving frames.
//
// StreamReader must be used carefully, as it can clog the write pipeline if the caller
// does not receive the incoming frames fast enough. It's recommended that you use a
// buffered channel for the readers output.
//
// Issuing a new StreamReaderRequest updates the set of channels the stream reader
// subscribes to.
//
// To stop receiving values, simply close the inlet of the reader. The reader will then
// gracefully exit and close its output channel.
type StreamReader = confluence.Segment[StreamReaderRequest, StreamReaderResponse]

// NewStreamReader opens a new StreamReader using the given configuration. To start
// receiving frames, call StreamReader.Flow. The provided context is only used for
// opening the reader, and cancelling it has no implications after NewStreamReader
// returns.
func (db *DB) NewStreamReader(ctx context.Context, cfg StreamReaderConfig) (StreamReader, error) {
	return &streamReader{
		StreamReaderConfig: cfg,
		relay:              db.relay,
	}, nil
}

type streamReader struct {
	StreamReaderConfig
	confluence.AbstractLinear[StreamReaderRequest, StreamReaderResponse]
	relay *relay
}

var _ StreamReader = (*streamReader)(nil)

// Flow implements confluence.Flow.
func (r *streamReader) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(r.Out)
	frames, disconnect := r.relay.connect(1)
	ctx.Go(func(ctx context.Context) error {
		defer disconnect()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req, ok := <-r.In.Outlet():
				if !ok {
					return nil
				}
				r.Channels = req.Channels
			case f := <-frames.Outlet():
				filtered := f.FilterKeys(r.Channels)
				if len(filtered.Keys) != 0 {
					r.Out.Inlet() <- StreamReaderResponse{Frame: filtered}
				}
			}
		}
	}, o.Signal...)
}
