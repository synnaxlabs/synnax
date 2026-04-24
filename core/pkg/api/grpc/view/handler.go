// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package view

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/view"
	"github.com/synnaxlabs/synnax/pkg/service/view/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		view.CreateRequest,
		*CreateRequest,
		view.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		view.RetrieveRequest,
		*RetrieveRequest,
		view.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		view.DeleteRequest,
		*DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
)

var (
	_ grpc.Translator[view.CreateRequest, *CreateRequest]       = createRequestTranslator{}
	_ grpc.Translator[view.CreateResponse, *CreateResponse]     = createResponseTranslator{}
	_ grpc.Translator[view.RetrieveRequest, *RetrieveRequest]   = retrieveRequestTranslator{}
	_ grpc.Translator[view.RetrieveResponse, *RetrieveResponse] = retrieveResponseTranslator{}
	_ grpc.Translator[view.DeleteRequest, *DeleteRequest]       = deleteRequestTranslator{}
)

func (createRequestTranslator) Forward(_ context.Context, req view.CreateRequest) (*CreateRequest, error) {
	views, err := pb.ViewsToPB(req.Views)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Views: views}, nil
}

func (createRequestTranslator) Backward(_ context.Context, req *CreateRequest) (view.CreateRequest, error) {
	views, err := pb.ViewsFromPB(req.GetViews())
	if err != nil {
		return view.CreateRequest{}, err
	}
	return view.CreateRequest{Views: views}, nil
}

func (createResponseTranslator) Forward(_ context.Context, res view.CreateResponse) (*CreateResponse, error) {
	views, err := pb.ViewsToPB(res.Views)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Views: views}, nil
}

func (createResponseTranslator) Backward(_ context.Context, res *CreateResponse) (view.CreateResponse, error) {
	views, err := pb.ViewsFromPB(res.GetViews())
	if err != nil {
		return view.CreateResponse{}, err
	}
	return view.CreateResponse{Views: views}, nil
}

func (retrieveRequestTranslator) Forward(_ context.Context, req view.RetrieveRequest) (*RetrieveRequest, error) {
	keys := make([]string, len(req.Keys))
	for i, k := range req.Keys {
		keys[i] = k.String()
	}
	return &RetrieveRequest{
		Keys:       keys,
		Types:      req.Types,
		SearchTerm: req.SearchTerm,
		Limit:      int32(req.Limit),
		Offset:     int32(req.Offset),
	}, nil
}

func (retrieveRequestTranslator) Backward(_ context.Context, req *RetrieveRequest) (view.RetrieveRequest, error) {
	keys := make([]uuid.UUID, len(req.GetKeys()))
	for i, k := range req.GetKeys() {
		parsed, err := uuid.Parse(k)
		if err != nil {
			return view.RetrieveRequest{}, err
		}
		keys[i] = parsed
	}
	return view.RetrieveRequest{
		Keys:       keys,
		Types:      req.GetTypes(),
		SearchTerm: req.GetSearchTerm(),
		Limit:      int(req.GetLimit()),
		Offset:     int(req.GetOffset()),
	}, nil
}

func (retrieveResponseTranslator) Forward(_ context.Context, res view.RetrieveResponse) (*RetrieveResponse, error) {
	views, err := pb.ViewsToPB(res.Views)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Views: views}, nil
}

func (retrieveResponseTranslator) Backward(_ context.Context, res *RetrieveResponse) (view.RetrieveResponse, error) {
	views, err := pb.ViewsFromPB(res.GetViews())
	if err != nil {
		return view.RetrieveResponse{}, err
	}
	return view.RetrieveResponse{Views: views}, nil
}

func (deleteRequestTranslator) Forward(_ context.Context, req view.DeleteRequest) (*DeleteRequest, error) {
	keys := make([]string, len(req.Keys))
	for i, k := range req.Keys {
		keys[i] = k.String()
	}
	return &DeleteRequest{Keys: keys}, nil
}

func (deleteRequestTranslator) Backward(_ context.Context, req *DeleteRequest) (view.DeleteRequest, error) {
	keys := make([]uuid.UUID, len(req.GetKeys()))
	for i, k := range req.GetKeys() {
		parsed, err := uuid.Parse(k)
		if err != nil {
			return view.DeleteRequest{}, err
		}
		keys[i] = parsed
	}
	return view.DeleteRequest{Keys: keys}, nil
}

func New(a *api.Transport) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &ViewCreateService_ServiceDesc,
	}
	a.ViewCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &ViewRetrieveService_ServiceDesc,
	}
	a.ViewRetrieve = retrieve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &ViewDeleteService_ServiceDesc,
	}
	a.ViewDelete = del

	return grpc.CompoundBindableTransport{
		create,
		retrieve,
		del,
	}
}
