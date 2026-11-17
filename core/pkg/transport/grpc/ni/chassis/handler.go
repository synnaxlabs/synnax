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
	"github.com/synnaxlabs/x/errors"
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
	_ grpc.BackwardTranslator[chassis.CreateRequest, *CreateRequest]         = createRequestTranslator{}
	_ grpc.ForwardTranslator[chassis.CreateResponse, *CreateResponse]        = createResponseTranslator{}
	_ grpc.BackwardTranslator[chassis.RetrieveRequest, *pb.RetrieveRequest]  = retrieveRequestTranslator{}
	_ grpc.ForwardTranslator[chassis.RetrieveResponse, *pb.RetrieveResponse] = retrieveResponseTranslator{}
	_ grpc.BackwardTranslator[chassis.UpdateRequest, *UpdateRequest]         = updateRequestTranslator{}
	_ grpc.BackwardTranslator[chassis.DeleteRequest, *DeleteRequest]         = deleteRequestTranslator{}
)

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
func New(t *api.Transport, errs *errors.Codec) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &NIChassisCreateService_ServiceDesc,
		Errors:             errs,
	}
	t.NIChassisCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &NIChassisRetrieveService_ServiceDesc,
		Errors:             errs,
	}
	t.NIChassisRetrieve = retrieve
	update := &updateServer{
		RequestTranslator:  updateRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &NIChassisUpdateService_ServiceDesc,
		Errors:             errs,
	}
	t.NIChassisUpdate = update
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &NIChassisDeleteService_ServiceDesc,
		Errors:             errs,
	}
	t.NIChassisDelete = del
	return grpc.CompoundBindableTransport{create, retrieve, update, del}
}
