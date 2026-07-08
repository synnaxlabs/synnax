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

// StreamerRequest can be used to update the channel set a Streamer subscribes
// to.
type StreamerRequest struct {
	// Channels sets the channels the Streamer subscribes to.
	Channels []channel.Key
}

// StreamerConfig sets the configuration parameters used when opening the Streamer.
type StreamerConfig struct {
	// Channels sets the channels the Streamer subscribes to.
	Channels []channel.Key
	// OnSuccessfulStart is closed when the Streamer is successfully opened.
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

// Streamer allows the caller to tap into the DB's write pipeline using a confluence
// Segment based interface. To use a Streamer, call DB.NewStreamer with a list of
// channels whose series you'd like to receive. Then, call Streamer.Flow to start
// receiving frames.
//
// Streamer must be used carefully, as it can clog the write pipeline if the caller does
// not receive the incoming frames fast enough. It's recommended that you use a buffered
// channel for the readers output.
//
// Issuing a new StreamerRequest updates the set of channels the stream reader
// subscribes to.
//
// To stop receiving values, simply close the inlet of the streamer. The streamer will
// then gracefully exit and close its output channel.
type Streamer[RQ any, RS any] interface {
	confluence.Segment[RQ, RS]
	// SetChannels replaces the set of channels the Streamer subscribes to. Unlike a
	// StreamerRequest, which is applied asynchronously by the streaming goroutine,
	// the new set applies to every frame written after SetChannels returns, so it
	// can be used as a readiness barrier. Callers should update the channel set
	// through SetChannels or through requests, not both.
	SetChannels([]channel.Key)
}

func passThroughStreamerRequestTranslator(req StreamerRequest) StreamerRequest {
	return req
}

func passThroughStreamerResponseTranslator(res StreamerResponse) StreamerResponse {
	return res
}

// NewStreamer opens a new Streamer using the given configuration. To start receiving
// frames, call Streamer.Flow. The provided context is only used for opening the
// streamer, and cancelling it has no implications after NewStreamer returns.
func (db *DB) NewStreamer(ctx context.Context, cfg StreamerConfig) (Streamer[StreamerRequest, StreamerResponse], error) {
	return NewTranslatedStreamer(
		db,
		cfg,
		passThroughStreamerRequestTranslator,
		passThroughStreamerResponseTranslator,
	)

}

func NewTranslatedStreamer[I any, O any](
	db *DB,
	cfg StreamerConfig,
	translateRequest func(I) StreamerRequest,
	translateResponse func(StreamerResponse) O,
) (Streamer[I, O], error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}
	s := &streamer[I, O]{
		relay:             db.relay,
		translateResponse: translateResponse,
		translateRequest:  translateRequest,
		sendOpenAck:       cfg.SendOpenAck,
	}
	s.SetChannels(cfg.Channels)
	return s, nil
}

type streamer[I any, O any] struct {
	confluence.AbstractLinear[I, O]
	relay             *relay
	translateRequest  func(I) StreamerRequest
	translateResponse func(StreamerResponse) O
	// channels is the currently subscribed channel set. Stored atomically so
	// SetChannels can apply updates synchronously while Flow's delivery loop reads the
	// set on every frame.
	channels atomic.Pointer[[]channel.Key]
	// sendOpenAck indicates whether to emit an empty response once the streamer starts
	// flowing.
	sendOpenAck bool
}

// SetChannels implements Streamer.
func (s *streamer[I, O]) SetChannels(keys []channel.Key) { s.channels.Store(&keys) }

var _ Streamer[StreamerRequest, StreamerResponse] = (*streamer[StreamerRequest, StreamerResponse])(nil)

// Flow implements confluence.Flow.
func (s *streamer[I, O]) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(s.Out)
	frames, disconnect := s.relay.connect()
	sCtx.Go(func(ctx context.Context) error {
		defer disconnect()
		if s.sendOpenAck {
			if err := signal.SendUnderContext(
				ctx,
				s.Out.Inlet(),
				s.translateResponse(StreamerResponse{}),
			); err != nil {
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
				s.SetChannels(s.translateRequest(req).Channels)
			case rf := <-frames.Outlet():
				if filtered := rf.frame.KeepKeys(*s.channels.Load()); !filtered.Empty() {
					if err := signal.SendUnderContext(
						ctx,
						s.Out.Inlet(),
						s.translateResponse(StreamerResponse{Frame: filtered, Group: rf.group}),
					); err != nil {
						return err
					}
				}
			}
		}
	}, o.Signal...)
}
