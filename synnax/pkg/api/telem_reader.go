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
)

type FrameReaderRequest = framer.StreamReaderRequest
type FrameReaderResponse = framer.StreamReaderResponse
type FrameReaderStream = freighter.ServerStream[FrameReaderRequest, FrameReaderResponse]

func (s *TelemService) Read(ctx context.Context, stream FrameReaderStream) errors.Typed {
	reader, err := s.openReader(ctx, stream)
	if err.Occurred() {
		return err
	}
	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(s.Instrumentation))
	defer cancel()
	receiver := &freightfluence.Receiver[FrameReaderRequest]{
		Receiver: stream,
	}
	sender := &freightfluence.TransformSender[FrameReaderResponse, FrameReaderResponse]{
		Sender: freighter.SenderNopCloser[FrameReaderResponse]{StreamSender: stream},
		Transform: func(ctx context.Context, res FrameReaderResponse) (FrameReaderResponse, bool, error) {
			if res.Error != nil {
				res.Error = ferrors.Encode(res.Error)
			}
			return res, true, nil
		},
	}
	pipe := plumber.New()
	plumber.SetSegment[FrameReaderRequest, FrameReaderResponse](pipe, "reader", reader)
	plumber.SetSink[FrameReaderResponse](pipe, "sender", sender)
	plumber.SetSource[FrameReaderRequest](pipe, "receiver", receiver)
	plumber.MustConnect[FrameReaderResponse](pipe, "reader", "sender", 1)
	plumber.MustConnect[FrameReaderRequest](pipe, "receiver", "reader", 1)
	pipe.Flow(sCtx, confluence.CloseInletsOnExit())
	return errors.MaybeUnexpected(sCtx.Wait())
}

func (s *TelemService) openReader(ctx context.Context, stream FrameReaderStream) (framer.StreamReader, errors.Typed) {
	req, err := stream.Receive()
	if err != nil {
		return nil, errors.Unexpected(err)
	}
	reader, err := s.Framer.NewStreamReader(ctx, framer.StreamReaderConfig{
		Start: req.Start,
		Keys:  req.Keys,
	})
	return reader, errors.MaybeUnexpected(err)
}
