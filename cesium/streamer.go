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

	"github.com/synnaxlabs/cesium/internal/channel"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/signal"
)

// StreamerRequest can be used to update the channel set a Streamer subscribes to.
type StreamerRequest struct {
	// Channels sets the channels the Streamer subscribes to. Unlike the initial
	// StreamerConfig.Channels, the set is applied as a pure filter: keys that do not
	// exist in the database yield no error and no frames.
	Channels []channel.Key
}

// StreamerConfig sets the configuration parameters used when opening the Streamer.
type StreamerConfig struct {
	// Channels sets the channels the Streamer initially subscribes to. Every channel
	// must exist in the database.
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
// subscribes to. A request applies to every frame written after the request is sent, so
// it can be used as a readiness barrier.
//
// To stop receiving values, simply close the inlet of the streamer. The streamer will
// then gracefully exit and close its output channel.
type Streamer[RQ any, RS any] = confluence.Segment[RQ, RS]

func passThroughStreamerRequestTranslator(req StreamerRequest) StreamerRequest {
	return req
}

func passThroughStreamerResponseTranslator(res StreamerResponse) StreamerResponse {
	return res
}

// NewStreamer opens a new Streamer using the given configuration. To start receiving
// frames, call Streamer.Flow. The provided context is only used for opening the
// streamer, and cancelling it has no implications after NewStreamer returns. It returns
// ErrChannelNotFound if any of the configured channels do not exist in the database.
func (db *DB) NewStreamer(ctx context.Context, cfg StreamerConfig) (Streamer[StreamerRequest, StreamerResponse], error) {
	return NewTranslatedStreamer(
		db,
		cfg,
		passThroughStreamerRequestTranslator,
		passThroughStreamerResponseTranslator,
	)
}

// NewTranslatedStreamer opens a new Streamer whose requests and responses are
// translated to the given input and output types. It allows callers to consume the
// streamer in their own types without intermediate translation segments. It returns
// ErrChannelNotFound if any of the configured channels do not exist in the database.
func NewTranslatedStreamer[I any, O any](
	db *DB,
	cfg StreamerConfig,
	translateRequest func(I) StreamerRequest,
	translateResponse func(StreamerResponse) O,
) (Streamer[I, O], error) {
	if db.closed.Load() {
		return nil, ErrDBClosed
	}
	db.mu.RLock()
	defer db.mu.RUnlock()
	for _, key := range cfg.Channels {
		if _, ok := db.mu.dbs.unary[key]; ok {
			continue
		}
		if _, ok := db.mu.dbs.virtual[key]; ok {
			continue
		}
		return nil, channel.NewNotFoundError(key)
	}
	return &streamer[I, O]{
		StreamerConfig:    cfg,
		relay:             db.relay,
		translateRequest:  translateRequest,
		translateResponse: translateResponse,
	}, nil
}

type streamer[I any, O any] struct {
	confluence.AbstractLinear[I, O]
	relay             *relay
	translateRequest  func(I) StreamerRequest
	translateResponse func(StreamerResponse) O
	StreamerConfig
}

var _ Streamer[StreamerRequest, StreamerResponse] = (*streamer[StreamerRequest, StreamerResponse])(nil)

// Flow implements confluence.Flow.
func (s *streamer[I, O]) Flow(sCtx signal.Context, opts ...confluence.Option) {
	o := confluence.NewOptions(opts)
	o.AttachClosables(s.Out)
	frames, disconnect := s.relay.connect()
	sCtx.Go(func(ctx context.Context) error {
		defer disconnect()
		if s.SendOpenAck {
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
				s.Channels = s.translateRequest(req).Channels
			case rf := <-frames.Outlet():
				// Apply any queued subscription updates before filtering. A request
				// sent before a frame was written is guaranteed to already be in the
				// inlet, so draining here makes requests a readiness barrier: the new
				// channel set applies to every frame written after the request send
				// completes, regardless of which select case fires first.
				if ok := s.drainRequests(); !ok {
					return nil
				}
				if filtered := rf.frame.KeepKeys(s.Channels); !filtered.Empty() {
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

// drainRequests applies all pending subscription updates, returning false if the
// request inlet has been closed and the streamer should exit.
func (s *streamer[I, O]) drainRequests() bool {
	for {
		select {
		case req, ok := <-s.In.Outlet():
			if !ok {
				return false
			}
			s.Channels = s.translateRequest(req).Channels
		default:
			return true
		}
	}
}
