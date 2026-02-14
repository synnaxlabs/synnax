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

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apichannel "github.com/synnaxlabs/synnax/pkg/api/channel"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/cluster"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
	createServer               = fgrpc.UnaryServer[
		apichannel.CreateRequest,
		*gapi.ChannelCreateRequest,
		apichannel.CreateResponse,
		*gapi.ChannelCreateResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apichannel.RetrieveRequest,
		*gapi.ChannelRetrieveRequest,
		apichannel.RetrieveResponse,
		*gapi.ChannelRetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apichannel.DeleteRequest,
		*gapi.ChannelDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[apichannel.CreateRequest, *gapi.ChannelCreateRequest]       = (*createRequestTranslator)(nil)
	_ fgrpc.Translator[apichannel.CreateResponse, *gapi.ChannelCreateResponse]     = (*createResponseTranslator)(nil)
	_ fgrpc.Translator[apichannel.RetrieveRequest, *gapi.ChannelRetrieveRequest]   = (*retrieveRequestTranslator)(nil)
	_ fgrpc.Translator[apichannel.RetrieveResponse, *gapi.ChannelRetrieveResponse] = (*retrieveResponseTranslator)(nil)
	_ fgrpc.Translator[apichannel.CreateRequest, *gapi.ChannelCreateRequest]       = (*createRequestTranslator)(nil)
)

func TranslateKeysForward(keys []distchannel.Key) []uint32 {
	return unsafe.ReinterpretSlice[distchannel.Key, uint32](keys)
}

func TranslateKeysBackward(keys []uint32) []distchannel.Key {
	return unsafe.ReinterpretSlice[uint32, distchannel.Key](keys)
}

func (t createRequestTranslator) Forward(
	_ context.Context,
	msg apichannel.CreateRequest,
) (*gapi.ChannelCreateRequest, error) {
	return &gapi.ChannelCreateRequest{
		Channels:             lo.Map(msg.Channels, TranslateForward),
		RetrieveIfNameExists: msg.RetrieveIfNameExists,
	}, nil
}

func (t createRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.ChannelCreateRequest,
) (apichannel.CreateRequest, error) {
	return apichannel.CreateRequest{
		Channels:             lo.Map(msg.Channels, TranslateBackward),
		RetrieveIfNameExists: msg.RetrieveIfNameExists,
	}, nil
}

func (t createResponseTranslator) Forward(
	_ context.Context,
	msg apichannel.CreateResponse,
) (*gapi.ChannelCreateResponse, error) {
	return &gapi.ChannelCreateResponse{
		Channels: lo.Map(msg.Channels, TranslateForward),
	}, nil
}

func (t createResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.ChannelCreateResponse,
) (apichannel.CreateResponse, error) {
	return apichannel.CreateResponse{Channels: lo.Map(msg.Channels, TranslateBackward)}, nil
}

func (t retrieveRequestTranslator) Forward(
	_ context.Context,
	msg apichannel.RetrieveRequest,
) (*gapi.ChannelRetrieveRequest, error) {
	return &gapi.ChannelRetrieveRequest{
		NodeKey: uint32(msg.NodeKey),
		Names:   msg.Names,
		Search:  msg.SearchTerm,
		Keys:    unsafe.ReinterpretSlice[distchannel.Key, uint32](msg.Keys),
	}, nil
}

func (t retrieveRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.ChannelRetrieveRequest,
) (apichannel.RetrieveRequest, error) {
	return apichannel.RetrieveRequest{
		NodeKey:    cluster.NodeKey(msg.NodeKey),
		Names:      msg.Names,
		SearchTerm: msg.Search,
		Keys:       unsafe.ReinterpretSlice[uint32, distchannel.Key](msg.Keys),
	}, nil
}

func (t retrieveResponseTranslator) Forward(
	_ context.Context,
	msg apichannel.RetrieveResponse,
) (*gapi.ChannelRetrieveResponse, error) {
	return &gapi.ChannelRetrieveResponse{Channels: lo.Map(msg.Channels, TranslateForward)}, nil
}

func (t retrieveResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.ChannelRetrieveResponse,
) (apichannel.RetrieveResponse, error) {
	return apichannel.RetrieveResponse{Channels: lo.Map(msg.Channels, TranslateBackward)}, nil
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	msg apichannel.DeleteRequest,
) (*gapi.ChannelDeleteRequest, error) {
	return &gapi.ChannelDeleteRequest{
		Keys:  TranslateKeysForward(msg.Keys),
		Names: msg.Names,
	}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.ChannelDeleteRequest,
) (apichannel.DeleteRequest, error) {
	return apichannel.DeleteRequest{
		Keys:  TranslateKeysBackward(msg.Keys),
		Names: msg.Names,
	}, nil
}

func TranslateForward(
	msg apichannel.Channel,
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

func TranslateBackward(
	msg *gapi.Channel,
	_ int,
) apichannel.Channel {
	return apichannel.Channel{
		Key:         distchannel.Key(msg.Key),
		Name:        msg.Name,
		Leaseholder: cluster.NodeKey(msg.Leaseholder),
		DataType:    telem.DataType(msg.DataType),
		Density:     telem.Density(msg.Density),
		IsIndex:     msg.IsIndex,
		Index:       distchannel.Key(msg.Index),
		Virtual:     msg.IsVirtual,
	}
}

func New(a *api.Transport) fgrpc.BindableTransport {
	c := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &gapi.ChannelCreateService_ServiceDesc,
	}
	r := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &gapi.ChannelRetrieveService_ServiceDesc,
	}
	d := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.ChannelDeleteService_ServiceDesc,
	}
	a.ChannelCreate = c
	a.ChannelRetrieve = r
	a.ChannelDelete = d
	return fgrpc.CompoundBindableTransport{c, r, d}
}
