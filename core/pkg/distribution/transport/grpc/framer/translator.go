// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"

	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	framerv1 "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer/v1"
	"github.com/synnaxlabs/synnax/pkg/storage/ts"
	"github.com/synnaxlabs/x/config"
	"github.com/synnaxlabs/x/control"
	"github.com/synnaxlabs/x/telem"
)

var (
	_ fgrpc.Translator[writer.Request, *framerv1.WriterRequest]       = (*writerRequestTranslator)(nil)
	_ fgrpc.Translator[writer.Response, *framerv1.WriterResponse]     = (*writerResponseTranslator)(nil)
	_ fgrpc.Translator[iterator.Request, *framerv1.IteratorRequest]   = (*iteratorRequestTranslator)(nil)
	_ fgrpc.Translator[iterator.Response, *framerv1.IteratorResponse] = (*iteratorResponseTranslator)(nil)
	_ fgrpc.Translator[relay.Request, *framerv1.RelayRequest]         = (*relayRequestTranslator)(nil)
	_ fgrpc.Translator[relay.Response, *framerv1.RelayResponse]       = (*relayResponseTranslator)(nil)
	_ fgrpc.Translator[deleter.Request, *framerv1.DeleteRequest]      = (*deleteRequestTranslator)(nil)
)

type writerRequestTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (writerRequestTranslator) Backward(
	_ context.Context,
	req *framerv1.WriterRequest,
) (writer.Request, error) {
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
		Frame: translateFrameForward(req.Frame),
	}, nil
}

// Forward implements the fgrpc.Translator interface.
func (writerRequestTranslator) Forward(
	_ context.Context,
	req writer.Request,
) (*framerv1.WriterRequest, error) {
	cfg := &framerv1.WriterConfig{
		ControlSubject: &control.ControlSubject{
			Key:  req.Config.ControlSubject.Key,
			Name: req.Config.ControlSubject.Name,
		},
		Keys:  req.Config.Keys.Uint32(),
		Start: int64(req.Config.Start),
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
	return &framerv1.WriterRequest{
		Command: int32(req.Command),
		Config:  cfg,
		Frame:   translateFrameBackward(req.Frame),
	}, nil
}

type writerResponseTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (writerResponseTranslator) Backward(
	ctx context.Context,
	res *framerv1.WriterResponse,
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
func (writerResponseTranslator) Forward(
	ctx context.Context,
	res writer.Response,
) (*framerv1.WriterResponse, error) {
	return &framerv1.WriterResponse{
		Command:    int32(res.Command),
		SeqNum:     int32(res.SeqNum),
		NodeKey:    int32(res.NodeKey),
		End:        int64(res.End),
		Authorized: res.Authorized,
	}, nil
}

type iteratorRequestTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (iteratorRequestTranslator) Backward(
	_ context.Context,
	req *framerv1.IteratorRequest,
) (iterator.Request, error) {
	return iterator.Request{
		Command:   iterator.Command(req.Command),
		Span:      telem.TimeSpan(req.Span),
		Bounds:    telem.TranslateTimeRangeBackward(req.Bounds),
		Stamp:     telem.TimeStamp(req.Stamp),
		Keys:      channel.KeysFromUint32(req.Keys),
		ChunkSize: req.ChunkSize,
		SeqNum:    int(req.SeqNum),
	}, nil
}

// Forward implements the fgrpc.Translator interface.
func (iteratorRequestTranslator) Forward(
	_ context.Context,
	req iterator.Request,
) (*framerv1.IteratorRequest, error) {
	return &framerv1.IteratorRequest{
		Command:   int32(req.Command),
		Span:      int64(req.Span),
		Bounds:    telem.TranslateTimeRangeForward(req.Bounds),
		Stamp:     int64(req.Stamp),
		Keys:      req.Keys.Uint32(),
		ChunkSize: req.ChunkSize,
		SeqNum:    int32(req.SeqNum),
	}, nil
}

type iteratorResponseTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (iteratorResponseTranslator) Backward(
	ctx context.Context,
	res *framerv1.IteratorResponse,
) (iterator.Response, error) {
	return iterator.Response{
		Variant: iterator.ResponseVariant(res.Variant),
		NodeKey: cluster.NodeKey(res.NodeKey),
		Ack:     res.Ack,
		SeqNum:  int(res.SeqNum),
		Command: iterator.Command(res.Command),
		Error:   fgrpc.DecodeError(ctx, res.Error),
		Frame:   translateFrameForward(res.Frame),
	}, nil
}

// Forward implements the fgrpc.Translator interface.
func (iteratorResponseTranslator) Forward(
	ctx context.Context,
	res iterator.Response,
) (*framerv1.IteratorResponse, error) {
	return &framerv1.IteratorResponse{
		Variant: int32(res.Variant),
		NodeKey: int32(res.NodeKey),
		Ack:     res.Ack,
		SeqNum:  int32(res.SeqNum),
		Command: int32(res.Command),
		Error:   fgrpc.EncodeError(ctx, res.Error, true),
		Frame:   translateFrameBackward(res.Frame),
	}, nil
}

type relayRequestTranslator struct{}

func (w relayRequestTranslator) Backward(
	_ context.Context,
	req *framerv1.RelayRequest,
) (relay.Request, error) {
	return relay.Request{Keys: channel.KeysFromUint32(req.Keys)}, nil
}

func (w relayRequestTranslator) Forward(
	_ context.Context,
	req relay.Request,
) (*framerv1.RelayRequest, error) {
	return &framerv1.RelayRequest{Keys: req.Keys.Uint32()}, nil
}

type relayResponseTranslator struct{}

func (w relayResponseTranslator) Backward(
	ctx context.Context,
	res *framerv1.RelayResponse,
) (relay.Response, error) {
	return relay.Response{Frame: translateFrameForward(res.Frame)}, nil
}

func (w relayResponseTranslator) Forward(
	ctx context.Context,
	res relay.Response,
) (*framerv1.RelayResponse, error) {
	return &framerv1.RelayResponse{Frame: translateFrameBackward(res.Frame)}, nil
}

func translateFrameForward(frame *telem.PBFrame) framer.Frame {
	keys := channel.KeysFromUint32(frame.Keys)
	series := telem.TranslateManySeriesBackward(frame.Series)
	return core.MultiFrame(keys, series)
}

func translateFrameBackward(frame framer.Frame) *telem.PBFrame {
	return &telem.PBFrame{
		Keys:   channel.Keys(frame.KeysSlice()).Uint32(),
		Series: telem.TranslateManySeriesForward(frame.SeriesSlice()),
	}
}

type deleteRequestTranslator struct{}

func (r deleteRequestTranslator) Forward(
	_ context.Context,
	msg deleter.Request,
) (*framerv1.DeleteRequest, error) {
	return &framerv1.DeleteRequest{
		Keys:   msg.Keys.Uint32(),
		Names:  msg.Names,
		Bounds: telem.TranslateTimeRangeForward(msg.Bounds),
	}, nil
}

func (r deleteRequestTranslator) Backward(
	_ context.Context,
	msg *framerv1.DeleteRequest,
) (deleter.Request, error) {
	return deleter.Request{
		Keys:   channel.KeysFromUint32(msg.Keys),
		Names:  msg.Names,
		Bounds: telem.TranslateTimeRangeBackward(msg.Bounds),
	}, nil
}
