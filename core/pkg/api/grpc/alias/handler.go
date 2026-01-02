// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apialias "github.com/synnaxlabs/synnax/pkg/api/alias"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	setServer = fgrpc.UnaryServer[
		apialias.SetRequest,
		*AliasSetRequest,
		types.Nil,
		*emptypb.Empty,
	]
	resolveServer = fgrpc.UnaryServer[
		apialias.ResolveRequest,
		*AliasResolveRequest,
		apialias.ResolveResponse,
		*AliasResolveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apialias.DeleteRequest,
		*AliasDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	listServer = fgrpc.UnaryServer[
		apialias.ListRequest,
		*AliasListRequest,
		apialias.ListResponse,
		*AliasListResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apialias.RetrieveRequest,
		*AliasRetrieveRequest,
		apialias.RetrieveResponse,
		*AliasRetrieveResponse,
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
	_ fgrpc.Translator[apialias.SetRequest, *AliasSetRequest]             = (*setRequestTranslator)(nil)
	_ fgrpc.Translator[apialias.ResolveRequest, *AliasResolveRequest]     = (*resolveRequestTranslator)(nil)
	_ fgrpc.Translator[apialias.ResolveResponse, *AliasResolveResponse]   = (*resolveResponseTranslator)(nil)
	_ fgrpc.Translator[apialias.DeleteRequest, *AliasDeleteRequest]       = (*deleteRequestTranslator)(nil)
	_ fgrpc.Translator[apialias.ListRequest, *AliasListRequest]           = (*listRequestTranslator)(nil)
	_ fgrpc.Translator[apialias.ListResponse, *AliasListResponse]         = (*listResponseTranslator)(nil)
	_ fgrpc.Translator[apialias.RetrieveRequest, *AliasRetrieveRequest]   = (*retrieveRequestTranslator)(nil)
	_ fgrpc.Translator[apialias.RetrieveResponse, *AliasRetrieveResponse] = (*retrieveResponseTranslator)(nil)
)

func (t setRequestTranslator) Forward(
	_ context.Context,
	r apialias.SetRequest,
) (*AliasSetRequest, error) {
	return &AliasSetRequest{
		Range:   r.Range.String(),
		Aliases: unsafe.ReinterpretMapKeys[channel.Key, uint32, string](r.Aliases),
	}, nil
}

func (t setRequestTranslator) Backward(
	_ context.Context,
	r *AliasSetRequest,
) (apialias.SetRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apialias.SetRequest{
		Range:   key,
		Aliases: unsafe.ReinterpretMapKeys[uint32, channel.Key, string](r.Aliases),
	}, err
}

func (t resolveRequestTranslator) Forward(
	_ context.Context,
	r apialias.ResolveRequest,
) (*AliasResolveRequest, error) {
	return &AliasResolveRequest{
		Range:   r.Range.String(),
		Aliases: r.Aliases,
	}, nil
}

func (t resolveRequestTranslator) Backward(
	_ context.Context,
	r *AliasResolveRequest,
) (apialias.ResolveRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apialias.ResolveRequest{
		Range:   key,
		Aliases: r.Aliases,
	}, err
}

func (t resolveResponseTranslator) Forward(
	_ context.Context,
	r apialias.ResolveResponse,
) (*AliasResolveResponse, error) {
	return &AliasResolveResponse{
		Aliases: unsafe.ReinterpretMapValues[string, channel.Key, uint32](r.Aliases),
	}, nil
}

func (t resolveResponseTranslator) Backward(
	_ context.Context,
	r *AliasResolveResponse,
) (apialias.ResolveResponse, error) {
	return apialias.ResolveResponse{
		Aliases: unsafe.ReinterpretMapValues[string, uint32, channel.Key](r.Aliases),
	}, nil
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	r apialias.DeleteRequest,
) (*AliasDeleteRequest, error) {
	return &AliasDeleteRequest{
		Range:    r.Range.String(),
		Channels: unsafe.ReinterpretSlice[channel.Key, uint32](r.Channels),
	}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	r *AliasDeleteRequest,
) (apialias.DeleteRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apialias.DeleteRequest{
		Range:    key,
		Channels: unsafe.ReinterpretSlice[uint32, channel.Key](r.Channels),
	}, err
}

func (t listRequestTranslator) Forward(
	_ context.Context,
	r apialias.ListRequest,
) (*AliasListRequest, error) {
	return &AliasListRequest{
		Range: r.Range.String(),
	}, nil
}

func (t listRequestTranslator) Backward(
	_ context.Context,
	r *AliasListRequest,
) (apialias.ListRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apialias.ListRequest{
		Range: key,
	}, err
}

func (t listResponseTranslator) Forward(
	_ context.Context,
	r apialias.ListResponse,
) (*AliasListResponse, error) {
	return &AliasListResponse{
		Aliases: unsafe.ReinterpretMapKeys[channel.Key, uint32, string](r.Aliases),
	}, nil
}

func (t listResponseTranslator) Backward(
	_ context.Context,
	r *AliasListResponse,
) (apialias.ListResponse, error) {
	return apialias.ListResponse{
		Aliases: unsafe.ReinterpretMapKeys[uint32, channel.Key, string](r.Aliases),
	}, nil
}

func (t retrieveRequestTranslator) Forward(
	_ context.Context,
	r apialias.RetrieveRequest,
) (*AliasRetrieveRequest, error) {
	return &AliasRetrieveRequest{
		Range:    r.Range.String(),
		Channels: unsafe.ReinterpretSlice[channel.Key, uint32](r.Channels),
	}, nil
}

func (t retrieveRequestTranslator) Backward(
	_ context.Context,
	r *AliasRetrieveRequest,
) (apialias.RetrieveRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apialias.RetrieveRequest{
		Range:    key,
		Channels: unsafe.ReinterpretSlice[uint32, channel.Key](r.Channels),
	}, err
}

func (t retrieveResponseTranslator) Forward(
	_ context.Context,
	r apialias.RetrieveResponse,
) (*AliasRetrieveResponse, error) {
	return &AliasRetrieveResponse{
		Aliases: unsafe.ReinterpretMapKeys[channel.Key, uint32, string](r.Aliases),
	}, nil
}

func (t retrieveResponseTranslator) Backward(
	_ context.Context,
	r *AliasRetrieveResponse,
) (apialias.RetrieveResponse, error) {
	return apialias.RetrieveResponse{
		Aliases: unsafe.ReinterpretMapKeys[uint32, channel.Key, string](r.Aliases),
	}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	set := &setServer{
		RequestTranslator:  setRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &AliasSetService_ServiceDesc,
	}
	a.AliasSet = set
	resolve := &resolveServer{
		RequestTranslator:  resolveRequestTranslator{},
		ResponseTranslator: resolveResponseTranslator{},
		ServiceDesc:        &AliasResolveService_ServiceDesc,
	}
	a.AliasResolve = resolve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &AliasDeleteService_ServiceDesc,
	}
	a.AliasDelete = del
	list := &listServer{
		RequestTranslator:  listRequestTranslator{},
		ResponseTranslator: listResponseTranslator{},
		ServiceDesc:        &AliasListService_ServiceDesc,
	}
	a.AliasList = list
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &AliasRetrieveService_ServiceDesc,
	}
	a.AliasRetrieve = retrieve
	return fgrpc.CompoundBindableTransport{set, resolve, del, list, retrieve}
}
