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

	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/rack"
	"github.com/synnaxlabs/synnax/pkg/service/rack/pb"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		rack.CreateRequest,
		*CreateRequest,
		rack.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		rack.RetrieveRequest,
		*RetrieveRequest,
		rack.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		rack.DeleteRequest,
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
	_ grpc.Translator[rack.CreateRequest, *CreateRequest]       = createRequestTranslator{}
	_ grpc.Translator[rack.CreateResponse, *CreateResponse]     = createResponseTranslator{}
	_ grpc.Translator[rack.RetrieveRequest, *RetrieveRequest]   = retrieveRequestTranslator{}
	_ grpc.Translator[rack.RetrieveResponse, *RetrieveResponse] = retrieveResponseTranslator{}
	_ grpc.Translator[rack.DeleteRequest, *DeleteRequest]       = deleteRequestTranslator{}
)

func (createRequestTranslator) Forward(_ context.Context, req rack.CreateRequest) (*CreateRequest, error) {
	racks, err := pb.RacksToPB(req.Racks)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Racks: racks}, nil
}

func (createRequestTranslator) Backward(
	_ context.Context,
	req *CreateRequest,
) (rack.CreateRequest, error) {
	racks, err := pb.RacksFromPB(req.Racks)
	if err != nil {
		return rack.CreateRequest{}, err
	}
	return rack.CreateRequest{Racks: racks}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	res rack.CreateResponse,
) (*CreateResponse, error) {
	racks, err := pb.RacksToPB(res.Racks)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Racks: racks}, nil
}

func (createResponseTranslator) Backward(
	_ context.Context,
	res *CreateResponse,
) (rack.CreateResponse, error) {
	racks, err := pb.RacksFromPB(res.Racks)
	if err != nil {
		return rack.CreateResponse{}, err
	}
	return rack.CreateResponse{Racks: racks}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	req rack.RetrieveRequest,
) (*RetrieveRequest, error) {
	return &RetrieveRequest{
		Keys:          unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys),
		Names:         req.Names,
		Integration:   req.Integration,
		IncludeStatus: req.IncludeStatus,
	}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	req *RetrieveRequest,
) (rack.RetrieveRequest, error) {
	return rack.RetrieveRequest{
		Keys:          unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys),
		Names:         req.Names,
		Integration:   req.Integration,
		IncludeStatus: req.IncludeStatus,
	}, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	res rack.RetrieveResponse,
) (*RetrieveResponse, error) {
	racks, err := pb.RacksToPB(res.Racks)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Racks: racks}, nil
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	res *RetrieveResponse,
) (rack.RetrieveResponse, error) {
	racks, err := pb.RacksFromPB(res.Racks)
	if err != nil {
		return rack.RetrieveResponse{}, err
	}
	return rack.RetrieveResponse{Racks: racks}, nil
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	req rack.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{
		Keys: unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys),
	}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	req *DeleteRequest,
) (rack.DeleteRequest, error) {
	return rack.DeleteRequest{
		Keys: unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys),
	}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &RackCreateService_ServiceDesc,
	}
	t.RackCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &RackRetrieveService_ServiceDesc,
	}
	t.RackRetrieve = retrieve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &RackDeleteService_ServiceDesc,
	}
	t.RackDelete = del

	return grpc.CompoundBindableTransport{create, retrieve, del}
}
