// Copyright 2026 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	rackpb "github.com/synnaxlabs/synnax/pkg/service/rack/pb"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = fgrpc.UnaryServer[
		apirack.CreateRequest,
		*CreateRequest,
		apirack.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apirack.RetrieveRequest,
		*RetrieveRequest,
		apirack.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apirack.DeleteRequest,
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
	_ fgrpc.Translator[apirack.CreateRequest, *CreateRequest]       = createRequestTranslator{}
	_ fgrpc.Translator[apirack.CreateResponse, *CreateResponse]     = createResponseTranslator{}
	_ fgrpc.Translator[apirack.RetrieveRequest, *RetrieveRequest]   = retrieveRequestTranslator{}
	_ fgrpc.Translator[apirack.RetrieveResponse, *RetrieveResponse] = retrieveResponseTranslator{}
	_ fgrpc.Translator[apirack.DeleteRequest, *DeleteRequest]       = deleteRequestTranslator{}
)

func (createRequestTranslator) Forward(ctx context.Context, req apirack.CreateRequest) (*CreateRequest, error) {
	racks, err := rackpb.RacksToPB(ctx, req.Racks)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Racks: racks}, nil
}

func (createRequestTranslator) Backward(ctx context.Context, req *CreateRequest) (apirack.CreateRequest, error) {
	racks, err := rackpb.RacksFromPB(ctx, req.Racks)
	if err != nil {
		return apirack.CreateRequest{}, err
	}
	return apirack.CreateRequest{Racks: racks}, nil
}

func (createResponseTranslator) Forward(ctx context.Context, res apirack.CreateResponse) (*CreateResponse, error) {
	racks, err := rackpb.RacksToPB(ctx, res.Racks)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Racks: racks}, nil
}

func (createResponseTranslator) Backward(ctx context.Context, res *CreateResponse) (apirack.CreateResponse, error) {
	racks, err := rackpb.RacksFromPB(ctx, res.Racks)
	if err != nil {
		return apirack.CreateResponse{}, err
	}
	return apirack.CreateResponse{Racks: racks}, nil
}

func (retrieveRequestTranslator) Forward(_ context.Context, req apirack.RetrieveRequest) (*RetrieveRequest, error) {
	return &RetrieveRequest{
		Keys:  unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys),
		Names: req.Names,
	}, nil
}

func (retrieveRequestTranslator) Backward(_ context.Context, req *RetrieveRequest) (apirack.RetrieveRequest, error) {
	return apirack.RetrieveRequest{
		Keys:  unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys),
		Names: req.Names,
	}, nil
}

func (retrieveResponseTranslator) Forward(ctx context.Context, res apirack.RetrieveResponse) (*RetrieveResponse, error) {
	racks, err := rackpb.RacksToPB(ctx, res.Racks)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Racks: racks}, nil
}

func (retrieveResponseTranslator) Backward(ctx context.Context, res *RetrieveResponse) (apirack.RetrieveResponse, error) {
	racks, err := rackpb.RacksFromPB(ctx, res.Racks)
	if err != nil {
		return apirack.RetrieveResponse{}, err
	}
	return apirack.RetrieveResponse{Racks: racks}, nil
}

func (deleteRequestTranslator) Forward(_ context.Context, req apirack.DeleteRequest) (*DeleteRequest, error) {
	return &DeleteRequest{Keys: unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys)}, nil
}

func (deleteRequestTranslator) Backward(_ context.Context, req *DeleteRequest) (apirack.DeleteRequest, error) {
	return apirack.DeleteRequest{Keys: unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys)}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &RackCreateService_ServiceDesc,
	}
	a.RackCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &RackRetrieveService_ServiceDesc,
	}
	a.RackRetrieve = retrieve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &RackDeleteService_ServiceDesc,
	}
	a.RackDelete = del

	return fgrpc.CompoundBindableTransport{
		create,
		retrieve,
		del,
	}
}
