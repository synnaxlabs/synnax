// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package alias

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/alias"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	setServer = grpc.UnaryServer[
		alias.SetRequest,
		*SetRequest,
		types.Nil,
		*emptypb.Empty,
	]
	resolveServer = grpc.UnaryServer[
		alias.ResolveRequest,
		*ResolveRequest,
		alias.ResolveResponse,
		*ResolveResponse,
	]
	deleteServer = grpc.UnaryServer[
		alias.DeleteRequest,
		*DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	listServer = grpc.UnaryServer[
		alias.ListRequest,
		*ListRequest,
		alias.ListResponse,
		*ListResponse,
	]
	retrieveServer = grpc.UnaryServer[
		alias.RetrieveRequest,
		*RetrieveRequest,
		alias.RetrieveResponse,
		*RetrieveResponse,
	]
)

type (
	setRequestTranslator       struct{}
	resolveRequestTranslator   struct{}
	resolveResponseTranslator  struct{}
	deleteRequestTranslator    struct{}
	listRequestTranslator      struct{}
	listResponseTranslator     struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
)

var (
	_ grpc.Translator[alias.SetRequest, *SetRequest]             = (*setRequestTranslator)(nil)
	_ grpc.Translator[alias.ResolveRequest, *ResolveRequest]     = (*resolveRequestTranslator)(nil)
	_ grpc.Translator[alias.ResolveResponse, *ResolveResponse]   = (*resolveResponseTranslator)(nil)
	_ grpc.Translator[alias.DeleteRequest, *DeleteRequest]       = (*deleteRequestTranslator)(nil)
	_ grpc.Translator[alias.ListRequest, *ListRequest]           = (*listRequestTranslator)(nil)
	_ grpc.Translator[alias.ListResponse, *ListResponse]         = (*listResponseTranslator)(nil)
	_ grpc.Translator[alias.RetrieveRequest, *RetrieveRequest]   = (*retrieveRequestTranslator)(nil)
	_ grpc.Translator[alias.RetrieveResponse, *RetrieveResponse] = (*retrieveResponseTranslator)(nil)
)

func (setRequestTranslator) Forward(
	_ context.Context,
	r alias.SetRequest,
) (*SetRequest, error) {
	return &SetRequest{
		Range:   r.Range.String(),
		Aliases: unsafe.ReinterpretMapKeys[channel.Key, uint32](r.Aliases),
	}, nil
}

func (setRequestTranslator) Backward(
	_ context.Context,
	r *SetRequest,
) (alias.SetRequest, error) {
	key, err := uuid.Parse(r.Range)
	if err != nil {
		return alias.SetRequest{}, err
	}
	return alias.SetRequest{
		Range:   key,
		Aliases: unsafe.ReinterpretMapKeys[uint32, channel.Key](r.Aliases),
	}, nil
}

func (resolveRequestTranslator) Forward(
	_ context.Context,
	r alias.ResolveRequest,
) (*ResolveRequest, error) {
	return &ResolveRequest{Range: r.Range.String(), Aliases: r.Aliases}, nil
}

func (resolveRequestTranslator) Backward(
	_ context.Context,
	r *ResolveRequest,
) (alias.ResolveRequest, error) {
	key, err := uuid.Parse(r.Range)
	if err != nil {
		return alias.ResolveRequest{}, err
	}
	return alias.ResolveRequest{Range: key, Aliases: r.Aliases}, nil
}

func (resolveResponseTranslator) Forward(
	_ context.Context,
	r alias.ResolveResponse,
) (*ResolveResponse, error) {
	return &ResolveResponse{
		Aliases: unsafe.ReinterpretMapValues[string, channel.Key, uint32](r.Aliases),
	}, nil
}

func (resolveResponseTranslator) Backward(
	_ context.Context,
	r *ResolveResponse,
) (alias.ResolveResponse, error) {
	return alias.ResolveResponse{
		Aliases: unsafe.ReinterpretMapValues[string, uint32, channel.Key](r.Aliases),
	}, nil
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	r alias.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{
		Range:    r.Range.String(),
		Channels: unsafe.ReinterpretSlice[channel.Key, uint32](r.Channels),
	}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	r *DeleteRequest,
) (alias.DeleteRequest, error) {
	key, err := uuid.Parse(r.Range)
	if err != nil {
		return alias.DeleteRequest{}, err
	}
	return alias.DeleteRequest{
		Range:    key,
		Channels: unsafe.ReinterpretSlice[uint32, channel.Key](r.Channels),
	}, nil
}

func (listRequestTranslator) Forward(
	_ context.Context,
	r alias.ListRequest,
) (*ListRequest, error) {
	return &ListRequest{Range: r.Range.String()}, nil
}

func (listRequestTranslator) Backward(
	_ context.Context,
	r *ListRequest,
) (alias.ListRequest, error) {
	key, err := uuid.Parse(r.Range)
	if err != nil {
		return alias.ListRequest{}, err
	}
	return alias.ListRequest{Range: key}, nil
}

func (listResponseTranslator) Forward(
	_ context.Context,
	r alias.ListResponse,
) (*ListResponse, error) {
	return &ListResponse{
		Aliases: unsafe.ReinterpretMapKeys[channel.Key, uint32](r.Aliases),
	}, nil
}

func (listResponseTranslator) Backward(
	_ context.Context,
	r *ListResponse,
) (alias.ListResponse, error) {
	return alias.ListResponse{
		Aliases: unsafe.ReinterpretMapKeys[uint32, channel.Key](r.Aliases),
	}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	r alias.RetrieveRequest,
) (*RetrieveRequest, error) {
	return &RetrieveRequest{
		Range:    r.Range.String(),
		Channels: unsafe.ReinterpretSlice[channel.Key, uint32](r.Channels),
	}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	r *RetrieveRequest,
) (alias.RetrieveRequest, error) {
	key, err := uuid.Parse(r.Range)
	if err != nil {
		return alias.RetrieveRequest{}, err
	}
	return alias.RetrieveRequest{
		Range:    key,
		Channels: unsafe.ReinterpretSlice[uint32, channel.Key](r.Channels),
	}, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	r alias.RetrieveResponse,
) (*RetrieveResponse, error) {
	return &RetrieveResponse{
		Aliases: unsafe.ReinterpretMapKeys[channel.Key, uint32](r.Aliases),
	}, nil
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	r *RetrieveResponse,
) (alias.RetrieveResponse, error) {
	return alias.RetrieveResponse{
		Aliases: unsafe.ReinterpretMapKeys[uint32, channel.Key](r.Aliases),
	}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
	set := &setServer{
		RequestTranslator:  setRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &AliasSetService_ServiceDesc,
	}
	t.AliasSet = set
	resolve := &resolveServer{
		RequestTranslator:  resolveRequestTranslator{},
		ResponseTranslator: resolveResponseTranslator{},
		ServiceDesc:        &AliasResolveService_ServiceDesc,
	}
	t.AliasResolve = resolve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &AliasDeleteService_ServiceDesc,
	}
	t.AliasDelete = del
	list := &listServer{
		RequestTranslator:  listRequestTranslator{},
		ResponseTranslator: listResponseTranslator{},
		ServiceDesc:        &AliasListService_ServiceDesc,
	}
	t.AliasList = list
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &AliasRetrieveService_ServiceDesc,
	}
	t.AliasRetrieve = retrieve
	return grpc.CompoundBindableTransport{set, resolve, del, list, retrieve}
}
