// Copyright 2026 Synnax Labs, Inc.
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
	"go/types"

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	frameWriterRequestTranslator struct {
		codec *codec.Codec
	}
	frameWriterResponseTranslator   struct{}
	frameIteratorRequestTranslator  struct{}
	frameIteratorResponseTranslator struct{}
	frameStreamerRequestTranslator  struct {
		codec *codec.Codec
	}
	frameStreamerResponseTranslator struct {
		codec *codec.Codec
	}
	FrameDeleteRequestTranslator struct{}
	framerWriterServerCore       = fgrpc.StreamServerCore[
		api.FrameWriterRequest,
		*gapi.FrameWriterRequest,
		api.FrameWriterResponse,
		*gapi.FrameWriterResponse,
	]
	frameIteratorServerCore = fgrpc.StreamServerCore[
		api.FrameIteratorRequest,
		*gapi.FrameIteratorRequest,
		api.FrameIteratorResponse,
		*gapi.FrameIteratorResponse,
	]
	frameStreamerServerCore = fgrpc.StreamServerCore[
		api.FrameStreamerRequest,
		*gapi.FrameStreamerRequest,
		api.FrameStreamerResponse,
		*gapi.FrameStreamerResponse,
	]
	frameDeleteServer = fgrpc.UnaryServer[
		api.FrameDeleteRequest,
		*gapi.FrameDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[api.FrameWriterRequest, *gapi.FrameWriterRequest]       = (*frameWriterRequestTranslator)(nil)
	_ fgrpc.Translator[api.FrameWriterResponse, *gapi.FrameWriterResponse]     = (*frameWriterResponseTranslator)(nil)
	_ fgrpc.Translator[api.FrameIteratorRequest, *gapi.FrameIteratorRequest]   = (*frameIteratorRequestTranslator)(nil)
	_ fgrpc.Translator[api.FrameIteratorResponse, *gapi.FrameIteratorResponse] = (*frameIteratorResponseTranslator)(nil)
	_ fgrpc.Translator[api.FrameStreamerRequest, *gapi.FrameStreamerRequest]   = (*frameStreamerRequestTranslator)(nil)
	_ fgrpc.Translator[api.FrameStreamerResponse, *gapi.FrameStreamerResponse] = (*frameStreamerResponseTranslator)(nil)
	_ fgrpc.Translator[api.FrameDeleteRequest, *gapi.FrameDeleteRequest]       = (*FrameDeleteRequestTranslator)(nil)
)

func translateFrameForward(f api.Frame) *telem.PBFrame {
	return &telem.PBFrame{
		Keys:   translateChannelKeysForward(f.KeysSlice()),
		Series: telem.TranslateManySeriesForward(f.SeriesSlice()),
	}
}

func translateFrameBackward(f *telem.PBFrame) api.Frame {
	if f == nil {
		return api.Frame{}
	}
	return frame.NewMulti(
		translateChannelKeysBackward(f.Keys),
		telem.TranslateManySeriesBackward(f.Series),
	)
}

func translateControlSubjectForward(cs control.Subject) *control.ControlSubject {
	return &control.ControlSubject{
		Key:  cs.Key,
		Name: cs.Name,
	}
}

func translateControlSubjectBackward(cs *control.ControlSubject) (of control.Subject) {
	if cs == nil {
		return
	}
	of.Key = cs.Key
	of.Name = cs.Name
	return
}

func (t frameWriterRequestTranslator) Forward(
	ctx context.Context,
	msg api.FrameWriterRequest,
) (*gapi.FrameWriterRequest, error) {
	r := &gapi.FrameWriterRequest{
		Command: int32(msg.Command),
		Config: &gapi.FrameWriterConfig{
			Keys:                     translateChannelKeysForward(msg.Config.Keys),
			Start:                    int64(msg.Config.Start),
			Mode:                     int32(msg.Config.Mode),
			Authorities:              msg.Config.Authorities,
			EnableAutoCommit:         msg.Config.EnableAutoCommit,
			AutoIndexPersistInterval: int64(msg.Config.AutoIndexPersistInterval),
			ControlSubject:           translateControlSubjectForward(msg.Config.ControlSubject),
			ErrOnUnauthorized:        msg.Config.ErrOnUnauthorized,
		},
		Frame: translateFrameForward(msg.Frame),
	}
	var err error
	r.Buffer, err = t.codec.Encode(ctx, msg.Frame)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (t frameWriterRequestTranslator) Backward(
	ctx context.Context,
	msg *gapi.FrameWriterRequest,
) (r api.FrameWriterRequest, err error) {
	if msg == nil {
		return
	}
	r.Command = writer.Command(msg.Command)
	if msg.Config != nil {
		keys := translateChannelKeysBackward(msg.Config.Keys)
		r.Config = api.FrameWriterConfig{
			Keys:                     keys,
			Start:                    telem.TimeStamp(msg.Config.Start),
			Mode:                     writer.Mode(msg.Config.Mode),
			Authorities:              msg.Config.Authorities,
			EnableAutoCommit:         msg.Config.EnableAutoCommit,
			AutoIndexPersistInterval: telem.TimeSpan(msg.Config.AutoIndexPersistInterval),
			ControlSubject:           translateControlSubjectBackward(msg.Config.ControlSubject),
			ErrOnUnauthorized:        msg.Config.ErrOnUnauthorized,
		}
		if err = t.codec.Update(ctx, keys); err != nil {
			return r, err
		}
	}
	r.Frame = translateFrameBackward(msg.Frame)
	if t.codec != nil && len(msg.Buffer) > 0 {
		r.Frame, err = t.codec.Decode(msg.Buffer)
	}
	return r, err
}

func (t frameWriterResponseTranslator) Forward(
	ctx context.Context,
	msg api.FrameWriterResponse,
) (*gapi.FrameWriterResponse, error) {
	return &gapi.FrameWriterResponse{
		Command: int32(msg.Command),
		End:     int64(msg.End),
		Error:   errors.TranslatePayloadForward(msg.Err),
	}, nil
}

func (t frameWriterResponseTranslator) Backward(
	ctx context.Context,
	msg *gapi.FrameWriterResponse,
) (api.FrameWriterResponse, error) {
	return api.FrameWriterResponse{
		Command: writer.Command(msg.Command),
		End:     telem.TimeStamp(msg.End),
		Err:     errors.TranslatePayloadBackward(msg.Error),
	}, nil
}

func (t frameIteratorRequestTranslator) Forward(
	ctx context.Context,
	msg api.FrameIteratorRequest,
) (*gapi.FrameIteratorRequest, error) {
	return &gapi.FrameIteratorRequest{
		Command:   int32(msg.Command),
		Span:      int64(msg.Span),
		Range:     telem.TranslateTimeRangeForward(msg.Bounds),
		Keys:      translateChannelKeysForward(msg.Keys),
		Stamp:     int64(msg.Stamp),
		ChunkSize: msg.ChunkSize,
	}, nil
}

func (t frameIteratorRequestTranslator) Backward(
	ctx context.Context,
	msg *gapi.FrameIteratorRequest,
) (api.FrameIteratorRequest, error) {
	return api.FrameIteratorRequest{
		Command:   iterator.Command(msg.Command),
		Span:      telem.TimeSpan(msg.Span),
		Bounds:    telem.TranslateTimeRangeBackward(msg.Range),
		Keys:      translateChannelKeysBackward(msg.Keys),
		Stamp:     telem.TimeStamp(msg.Stamp),
		ChunkSize: msg.ChunkSize,
	}, nil
}

func (t frameIteratorResponseTranslator) Forward(
	ctx context.Context,
	msg api.FrameIteratorResponse,
) (*gapi.FrameIteratorResponse, error) {
	return &gapi.FrameIteratorResponse{
		Variant: int32(msg.Variant),
		Command: int32(msg.Command),
		NodeKey: int32(msg.NodeKey),
		Ack:     msg.Ack,
		SeqNum:  int32(msg.SeqNum),
		Frame:   translateFrameForward(msg.Frame),
		Error:   fgrpc.EncodeError(ctx, msg.Error, false),
	}, nil
}

func (t frameIteratorResponseTranslator) Backward(
	ctx context.Context,
	msg *gapi.FrameIteratorResponse,
) (api.FrameIteratorResponse, error) {
	return api.FrameIteratorResponse{
		Variant: iterator.ResponseVariant(msg.Variant),
		Command: iterator.Command(msg.Command),
		NodeKey: cluster.NodeKey(msg.NodeKey),
		Ack:     msg.Ack,
		SeqNum:  int(msg.SeqNum),
		Frame:   translateFrameBackward(msg.Frame),
		Error:   fgrpc.DecodeError(ctx, msg.Error),
	}, nil
}

func (t frameStreamerRequestTranslator) Forward(
	ctx context.Context,
	msg api.FrameStreamerRequest,
) (*gapi.FrameStreamerRequest, error) {
	return &gapi.FrameStreamerRequest{
		Keys:             translateChannelKeysForward(msg.Keys),
		DownsampleFactor: int32(msg.DownsampleFactor),
		ThrottleRateHz:   float64(msg.ThrottleRate),
	}, nil
}

func (t frameStreamerRequestTranslator) Backward(
	ctx context.Context,
	msg *gapi.FrameStreamerRequest,
) (api.FrameStreamerRequest, error) {
	rq := api.FrameStreamerRequest{
		Keys:             translateChannelKeysBackward(msg.Keys),
		DownsampleFactor: int(msg.DownsampleFactor),
		ThrottleRate:     telem.Rate(msg.ThrottleRateHz),
	}
	if msg.EnableExperimentalCodec {
		return rq, t.codec.Update(ctx, rq.Keys)
	}
	return rq, nil
}

func (t frameStreamerResponseTranslator) Forward(
	ctx context.Context,
	msg api.FrameStreamerResponse,
) (res *gapi.FrameStreamerResponse, err error) {
	res = &gapi.FrameStreamerResponse{}
	if t.codec.Initialized() {
		res.Buffer, err = t.codec.Encode(ctx, msg.Frame)
		return
	}
	res.Frame = translateFrameForward(msg.Frame)
	return
}

func (t frameStreamerResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.FrameStreamerResponse,
) (api.FrameStreamerResponse, error) {
	return api.FrameStreamerResponse{Frame: translateFrameBackward(msg.Frame)}, nil
}

func (t FrameDeleteRequestTranslator) Forward(
	_ context.Context,
	msg api.FrameDeleteRequest,
) (*gapi.FrameDeleteRequest, error) {
	return &gapi.FrameDeleteRequest{
		Keys:   msg.Keys.Uint32(),
		Names:  msg.Names,
		Bounds: telem.TranslateTimeRangeForward(msg.Bounds),
	}, nil
}

func (t FrameDeleteRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.FrameDeleteRequest,
) (api.FrameDeleteRequest, error) {
	return api.FrameDeleteRequest{
		Keys:   channel.KeysFromUint32(msg.Keys),
		Names:  msg.Names,
		Bounds: telem.TranslateTimeRangeBackward(msg.Bounds),
	}, nil
}

type writerServer struct{ *framerWriterServerCore }

func (f *writerServer) Exec(
	server gapi.FrameWriterService_ExecServer,
) error {
	return f.Handler(server.Context(), server)
}

func (f *writerServer) BindTo(reg grpc.ServiceRegistrar) {
	gapi.RegisterFrameWriterServiceServer(reg, f)
}

type iteratorServer struct{ *frameIteratorServerCore }

func (f *iteratorServer) Exec(
	server gapi.FrameIteratorService_ExecServer,
) error {
	return f.Handler(server.Context(), server)
}

func (f *iteratorServer) BindTo(reg grpc.ServiceRegistrar) {
	gapi.RegisterFrameIteratorServiceServer(reg, f)
}

type streamerServer struct{ *frameStreamerServerCore }

func (f *streamerServer) Exec(
	stream gapi.FrameStreamerService_ExecServer,
) error {
	return f.Handler(stream.Context(), stream)
}

func (f *streamerServer) BindTo(reg grpc.ServiceRegistrar) {
	gapi.RegisterFrameStreamerServiceServer(reg, f)
}

func newFramer(a *api.Transport, channelSvc *channel.Service) fgrpc.BindableTransport {
	var (
		ws = &writerServer{
			framerWriterServerCore: &framerWriterServerCore{
				ResponseTranslator: frameWriterResponseTranslator{},
				CreateTranslators: func() (
					fgrpc.Translator[api.FrameWriterRequest, *gapi.FrameWriterRequest],
					fgrpc.Translator[api.FrameWriterResponse, *gapi.FrameWriterResponse],
				) {
					codec := codec.NewDynamic(channelSvc)
					return frameWriterRequestTranslator{codec: codec}, frameWriterResponseTranslator{}
				},
				ServiceDesc: &gapi.FrameWriterService_ServiceDesc,
			},
		}
		is = &iteratorServer{
			frameIteratorServerCore: &frameIteratorServerCore{
				RequestTranslator:  frameIteratorRequestTranslator{},
				ResponseTranslator: frameIteratorResponseTranslator{},
				ServiceDesc:        &gapi.FrameIteratorService_ServiceDesc,
			},
		}
		ss = &streamerServer{
			frameStreamerServerCore: &frameStreamerServerCore{
				CreateTranslators: func() (fgrpc.Translator[api.FrameStreamerRequest, *gapi.FrameStreamerRequest], fgrpc.Translator[api.FrameStreamerResponse, *gapi.FrameStreamerResponse]) {
					codec := codec.NewDynamic(channelSvc)
					return frameStreamerRequestTranslator{codec: codec}, frameStreamerResponseTranslator{codec: codec}
				},
				ServiceDesc: &gapi.FrameStreamerService_ServiceDesc,
			},
		}
		ds = &frameDeleteServer{
			RequestTranslator: FrameDeleteRequestTranslator{},
			ServiceDesc:       &gapi.FrameDeleteService_ServiceDesc,
		}
	)
	a.FrameStreamer = ss
	a.FrameWriter = ws
	a.FrameIterator = is
	a.FrameDelete = ds
	return fgrpc.CompoundBindableTransport{ws, is, ss}
}
