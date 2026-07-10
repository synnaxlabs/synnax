// Copyright 2026 Synnax Labs, Inc.
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
	"sync/atomic"

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

// StreamerConfig sets the configuration parameters used when opening the Streamer.
type StreamerConfig struct {
	// Channels sets the channels the Streamer initially subscribes to.
	Channels []channel.Key
	// SendOpenAck emits an empty response once the streamer starts flowing.
	SendOpenAck bool
}

// StreamerResponse contains a frame representing the series of all subscribed channels.
// This Frame is guaranteed to only contain data for the channels that are currently
// subscribed to.
type StreamerResponse struct {
	// Frame is the frame containing the channel data.
	Frame Frame
	// Group is the group identifier of the writer that produced this frame.
	Group uint32
}

// Streamer allows the caller to tap into the DB's write pipeline, emitting a response
// for every frame written to its subscribed channels. To use a Streamer, call
// DB.NewStreamer with a list of channels whose series you'd like to receive. Then, call
// Streamer.Flow to start receiving frames.
//
// Streamer must be used carefully, as it can clog the write pipeline if the caller does
// not receive the incoming frames fast enough. It's recommended that you use a buffered
// channel for the readers output.
//
// To stop the Streamer, cancel the signal context passed to Flow.
type Streamer[O any] interface {
	confluence.Source[O]
	// SetChannels replaces the set of channels the Streamer subscribes to. The new set
	// applies to every frame written after SetChannels returns, so it can be used as a
	// readiness barrier.
	SetChannels([]channel.Key)
}

// NewStreamer opens a new Streamer using the given configuration. To start receiving
// frames, call Streamer.Flow.
func (db *DB) NewStreamer(cfg StreamerConfig) (Streamer[StreamerResponse], error) {
	return NewTranslatedStreamer(db, cfg, func(res StreamerResponse) StreamerResponse {
		return res
	})
}

// NewTranslatedStreamer opens a new Streamer whose responses are translated to the
// given output type before being sent to the streamer's outlet. It allows callers to
// consume the streamer's output in their own type without an intermediate translation
// segment.
func NewTranslatedStreamer[O any](
	db *DB,
	cfg StreamerConfig,
	translate func(StreamerResponse) O,
) (Streamer[O], error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}
	s := &streamer[O]{relay: db.relay, translate: translate, sendOpenAck: cfg.SendOpenAck}
	s.SetChannels(cfg.Channels)
	return s, nil
}

type streamer[O any] struct {
	confluence.AbstractUnarySource[O]
	relay     *relay
	translate func(StreamerResponse) O
	// channels is the currently subscribed channel set. Stored atomically so
	// SetChannels can apply updates synchronously while Flow's delivery loop reads the
	// set on every frame.
	channels atomic.Pointer[[]channel.Key]
	// sendOpenAck indicates whether to emit an empty response once the streamer starts
	// flowing.
	sendOpenAck bool
}

var _ Streamer[StreamerResponse] = (*streamer[StreamerResponse])(nil)

// SetChannels implements Streamer.
func (s *streamer[O]) SetChannels(keys []channel.Key) { s.channels.Store(&keys) }

// Flow implements confluence.Flow.
func (s *streamer[O]) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(s.Out)
	frames, disconnect := s.relay.connect()
	sCtx.Go(func(ctx context.Context) error {
		defer disconnect()
		if s.sendOpenAck {
			if err := signal.SendUnderContext(
				ctx,
				s.Out.Inlet(),
				s.translate(StreamerResponse{}),
			); err != nil {
				return err
			}
		}
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case rf := <-frames.Outlet():
				if filtered := rf.frame.KeepKeys(*s.channels.Load()); !filtered.Empty() {
					if err := signal.SendUnderContext(
						ctx,
						s.Out.Inlet(),
						s.translate(StreamerResponse{Frame: filtered, Group: rf.group}),
					); err != nil {
						return err
					}
				}
			}
		}
	}, o.Signal...)
}
