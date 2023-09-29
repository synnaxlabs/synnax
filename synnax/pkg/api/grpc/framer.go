// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import (
	"context"
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/gen/proto/go/apiv1"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/telem/telempb"
	"google.golang.org/grpc"
)

type (
	frameWriterRequestTranslator    struct{}
	frameWriterResponseTranslator   struct{}
	frameIteratorRequestTranslator  struct{}
	frameIteratorResponseTranslator struct{}
	frameStreamerRequestTranslator  struct{}
	frameStreamerResponseTranslator struct{}
	writerServer                    = fgrpc.StreamServerCore[
		api.FrameWriterRequest,
		*gapi.FrameWriterRequest,
		api.FrameWriterResponse,
		*gapi.FrameWriterResponse,
	]
	iteratorServer = fgrpc.StreamServerCore[
		api.FrameIteratorRequest,
		*gapi.FrameIteratorRequest,
		api.FrameIteratorResponse,
		*gapi.FrameIteratorResponse,
	]
	streamerServer = fgrpc.StreamServerCore[
		api.FrameStreamerRequest,
		*gapi.FrameStreamerRequest,
		api.FrameStreamerResponse,
		*gapi.FrameStreamerResponse,
	]
)

var (
	_ fgrpc.Translator[api.FrameWriterRequest, *gapi.FrameWriterRequest]       = (*frameWriterRequestTranslator)(nil)
	_ fgrpc.Translator[api.FrameWriterResponse, *gapi.FrameWriterResponse]     = (*frameWriterResponseTranslator)(nil)
	_ fgrpc.Translator[api.FrameIteratorRequest, *gapi.FrameIteratorRequest]   = (*frameIteratorRequestTranslator)(nil)
	_ fgrpc.Translator[api.FrameIteratorResponse, *gapi.FrameIteratorResponse] = (*frameIteratorResponseTranslator)(nil)
	_ fgrpc.Translator[api.FrameStreamerRequest, *gapi.FrameStreamerRequest]   = (*frameStreamerRequestTranslator)(nil)
	_ fgrpc.Translator[api.FrameStreamerResponse, *gapi.FrameStreamerResponse] = (*frameStreamerResponseTranslator)(nil)
	_ gapi.FrameServiceServer                                                  = (*framerServer)(nil)
)

func translateFrameForward(f api.Frame) *gapi.Frame {
	return &gapi.Frame{
		Keys:   translateChannelKeysForward(f.Keys),
		Series: telempb.TranslateManySeriesForward(f.Series),
	}
}

func translateFrameBackward(f *gapi.Frame) api.Frame {
	return api.Frame{
		Keys:   translateChannelKeysBackward(f.Keys),
		Series: telempb.TranslateManySeriesBackward(f.Series),
	}
}

func (t frameWriterRequestTranslator) Forward(
	_ context.Context,
	msg api.FrameWriterRequest,
) (*gapi.FrameWriterRequest, error) {
	return &gapi.FrameWriterRequest{
		Command: int32(msg.Command),
		Config: &gapi.FrameWriterConfig{
			Keys:  translateChannelKeysForward(msg.Config.Keys),
			Start: int64(msg.Config.Start),
		},
		Frame: translateFrameForward(msg.Frame),
	}, nil
}

func (t frameWriterRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.FrameWriterRequest,
) (api.FrameWriterRequest, error) {
	return api.FrameWriterRequest{
		Command: writer.Command(msg.Command),
		Config: api.FrameWriterConfig{
			Keys:  translateChannelKeysBackward(msg.Config.Keys),
			Start: telem.TimeStamp(msg.Config.Start),
		},
		Frame: translateFrameBackward(msg.Frame),
	}, nil
}

func (t frameWriterResponseTranslator) Forward(
	_ context.Context,
	msg api.FrameWriterResponse,
) (*gapi.FrameWriterResponse, error) {
	return &gapi.FrameWriterResponse{
		Command: int32(msg.Command),
		Ack:     msg.Ack,
		Counter: int32(msg.SeqNum),
		NodeKey: int32(msg.NodeKey),
		Error:   fgrpc.EncodeError(msg.Error),
		End:     int64(msg.End),
	}, nil
}

func (t frameWriterResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.FrameWriterResponse,
) (api.FrameWriterResponse, error) {
	return api.FrameWriterResponse{
		Command: writer.Command(msg.Command),
		Ack:     msg.Ack,
		SeqNum:  int(msg.Counter),
		NodeKey: core.NodeKey(msg.NodeKey),
		Error:   fgrpc.DecodeError(msg.Error),
		End:     telem.TimeStamp(msg.End),
	}, nil
}

func (t frameIteratorRequestTranslator) Forward(
	_ context.Context,
	msg api.FrameIteratorRequest,
) (*gapi.FrameIteratorRequest, error) {
	return &gapi.FrameIteratorRequest{
		Command: int32(msg.Command),
		Span:    int64(msg.Span),
		Range:   telempb.TranslateTimeRangeForward(msg.Bounds),
		Keys:    translateChannelKeysForward(msg.Keys),
		Stamp:   int64(msg.Stamp),
	}, nil
}

func (t frameIteratorRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.FrameIteratorRequest,
) (api.FrameIteratorRequest, error) {
	return api.FrameIteratorRequest{
		Command: iterator.Command(msg.Command),
		Span:    telem.TimeSpan(msg.Span),
		Bounds:  telempb.TranslateTimeRangeBackward(msg.Range),
		Keys:    translateChannelKeysBackward(msg.Keys),
		Stamp:   telem.TimeStamp(msg.Stamp),
	}, nil
}

func (t frameIteratorResponseTranslator) Forward(
	_ context.Context,
	msg api.FrameIteratorResponse,
) (*gapi.FrameIteratorResponse, error) {
	return &gapi.FrameIteratorResponse{
		Variant: int32(msg.Variant),
		Command: int32(msg.Command),
		NodeKey: int32(msg.NodeKey),
		Ack:     msg.Ack,
		SeqNum:  int32(msg.SeqNum),
		Frame:   translateFrameForward(msg.Frame),
		Error:   fgrpc.EncodeError(msg.Error),
	}, nil
}

func (t frameIteratorResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.FrameIteratorResponse,
) (api.FrameIteratorResponse, error) {
	return api.FrameIteratorResponse{
		Variant: iterator.ResponseVariant(msg.Variant),
		Command: iterator.Command(msg.Command),
		NodeKey: core.NodeKey(msg.NodeKey),
		Ack:     msg.Ack,
		SeqNum:  int(msg.SeqNum),
		Frame:   translateFrameBackward(msg.Frame),
		Error:   fgrpc.DecodeError(msg.Error),
	}, nil
}

func (t frameStreamerRequestTranslator) Forward(
	_ context.Context,
	msg api.FrameStreamerRequest,
) (*gapi.FrameStreamerRequest, error) {
	return &gapi.FrameStreamerRequest{
		Start: int64(msg.Start),
		Keys:  translateChannelKeysForward(msg.Keys),
	}, nil
}

func (t frameStreamerRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.FrameStreamerRequest,
) (api.FrameStreamerRequest, error) {
	return api.FrameStreamerRequest{
		Start: telem.TimeStamp(msg.Start),
		Keys:  translateChannelKeysBackward(msg.Keys),
	}, nil
}

func (t frameStreamerResponseTranslator) Forward(
	_ context.Context,
	msg api.FrameStreamerResponse,
) (*gapi.FrameStreamerResponse, error) {
	return &gapi.FrameStreamerResponse{
		Frame: translateFrameForward(msg.Frame),
		Error: fgrpc.EncodeError(msg.Error),
	}, nil
}

func (t frameStreamerResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.FrameStreamerResponse,
) (api.FrameStreamerResponse, error) {
	return api.FrameStreamerResponse{
		Frame: translateFrameBackward(msg.Frame),
		Error: fgrpc.DecodeError(msg.Error),
	}, nil
}

type framerServer struct {
	writerServer   *writerServer
	iteratorServer *iteratorServer
	streamerServer *streamerServer
}

func (f *framerServer) Report() alamos.Report {
	return f.writerServer.Report()
}

func (f *framerServer) Write(
	server gapi.FrameService_WriteServer,
) error {
	return f.writerServer.Handler(server.Context(), f.writerServer.Server(server))
}

func (f *framerServer) Iterate(
	server gapi.FrameService_IterateServer,
) error {
	return f.iteratorServer.Handler(server.Context(), f.iteratorServer.Server(server))
}

func (f *framerServer) Stream(
	server gapi.FrameService_StreamServer,
) error {
	return f.streamerServer.Handler(server.Context(), f.streamerServer.Server(server))
}

func (f *framerServer) BindTo(server grpc.ServiceRegistrar) {
	gapi.RegisterFrameServiceServer(server, f)
}

func (f *framerServer) Use(middleware ...freighter.Middleware) {
	f.writerServer.Use(middleware...)
	f.iteratorServer.Use(middleware...)
	f.streamerServer.Use(middleware...)
}

func newFramer(a *api.Transport) fgrpc.BindableTransport {
	var s = &framerServer{}
	s.writerServer = &writerServer{
		RequestTranslator:  frameWriterRequestTranslator{},
		ResponseTranslator: frameWriterResponseTranslator{},
		ServiceDesc:        &gapi.FrameService_ServiceDesc,
	}
	s.iteratorServer = &iteratorServer{
		RequestTranslator:  frameIteratorRequestTranslator{},
		ResponseTranslator: frameIteratorResponseTranslator{},
		ServiceDesc:        &gapi.FrameService_ServiceDesc,
	}
	s.streamerServer = &streamerServer{
		RequestTranslator:  frameStreamerRequestTranslator{},
		ResponseTranslator: frameStreamerResponseTranslator{},
		ServiceDesc:        &gapi.FrameService_ServiceDesc,
	}
	a.FrameStreamer = s.streamerServer
	a.FrameWriter = s.writerServer
	a.FrameIterator = s.iteratorServer
	return s
}
