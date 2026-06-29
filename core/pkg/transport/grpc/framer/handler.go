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

	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distframer "github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/codec"
	"github.com/synnaxlabs/synnax/pkg/service/node"
	controlpb "github.com/synnaxlabs/x/control/pb"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	telempb "github.com/synnaxlabs/x/telem/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	frameWriterRequestTranslator    struct{ codec *codec.Codec }
	frameWriterResponseTranslator   struct{}
	frameIteratorRequestTranslator  struct{ codec *codec.Codec }
	frameIteratorResponseTranslator struct{ codec *codec.Codec }
	frameStreamerRequestTranslator  struct{ codec *codec.Codec }
	frameStreamerResponseTranslator struct{ codec *codec.Codec }
	frameDeleteRequestTranslator    struct{}
	framerWriterServerCore          = fgrpc.StreamServerCore[
		framer.WriterRequest,
		*WriterRequest,
		framer.WriterResponse,
		*WriterResponse,
	]
	frameIteratorServerCore = fgrpc.StreamServerCore[
		framer.IteratorRequest,
		*IteratorRequest,
		framer.IteratorResponse,
		*IteratorResponse,
	]
	frameStreamerServerCore = fgrpc.StreamServerCore[
		framer.StreamerRequest,
		*StreamerRequest,
		framer.StreamerResponse,
		*StreamerResponse,
	]
	frameDeleteServer = fgrpc.UnaryServer[
		framer.DeleteRequest,
		*DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[framer.WriterRequest, *WriterRequest]       = (*frameWriterRequestTranslator)(nil)
	_ fgrpc.Translator[framer.WriterResponse, *WriterResponse]     = (*frameWriterResponseTranslator)(nil)
	_ fgrpc.Translator[framer.IteratorRequest, *IteratorRequest]   = (*frameIteratorRequestTranslator)(nil)
	_ fgrpc.Translator[framer.IteratorResponse, *IteratorResponse] = (*frameIteratorResponseTranslator)(nil)
	_ fgrpc.Translator[framer.StreamerRequest, *StreamerRequest]   = (*frameStreamerRequestTranslator)(nil)
	_ fgrpc.Translator[framer.StreamerResponse, *StreamerResponse] = (*frameStreamerResponseTranslator)(nil)
	_ fgrpc.Translator[framer.DeleteRequest, *DeleteRequest]       = (*frameDeleteRequestTranslator)(nil)
)

func (t frameWriterRequestTranslator) Forward(
	ctx context.Context,
	msg framer.WriterRequest,
) (*WriterRequest, error) {
	subj, err := controlpb.SubjectToPB(msg.Config.ControlSubject)
	if err != nil {
		return nil, err
	}
	r := &WriterRequest{
		Command: int32(msg.Command),
		Config: &WriterConfig{
			Keys:                     msg.Config.Keys.Uint32(),
			Start:                    int64(msg.Config.Start),
			Mode:                     int32(msg.Config.Mode),
			Authorities:              msg.Config.Authorities,
			EnableAutoCommit:         msg.Config.EnableAutoCommit,
			AutoIndexPersistInterval: int64(msg.Config.AutoIndexPersistInterval),
			ControlSubject:           subj,
			ErrOnUnauthorized:        msg.Config.ErrOnUnauthorized,
			AutoIndex:                msg.Config.AutoIndex,
		},
	}
	if t.codec != nil && t.codec.Initialized() && !msg.Frame.Empty() {
		if r.Buffer, err = t.codec.Encode(ctx, msg.Frame); err != nil {
			return nil, err
		}
		return r, nil
	}
	if r.Frame, err = telempb.FrameToPB(msg.Frame.Frame); err != nil {
		return nil, err
	}
	return r, nil
}

func (t frameWriterRequestTranslator) Backward(
	ctx context.Context,
	msg *WriterRequest,
) (framer.WriterRequest, error) {
	if msg == nil {
		return framer.WriterRequest{}, nil
	}
	r := framer.WriterRequest{Command: distframer.WriterCommand(msg.Command)}
	if msg.Config != nil {
		subj, err := controlpb.SubjectFromPB(msg.Config.ControlSubject)
		if err != nil {
			return framer.WriterRequest{}, err
		}
		keys := channel.KeysFromUint32(msg.Config.Keys)
		r.Config = framer.WriterConfig{
			Keys:                     keys,
			Start:                    telem.TimeStamp(msg.Config.Start),
			Mode:                     distframer.WriterMode(msg.Config.Mode),
			Authorities:              msg.Config.Authorities,
			EnableAutoCommit:         msg.Config.EnableAutoCommit,
			AutoIndexPersistInterval: telem.TimeSpan(msg.Config.AutoIndexPersistInterval),
			ControlSubject:           subj,
			ErrOnUnauthorized:        msg.Config.ErrOnUnauthorized,
			AutoIndex:                msg.Config.AutoIndex,
		}
		if t.codec != nil {
			if err = t.codec.Update(ctx, keys); err != nil {
				return framer.WriterRequest{}, err
			}
		}
	}
	if t.codec != nil && len(msg.Buffer) > 0 {
		fr, err := t.codec.Decode(msg.Buffer)
		if err != nil {
			return framer.WriterRequest{}, err
		}
		r.Frame = fr
		return r, nil
	}
	frm, err := telempb.FrameFromPB[channel.Key](msg.Frame)
	if err != nil {
		return framer.WriterRequest{}, err
	}
	r.Frame = framer.Frame{Frame: frm}
	return r, nil
}

func (frameWriterResponseTranslator) Forward(
	_ context.Context,
	msg framer.WriterResponse,
) (*WriterResponse, error) {
	return &WriterResponse{
		Command: int32(msg.Command),
		End:     int64(msg.End),
		Error:   errors.TranslatePayloadForward(msg.Err),
	}, nil
}

func (frameWriterResponseTranslator) Backward(
	_ context.Context,
	msg *WriterResponse,
) (framer.WriterResponse, error) {
	return framer.WriterResponse{
		Command: distframer.WriterCommand(msg.Command),
		End:     telem.TimeStamp(msg.End),
		Err:     errors.TranslatePayloadBackward(msg.Error),
	}, nil
}

func (frameIteratorRequestTranslator) Forward(
	_ context.Context,
	msg framer.IteratorRequest,
) (*IteratorRequest, error) {
	tr, err := telempb.TimeRangeToPB(msg.Bounds)
	if err != nil {
		return nil, err
	}
	return &IteratorRequest{
		Command:   int32(msg.Command),
		Span:      int64(msg.Span),
		Range:     tr,
		Keys:      msg.Keys.Uint32(),
		Stamp:     int64(msg.Stamp),
		ChunkSize: msg.ChunkSize,
	}, nil
}

func (t frameIteratorRequestTranslator) Backward(
	ctx context.Context,
	msg *IteratorRequest,
) (framer.IteratorRequest, error) {
	tr, err := telempb.TimeRangeFromPB(msg.Range)
	if err != nil {
		return framer.IteratorRequest{}, err
	}
	keys := channel.KeysFromUint32(msg.Keys)
	if t.codec != nil && len(keys) > 0 {
		if err = t.codec.Update(ctx, keys); err != nil {
			return framer.IteratorRequest{}, err
		}
	}
	return framer.IteratorRequest{
		Command:   distframer.IteratorCommand(msg.Command),
		Span:      telem.TimeSpan(msg.Span),
		Bounds:    tr,
		Keys:      keys,
		Stamp:     telem.TimeStamp(msg.Stamp),
		ChunkSize: msg.ChunkSize,
	}, nil
}

func (t frameIteratorResponseTranslator) Forward(
	ctx context.Context,
	msg framer.IteratorResponse,
) (*IteratorResponse, error) {
	res := &IteratorResponse{
		Variant: int32(msg.Variant),
		Command: int32(msg.Command),
		NodeKey: int32(msg.NodeKey),
		Ack:     msg.Ack,
		SeqNum:  int32(msg.SeqNum),
		Error:   fgrpc.EncodeError(ctx, msg.Error, false),
	}
	if t.codec != nil &&
		t.codec.Initialized() &&
		msg.Variant == distframer.IteratorResponseVariantData &&
		!msg.Frame.Empty() {
		buf, err := t.codec.Encode(ctx, msg.Frame)
		if err != nil {
			return nil, err
		}
		res.Buffer = buf
		return res, nil
	}
	frm, err := telempb.FrameToPB(msg.Frame.Frame)
	if err != nil {
		return nil, err
	}
	res.Frame = frm
	return res, nil
}

func (t frameIteratorResponseTranslator) Backward(
	ctx context.Context,
	msg *IteratorResponse,
) (framer.IteratorResponse, error) {
	res := framer.IteratorResponse{
		Variant: distframer.IteratorResponseVariant(msg.Variant),
		Command: distframer.IteratorCommand(msg.Command),
		NodeKey: node.Key(msg.NodeKey),
		Ack:     msg.Ack,
		SeqNum:  int(msg.SeqNum),
	}
	if msg.Error != nil {
		res.Error = fgrpc.DecodeError(ctx, msg.Error)
	}
	if t.codec != nil && len(msg.Buffer) > 0 {
		fr, err := t.codec.Decode(msg.Buffer)
		if err != nil {
			return framer.IteratorResponse{}, err
		}
		res.Frame = fr
		return res, nil
	}
	frm, err := telempb.FrameFromPB[channel.Key](msg.Frame)
	if err != nil {
		return framer.IteratorResponse{}, err
	}
	res.Frame = framer.Frame{Frame: frm}
	return res, nil
}

func (frameStreamerRequestTranslator) Forward(
	_ context.Context,
	msg framer.StreamerRequest,
) (*StreamerRequest, error) {
	return &StreamerRequest{
		Keys:             msg.Keys.Uint32(),
		DownsampleFactor: int32(msg.DownsampleFactor),
		ThrottleRateHz:   float64(msg.ThrottleRate),
		ExcludeGroups:    msg.ExcludeGroups,
	}, nil
}

func (t frameStreamerRequestTranslator) Backward(
	ctx context.Context,
	msg *StreamerRequest,
) (framer.StreamerRequest, error) {
	rq := framer.StreamerRequest{
		Keys:             channel.KeysFromUint32(msg.Keys),
		DownsampleFactor: int(msg.DownsampleFactor),
		ThrottleRate:     telem.Rate(msg.ThrottleRateHz),
		ExcludeGroups:    msg.ExcludeGroups,
	}
	if t.codec != nil && len(rq.Keys) > 0 {
		if err := t.codec.Update(ctx, rq.Keys); err != nil {
			return framer.StreamerRequest{}, err
		}
	}
	return rq, nil
}

func (t frameStreamerResponseTranslator) Forward(
	ctx context.Context,
	msg framer.StreamerResponse,
) (*StreamerResponse, error) {
	if t.codec != nil && t.codec.Initialized() {
		buf, err := t.codec.Encode(ctx, msg.Frame)
		if err != nil {
			return nil, err
		}
		return &StreamerResponse{Buffer: buf}, nil
	}
	frm, err := telempb.FrameToPB(msg.Frame.Frame)
	if err != nil {
		return nil, err
	}
	return &StreamerResponse{Frame: frm}, nil
}

func (t frameStreamerResponseTranslator) Backward(
	_ context.Context,
	msg *StreamerResponse,
) (framer.StreamerResponse, error) {
	if t.codec != nil && len(msg.Buffer) > 0 {
		fr, err := t.codec.Decode(msg.Buffer)
		if err != nil {
			return framer.StreamerResponse{}, err
		}
		return framer.StreamerResponse{Frame: fr}, nil
	}
	tr, err := telempb.FrameFromPB[channel.Key](msg.Frame)
	if err != nil {
		return framer.StreamerResponse{}, err
	}
	return framer.StreamerResponse{Frame: framer.Frame{Frame: tr}}, nil
}

func (frameDeleteRequestTranslator) Forward(
	_ context.Context,
	msg framer.DeleteRequest,
) (*DeleteRequest, error) {
	tr, err := telempb.TimeRangeToPB(msg.Bounds)
	if err != nil {
		return nil, err
	}
	return &DeleteRequest{Keys: msg.Keys.Uint32(), Names: msg.Names, Bounds: tr}, nil
}

func (frameDeleteRequestTranslator) Backward(
	_ context.Context,
	msg *DeleteRequest,
) (framer.DeleteRequest, error) {
	tr, err := telempb.TimeRangeFromPB(msg.Bounds)
	if err != nil {
		return framer.DeleteRequest{}, err
	}
	return framer.DeleteRequest{
		Keys:   channel.KeysFromUint32(msg.Keys),
		Names:  msg.Names,
		Bounds: tr,
	}, nil
}

type writerServer struct{ *framerWriterServerCore }

func (f *writerServer) Exec(server FrameWriterService_ExecServer) error {
	return f.Handler(server.Context(), server)
}

func (f *writerServer) BindTo(reg grpc.ServiceRegistrar) {
	RegisterFrameWriterServiceServer(reg, f)
}

type iteratorServer struct{ *frameIteratorServerCore }

func (f *iteratorServer) Exec(server FrameIteratorService_ExecServer) error {
	return f.Handler(server.Context(), server)
}

func (f *iteratorServer) BindTo(reg grpc.ServiceRegistrar) {
	RegisterFrameIteratorServiceServer(reg, f)
}

type streamerServer struct{ *frameStreamerServerCore }

func (f *streamerServer) Exec(stream FrameStreamerService_ExecServer) error {
	return f.Handler(stream.Context(), stream)
}

func (f *streamerServer) BindTo(reg grpc.ServiceRegistrar) {
	RegisterFrameStreamerServiceServer(reg, f)
}

func New(t *api.Transport, channelSvc *channel.Service) fgrpc.BindableTransport {
	var (
		ws = &writerServer{
			framerWriterServerCore: &framerWriterServerCore{
				CreateTranslators: func() (
					fgrpc.Translator[framer.WriterRequest, *WriterRequest],
					fgrpc.Translator[framer.WriterResponse, *WriterResponse],
				) {
					codec := codec.NewDynamic(channelSvc)
					return frameWriterRequestTranslator{codec: codec}, frameWriterResponseTranslator{}
				},
				ServiceDesc: &FrameWriterService_ServiceDesc,
			},
		}
		is = &iteratorServer{
			frameIteratorServerCore: &frameIteratorServerCore{
				CreateTranslators: func() (
					fgrpc.Translator[framer.IteratorRequest, *IteratorRequest],
					fgrpc.Translator[framer.IteratorResponse, *IteratorResponse],
				) {
					codec := codec.NewDynamic(channelSvc)
					return frameIteratorRequestTranslator{codec: codec},
						frameIteratorResponseTranslator{codec: codec}
				},
				ServiceDesc: &FrameIteratorService_ServiceDesc,
			},
		}
		ss = &streamerServer{
			frameStreamerServerCore: &frameStreamerServerCore{
				CreateTranslators: func() (
					fgrpc.Translator[framer.StreamerRequest, *StreamerRequest],
					fgrpc.Translator[framer.StreamerResponse, *StreamerResponse],
				) {
					codec := codec.NewDynamic(channelSvc)
					return frameStreamerRequestTranslator{codec: codec}, frameStreamerResponseTranslator{codec: codec}
				},
				ServiceDesc: &FrameStreamerService_ServiceDesc,
			},
		}
		ds = &frameDeleteServer{
			RequestTranslator: frameDeleteRequestTranslator{},
			ServiceDesc:       &FrameDeleteService_ServiceDesc,
		}
	)
	t.FrameStreamer = ss
	t.FrameWriter = ws
	t.FrameIterator = is
	t.FrameDelete = ds
	return fgrpc.CompoundBindableTransport{ws, is, ss, ds}
}
