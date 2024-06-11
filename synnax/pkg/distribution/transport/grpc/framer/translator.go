// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	dcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	tsv1 "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer/v1"
	"github.com/synnaxlabs/x/telem"
)

var (
	_ fgrpc.Translator[writer.Request, *tsv1.WriterRequest]       = (*writerRequestTranslator)(nil)
	_ fgrpc.Translator[writer.Response, *tsv1.WriterResponse]     = (*writerResponseTranslator)(nil)
	_ fgrpc.Translator[iterator.Request, *tsv1.IteratorRequest]   = (*iteratorRequestTranslator)(nil)
	_ fgrpc.Translator[iterator.Response, *tsv1.IteratorResponse] = (*iteratorResponseTranslator)(nil)
	_ fgrpc.Translator[relay.Request, *tsv1.RelayRequest]         = (*relayRequestTranslator)(nil)
	_ fgrpc.Translator[relay.Response, *tsv1.RelayResponse]       = (*relayResponseTranslator)(nil)
	_ fgrpc.Translator[deleter.Request, *tsv1.DeleteRequest]      = (*deleteRequestTranslator)(nil)
)

type writerRequestTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (writerRequestTranslator) Backward(
	_ context.Context,
	req *tsv1.WriterRequest,
) (writer.Request, error) {
	return writer.Request{
		Command: writer.Command(req.Command),
		Config: writer.Config{
			Keys:  channel.KeysFromUint32(req.Config.Keys),
			Start: telem.TimeStamp(req.Config.Start),
		},
		Frame: translateFrameForward(req.Frame),
	}, nil
}

// Forward implements the fgrpc.Translator interface.
func (writerRequestTranslator) Forward(
	_ context.Context,
	req writer.Request,
) (*tsv1.WriterRequest, error) {
	return &tsv1.WriterRequest{
		Command: int32(req.Command),
		Config: &tsv1.WriterConfig{
			Keys:  req.Config.Keys.Uint32(),
			Start: int64(req.Config.Start),
		},
		Frame: translateFrameBackward(req.Frame),
	}, nil
}

type writerResponseTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (writerResponseTranslator) Backward(
	ctx context.Context,
	res *tsv1.WriterResponse,
) (writer.Response, error) {
	return writer.Response{
		Command: writer.Command(res.Command),
		SeqNum:  int(res.SeqNum),
		Ack:     res.Ack,
		Error:   fgrpc.DecodeError(ctx, res.Error),
	}, nil
}

// Forward implements the fgrpc.Translator interface.
func (writerResponseTranslator) Forward(
	ctx context.Context,
	res writer.Response,
) (*tsv1.WriterResponse, error) {
	return &tsv1.WriterResponse{
		Command: int32(res.Command),
		SeqNum:  int32(res.SeqNum),
		Ack:     res.Ack,
		Error:   fgrpc.EncodeError(ctx, res.Error, true),
	}, nil
}

type iteratorRequestTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (iteratorRequestTranslator) Backward(
	_ context.Context,
	req *tsv1.IteratorRequest,
) (iterator.Request, error) {
	return iterator.Request{
		Command: iterator.Command(req.Command),
		Span:    telem.TimeSpan(req.Span),
		Bounds:  telem.TranslateTimeRangeBackward(req.Bounds),
		Stamp:   telem.TimeStamp(req.Stamp),
		Keys:    channel.KeysFromUint32(req.Keys),
	}, nil
}

// Forward implements the fgrpc.Translator interface.
func (iteratorRequestTranslator) Forward(
	_ context.Context,
	req iterator.Request,
) (*tsv1.IteratorRequest, error) {
	return &tsv1.IteratorRequest{
		Command: int32(req.Command),
		Span:    int64(req.Span),
		Bounds:  telem.TranslateTimeRangeForward(req.Bounds),
		Stamp:   int64(req.Stamp),
		Keys:    req.Keys.Uint32(),
	}, nil
}

type iteratorResponseTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (iteratorResponseTranslator) Backward(
	ctx context.Context,
	res *tsv1.IteratorResponse,
) (iterator.Response, error) {
	return iterator.Response{
		Variant: iterator.ResponseVariant(res.Variant),
		NodeKey: dcore.NodeKey(res.NodeKey),
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
) (*tsv1.IteratorResponse, error) {
	return &tsv1.IteratorResponse{
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
	req *tsv1.RelayRequest,
) (relay.Request, error) {
	return relay.Request{Keys: channel.KeysFromUint32(req.Keys)}, nil
}

func (w relayRequestTranslator) Forward(
	_ context.Context,
	req relay.Request,
) (*tsv1.RelayRequest, error) {
	return &tsv1.RelayRequest{Keys: req.Keys.Uint32()}, nil
}

type relayResponseTranslator struct{}

func (w relayResponseTranslator) Backward(
	ctx context.Context,
	res *tsv1.RelayResponse,
) (relay.Response, error) {
	return relay.Response{
		Frame: translateFrameForward(res.Frame),
		Error: fgrpc.DecodeError(ctx, res.Error),
	}, nil
}

func (w relayResponseTranslator) Forward(
	ctx context.Context,
	res relay.Response,
) (*tsv1.RelayResponse, error) {
	return &tsv1.RelayResponse{
		Frame: translateFrameBackward(res.Frame),
		Error: fgrpc.EncodeError(ctx, res.Error, true),
	}, nil
}

func translateFrameForward(frame *tsv1.Frame) framer.Frame {
	keys := channel.KeysFromUint32(frame.Keys)
	series := telem.TranslateManySeriesBackward(frame.Series)
	return framer.Frame{Keys: keys, Series: series}
}

func translateFrameBackward(frame framer.Frame) *tsv1.Frame {
	return &tsv1.Frame{
		Keys:   frame.Keys.Uint32(),
		Series: telem.TranslateManySeriesForward(frame.Series),
	}
}

type deleteRequestTranslator struct{}

func (r deleteRequestTranslator) Forward(
	_ context.Context,
	msg deleter.Request,
) (*tsv1.DeleteRequest, error) {
	return &tsv1.DeleteRequest{
		Keys:   msg.Keys.Uint32(),
		Names:  msg.Names,
		Bounds: telem.TranslateTimeRangeForward(msg.Bounds),
	}, nil
}

func (r deleteRequestTranslator) Backward(
	_ context.Context,
	msg *tsv1.DeleteRequest,
) (deleter.Request, error) {
	return deleter.Request{
		Keys:   channel.KeysFromUint32(msg.Keys),
		Names:  msg.Names,
		Bounds: telem.TranslateTimeRangeBackward(msg.Bounds),
	}, nil
}
