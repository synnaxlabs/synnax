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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/ferrors"
	"github.com/synnaxlabs/freighter/freightfluence"
	"github.com/synnaxlabs/synnax/pkg/api/errors"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/telem"
	"github.com/synnaxlabs/x/confluence"
	"github.com/synnaxlabs/x/confluence/plumber"
	"github.com/synnaxlabs/x/signal"
)

type TelemService struct {
	alamos.Instrumentation
	authProvider
	Framer *framer.Service
	Telem  *telem.Service
}

type LiveReaderRequest struct {
	Config telem.LiveReaderConfig
}
type LiveReaderResponse = telem.LiveReadResponse

func NewTelemService(p Provider) *TelemService {
	return &TelemService{
		Instrumentation: p.Instrumentation,
		Framer:          p.Config.Framer,
		authProvider:    p.auth,
	}
}

type LiveReaderStream = freighter.ServerStream[LiveReaderRequest, telem.LiveReadResponse]

func (s *TelemService) LiveRead(
	ctx context.Context,
	stream LiveReaderStream,
) errors.Typed {
	reader, err := s.openLiveReader(ctx, stream)
	if err.Occurred() {
		return err
	}

	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(s.Instrumentation))
	defer cancel()

	receiver := &freightfluence.TransformReceiver[telem.LiveReadRequest, LiveReaderRequest]{}
	receiver.Transform = func(ctx context.Context, req LiveReaderRequest) (telem.LiveReadRequest, bool, error) {
		return telem.LiveReadRequest{Keys: req.Config.Keys}, true, nil
	}
	sender := &freightfluence.TransformSender[telem.LiveReadResponse, telem.LiveReadResponse]{}
	sender.Transform = func(ctx context.Context, res telem.LiveReadResponse) (telem.LiveReadResponse, bool, error) {
		if res.Err != nil {
			res.Err = ferrors.Encode(res.Err)
		}
		return res, true, nil
	}

	pipe := plumber.New()
	plumber.SetSegment(pipe, "reader", reader)
	plumber.SetSink(pipe, "sender", sender)
	plumber.SetSource[telem.LiveReadRequest](pipe, "receiver", receiver)
	pipe.Flow(sCtx, confluence.CloseInletsOnExit())

	return errors.Unexpected(sCtx.Wait())
}

func (s *TelemService) openLiveReader(
	ctx context.Context,
	stream LiveReaderStream,
) (telem.LiveReader, errors.Typed) {
	req, err := stream.Receive()
	if err != nil {
		return nil, errors.Unexpected(err)
	}
	reader, err := s.Telem.NewLiveReader(ctx, req.Config)
	return reader, errors.MaybeUnexpected(err)
}

type LiveWriterRequest = framer.WriteRequest
type LiveWriterResponse = framer.WriteResponse
type LiveWriterStream = freighter.ServerStream[framer.WriteRequest, framer.WriteResponse]

func (s *TelemService) LiveWrite(
	ctx context.Context,
	stream LiveWriterStream,
) errors.Typed {
	sCtx, cancel := signal.WithCancel(ctx, signal.WithInstrumentation(s.Instrumentation))
	defer cancel()

	writer := s.Telem.NewLiveWriter()

	receiver := &freightfluence.Receiver[framer.WriteRequest]{Receiver: stream}
	sender := &freightfluence.Sender[framer.WriteResponse]{
		Sender: freighter.SenderNopCloser[framer.WriteResponse]{StreamSender: stream},
	}
	pipe := plumber.New()
	plumber.SetSegment(pipe, "writer", writer)
	plumber.SetSink(pipe, "sender", sender)
	plumber.SetSource[framer.WriteRequest](pipe, "receiver", receiver)
	pipe.Flow(sCtx, confluence.CloseInletsOnExit())

	return errors.MaybeUnexpected(sCtx.Wait())
}
