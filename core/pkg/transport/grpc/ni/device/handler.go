// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package device

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/ni/device"
	"github.com/synnaxlabs/synnax/pkg/api/ni/device/pb"
	svcdevice "github.com/synnaxlabs/synnax/pkg/service/ni/device"
	devicepb "github.com/synnaxlabs/synnax/pkg/service/ni/device/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		device.CreateRequest,
		*CreateRequest,
		device.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		device.RetrieveRequest,
		*pb.RetrieveRequest,
		device.RetrieveResponse,
		*pb.RetrieveResponse,
	]
	updateServer = grpc.UnaryServer[
		device.UpdateRequest,
		*UpdateRequest,
		struct{},
		*emptypb.Empty,
	]
	deleteServer = grpc.UnaryServer[
		device.DeleteRequest,
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
	_ grpc.Translator[device.CreateRequest, *CreateRequest]          = createRequestTranslator{}
	_ grpc.Translator[device.CreateResponse, *CreateResponse]        = createResponseTranslator{}
	_ grpc.Translator[device.RetrieveRequest, *pb.RetrieveRequest]   = retrieveRequestTranslator{}
	_ grpc.Translator[device.RetrieveResponse, *pb.RetrieveResponse] = retrieveResponseTranslator{}
	_ grpc.Translator[device.UpdateRequest, *UpdateRequest]          = updateRequestTranslator{}
	_ grpc.Translator[device.DeleteRequest, *DeleteRequest]          = deleteRequestTranslator{}
)

func (createRequestTranslator) Forward(
	_ context.Context,
	req device.CreateRequest,
) (*CreateRequest, error) {
	devices, err := devicepb.DevicesToPB(req.Devices)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Devices: devices}, nil
}

func (createRequestTranslator) Backward(
	_ context.Context,
	req *CreateRequest,
) (device.CreateRequest, error) {
	devices, err := devicepb.DevicesFromPB(req.Devices)
	if err != nil {
		return device.CreateRequest{}, err
	}
	return device.CreateRequest{Devices: devices}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	res device.CreateResponse,
) (*CreateResponse, error) {
	devices, err := devicepb.DevicesToPB(res.Devices)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Devices: devices}, nil
}

func (createResponseTranslator) Backward(
	_ context.Context,
	res *CreateResponse,
) (device.CreateResponse, error) {
	devices, err := devicepb.DevicesFromPB(res.Devices)
	if err != nil {
		return device.CreateResponse{}, err
	}
	return device.CreateResponse{Devices: devices}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	req device.RetrieveRequest,
) (*pb.RetrieveRequest, error) {
	return pb.RetrieveRequestToPB(req)
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	req *pb.RetrieveRequest,
) (device.RetrieveRequest, error) {
	return pb.RetrieveRequestFromPB(req)
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	res device.RetrieveResponse,
) (*pb.RetrieveResponse, error) {
	return pb.RetrieveResponseToPB(res)
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	res *pb.RetrieveResponse,
) (device.RetrieveResponse, error) {
	return pb.RetrieveResponseFromPB(res)
}

func (updateRequestTranslator) Forward(
	_ context.Context,
	req device.UpdateRequest,
) (*UpdateRequest, error) {
	update, err := devicepb.UpdateToPB(req.Update)
	if err != nil {
		return nil, err
	}
	return &UpdateRequest{Key: req.Key.String(), Update: update}, nil
}

func (updateRequestTranslator) Backward(
	_ context.Context,
	req *UpdateRequest,
) (device.UpdateRequest, error) {
	key, err := uuid.Parse(req.Key)
	if err != nil {
		return device.UpdateRequest{}, err
	}
	update, err := devicepb.UpdateFromPB(req.Update)
	if err != nil {
		return device.UpdateRequest{}, err
	}
	return device.UpdateRequest{Key: key, Update: update}, nil
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	req device.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{
		Keys: lo.Map(
			req.Keys,
			func(k svcdevice.Key, _ int) string { return k.String() },
		),
	}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	req *DeleteRequest,
) (device.DeleteRequest, error) {
	keys, err := lo.MapErr(req.Keys, func(k string, _ int) (svcdevice.Key, error) {
		return uuid.Parse(k)
	})
	if err != nil {
		return device.DeleteRequest{}, err
	}
	return device.DeleteRequest{Keys: keys}, nil
}

// New binds the NI device routes to t and returns their gRPC transport.
func New(t *api.Transport) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &NIDeviceCreateService_ServiceDesc,
	}
	t.NIDeviceCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &NIDeviceRetrieveService_ServiceDesc,
	}
	t.NIDeviceRetrieve = retrieve
	update := &updateServer{
		RequestTranslator:  updateRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &NIDeviceUpdateService_ServiceDesc,
	}
	t.NIDeviceUpdate = update
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &NIDeviceDeleteService_ServiceDesc,
	}
	t.NIDeviceDelete = del
	return grpc.CompoundBindableTransport{create, retrieve, update, del}
}
