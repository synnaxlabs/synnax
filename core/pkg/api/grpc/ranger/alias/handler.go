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
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	apialias "github.com/synnaxlabs/synnax/pkg/api/ranger/alias"
	distchannel "github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	setRequestTranslator      struct{}
	deleteRequestTranslator   struct{}
	resolveRequestTranslator  struct{}
	resolveResponseTranslator struct{}
	listRequestTranslator     struct{}
	listResponseTranslator    struct{}
	setServer                 = fgrpc.UnaryServer[
		apialias.SetRequest,
		*gapi.RangeAliasSetRequest,
		types.Nil,
		*emptypb.Empty,
	]
	deleteServer = fgrpc.UnaryServer[
		apialias.DeleteRequest,
		*gapi.RangeAliasDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	resolveServer = fgrpc.UnaryServer[
		apialias.ResolveRequest,
		*gapi.RangeAliasResolveRequest,
		apialias.ResolveResponse,
		*gapi.RangeAliasResolveResponse,
	]
	listServer = fgrpc.UnaryServer[
		apialias.ListRequest,
		*gapi.RangeAliasListRequest,
		apialias.ListResponse,
		*gapi.RangeAliasListResponse,
	]
)

var (
	_ fgrpc.Translator[apialias.SetRequest, *gapi.RangeAliasSetRequest]           = (*setRequestTranslator)(nil)
	_ fgrpc.Translator[apialias.DeleteRequest, *gapi.RangeAliasDeleteRequest]     = (*deleteRequestTranslator)(nil)
	_ fgrpc.Translator[apialias.ResolveRequest, *gapi.RangeAliasResolveRequest]   = (*resolveRequestTranslator)(nil)
	_ fgrpc.Translator[apialias.ResolveResponse, *gapi.RangeAliasResolveResponse] = (*resolveResponseTranslator)(nil)
	_ fgrpc.Translator[apialias.ListRequest, *gapi.RangeAliasListRequest]         = (*listRequestTranslator)(nil)
	_ fgrpc.Translator[apialias.ListResponse, *gapi.RangeAliasListResponse]       = (*listResponseTranslator)(nil)
)

func (t setRequestTranslator) Forward(
	_ context.Context,
	r apialias.SetRequest,
) (*gapi.RangeAliasSetRequest, error) {
	return &gapi.RangeAliasSetRequest{
		Range:   r.Range.String(),
		Aliases: unsafe.ReinterpretMapKeys[distchannel.Key, uint32, string](r.Aliases),
	}, nil
}

func (t setRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasSetRequest,
) (apialias.SetRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apialias.SetRequest{
		Range:   key,
		Aliases: unsafe.ReinterpretMapKeys[uint32, distchannel.Key, string](r.Aliases),
	}, err
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	r apialias.DeleteRequest,
) (*gapi.RangeAliasDeleteRequest, error) {
	return &gapi.RangeAliasDeleteRequest{
		Range:    r.Range.String(),
		Channels: unsafe.ReinterpretSlice[distchannel.Key, uint32](r.Channels),
	}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasDeleteRequest,
) (apialias.DeleteRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apialias.DeleteRequest{
		Range:    key,
		Channels: unsafe.ReinterpretSlice[uint32, distchannel.Key](r.Channels),
	}, err
}

func (t resolveRequestTranslator) Forward(
	_ context.Context,
	r apialias.ResolveRequest,
) (*gapi.RangeAliasResolveRequest, error) {
	return &gapi.RangeAliasResolveRequest{
		Range:   r.Range.String(),
		Aliases: r.Aliases,
	}, nil
}

func (t resolveRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasResolveRequest,
) (apialias.ResolveRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apialias.ResolveRequest{
		Range:   key,
		Aliases: r.Aliases,
	}, err
}

func (t listRequestTranslator) Forward(
	_ context.Context,
	r apialias.ListRequest,
) (*gapi.RangeAliasListRequest, error) {
	return &gapi.RangeAliasListRequest{
		Range: r.Range.String(),
	}, nil
}

func (t listRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasListRequest,
) (apialias.ListRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apialias.ListRequest{
		Range: key,
	}, err
}

func (t resolveResponseTranslator) Forward(
	_ context.Context,
	r apialias.ResolveResponse,
) (*gapi.RangeAliasResolveResponse, error) {
	return &gapi.RangeAliasResolveResponse{
		Aliases: unsafe.ReinterpretMapValues[string, distchannel.Key, uint32](r.Aliases),
	}, nil
}

func (t resolveResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasResolveResponse,
) (apialias.ResolveResponse, error) {
	return apialias.ResolveResponse{
		Aliases: unsafe.ReinterpretMapValues[string, uint32, distchannel.Key](r.Aliases),
	}, nil
}

func (t listResponseTranslator) Forward(
	_ context.Context,
	r apialias.ListResponse,
) (*gapi.RangeAliasListResponse, error) {
	return &gapi.RangeAliasListResponse{
		Aliases: unsafe.ReinterpretMapKeys[distchannel.Key, uint32, string](r.Aliases),
	}, nil
}

func (t listResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasListResponse,
) (apialias.ListResponse, error) {
	return apialias.ListResponse{
		Aliases: unsafe.ReinterpretMapKeys[uint32, distchannel.Key, string](r.Aliases),
	}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	aliasSet := &setServer{
		RequestTranslator:  setRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeAliasSetService_ServiceDesc,
	}
	a.AliasSet = aliasSet
	aliasDelete := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeAliasDeleteService_ServiceDesc,
	}
	a.AliasDelete = aliasDelete
	aliasResolve := &resolveServer{
		RequestTranslator:  resolveRequestTranslator{},
		ResponseTranslator: resolveResponseTranslator{},
		ServiceDesc:        &gapi.RangeAliasResolveService_ServiceDesc,
	}
	a.AliasResolve = aliasResolve
	aliasList := &listServer{
		RequestTranslator:  listRequestTranslator{},
		ResponseTranslator: listResponseTranslator{},
		ServiceDesc:        &gapi.RangeAliasListService_ServiceDesc,
	}
	a.AliasList = aliasList
	return fgrpc.CompoundBindableTransport{
		aliasSet,
		aliasDelete,
		aliasResolve,
		aliasList,
	}
}
