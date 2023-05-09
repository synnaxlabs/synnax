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

type StreamReaderRequest struct {
	Channels []core.ChannelKey
}

type StreamReaderConfig struct {
	Channels        []core.ChannelKey
	RelayBufferSize int
}

type StreamReaderResponse struct {
	Frame Frame
}

type StreamReader = confluence.Segment[StreamReaderRequest, StreamReaderResponse]

func (db *DB) NewStreamReader(ctx context.Context, cfg StreamReaderConfig) (StreamReader, error) {
	frames, disconnect := db.relay.connect(cfg.RelayBufferSize)
	return &streamReader{
		StreamReaderConfig: cfg,
		relay:              frames,
		disconnect:         disconnect,
	}, nil
}

type streamReader struct {
	StreamReaderConfig
	confluence.AbstractLinear[StreamReaderRequest, StreamReaderResponse]
	relay      confluence.Outlet[Frame]
	disconnect func()
}

func (r *streamReader) Flow(ctx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(r.Out)
	ctx.Go(func(ctx context.Context) error {
		defer r.disconnect()
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case req := <-r.In.Outlet():
				r.Channels = req.Channels
			case f, ok := <-r.relay.Outlet():
				if !ok {
					return nil
				}
				filtered := f.FilterKeys(r.Channels)
				if len(filtered.Keys) != 0 {
					r.Out.Inlet() <- StreamReaderResponse{Frame: filtered}
				}
			}
		}
	}, o.Signal...)
}
