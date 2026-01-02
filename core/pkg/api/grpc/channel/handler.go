// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	entitypb "github.com/synnaxlabs/synnax/pkg/api/channel/pb"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
	createServer = fgrpc.UnaryServer[
		apichannel.CreateRequest,
		*CreateRequest,
		apichannel.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apichannel.RetrieveRequest,
		*RetrieveRequest,
		apichannel.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apichannel.DeleteRequest,
		*DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[apichannel.CreateRequest, *CreateRequest]       = (*createRequestTranslator)(nil)
	_ fgrpc.Translator[apichannel.CreateResponse, *CreateResponse]     = (*createResponseTranslator)(nil)
	_ fgrpc.Translator[apichannel.RetrieveRequest, *RetrieveRequest]   = (*retrieveRequestTranslator)(nil)
	_ fgrpc.Translator[apichannel.RetrieveResponse, *RetrieveResponse] = (*retrieveResponseTranslator)(nil)
	_ fgrpc.Translator[apichannel.DeleteRequest, *DeleteRequest]       = (*deleteRequestTranslator)(nil)
)

func (t createRequestTranslator) Forward(
	ctx context.Context,
	msg apichannel.CreateRequest,
) (*CreateRequest, error) {
	channels, err := entitypb.ChannelsToPB(ctx, msg.Channels)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{
		Channels:             channels,
		RetrieveIfNameExists: msg.RetrieveIfNameExists,
	}, nil
}

func (t createRequestTranslator) Backward(
	ctx context.Context,
	msg *CreateRequest,
) (apichannel.CreateRequest, error) {
	channels, err := entitypb.ChannelsFromPB(ctx, msg.Channels)
	if err != nil {
		return apichannel.CreateRequest{}, err
	}
	return apichannel.CreateRequest{
		Channels:             channels,
		RetrieveIfNameExists: msg.RetrieveIfNameExists,
	}, nil
}

func (t createResponseTranslator) Forward(
	ctx context.Context,
	msg apichannel.CreateResponse,
) (*CreateResponse, error) {
	channels, err := entitypb.ChannelsToPB(ctx, msg.Channels)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Channels: channels}, nil
}

func (t createResponseTranslator) Backward(
	ctx context.Context,
	msg *CreateResponse,
) (apichannel.CreateResponse, error) {
	channels, err := entitypb.ChannelsFromPB(ctx, msg.Channels)
	if err != nil {
		return apichannel.CreateResponse{}, err
	}
	return apichannel.CreateResponse{Channels: channels}, nil
}

func (t retrieveRequestTranslator) Forward(
	_ context.Context,
	msg apichannel.RetrieveRequest,
) (*RetrieveRequest, error) {
	return &RetrieveRequest{
		NodeKey: uint32(msg.NodeKey),
		Names:   msg.Names,
		Search:  msg.SearchTerm,
		Keys:    unsafe.ReinterpretSlice[channel.Key, uint32](msg.Keys),
	}, nil
}

func (t retrieveRequestTranslator) Backward(
	_ context.Context,
	msg *RetrieveRequest,
) (apichannel.RetrieveRequest, error) {
	return apichannel.RetrieveRequest{
		NodeKey:    cluster.NodeKey(msg.NodeKey),
		Names:      msg.Names,
		SearchTerm: msg.Search,
		Keys:       unsafe.ReinterpretSlice[uint32, channel.Key](msg.Keys),
	}, nil
}

func (t retrieveResponseTranslator) Forward(
	ctx context.Context,
	msg apichannel.RetrieveResponse,
) (*RetrieveResponse, error) {
	channels, err := entitypb.ChannelsToPB(ctx, msg.Channels)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Channels: channels}, nil
}

func (t retrieveResponseTranslator) Backward(
	ctx context.Context,
	msg *RetrieveResponse,
) (apichannel.RetrieveResponse, error) {
	channels, err := entitypb.ChannelsFromPB(ctx, msg.Channels)
	if err != nil {
		return apichannel.RetrieveResponse{}, err
	}
	return apichannel.RetrieveResponse{Channels: channels}, nil
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	msg apichannel.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{
		Keys:  unsafe.ReinterpretSlice[channel.Key, uint32](msg.Keys),
		Names: msg.Names,
	}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	msg *DeleteRequest,
) (apichannel.DeleteRequest, error) {
	return apichannel.DeleteRequest{
		Keys:  unsafe.ReinterpretSlice[uint32, channel.Key](msg.Keys),
		Names: msg.Names,
	}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
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
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &ChannelDeleteService_ServiceDesc,
	}
	a.ChannelCreate = c
	a.ChannelRetrieve = r
	a.ChannelDelete = d
	return fgrpc.CompoundBindableTransport{c, r, d}
}
