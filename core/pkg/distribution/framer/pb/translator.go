// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/frame"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	controlpb "github.com/synnaxlabs/x/control/pb"
	"github.com/synnaxlabs/x/telem"
	telempb "github.com/synnaxlabs/x/telem/pb"
)

var (
	_ fgrpc.Translator[writer.Request, *WriterRequest]       = (*WriterRequestTranslator)(nil)
	_ fgrpc.Translator[writer.Response, *WriterResponse]     = (*WriterResponseTranslator)(nil)
	_ fgrpc.Translator[iterator.Request, *IteratorRequest]   = (*IteratorRequestTranslator)(nil)
	_ fgrpc.Translator[iterator.Response, *IteratorResponse] = (*IteratorResponseTranslator)(nil)
	_ fgrpc.Translator[relay.Request, *RelayRequest]         = (*RelayRequestTranslator)(nil)
	_ fgrpc.Translator[relay.Response, *RelayResponse]       = (*RelayResponseTranslator)(nil)
	_ fgrpc.Translator[deleter.Request, *DeleteRequest]      = (*DeleteRequestTranslator)(nil)
)

type WriterRequestTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (WriterRequestTranslator) Backward(
	ctx context.Context,
	req *WriterRequest,
) (writer.Request, error) {
	fr, err := translateFrameForward(ctx, req.Frame)
	if err != nil {
		return writer.Request{}, err
	}
	return writer.Request{
		Command: writer.Command(req.Command),
		Config: writer.Config{
			ControlSubject: control.Subject{
				Key:  req.Config.ControlSubject.Key,
				Name: req.Config.ControlSubject.Name,
			},
			Keys:  channel.KeysFromUint32(req.Config.Keys),
			Start: telem.TimeStamp(req.Config.Start),
			Authorities: lo.Map(req.Config.Authorities, func(auth uint32, _ int) control.Authority {
				return control.Authority(auth)
			}),
			ErrOnUnauthorized:        config.Bool(req.Config.ErrOnUnauthorized),
			Mode:                     ts.WriterMode(req.Config.Mode),
			EnableAutoCommit:         config.Bool(req.Config.EnableAutoCommit),
			AutoIndexPersistInterval: telem.TimeSpan(req.Config.AutoIndexPersistInterval),
		},
		Frame: fr,
	}, nil
}

// Forward implements the fgrpc.Translator interface.
func (WriterRequestTranslator) Forward(
	ctx context.Context,
	req writer.Request,
) (*WriterRequest, error) {
	subject, err := controlpb.SubjectToPB(ctx, req.Config.ControlSubject)
	if err != nil {
		return nil, err
	}
	cfg := &WriterConfig{
		ControlSubject: subject,
		Keys:           req.Config.Keys.Uint32(),
		Start:          int64(req.Config.Start),
		Authorities: lo.Map(req.Config.Authorities, func(auth control.Authority, _ int) uint32 {
			return uint32(auth)
		}),
		Mode:                     uint32(req.Config.Mode),
		AutoIndexPersistInterval: int64(req.Config.AutoIndexPersistInterval),
	}
	if req.Config.ErrOnUnauthorized != nil {
		cfg.ErrOnUnauthorized = *req.Config.ErrOnUnauthorized
	}
	if req.Config.EnableAutoCommit != nil {
		cfg.EnableAutoCommit = *req.Config.EnableAutoCommit
	}
	fr, err := translateFrameBackward(ctx, req.Frame)
	if err != nil {
		return nil, err
	}
	return &WriterRequest{
		Command: int32(req.Command),
		Config:  cfg,
		Frame:   fr,
	}, nil
}

type WriterResponseTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (WriterResponseTranslator) Backward(
	ctx context.Context,
	res *WriterResponse,
) (writer.Response, error) {
	return writer.Response{
		Command:    writer.Command(res.Command),
		SeqNum:     int(res.SeqNum),
		NodeKey:    cluster.NodeKey(res.NodeKey),
		Authorized: res.Authorized,
		End:        telem.TimeStamp(res.End),
	}, nil
}

// Forward implements the fgrpc.Translator interface.
func (WriterResponseTranslator) Forward(
	ctx context.Context,
	res writer.Response,
) (*WriterResponse, error) {
	return &WriterResponse{
		Command:    int32(res.Command),
		SeqNum:     int32(res.SeqNum),
		NodeKey:    int32(res.NodeKey),
		End:        int64(res.End),
		Authorized: res.Authorized,
	}, nil
}

type IteratorRequestTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (IteratorRequestTranslator) Backward(
	ctx context.Context,
	req *IteratorRequest,
) (iterator.Request, error) {
	bounds, err := telempb.TimeRangeFromPB(ctx, req.Bounds)
	if err != nil {
		return iterator.Request{}, err
	}
	return iterator.Request{
		Command:   iterator.Command(req.Command),
		Span:      telem.TimeSpan(req.Span),
		Bounds:    bounds,
		Stamp:     telem.TimeStamp(req.Stamp),
		Keys:      channel.KeysFromUint32(req.Keys),
		ChunkSize: req.ChunkSize,
		SeqNum:    int(req.SeqNum),
	}, nil
}

// Forward implements the fgrpc.Translator interface.
func (IteratorRequestTranslator) Forward(
	ctx context.Context,
	req iterator.Request,
) (*IteratorRequest, error) {
	bounds, err := telempb.TimeRangeToPB(ctx, req.Bounds)
	if err != nil {
		return nil, err
	}
	return &IteratorRequest{
		Command:   int32(req.Command),
		Span:      int64(req.Span),
		Bounds:    bounds,
		Stamp:     int64(req.Stamp),
		Keys:      req.Keys.Uint32(),
		ChunkSize: req.ChunkSize,
		SeqNum:    int32(req.SeqNum),
	}, nil
}

type IteratorResponseTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (IteratorResponseTranslator) Backward(
	ctx context.Context,
	res *IteratorResponse,
) (iterator.Response, error) {
	fr, err := translateFrameForward(ctx, res.Frame)
	if err != nil {
		return iterator.Response{}, err
	}
	return iterator.Response{
		Variant: iterator.ResponseVariant(res.Variant),
		NodeKey: cluster.NodeKey(res.NodeKey),
		Ack:     res.Ack,
		SeqNum:  int(res.SeqNum),
		Command: iterator.Command(res.Command),
		Error:   fgrpc.DecodeError(ctx, res.Error),
		Frame:   fr,
	}, nil
}

// Forward implements the fgrpc.Translator interface.
func (IteratorResponseTranslator) Forward(
	ctx context.Context,
	res iterator.Response,
) (*IteratorResponse, error) {
	fr, err := translateFrameBackward(ctx, res.Frame)
	if err != nil {
		return nil, err
	}
	return &IteratorResponse{
		Variant: int32(res.Variant),
		NodeKey: int32(res.NodeKey),
		Ack:     res.Ack,
		SeqNum:  int32(res.SeqNum),
		Command: int32(res.Command),
		Error:   fgrpc.EncodeError(ctx, res.Error, true),
		Frame:   fr,
	}, nil
}

type RelayRequestTranslator struct{}

func (w RelayRequestTranslator) Backward(
	_ context.Context,
	req *RelayRequest,
) (relay.Request, error) {
	return relay.Request{Keys: channel.KeysFromUint32(req.Keys)}, nil
}

func (w RelayRequestTranslator) Forward(
	_ context.Context,
	req relay.Request,
) (*RelayRequest, error) {
	return &RelayRequest{Keys: req.Keys.Uint32()}, nil
}

type RelayResponseTranslator struct{}

func (w RelayResponseTranslator) Backward(
	ctx context.Context,
	res *RelayResponse,
) (relay.Response, error) {
	fr, err := translateFrameForward(ctx, res.Frame)
	if err != nil {
		return relay.Response{}, err
	}
	return relay.Response{Frame: fr}, nil
}

func (w RelayResponseTranslator) Forward(
	ctx context.Context,
	res relay.Response,
) (*RelayResponse, error) {
	fr, err := translateFrameBackward(ctx, res.Frame)
	if err != nil {
		return nil, err
	}
	return &RelayResponse{Frame: fr}, nil
}

func translateFrameForward(ctx context.Context, fr *telempb.Frame) (framer.Frame, error) {
	telemFr, err := telempb.FrameFromPB[channel.Key](ctx, fr)
	if err != nil {
		return framer.Frame{}, err
	}
	return frame.Frame{Frame: telemFr}, nil
}

func translateFrameBackward(ctx context.Context, fr framer.Frame) (*telempb.Frame, error) {
	return telempb.FrameToPB(ctx, fr.Frame)
}

type DeleteRequestTranslator struct{}

func (r DeleteRequestTranslator) Forward(
	ctx context.Context,
	msg deleter.Request,
) (*DeleteRequest, error) {
	bounds, err := telempb.TimeRangeToPB(ctx, msg.Bounds)
	if err != nil {
		return nil, err
	}
	return &DeleteRequest{
		Keys:   msg.Keys.Uint32(),
		Names:  msg.Names,
		Bounds: bounds,
	}, nil
}

func (r DeleteRequestTranslator) Backward(
	ctx context.Context,
	msg *DeleteRequest,
) (deleter.Request, error) {
	bounds, err := telempb.TimeRangeFromPB(ctx, msg.Bounds)
	if err != nil {
		return deleter.Request{}, err
	}
	return deleter.Request{
		Keys:   channel.KeysFromUint32(msg.Keys),
		Names:  msg.Names,
		Bounds: bounds,
	}, nil
}
