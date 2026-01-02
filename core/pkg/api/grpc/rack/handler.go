// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apirack "github.com/synnaxlabs/synnax/pkg/api/rack"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	rackpb "github.com/synnaxlabs/synnax/pkg/service/rack/pb"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = fgrpc.UnaryServer[
		apirack.CreateRequest,
		*gapi.RackCreateRequest,
		apirack.CreateResponse,
		*gapi.RackCreateResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apirack.RetrieveRequest,
		*gapi.RackRetrieveRequest,
		apirack.RetrieveResponse,
		*gapi.RackRetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apirack.DeleteRequest,
		*gapi.RackDeleteRequest,
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
	_ fgrpc.Translator[apirack.CreateRequest, *gapi.RackCreateRequest]       = createRequestTranslator{}
	_ fgrpc.Translator[apirack.CreateResponse, *gapi.RackCreateResponse]     = createResponseTranslator{}
	_ fgrpc.Translator[apirack.RetrieveRequest, *gapi.RackRetrieveRequest]   = retrieveRequestTranslator{}
	_ fgrpc.Translator[apirack.RetrieveResponse, *gapi.RackRetrieveResponse] = retrieveResponseTranslator{}
	_ fgrpc.Translator[apirack.DeleteRequest, *gapi.RackDeleteRequest]       = deleteRequestTranslator{}
)

func (createRequestTranslator) Forward(ctx context.Context, req apirack.CreateRequest) (*gapi.RackCreateRequest, error) {
	racks, err := rackpb.RacksToPB(ctx, req.Racks)
	if err != nil {
		return nil, err
	}
	return &gapi.RackCreateRequest{Racks: racks}, nil
}

func (createRequestTranslator) Backward(ctx context.Context, req *gapi.RackCreateRequest) (apirack.CreateRequest, error) {
	racks, err := rackpb.RacksFromPB(ctx, req.Racks)
	if err != nil {
		return apirack.CreateRequest{}, err
	}
	return apirack.CreateRequest{Racks: racks}, nil
}

func (createResponseTranslator) Forward(ctx context.Context, res apirack.CreateResponse) (*gapi.RackCreateResponse, error) {
	racks, err := rackpb.RacksToPB(ctx, res.Racks)
	if err != nil {
		return nil, err
	}
	return &gapi.RackCreateResponse{Racks: racks}, nil
}

func (createResponseTranslator) Backward(ctx context.Context, res *gapi.RackCreateResponse) (apirack.CreateResponse, error) {
	racks, err := rackpb.RacksFromPB(ctx, res.Racks)
	if err != nil {
		return apirack.CreateResponse{}, err
	}
	return apirack.CreateResponse{Racks: racks}, nil
}

func (retrieveRequestTranslator) Forward(_ context.Context, req apirack.RetrieveRequest) (*gapi.RackRetrieveRequest, error) {
	return &gapi.RackRetrieveRequest{
		Keys:  unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys),
		Names: req.Names,
	}, nil
}

func (retrieveRequestTranslator) Backward(_ context.Context, req *gapi.RackRetrieveRequest) (apirack.RetrieveRequest, error) {
	return apirack.RetrieveRequest{
		Keys:  unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys),
		Names: req.Names,
	}, nil
}

func (retrieveResponseTranslator) Forward(ctx context.Context, res apirack.RetrieveResponse) (*gapi.RackRetrieveResponse, error) {
	racks, err := rackpb.RacksToPB(ctx, res.Racks)
	if err != nil {
		return nil, err
	}
	return &gapi.RackRetrieveResponse{Racks: racks}, nil
}

func (retrieveResponseTranslator) Backward(ctx context.Context, res *gapi.RackRetrieveResponse) (apirack.RetrieveResponse, error) {
	racks, err := rackpb.RacksFromPB(ctx, res.Racks)
	if err != nil {
		return apirack.RetrieveResponse{}, err
	}
	return apirack.RetrieveResponse{Racks: racks}, nil
}

func (deleteRequestTranslator) Forward(_ context.Context, req apirack.DeleteRequest) (*gapi.RackDeleteRequest, error) {
	return &gapi.RackDeleteRequest{Keys: unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys)}, nil
}

func (deleteRequestTranslator) Backward(_ context.Context, req *gapi.RackDeleteRequest) (apirack.DeleteRequest, error) {
	return apirack.DeleteRequest{Keys: unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys)}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &gapi.RackCreateService_ServiceDesc,
	}
	a.RackCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &gapi.RackRetrieveService_ServiceDesc,
	}
	a.RackRetrieve = retrieve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RackDeleteService_ServiceDesc,
	}
	a.RackDelete = del

	return fgrpc.CompoundBindableTransport{
		create,
		retrieve,
		del,
	}
}
