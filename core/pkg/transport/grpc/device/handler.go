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

	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/api/device/pb"
	devicepb "github.com/synnaxlabs/synnax/pkg/service/legacy/device/pb"
	xpb "github.com/synnaxlabs/x/pb"
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
		*RetrieveRequest,
		device.RetrieveResponse,
		*pb.RetrieveResponse,
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
	deleteRequestTranslator    struct{}
)

var (
	_ grpc.Translator[device.CreateRequest, *CreateRequest]          = createRequestTranslator{}
	_ grpc.Translator[device.CreateResponse, *CreateResponse]        = createResponseTranslator{}
	_ grpc.Translator[device.RetrieveRequest, *RetrieveRequest]      = retrieveRequestTranslator{}
	_ grpc.Translator[device.RetrieveResponse, *pb.RetrieveResponse] = retrieveResponseTranslator{}
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
	r device.RetrieveRequest,
) (*RetrieveRequest, error) {
	req, err := pb.RetrieveRequestToPB(r)
	if err != nil {
		return nil, err
	}
	return &RetrieveRequest{Request: req}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	r *RetrieveRequest,
) (device.RetrieveRequest, error) {
	// A client older than the Core sends the reserved fields, which would otherwise
	// select every device.
	if err := xpb.RejectUnknown(r); err != nil {
		return device.RetrieveRequest{}, err
	}
	return pb.RetrieveRequestFromPB(r.Request)
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

func (deleteRequestTranslator) Forward(
	_ context.Context,
	req device.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{Keys: req.Keys}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	req *DeleteRequest,
) (device.DeleteRequest, error) {
	return device.DeleteRequest{Keys: req.Keys}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &DeviceCreateService_ServiceDesc,
	}
	t.DeviceCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &DeviceRetrieveService_ServiceDesc,
	}
	t.DeviceRetrieve = retrieve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &DeviceDeleteService_ServiceDesc,
	}
	t.DeviceDelete = del

	return grpc.CompoundBindableTransport{create, retrieve, del}
}
