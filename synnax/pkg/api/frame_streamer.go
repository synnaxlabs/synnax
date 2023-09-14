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
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
	"github.com/sirupsen/logrus"
)

type (
	FrameStreamerRequest  = framer.StreamerRequest
	FrameStreamerResponse = framer.StreamerResponse
	StreamerStream        = freighter.ServerStream[FrameStreamerRequest, FrameStreamerResponse]
)

func (s *FrameService) Stream(ctx context.Context, stream StreamerStream) errors.Typed {
	reader, err := s.openReader(ctx, stream)
	if err.Occurred() {
		return err
	}
	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(s.Instrumentation))
	defer cancel()
	receiver := &freightfluence.Receiver[FrameStreamerRequest]{
		Receiver: stream,
	}
	sender := &freightfluence.TransformSender[FrameStreamerResponse, FrameStreamerResponse]{
		Sender: freighter.SenderNopCloser[FrameStreamerResponse]{StreamSender: stream},
		Transform: func(ctx context.Context, res FrameStreamerResponse) (FrameStreamerResponse, bool, error) {
			if res.Error != nil {
				res.Error = ferrors.Encode(res.Error)
			}
			return res, true, nil
		},
	}
	pipe := plumber.New()
	plumber.SetSegment[FrameStreamerRequest, FrameStreamerResponse](pipe, "reader", reader)
	plumber.SetSink[FrameStreamerResponse](pipe, "sender", sender)
	plumber.SetSource[FrameStreamerRequest](pipe, "receiver", receiver)
	plumber.MustConnect[FrameStreamerResponse](pipe, "reader", "sender", 1)
	plumber.MustConnect[FrameStreamerRequest](pipe, "receiver", "reader", 1)
	pipe.Flow(sCtx, confluence.CloseInletsOnExit())
	end := errors.MaybeUnexpected(sCtx.Wait())
	logrus.Info("Stream ended")
	return end
}

func (s *FrameService) openReader(ctx context.Context, stream StreamerStream) (framer.Streamer, errors.Typed) {
	req, err := stream.Receive()
	if err != nil {
		return nil, errors.Unexpected(err)
	}
	reader, err := s.Internal.NewStreamer(ctx, framer.StreamerConfig{
		Start: req.Start,
		Keys:  req.Keys,
	})
	return reader, errors.MaybeUnexpected(err)
}
