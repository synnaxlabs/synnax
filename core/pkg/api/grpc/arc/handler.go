// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apiarc "github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
	createServer               = fgrpc.UnaryServer[
		apiarc.CreateRequest,
		*CreateRequest,
		apiarc.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apiarc.RetrieveRequest,
		*RetrieveRequest,
		apiarc.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apiarc.DeleteRequest,
		*DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[apiarc.CreateRequest, *CreateRequest]       = (*createRequestTranslator)(nil)
	_ fgrpc.Translator[apiarc.CreateResponse, *CreateResponse]     = (*createResponseTranslator)(nil)
	_ fgrpc.Translator[apiarc.RetrieveRequest, *RetrieveRequest]   = (*retrieveRequestTranslator)(nil)
	_ fgrpc.Translator[apiarc.RetrieveResponse, *RetrieveResponse] = (*retrieveResponseTranslator)(nil)
	_ fgrpc.Translator[apiarc.DeleteRequest, *DeleteRequest]       = (*deleteRequestTranslator)(nil)
)

func (t createRequestTranslator) Forward(
	ctx context.Context,
	msg apiarc.CreateRequest,
) (*CreateRequest, error) {
	arcs, err := pb.ArcsToPB(ctx, msg.Arcs)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Arcs: arcs}, nil
}

func (t createRequestTranslator) Backward(
	ctx context.Context,
	msg *CreateRequest,
) (apiarc.CreateRequest, error) {
	arcs, err := pb.ArcsFromPB(ctx, msg.Arcs)
	if err != nil {
		return apiarc.CreateResponse{}, err
	}
	return apiarc.CreateRequest{Arcs: arcs}, nil
}

func (t createResponseTranslator) Forward(
	ctx context.Context,
	msg apiarc.CreateResponse,
) (*CreateResponse, error) {
	pbArcs, err := pb.ArcsToPB(ctx, msg.Arcs)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Arcs: pbArcs}, nil
}

func (t createResponseTranslator) Backward(
	ctx context.Context,
	msg *CreateResponse,
) (apiarc.CreateResponse, error) {
	arcs, err := pb.ArcsFromPB(ctx, msg.Arcs)
	if err != nil {
		return apiarc.CreateResponse{}, nil
	}
	return apiarc.CreateResponse{Arcs: arcs}, nil
}

func (t retrieveRequestTranslator) Forward(
	_ context.Context,
	msg apiarc.RetrieveRequest,
) (*RetrieveRequest, error) {
	keys := lo.Map(msg.Keys, func(k uuid.UUID, _ int) string { return k.String() })
	return &RetrieveRequest{
		Keys:          keys,
		Names:         msg.Names,
		SearchTerm:    msg.SearchTerm,
		Limit:         int32(msg.Limit),
		Offset:        int32(msg.Offset),
		IncludeStatus: msg.IncludeStatus,
		Compile:       msg.Compile,
	}, nil
}

func (t retrieveRequestTranslator) Backward(
	_ context.Context,
	msg *RetrieveRequest,
) (apiarc.RetrieveRequest, error) {
	keys := make([]uuid.UUID, 0, len(msg.Keys))
	for _, keyStr := range msg.Keys {
		key, err := uuid.Parse(keyStr)
		if err != nil {
			return apiarc.RetrieveRequest{}, err
		}
		keys = append(keys, key)
	}
	return apiarc.RetrieveRequest{
		Keys:          keys,
		Names:         msg.Names,
		SearchTerm:    msg.SearchTerm,
		Limit:         int(msg.Limit),
		Offset:        int(msg.Offset),
		IncludeStatus: msg.IncludeStatus,
		Compile:       msg.Compile,
	}, nil
}

func (t retrieveResponseTranslator) Forward(
	ctx context.Context,
	msg apiarc.RetrieveResponse,
) (*RetrieveResponse, error) {
	arcs, err := pb.ArcsToPB(ctx, msg.Arcs)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Arcs: arcs}, nil
}

func (t retrieveResponseTranslator) Backward(
	ctx context.Context,
	msg *RetrieveResponse,
) (apiarc.RetrieveResponse, error) {
	arcs, err := pb.ArcsFromPB(ctx, msg.Arcs)
	if err != nil {
		return apiarc.RetrieveResponse{}, err
	}
	return apiarc.RetrieveResponse{Arcs: arcs}, nil
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	msg apiarc.DeleteRequest,
) (*DeleteRequest, error) {
	keys := lo.Map(msg.Keys, func(k uuid.UUID, _ int) string { return k.String() })
	return &DeleteRequest{Keys: keys}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	msg *DeleteRequest,
) (apiarc.DeleteRequest, error) {
	keys := make([]uuid.UUID, 0, len(msg.Keys))
	for _, keyStr := range msg.Keys {
		key, err := uuid.Parse(keyStr)
		if err != nil {
			return apiarc.DeleteRequest{}, err
		}
		keys = append(keys, key)
	}
	return apiarc.DeleteRequest{Keys: keys}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	c := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &ArcCreateService_ServiceDesc,
	}
	r := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &ArcRetrieveService_ServiceDesc,
	}
	d := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &ArcDeleteService_ServiceDesc,
	}
	a.ArcCreate = c
	a.ArcRetrieve = r
	a.ArcDelete = d
	return fgrpc.CompoundBindableTransport{c, r, d}
}
