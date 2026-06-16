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
	"go/types"

	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/channel"
	"github.com/synnaxlabs/synnax/pkg/api/channel/pb"
	"github.com/synnaxlabs/synnax/pkg/service/node"
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
		channel.RetrieveRequest,
		*RetrieveRequest,
		channel.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		channel.DeleteRequest,
		*DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ grpc.Translator[channel.CreateRequest, *CreateRequest]       = (*createRequestTranslator)(nil)
	_ grpc.Translator[channel.CreateResponse, *CreateResponse]     = (*createResponseTranslator)(nil)
	_ grpc.Translator[channel.RetrieveRequest, *RetrieveRequest]   = (*retrieveRequestTranslator)(nil)
	_ grpc.Translator[channel.RetrieveResponse, *RetrieveResponse] = (*retrieveResponseTranslator)(nil)
	_ grpc.Translator[channel.DeleteRequest, *DeleteRequest]       = (*deleteRequestTranslator)(nil)
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
	msg channel.RetrieveRequest,
) (*RetrieveRequest, error) {
	return &RetrieveRequest{
		NodeKey: uint32(msg.NodeKey),
		Names:   msg.Names,
		Search:  msg.SearchTerm,
		Keys:    unsafe.ReinterpretSlice[channel.Key, uint32](msg.Keys),
	}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	msg *RetrieveRequest,
) (channel.RetrieveRequest, error) {
	return channel.RetrieveRequest{
		NodeKey:    node.Key(msg.NodeKey),
		Names:      msg.Names,
		SearchTerm: msg.Search,
		Keys:       unsafe.ReinterpretSlice[uint32, channel.Key](msg.Keys),
	}, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	msg channel.RetrieveResponse,
) (*RetrieveResponse, error) {
	channels, err := pb.ChannelsToPB(msg.Channels)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Channels: channels}, nil
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	msg *RetrieveResponse,
) (channel.RetrieveResponse, error) {
	channels, err := pb.ChannelsFromPB(msg.Channels)
	if err != nil {
		return channel.RetrieveResponse{}, err
	}
	return channel.RetrieveResponse{Channels: channels}, nil
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
