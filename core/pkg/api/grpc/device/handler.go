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

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/device"
	"github.com/synnaxlabs/synnax/pkg/service/device/pb"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = fgrpc.UnaryServer[
		device.CreateRequest,
		*CreateRequest,
		device.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		device.RetrieveRequest,
		*RetrieveRequest,
		device.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
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
	_ fgrpc.Translator[device.CreateRequest, *CreateRequest]       = createRequestTranslator{}
	_ fgrpc.Translator[device.CreateResponse, *CreateResponse]     = createResponseTranslator{}
	_ fgrpc.Translator[device.RetrieveRequest, *RetrieveRequest]   = retrieveRequestTranslator{}
	_ fgrpc.Translator[device.RetrieveResponse, *RetrieveResponse] = retrieveResponseTranslator{}
	_ fgrpc.Translator[device.DeleteRequest, *DeleteRequest]       = deleteRequestTranslator{}
)

func (createRequestTranslator) Forward(ctx context.Context, req device.CreateRequest) (*CreateRequest, error) {
	devices, err := pb.DevicesToPB(ctx, req.Devices)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Devices: devices}, nil
}

func (createRequestTranslator) Backward(ctx context.Context, req *CreateRequest) (device.CreateRequest, error) {
	devices, err := pb.DevicesFromPB(ctx, req.Devices)
	if err != nil {
		return device.CreateRequest{}, err
	}
	return device.CreateRequest{Devices: devices}, nil
}

func (createResponseTranslator) Forward(ctx context.Context, res device.CreateResponse) (*CreateResponse, error) {
	devices, err := pb.DevicesToPB(ctx, res.Devices)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Devices: devices}, nil
}

func (createResponseTranslator) Backward(ctx context.Context, res *CreateResponse) (device.CreateResponse, error) {
	devices, err := pb.DevicesFromPB(ctx, res.Devices)
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
	}, nil
}

func (retrieveRequestTranslator) Backward(_ context.Context, req *RetrieveRequest) (device.RetrieveRequest, error) {
	return device.RetrieveRequest{
		Keys:           req.Keys,
		Names:          req.Names,
		Makes:          req.Makes,
		Models:         req.Models,
		Locations:      req.Locations,
		SearchTerm:     req.Search,
		Limit:          int(req.Limit),
		Racks:          unsafe.ReinterpretSlice[uint32, rack.Key](req.Racks),
		Offset:         int(req.Offset),
		IgnoreNotFound: req.IgnoreNotFound,
		IncludeStatus:  req.IncludeStatus,
	}, nil
}

func (retrieveResponseTranslator) Forward(ctx context.Context, res device.RetrieveResponse) (*RetrieveResponse, error) {
	devices, err := pb.DevicesToPB(ctx, res.Devices)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Devices: devices}, nil
}

func (retrieveResponseTranslator) Backward(ctx context.Context, res *RetrieveResponse) (device.RetrieveResponse, error) {
	devices, err := pb.DevicesFromPB(ctx, res.Devices)
	if err != nil {
		return device.RetrieveResponse{}, err
	}
	return device.RetrieveResponse{Devices: devices}, nil
}

func (deleteRequestTranslator) Forward(_ context.Context, req device.DeleteRequest) (*DeleteRequest, error) {
	return &DeleteRequest{Keys: req.Keys}, nil
}

func (deleteRequestTranslator) Backward(_ context.Context, req *DeleteRequest) (device.DeleteRequest, error) {
	return device.DeleteRequest{Keys: req.Keys}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
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
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &DeviceDeleteService_ServiceDesc,
	}
	a.DeviceDelete = del

	return fgrpc.CompoundBindableTransport{
		create,
		retrieve,
		del,
	}
}
