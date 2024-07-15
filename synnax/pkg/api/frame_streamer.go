// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package api

import (
	"context"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/signal"
)

type (
	FrameStreamerRequest  = framer.StreamerRequest
	FrameStreamerResponse = framer.StreamerResponse
	StreamerStream        = freighter.ServerStream[FrameStreamerRequest, FrameStreamerResponse]
)

func (s *FrameService) Stream(ctx context.Context, stream StreamerStream) error {
	streamer, err := s.openStreamer(ctx, stream)
	if err != nil {
		return err
	}
	var (
		sCtx, cancel = signal.WithCancel(ctx, signal.WithInstrumentation(s.Instrumentation.Child("frame_streamer")))
		receiver     = &freightfluence.Receiver[FrameStreamerRequest]{Receiver: stream}
		sender       = &freightfluence.TransformSender[FrameStreamerResponse, FrameStreamerResponse]{
			Sender: freighter.SenderNopCloser[FrameStreamerResponse]{StreamSender: stream},
			Transform: func(ctx context.Context, res FrameStreamerResponse) (FrameStreamerResponse, bool, error) {
				if res.Error != nil {
					res.Error = errors.Encode(ctx, res.Error, false)
				}
				return res, true, nil
			},
		}
		pipe = plumber.New()
	)
	defer cancel()
	plumber.SetSegment[FrameStreamerRequest, FrameStreamerResponse](pipe, "streamer", streamer)
	plumber.SetSink[FrameStreamerResponse](pipe, "sender", sender)
	plumber.SetSource[FrameStreamerRequest](pipe, "receiver", receiver)
	plumber.MustConnect[FrameStreamerResponse](pipe, "streamer", "sender", 70)
	plumber.MustConnect[FrameStreamerRequest](pipe, "receiver", "streamer", 70)
	pipe.Flow(sCtx, confluence.CloseInletsOnExit(), confluence.RecoverWithErrOnPanic())
	return sCtx.Wait()
}

func (s *FrameService) openStreamer(ctx context.Context, stream StreamerStream) (framer.Streamer, error) {
	req, err := stream.Receive()
	if err != nil {
		return nil, err
	}
	reader, err := s.Internal.NewStreamer(ctx, framer.StreamerConfig{
		Start: req.Start,
		Keys:  req.Keys,
	})
	return reader, err
}
