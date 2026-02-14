// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apiframer "github.com/synnaxlabs/synnax/pkg/api/framer"
	channelgrpc "github.com/synnaxlabs/synnax/pkg/api/grpc/channel"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
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
	writerRequestTranslator struct {
		codec *codec.Codec
	}
	writerResponseTranslator   struct{}
	iteratorRequestTranslator  struct{}
	iteratorResponseTranslator struct{}
	streamerRequestTranslator  struct {
		codec *codec.Codec
	}
	streamerResponseTranslator struct {
		codec *codec.Codec
	}
	deleteRequestTranslator struct{}
	writerServerCore        = fgrpc.StreamServerCore[
		apiframer.WriterRequest,
		*gapi.FrameWriterRequest,
		apiframer.WriterResponse,
		*gapi.FrameWriterResponse,
	]
	iteratorServerCore = fgrpc.StreamServerCore[
		apiframer.IteratorRequest,
		*gapi.FrameIteratorRequest,
		apiframer.IteratorResponse,
		*gapi.FrameIteratorResponse,
	]
	streamerServerCore = fgrpc.StreamServerCore[
		apiframer.StreamerRequest,
		*gapi.FrameStreamerRequest,
		apiframer.StreamerResponse,
		*gapi.FrameStreamerResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apiframer.DeleteRequest,
		*gapi.FrameDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[apiframer.WriterRequest, *gapi.FrameWriterRequest]       = (*writerRequestTranslator)(nil)
	_ fgrpc.Translator[apiframer.WriterResponse, *gapi.FrameWriterResponse]     = (*writerResponseTranslator)(nil)
	_ fgrpc.Translator[apiframer.IteratorRequest, *gapi.FrameIteratorRequest]   = (*iteratorRequestTranslator)(nil)
	_ fgrpc.Translator[apiframer.IteratorResponse, *gapi.FrameIteratorResponse] = (*iteratorResponseTranslator)(nil)
	_ fgrpc.Translator[apiframer.StreamerRequest, *gapi.FrameStreamerRequest]   = (*streamerRequestTranslator)(nil)
	_ fgrpc.Translator[apiframer.StreamerResponse, *gapi.FrameStreamerResponse] = (*streamerResponseTranslator)(nil)
	_ fgrpc.Translator[apiframer.DeleteRequest, *gapi.FrameDeleteRequest]       = (*deleteRequestTranslator)(nil)
)

func translateForward(f apiframer.Frame) *telem.PBFrame {
	return &telem.PBFrame{
		Keys:   channelgrpc.TranslateKeysForward(f.KeysSlice()),
		Series: telem.TranslateManySeriesForward(f.SeriesSlice()),
	}
}

func translateBackward(f *telem.PBFrame) apiframer.Frame {
	if f == nil {
		return apiframer.Frame{}
	}
	return frame.NewMulti(
		channelgrpc.TranslateKeysBackward(f.Keys),
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

func (t writerRequestTranslator) Forward(
	ctx context.Context,
	msg apiframer.WriterRequest,
) (*gapi.FrameWriterRequest, error) {
	r := &gapi.FrameWriterRequest{
		Command: int32(msg.Command),
		Config: &gapi.FrameWriterConfig{
			Keys:                     channelgrpc.TranslateKeysForward(msg.Config.Keys),
			Start:                    int64(msg.Config.Start),
			Mode:                     int32(msg.Config.Mode),
			Authorities:              msg.Config.Authorities,
			EnableAutoCommit:         msg.Config.EnableAutoCommit,
			AutoIndexPersistInterval: int64(msg.Config.AutoIndexPersistInterval),
			ControlSubject:           translateControlSubjectForward(msg.Config.ControlSubject),
			ErrOnUnauthorized:        msg.Config.ErrOnUnauthorized,
		},
		Frame: translateForward(msg.Frame),
	}
	var err error
	r.Buffer, err = t.codec.Encode(ctx, msg.Frame)
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (t writerRequestTranslator) Backward(
	ctx context.Context,
	msg *gapi.FrameWriterRequest,
) (r apiframer.WriterRequest, err error) {
	if msg == nil {
		return
	}
	r.Command = writer.Command(msg.Command)
	if msg.Config != nil {
		keys := channelgrpc.TranslateKeysBackward(msg.Config.Keys)
		r.Config = apiframer.WriterConfig{
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
	r.Frame = translateBackward(msg.Frame)
	if t.codec != nil && len(msg.Buffer) > 0 {
		r.Frame, err = t.codec.Decode(msg.Buffer)
	}
	return r, err
}

func (t writerResponseTranslator) Forward(
	ctx context.Context,
	msg apiframer.WriterResponse,
) (*gapi.FrameWriterResponse, error) {
	return &gapi.FrameWriterResponse{
		Command: int32(msg.Command),
		End:     int64(msg.End),
		Error:   errors.TranslatePayloadForward(msg.Err),
	}, nil
}

func (t writerResponseTranslator) Backward(
	ctx context.Context,
	msg *gapi.FrameWriterResponse,
) (apiframer.WriterResponse, error) {
	return apiframer.WriterResponse{
		Command: writer.Command(msg.Command),
		End:     telem.TimeStamp(msg.End),
		Err:     errors.TranslatePayloadBackward(msg.Error),
	}, nil
}

func (t iteratorRequestTranslator) Forward(
	ctx context.Context,
	msg apiframer.IteratorRequest,
) (*gapi.FrameIteratorRequest, error) {
	return &gapi.FrameIteratorRequest{
		Command:   int32(msg.Command),
		Span:      int64(msg.Span),
		Range:     telem.TranslateTimeRangeForward(msg.Bounds),
		Keys:      channelgrpc.TranslateKeysForward(msg.Keys),
		Stamp:     int64(msg.Stamp),
		ChunkSize: msg.ChunkSize,
	}, nil
}

func (t iteratorRequestTranslator) Backward(
	ctx context.Context,
	msg *gapi.FrameIteratorRequest,
) (apiframer.IteratorRequest, error) {
	return apiframer.IteratorRequest{
		Command:   iterator.Command(msg.Command),
		Span:      telem.TimeSpan(msg.Span),
		Bounds:    telem.TranslateTimeRangeBackward(msg.Range),
		Keys:      channelgrpc.TranslateKeysBackward(msg.Keys),
		Stamp:     telem.TimeStamp(msg.Stamp),
		ChunkSize: msg.ChunkSize,
	}, nil
}

func (t iteratorResponseTranslator) Forward(
	ctx context.Context,
	msg apiframer.IteratorResponse,
) (*gapi.FrameIteratorResponse, error) {
	return &gapi.FrameIteratorResponse{
		Variant: int32(msg.Variant),
		Command: int32(msg.Command),
		NodeKey: int32(msg.NodeKey),
		Ack:     msg.Ack,
		SeqNum:  int32(msg.SeqNum),
		Frame:   translateForward(msg.Frame),
		Error:   fgrpc.EncodeError(ctx, msg.Error, false),
	}, nil
}

func (t iteratorResponseTranslator) Backward(
	ctx context.Context,
	msg *gapi.FrameIteratorResponse,
) (apiframer.IteratorResponse, error) {
	return apiframer.IteratorResponse{
		Variant: iterator.ResponseVariant(msg.Variant),
		Command: iterator.Command(msg.Command),
		NodeKey: cluster.NodeKey(msg.NodeKey),
		Ack:     msg.Ack,
		SeqNum:  int(msg.SeqNum),
		Frame:   translateBackward(msg.Frame),
		Error:   fgrpc.DecodeError(ctx, msg.Error),
	}, nil
}

func (t streamerRequestTranslator) Forward(
	ctx context.Context,
	msg apiframer.StreamerRequest,
) (*gapi.FrameStreamerRequest, error) {
	return &gapi.FrameStreamerRequest{
		Keys:             channelgrpc.TranslateKeysForward(msg.Keys),
		DownsampleFactor: int32(msg.DownsampleFactor),
		ThrottleRateHz:   float64(msg.ThrottleRate),
	}, nil
}

func (t streamerRequestTranslator) Backward(
	ctx context.Context,
	msg *gapi.FrameStreamerRequest,
) (apiframer.StreamerRequest, error) {
	rq := apiframer.StreamerRequest{
		Keys:             channelgrpc.TranslateKeysBackward(msg.Keys),
		DownsampleFactor: int(msg.DownsampleFactor),
		ThrottleRate:     telem.Rate(msg.ThrottleRateHz),
	}
	if msg.EnableExperimentalCodec {
		return rq, t.codec.Update(ctx, rq.Keys)
	}
	return rq, nil
}

func (t streamerResponseTranslator) Forward(
	ctx context.Context,
	msg apiframer.StreamerResponse,
) (res *gapi.FrameStreamerResponse, err error) {
	res = &gapi.FrameStreamerResponse{}
	if t.codec.Initialized() {
		res.Buffer, err = t.codec.Encode(ctx, msg.Frame)
		return
	}
	res.Frame = translateForward(msg.Frame)
	return
}

func (t streamerResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.FrameStreamerResponse,
) (apiframer.StreamerResponse, error) {
	return apiframer.StreamerResponse{Frame: translateBackward(msg.Frame)}, nil
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	msg apiframer.DeleteRequest,
) (*gapi.FrameDeleteRequest, error) {
	return &gapi.FrameDeleteRequest{
		Keys:   msg.Keys.Uint32(),
		Names:  msg.Names,
		Bounds: telem.TranslateTimeRangeForward(msg.Bounds),
	}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.FrameDeleteRequest,
) (apiframer.DeleteRequest, error) {
	return apiframer.DeleteRequest{
		Keys:   distchannel.KeysFromUint32(msg.Keys),
		Names:  msg.Names,
		Bounds: telem.TranslateTimeRangeBackward(msg.Bounds),
	}, nil
}

type writerServer struct{ *writerServerCore }

func (f *writerServer) Exec(
	server gapi.FrameWriterService_ExecServer,
) error {
	return f.Handler(server.Context(), server)
}

func (f *writerServer) BindTo(reg grpc.ServiceRegistrar) {
	gapi.RegisterFrameWriterServiceServer(reg, f)
}

type iteratorServer struct{ *iteratorServerCore }

func (f *iteratorServer) Exec(
	server gapi.FrameIteratorService_ExecServer,
) error {
	return f.Handler(server.Context(), server)
}

func (f *iteratorServer) BindTo(reg grpc.ServiceRegistrar) {
	gapi.RegisterFrameIteratorServiceServer(reg, f)
}

type streamerServer struct{ *streamerServerCore }

func (f *streamerServer) Exec(
	stream gapi.FrameStreamerService_ExecServer,
) error {
	return f.Handler(stream.Context(), stream)
}

func (f *streamerServer) BindTo(reg grpc.ServiceRegistrar) {
	gapi.RegisterFrameStreamerServiceServer(reg, f)
}

func New(a *api.Transport, channelSvc *distchannel.Service) fgrpc.BindableTransport {
	var (
		ws = &writerServer{
			writerServerCore: &writerServerCore{
				ResponseTranslator: writerResponseTranslator{},
				CreateTranslators: func() (
					fgrpc.Translator[apiframer.WriterRequest, *gapi.FrameWriterRequest],
					fgrpc.Translator[apiframer.WriterResponse, *gapi.FrameWriterResponse],
				) {
					codec := codec.NewDynamic(channelSvc)
					return writerRequestTranslator{codec: codec}, writerResponseTranslator{}
				},
				ServiceDesc: &gapi.FrameWriterService_ServiceDesc,
			},
		}
		is = &iteratorServer{
			iteratorServerCore: &iteratorServerCore{
				RequestTranslator:  iteratorRequestTranslator{},
				ResponseTranslator: iteratorResponseTranslator{},
				ServiceDesc:        &gapi.FrameIteratorService_ServiceDesc,
			},
		}
		ss = &streamerServer{
			streamerServerCore: &streamerServerCore{
				CreateTranslators: func() (fgrpc.Translator[apiframer.StreamerRequest, *gapi.FrameStreamerRequest], fgrpc.Translator[apiframer.StreamerResponse, *gapi.FrameStreamerResponse]) {
					codec := codec.NewDynamic(channelSvc)
					return streamerRequestTranslator{codec: codec}, streamerResponseTranslator{codec: codec}
				},
				ServiceDesc: &gapi.FrameStreamerService_ServiceDesc,
			},
		}
		ds = &deleteServer{
			RequestTranslator: deleteRequestTranslator{},
			ServiceDesc:       &gapi.FrameDeleteService_ServiceDesc,
		}
	)
	a.FrameStreamer = ss
	a.FrameWriter = ws
	a.FrameIterator = is
	a.FrameDelete = ds
	return fgrpc.CompoundBindableTransport{ws, is, ss}
}
