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
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/gen/go/v1"
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
	writerServerCore                = fgrpc.StreamServerCore[
		api.FrameWriterRequest,
		*gapi.FrameWriterRequest,
		api.FrameWriterResponse,
		*gapi.FrameWriterResponse,
	]
	iteratorServerCore = fgrpc.StreamServerCore[
		api.FrameIteratorRequest,
		*gapi.FrameIteratorRequest,
		api.FrameIteratorResponse,
		*gapi.FrameIteratorResponse,
	]
	streamServerCore = fgrpc.StreamServerCore[
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
)

func translateFrameForward(f api.Frame) *gapi.Frame {
	return &gapi.Frame{
		Keys:   translateChannelKeysForward(f.Keys),
		Series: telempb.TranslateManySeriesForward(f.Series),
	}
}

func translateFrameBackward(f *gapi.Frame) (of api.Frame) {
	if f == nil {
		return
	}
	of.Keys = translateChannelKeysBackward(f.Keys)
	of.Series = telempb.TranslateManySeriesBackward(f.Series)
	return
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
) (r api.FrameWriterRequest, err error) {
	if msg == nil {
		return
	}
	r.Command = writer.Command(msg.Command)
	if msg.Config != nil {
		r.Config = api.FrameWriterConfig{
			Keys:  translateChannelKeysBackward(msg.Config.Keys),
			Start: telem.TimeStamp(msg.Config.Start),
		}
	}
	r.Frame = translateFrameBackward(msg.Frame)
	return
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

type writerServer struct{ *writerServerCore }

func (f *writerServer) Exec(
	server gapi.FrameWriterService_ExecServer,
) error {
	return f.Handler(server.Context(), f.Server(server))
}

func (f *writerServer) BindTo(reg grpc.ServiceRegistrar) {
	gapi.RegisterFrameWriterServiceServer(reg, f)
}

type iteratorServer struct{ *iteratorServerCore }

func (f *iteratorServer) Exec(
	server gapi.FrameIteratorService_ExecServer,
) error {
	return f.Handler(server.Context(), f.Server(server))
}

func (f *iteratorServer) BindTo(reg grpc.ServiceRegistrar) {
	gapi.RegisterFrameIteratorServiceServer(reg, f)
}

type streamerServer struct{ *streamServerCore }

func (f *streamerServer) Exec(
	server gapi.FrameStreamerService_ExecServer,
) error {
	return f.Handler(server.Context(), f.Server(server))
}

func (f *streamerServer) BindTo(reg grpc.ServiceRegistrar) {
	gapi.RegisterFrameStreamerServiceServer(reg, f)
}

func newFramer(a *api.Transport) fgrpc.BindableTransport {
	var ws = &writerServer{
		writerServerCore: &writerServerCore{
			RequestTranslator:  frameWriterRequestTranslator{},
			ResponseTranslator: frameWriterResponseTranslator{},
			ServiceDesc:        &gapi.FrameWriterService_ServiceDesc,
		},
	}
	var is = &iteratorServer{
		iteratorServerCore: &iteratorServerCore{
			RequestTranslator:  frameIteratorRequestTranslator{},
			ResponseTranslator: frameIteratorResponseTranslator{},
			ServiceDesc:        &gapi.FrameIteratorService_ServiceDesc,
		},
	}
	var ss = &streamerServer{
		streamServerCore: &streamServerCore{
			RequestTranslator:  frameStreamerRequestTranslator{},
			ResponseTranslator: frameStreamerResponseTranslator{},
			ServiceDesc:        &gapi.FrameStreamerService_ServiceDesc,
		},
	}
	a.FrameStreamer = ss
	a.FrameWriter = ws
	a.FrameIterator = is
	return fgrpc.CompoundBindableTransport{ws, is, ss}
}
