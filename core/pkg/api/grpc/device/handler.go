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
	"go/types"

	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/service/device/pb"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/unsafe"
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
		*RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		device.DeleteRequest,
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
	_ grpc.Translator[device.CreateRequest, *CreateRequest]       = createRequestTranslator{}
	_ grpc.Translator[device.CreateResponse, *CreateResponse]     = createResponseTranslator{}
	_ grpc.Translator[device.RetrieveRequest, *RetrieveRequest]   = retrieveRequestTranslator{}
	_ grpc.Translator[device.RetrieveResponse, *RetrieveResponse] = retrieveResponseTranslator{}
	_ grpc.Translator[device.DeleteRequest, *DeleteRequest]       = deleteRequestTranslator{}
)

func (createRequestTranslator) Forward(_ context.Context, req device.CreateRequest) (*CreateRequest, error) {
	devices, err := pb.DevicesToPB(req.Devices)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Devices: devices}, nil
}

func (createRequestTranslator) Backward(_ context.Context, req *CreateRequest) (device.CreateRequest, error) {
	devices, err := pb.DevicesFromPB(req.GetDevices())
	if err != nil {
		return device.CreateRequest{}, err
	}
	return device.CreateRequest{Devices: devices}, nil
}

func (createResponseTranslator) Forward(_ context.Context, res device.CreateResponse) (*CreateResponse, error) {
	devices, err := pb.DevicesToPB(res.Devices)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Devices: devices}, nil
}

func (createResponseTranslator) Backward(_ context.Context, res *CreateResponse) (device.CreateResponse, error) {
	devices, err := pb.DevicesFromPB(res.GetDevices())
	if err != nil {
		return device.CreateResponse{}, err
	}
	return device.CreateResponse{Devices: devices}, nil
}

func (retrieveRequestTranslator) Forward(_ context.Context, req device.RetrieveRequest) (*RetrieveRequest, error) {
	return &RetrieveRequest{
		Keys:           req.Keys,
		Names:          req.Names,
		Makes:          req.Makes,
		Models:         req.Models,
		Locations:      req.Locations,
		Search:         req.SearchTerm,
		Racks:          unsafe.ReinterpretSlice[rack.Key, uint32](req.Racks),
		Limit:          uint32(req.Limit),
		Offset:         uint32(req.Offset),
		IgnoreNotFound: req.IgnoreNotFound,
		IncludeStatus:  req.IncludeStatus,
		IncludeParent:  req.IncludeParent,
	}, nil
}

func (retrieveRequestTranslator) Backward(_ context.Context, req *RetrieveRequest) (device.RetrieveRequest, error) {
	return device.RetrieveRequest{
		Keys:           req.GetKeys(),
		Names:          req.GetNames(),
		Makes:          req.GetMakes(),
		Models:         req.GetModels(),
		Locations:      req.GetLocations(),
		SearchTerm:     req.GetSearch(),
		Limit:          int(req.GetLimit()),
		Racks:          unsafe.ReinterpretSlice[uint32, rack.Key](req.GetRacks()),
		Offset:         int(req.GetOffset()),
		IgnoreNotFound: req.GetIgnoreNotFound(),
		IncludeStatus:  req.GetIncludeStatus(),
		IncludeParent:  req.GetIncludeParent(),
	}, nil
}

func (retrieveResponseTranslator) Forward(_ context.Context, res device.RetrieveResponse) (*RetrieveResponse, error) {
	devices, err := pb.DevicesToPB(res.Devices)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Devices: devices}, nil
}

func (retrieveResponseTranslator) Backward(_ context.Context, res *RetrieveResponse) (device.RetrieveResponse, error) {
	devices, err := pb.DevicesFromPB(res.GetDevices())
	if err != nil {
		return device.RetrieveResponse{}, err
	}
	return device.RetrieveResponse{Devices: devices}, nil
}

func (deleteRequestTranslator) Forward(_ context.Context, req device.DeleteRequest) (*DeleteRequest, error) {
	return &DeleteRequest{Keys: req.Keys}, nil
}

func (deleteRequestTranslator) Backward(_ context.Context, req *DeleteRequest) (device.DeleteRequest, error) {
	return device.DeleteRequest{Keys: req.GetKeys()}, nil
}

func New(a *api.Transport) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &DeviceCreateService_ServiceDesc,
	}
	a.DeviceCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &DeviceRetrieveService_ServiceDesc,
	}
	a.DeviceRetrieve = retrieve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &DeviceDeleteService_ServiceDesc,
	}
	a.DeviceDelete = del

	return grpc.CompoundBindableTransport{
		create,
		retrieve,
		del,
	}
}
