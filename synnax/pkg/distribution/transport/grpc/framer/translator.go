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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	tsv1 "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/gen/proto/go/framer/v1"
	"github.com/synnaxlabs/x/telem"
)

var (
	_ fgrpc.Translator[writer.Request, *tsv1.WriterRequest]       = (*writerRequestTranslator)(nil)
	_ fgrpc.Translator[writer.Response, *tsv1.WriterResponse]     = (*writerResponseTranslator)(nil)
	_ fgrpc.Translator[iterator.Request, *tsv1.IteratorRequest]   = (*iteratorRequestTranslator)(nil)
	_ fgrpc.Translator[iterator.Response, *tsv1.IteratorResponse] = (*iteratorResponseTranslator)(nil)
	_ fgrpc.Translator[relay.Request, *tsv1.RelayRequest]         = (*relayRequestTranslator)(nil)
	_ fgrpc.Translator[relay.Response, *tsv1.RelayResponse]       = (*relayResponseTranslator)(nil)
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
		Frame: tranFrameFwd(req.Frame),
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
		Frame: tranFrmBwd(req.Frame),
	}, nil
}

type writerResponseTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (writerResponseTranslator) Backward(
	_ context.Context,
	res *tsv1.WriterResponse,
) (writer.Response, error) {
	return writer.Response{
		Command: writer.Command(res.Command),
		SeqNum:  int(res.Counter),
		Ack:     res.Ack,
		Error:   fgrpc.DecodeError(res.Error),
	}, nil
}

// Forward implements the fgrpc.Translator interface.
func (writerResponseTranslator) Forward(
	_ context.Context,
	res writer.Response,
) (*tsv1.WriterResponse, error) {
	return &tsv1.WriterResponse{
		Command: int32(res.Command),
		Counter: int32(res.SeqNum),
		Ack:     res.Ack,
		Error:   fgrpc.EncodeError(res.Error),
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
		Bounds: telem.TimeRange{
			Start: telem.TimeStamp(req.Range.Start),
			End:   telem.TimeStamp(req.Range.End),
		},
		Stamp: telem.TimeStamp(req.Stamp),
		Keys:  channel.KeysFromUint32(req.Keys),
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
		Range: &tsv1.TimeRange{
			Start: int64(req.Bounds.Start),
			End:   int64(req.Bounds.End),
		},
		Stamp: int64(req.Stamp),
		Keys:  req.Keys.Uint32(),
	}, nil
}

type iteratorResponseTranslator struct{}

// Backward implements the fgrpc.Translator interface.
func (iteratorResponseTranslator) Backward(
	_ context.Context,
	res *tsv1.IteratorResponse,
) (iterator.Response, error) {
	return iterator.Response{
		Variant: iterator.ResponseVariant(res.Variant),
		NodeKey: dcore.NodeKey(res.NodeId),
		Ack:     res.Ack,
		SeqNum:  int(res.Counter),
		Command: iterator.Command(res.Command),
		Error:   fgrpc.DecodeError(res.Error),
		Frame:   tranFrameFwd(res.Frame),
	}, nil
}

// Forward implements the fgrpc.Translator interface.
func (iteratorResponseTranslator) Forward(
	_ context.Context,
	res iterator.Response,
) (*tsv1.IteratorResponse, error) {
	return &tsv1.IteratorResponse{
		Variant: int32(res.Variant),
		NodeId:  int32(res.NodeKey),
		Ack:     res.Ack,
		Counter: int32(res.SeqNum),
		Command: int32(res.Command),
		Error:   fgrpc.EncodeError(res.Error),
		Frame:   tranFrmBwd(res.Frame),
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
	_ context.Context,
	res *tsv1.RelayResponse,
) (relay.Response, error) {
	return relay.Response{
		Frame: tranFrameFwd(res.Frame),
		Error: fgrpc.DecodeError(res.Error),
	}, nil
}

func (w relayResponseTranslator) Forward(
	_ context.Context,
	res relay.Response,
) (*tsv1.RelayResponse, error) {
	return &tsv1.RelayResponse{
		Frame: tranFrmBwd(res.Frame),
		Error: fgrpc.EncodeError(res.Error),
	}, nil
}

func tranFrameFwd(frame *tsv1.Frame) framer.Frame {
	keys := channel.KeysFromUint32(frame.Keys)
	series := tranArrayFwd(frame.Series)
	return framer.Frame{Keys: keys, Series: series}
}

func tranFrmBwd(frame framer.Frame) *tsv1.Frame {
	return &tsv1.Frame{
		Keys:   frame.Keys.Uint32(),
		Series: tranArrBwd(frame.Series),
	}
}

func tranArrayFwd(series []*tsv1.Series) []telem.Series {
	LazyArrays := make([]telem.Series, len(series))
	for i, series := range series {
		LazyArrays[i] = telem.Series{
			DataType: telem.DataType(series.DataType),
			TimeRange: telem.TimeRange{
				Start: telem.TimeStamp(series.Range.Start),
				End:   telem.TimeStamp(series.Range.End),
			},
			Data: series.Data,
		}
	}
	return LazyArrays
}

func tranArrBwd(series []telem.Series) []*tsv1.Series {
	LazyArrays := make([]*tsv1.Series, len(series))
	for i, series := range series {
		LazyArrays[i] = &tsv1.Series{
			DataType: string(series.DataType),
			Range: &tsv1.TimeRange{
				Start: int64(series.TimeRange.Start),
				End:   int64(series.TimeRange.End),
			},
			Data: series.Data,
		}
	}
	return LazyArrays
}
