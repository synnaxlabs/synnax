// Copyright 2025 Synnax Labs, Inc.
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

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	channelCreateRequestTranslator    struct{}
	channelCreateResponseTranslator   struct{}
	channelRetrieveRequestTranslator  struct{}
	channelRetrieveResponseTranslator struct{}
	channelDeleteRequestTranslator    struct{}
	createServer                      = fgrpc.UnaryServer[
		api.ChannelCreateRequest,
		*gapi.ChannelCreateRequest,
		api.ChannelCreateResponse,
		*gapi.ChannelCreateResponse,
	]
	channelRetrieveServer = fgrpc.UnaryServer[
		api.ChannelRetrieveRequest,
		*gapi.ChannelRetrieveRequest,
		api.ChannelRetrieveResponse,
		*gapi.ChannelRetrieveResponse,
	]
	channelDeleteServer = fgrpc.UnaryServer[
		api.ChannelDeleteRequest,
		*gapi.ChannelDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[api.ChannelCreateRequest, *gapi.ChannelCreateRequest]       = (*channelCreateRequestTranslator)(nil)
	_ fgrpc.Translator[api.ChannelCreateResponse, *gapi.ChannelCreateResponse]     = (*channelCreateResponseTranslator)(nil)
	_ fgrpc.Translator[api.ChannelRetrieveRequest, *gapi.ChannelRetrieveRequest]   = (*channelRetrieveRequestTranslator)(nil)
	_ fgrpc.Translator[api.ChannelRetrieveResponse, *gapi.ChannelRetrieveResponse] = (*channelRetrieveResponseTranslator)(nil)
	_ fgrpc.Translator[api.ChannelCreateRequest, *gapi.ChannelCreateRequest]       = (*channelCreateRequestTranslator)(nil)
)

func translateChannelKeysForward(keys []channel.Key) []uint32 {
	return unsafe.ReinterpretSlice[channel.Key, uint32](keys)
}

func translateChannelKeysBackward(keys []uint32) []channel.Key {
	return unsafe.ReinterpretSlice[uint32, channel.Key](keys)
}

func (t channelCreateRequestTranslator) Forward(
	_ context.Context,
	msg api.ChannelCreateRequest,
) (*gapi.ChannelCreateRequest, error) {
	return &gapi.ChannelCreateRequest{
		Channels:             lo.Map(msg.Channels, translateChannelForward),
		RetrieveIfNameExists: msg.RetrieveIfNameExists,
	}, nil
}

func (t channelCreateRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.ChannelCreateRequest,
) (api.ChannelCreateRequest, error) {
	return api.ChannelCreateRequest{
		Channels:             lo.Map(msg.Channels, translateChannelBackward),
		RetrieveIfNameExists: msg.RetrieveIfNameExists,
	}, nil
}

func (t channelCreateResponseTranslator) Forward(
	_ context.Context,
	msg api.ChannelCreateResponse,
) (*gapi.ChannelCreateResponse, error) {
	return &gapi.ChannelCreateResponse{
		Channels: lo.Map(msg.Channels, translateChannelForward),
	}, nil
}

func (t channelCreateResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.ChannelCreateResponse,
) (api.ChannelCreateResponse, error) {
	return api.ChannelCreateResponse{Channels: lo.Map(msg.Channels, translateChannelBackward)}, nil
}

func (t channelRetrieveRequestTranslator) Forward(
	_ context.Context,
	msg api.ChannelRetrieveRequest,
) (*gapi.ChannelRetrieveRequest, error) {
	return &gapi.ChannelRetrieveRequest{
		NodeKey: uint32(msg.NodeKey),
		Names:   msg.Names,
		Search:  msg.SearchTerm,
		Keys:    unsafe.ReinterpretSlice[channel.Key, uint32](msg.Keys),
	}, nil
}

func (t channelRetrieveRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.ChannelRetrieveRequest,
) (api.ChannelRetrieveRequest, error) {
	return api.ChannelRetrieveRequest{
		NodeKey:    cluster.NodeKey(msg.NodeKey),
		Names:      msg.Names,
		SearchTerm: msg.Search,
		Keys:       unsafe.ReinterpretSlice[uint32, channel.Key](msg.Keys),
	}, nil
}

func (t channelRetrieveResponseTranslator) Forward(
	_ context.Context,
	msg api.ChannelRetrieveResponse,
) (*gapi.ChannelRetrieveResponse, error) {
	return &gapi.ChannelRetrieveResponse{Channels: lo.Map(msg.Channels, translateChannelForward)}, nil
}

func (t channelRetrieveResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.ChannelRetrieveResponse,
) (api.ChannelRetrieveResponse, error) {
	return api.ChannelRetrieveResponse{Channels: lo.Map(msg.Channels, translateChannelBackward)}, nil
}

func (t channelDeleteRequestTranslator) Forward(
	_ context.Context,
	msg api.ChannelDeleteRequest,
) (*gapi.ChannelDeleteRequest, error) {
	return &gapi.ChannelDeleteRequest{
		Keys:  translateChannelKeysForward(msg.Keys),
		Names: msg.Names,
	}, nil
}

func (t channelDeleteRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.ChannelDeleteRequest,
) (api.ChannelDeleteRequest, error) {
	return api.ChannelDeleteRequest{
		Keys:  translateChannelKeysBackward(msg.Keys),
		Names: msg.Names,
	}, nil
}

func translateChannelForward(
	msg api.Channel,
	_ int,
) *gapi.Channel {
	return &gapi.Channel{
		Key:         uint32(msg.Key),
		Name:        msg.Name,
		Leaseholder: uint32(msg.Leaseholder),
		DataType:    string(msg.DataType),
		Density:     int64(msg.Density),
		IsIndex:     msg.IsIndex,
		Index:       uint32(msg.Index),
		IsVirtual:   msg.Virtual,
	}
}

func translateChannelBackward(
	msg *gapi.Channel,
	_ int,
) api.Channel {
	return api.Channel{
		Key:         channel.Key(msg.Key),
		Name:        msg.Name,
		Leaseholder: cluster.NodeKey(msg.Leaseholder),
		DataType:    telem.DataType(msg.DataType),
		Density:     telem.Density(msg.Density),
		IsIndex:     msg.IsIndex,
		Index:       channel.Key(msg.Index),
		Virtual:     msg.IsVirtual,
	}
}

func newChannel(a *api.Transport) fgrpc.BindableTransport {
	c := &createServer{
		RequestTranslator:  channelCreateRequestTranslator{},
		ResponseTranslator: channelCreateResponseTranslator{},
		ServiceDesc:        &gapi.ChannelCreateService_ServiceDesc,
	}
	r := &channelRetrieveServer{
		RequestTranslator:  channelRetrieveRequestTranslator{},
		ResponseTranslator: channelRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.ChannelRetrieveService_ServiceDesc,
	}
	d := &channelDeleteServer{
		RequestTranslator:  channelDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.ChannelDeleteService_ServiceDesc,
	}
	a.ChannelCreate = c
	a.ChannelRetrieve = r
	a.ChannelDelete = d
	return fgrpc.CompoundBindableTransport{c, r, d}
}
