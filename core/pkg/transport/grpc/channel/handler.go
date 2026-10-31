// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"uuid"

	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/api/channel/pb"
	xpb "github.com/synnaxlabs/x/pb"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
	createServer               = grpc.UnaryServer[
		channel.CreateRequest,
		*CreateRequest,
		channel.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		channel.AliasedRetrieveRequest,
		*RetrieveRequest,
		channel.RetrieveResponse,
		*pb.RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		channel.DeleteRequest,
		*DeleteRequest,
		struct{},
		*emptypb.Empty,
	]
)

var (
	_ grpc.Translator[channel.CreateRequest, *CreateRequest] = (*createRequestTranslator)(
		nil,
	)
	_ grpc.Translator[channel.CreateResponse, *CreateResponse] = (*createResponseTranslator)(
		nil,
	)
	_ grpc.Translator[channel.AliasedRetrieveRequest, *RetrieveRequest] = (*retrieveRequestTranslator)(
		nil,
	)
	_ grpc.Translator[channel.RetrieveResponse, *pb.RetrieveResponse] = (*retrieveResponseTranslator)(
		nil,
	)
	_ grpc.Translator[channel.DeleteRequest, *DeleteRequest] = (*deleteRequestTranslator)(
		nil,
	)
)

func (createRequestTranslator) Forward(
	_ context.Context,
	msg channel.CreateRequest,
) (*CreateRequest, error) {
	channels, err := pb.ChannelsToPB(msg.Channels)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{
		Channels:             channels,
		RetrieveIfNameExists: msg.RetrieveIfNameExists,
	}, nil
}

func (createRequestTranslator) Backward(
	_ context.Context,
	msg *CreateRequest,
) (channel.CreateRequest, error) {
	channels, err := pb.ChannelsFromPB(msg.Channels)
	if err != nil {
		return channel.CreateRequest{}, err
	}
	return channel.CreateRequest{
		Channels:             channels,
		RetrieveIfNameExists: msg.RetrieveIfNameExists,
	}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	msg channel.CreateResponse,
) (*CreateResponse, error) {
	channels, err := pb.ChannelsToPB(msg.Channels)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Channels: channels}, nil
}

func (createResponseTranslator) Backward(
	_ context.Context,
	msg *CreateResponse,
) (channel.CreateResponse, error) {
	channels, err := pb.ChannelsFromPB(msg.Channels)
	if err != nil {
		return channel.CreateResponse{}, err
	}
	return channel.CreateResponse{Channels: channels}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	msg channel.AliasedRetrieveRequest,
) (*RetrieveRequest, error) {
	req, err := pb.RetrieveRequestToPB(msg.Request)
	if err != nil {
		return nil, err
	}
	res := &RetrieveRequest{Request: req}
	if msg.RangeKey != nil {
		res.RangeKey = new(msg.RangeKey.String())
	}
	return res, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	msg *RetrieveRequest,
) (channel.AliasedRetrieveRequest, error) {
	// A client older than the Core sends the reserved fields, which would otherwise
	// select every channel.
	if err := xpb.RejectUnknown(msg); err != nil {
		return channel.AliasedRetrieveRequest{}, err
	}
	req, err := pb.RetrieveRequestFromPB(msg.Request)
	if err != nil {
		return channel.AliasedRetrieveRequest{}, err
	}
	res := channel.AliasedRetrieveRequest{Request: req}
	if msg.RangeKey != nil {
		key, err := uuid.Parse(*msg.RangeKey)
		if err != nil {
			return channel.AliasedRetrieveRequest{}, err
		}
		res.RangeKey = &key
	}
	return res, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	msg channel.RetrieveResponse,
) (*pb.RetrieveResponse, error) {
	return pb.RetrieveResponseToPB(msg)
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	msg *pb.RetrieveResponse,
) (channel.RetrieveResponse, error) {
	return pb.RetrieveResponseFromPB(msg)
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	msg channel.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{
		Keys:  unsafe.ReinterpretSlice[channel.Key, uint32](msg.Keys),
		Names: msg.Names,
	}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	msg *DeleteRequest,
) (channel.DeleteRequest, error) {
	return channel.DeleteRequest{
		Keys:  unsafe.ReinterpretSlice[uint32, channel.Key](msg.Keys),
		Names: msg.Names,
	}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
	c := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &ChannelCreateService_ServiceDesc,
	}
	r := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &ChannelRetrieveService_ServiceDesc,
	}
	d := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &ChannelDeleteService_ServiceDesc,
	}
	t.ChannelCreate = c
	t.ChannelRetrieve = r
	t.ChannelDelete = d
	return grpc.CompoundBindableTransport{c, r, d}
}
