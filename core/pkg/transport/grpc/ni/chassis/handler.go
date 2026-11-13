// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package chassis

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/ni/chassis"
	"github.com/synnaxlabs/synnax/pkg/api/ni/chassis/pb"
	svcchassis "github.com/synnaxlabs/synnax/pkg/service/ni/chassis"
	chassispb "github.com/synnaxlabs/synnax/pkg/service/ni/chassis/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		chassis.CreateRequest,
		*CreateRequest,
		chassis.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		chassis.RetrieveRequest,
		*pb.RetrieveRequest,
		chassis.RetrieveResponse,
		*pb.RetrieveResponse,
	]
	updateServer = grpc.UnaryServer[
		chassis.UpdateRequest,
		*UpdateRequest,
		struct{},
		*emptypb.Empty,
	]
	deleteServer = grpc.UnaryServer[
		chassis.DeleteRequest,
		*DeleteRequest,
		struct{},
		*emptypb.Empty,
	]
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	updateRequestTranslator    struct{}
	deleteRequestTranslator    struct{}
)

var (
	_ grpc.Translator[chassis.CreateRequest, *CreateRequest]          = createRequestTranslator{}
	_ grpc.Translator[chassis.CreateResponse, *CreateResponse]        = createResponseTranslator{}
	_ grpc.Translator[chassis.RetrieveRequest, *pb.RetrieveRequest]   = retrieveRequestTranslator{}
	_ grpc.Translator[chassis.RetrieveResponse, *pb.RetrieveResponse] = retrieveResponseTranslator{}
	_ grpc.Translator[chassis.UpdateRequest, *UpdateRequest]          = updateRequestTranslator{}
	_ grpc.Translator[chassis.DeleteRequest, *DeleteRequest]          = deleteRequestTranslator{}
)

func (createRequestTranslator) Forward(
	_ context.Context,
	req chassis.CreateRequest,
) (*CreateRequest, error) {
	list, err := chassispb.ChassisListToPB(req.Chassis)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Chassis: list}, nil
}

func (createRequestTranslator) Backward(
	_ context.Context,
	req *CreateRequest,
) (chassis.CreateRequest, error) {
	list, err := chassispb.ChassisListFromPB(req.Chassis)
	if err != nil {
		return chassis.CreateRequest{}, err
	}
	return chassis.CreateRequest{Chassis: list}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	res chassis.CreateResponse,
) (*CreateResponse, error) {
	list, err := chassispb.ChassisListToPB(res.Chassis)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Chassis: list}, nil
}

func (createResponseTranslator) Backward(
	_ context.Context,
	res *CreateResponse,
) (chassis.CreateResponse, error) {
	list, err := chassispb.ChassisListFromPB(res.Chassis)
	if err != nil {
		return chassis.CreateResponse{}, err
	}
	return chassis.CreateResponse{Chassis: list}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	req chassis.RetrieveRequest,
) (*pb.RetrieveRequest, error) {
	return pb.RetrieveRequestToPB(req)
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	req *pb.RetrieveRequest,
) (chassis.RetrieveRequest, error) {
	return pb.RetrieveRequestFromPB(req)
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	res chassis.RetrieveResponse,
) (*pb.RetrieveResponse, error) {
	return pb.RetrieveResponseToPB(res)
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	res *pb.RetrieveResponse,
) (chassis.RetrieveResponse, error) {
	return pb.RetrieveResponseFromPB(res)
}

func (updateRequestTranslator) Forward(
	_ context.Context,
	req chassis.UpdateRequest,
) (*UpdateRequest, error) {
	update, err := chassispb.UpdateToPB(req.Update)
	if err != nil {
		return nil, err
	}
	return &UpdateRequest{Key: req.Key.String(), Update: update}, nil
}

func (updateRequestTranslator) Backward(
	_ context.Context,
	req *UpdateRequest,
) (chassis.UpdateRequest, error) {
	key, err := uuid.Parse(req.Key)
	if err != nil {
		return chassis.UpdateRequest{}, err
	}
	update, err := chassispb.UpdateFromPB(req.Update)
	if err != nil {
		return chassis.UpdateRequest{}, err
	}
	return chassis.UpdateRequest{Key: key, Update: update}, nil
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	req chassis.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{
		Keys: lo.Map(
			req.Keys,
			func(k svcchassis.Key, _ int) string { return k.String() },
		),
	}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	req *DeleteRequest,
) (chassis.DeleteRequest, error) {
	keys, err := lo.MapErr(req.Keys, func(k string, _ int) (svcchassis.Key, error) {
		return uuid.Parse(k)
	})
	if err != nil {
		return chassis.DeleteRequest{}, err
	}
	return chassis.DeleteRequest{Keys: keys}, nil
}

// New binds the NI chassis routes to t and returns their gRPC transport.
func New(t *api.Transport) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &NIChassisCreateService_ServiceDesc,
	}
	t.NIChassisCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &NIChassisRetrieveService_ServiceDesc,
	}
	t.NIChassisRetrieve = retrieve
	update := &updateServer{
		RequestTranslator:  updateRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &NIChassisUpdateService_ServiceDesc,
	}
	t.NIChassisUpdate = update
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &NIChassisDeleteService_ServiceDesc,
	}
	t.NIChassisDelete = del
	return grpc.CompoundBindableTransport{create, retrieve, update, del}
}
