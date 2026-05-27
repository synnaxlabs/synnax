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
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/service/arc/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
	createServer               = grpc.UnaryServer[
		arc.CreateRequest,
		*CreateRequest,
		arc.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		arc.RetrieveRequest,
		*RetrieveRequest,
		arc.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		arc.DeleteRequest,
		*DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ grpc.Translator[arc.CreateRequest, *CreateRequest]       = (*createRequestTranslator)(nil)
	_ grpc.Translator[arc.CreateResponse, *CreateResponse]     = (*createResponseTranslator)(nil)
	_ grpc.Translator[arc.RetrieveRequest, *RetrieveRequest]   = (*retrieveRequestTranslator)(nil)
	_ grpc.Translator[arc.RetrieveResponse, *RetrieveResponse] = (*retrieveResponseTranslator)(nil)
	_ grpc.Translator[arc.DeleteRequest, *DeleteRequest]       = (*deleteRequestTranslator)(nil)
)

func (createRequestTranslator) Forward(
	_ context.Context,
	msg arc.CreateRequest,
) (*CreateRequest, error) {
	arcs, err := pb.ArcsToPB(msg.Arcs)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Arcs: arcs}, nil
}

func (createRequestTranslator) Backward(
	_ context.Context,
	msg *CreateRequest,
) (arc.CreateRequest, error) {
	arcs, err := pb.ArcsFromPB(msg.Arcs)
	if err != nil {
		return arc.CreateRequest{}, err
	}
	return arc.CreateRequest{Arcs: arcs}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	msg arc.CreateResponse,
) (*CreateResponse, error) {
	pbArcs, err := pb.ArcsToPB(msg.Arcs)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Arcs: pbArcs}, nil
}

func (createResponseTranslator) Backward(
	_ context.Context,
	msg *CreateResponse,
) (arc.CreateResponse, error) {
	arcs, err := pb.ArcsFromPB(msg.Arcs)
	if err != nil {
		return arc.CreateResponse{}, err
	}
	return arc.CreateResponse{Arcs: arcs}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	msg arc.RetrieveRequest,
) (*RetrieveRequest, error) {
	keys := lo.Map(msg.Keys, func(k arc.Key, _ int) string { return k.String() })
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

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	msg *RetrieveRequest,
) (arc.RetrieveRequest, error) {
	keys := make([]arc.Key, 0, len(msg.Keys))
	for _, keyStr := range msg.Keys {
		key, err := uuid.Parse(keyStr)
		if err != nil {
			return arc.RetrieveRequest{}, err
		}
		keys = append(keys, key)
	}
	return arc.RetrieveRequest{
		Keys:          keys,
		Names:         msg.Names,
		SearchTerm:    msg.SearchTerm,
		Limit:         int(msg.Limit),
		Offset:        int(msg.Offset),
		IncludeStatus: msg.IncludeStatus,
		Compile:       msg.Compile,
	}, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	msg arc.RetrieveResponse,
) (*RetrieveResponse, error) {
	arcs, err := pb.ArcsToPB(msg.Arcs)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Arcs: arcs}, nil
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	msg *RetrieveResponse,
) (arc.RetrieveResponse, error) {
	arcs, err := pb.ArcsFromPB(msg.Arcs)
	if err != nil {
		return arc.RetrieveResponse{}, err
	}
	return arc.RetrieveResponse{Arcs: arcs}, nil
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	msg arc.DeleteRequest,
) (*DeleteRequest, error) {
	keys := lo.Map(msg.Keys, func(k arc.Key, _ int) string { return k.String() })
	return &DeleteRequest{Keys: keys}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	msg *DeleteRequest,
) (arc.DeleteRequest, error) {
	keys, err := lo.MapErr(msg.Keys, func(keyStr string, _ int) (arc.Key, error) {
		return uuid.Parse(keyStr)
	})
	if err != nil {
		return arc.DeleteRequest{}, err
	}
	return arc.DeleteRequest{Keys: keys}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
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
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &ArcDeleteService_ServiceDesc,
	}
	t.ArcCreate = c
	t.ArcRetrieve = r
	t.ArcDelete = d
	return grpc.CompoundBindableTransport{c, r, d}
}
