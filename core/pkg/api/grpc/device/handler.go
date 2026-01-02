// Copyright 2025 Synnax Labs, Inc.
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
	apidevice "github.com/synnaxlabs/synnax/pkg/api/device"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	devicepb "github.com/synnaxlabs/synnax/pkg/service/device/pb"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = fgrpc.UnaryServer[
		apidevice.CreateRequest,
		*gapi.DeviceCreateRequest,
		apidevice.CreateResponse,
		*gapi.DeviceCreateResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apidevice.RetrieveRequest,
		*gapi.DeviceRetrieveRequest,
		apidevice.RetrieveResponse,
		*gapi.DeviceRetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apidevice.DeleteRequest,
		*gapi.DeviceDeleteRequest,
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
	_ fgrpc.Translator[apidevice.CreateRequest, *gapi.DeviceCreateRequest]       = createRequestTranslator{}
	_ fgrpc.Translator[apidevice.CreateResponse, *gapi.DeviceCreateResponse]     = createResponseTranslator{}
	_ fgrpc.Translator[apidevice.RetrieveRequest, *gapi.DeviceRetrieveRequest]   = retrieveRequestTranslator{}
	_ fgrpc.Translator[apidevice.RetrieveResponse, *gapi.DeviceRetrieveResponse] = retrieveResponseTranslator{}
	_ fgrpc.Translator[apidevice.DeleteRequest, *gapi.DeviceDeleteRequest]       = deleteRequestTranslator{}
)

func (createRequestTranslator) Forward(ctx context.Context, req apidevice.CreateRequest) (*gapi.DeviceCreateRequest, error) {
	devices, err := devicepb.DevicesToPB(ctx, req.Devices)
	if err != nil {
		return nil, err
	}
	return &gapi.DeviceCreateRequest{Devices: devices}, nil
}

func (createRequestTranslator) Backward(ctx context.Context, req *gapi.DeviceCreateRequest) (apidevice.CreateRequest, error) {
	devices, err := devicepb.DevicesFromPB(ctx, req.Devices)
	if err != nil {
		return apidevice.CreateRequest{}, err
	}
	return apidevice.CreateRequest{Devices: devices}, nil
}

func (createResponseTranslator) Forward(ctx context.Context, res apidevice.CreateResponse) (*gapi.DeviceCreateResponse, error) {
	devices, err := devicepb.DevicesToPB(ctx, res.Devices)
	if err != nil {
		return nil, err
	}
	return &gapi.DeviceCreateResponse{Devices: devices}, nil
}

func (createResponseTranslator) Backward(ctx context.Context, res *gapi.DeviceCreateResponse) (apidevice.CreateResponse, error) {
	devices, err := devicepb.DevicesFromPB(ctx, res.Devices)
	if err != nil {
		return apidevice.CreateResponse{}, err
	}
	return apidevice.CreateResponse{Devices: devices}, nil
}

func (retrieveRequestTranslator) Forward(_ context.Context, req apidevice.RetrieveRequest) (*gapi.DeviceRetrieveRequest, error) {
	return &gapi.DeviceRetrieveRequest{
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

func (retrieveRequestTranslator) Backward(_ context.Context, req *gapi.DeviceRetrieveRequest) (apidevice.RetrieveRequest, error) {
	return apidevice.RetrieveRequest{
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

func (retrieveResponseTranslator) Forward(ctx context.Context, res apidevice.RetrieveResponse) (*gapi.DeviceRetrieveResponse, error) {
	devices, err := devicepb.DevicesToPB(ctx, res.Devices)
	if err != nil {
		return nil, err
	}
	return &gapi.DeviceRetrieveResponse{Devices: devices}, nil
}

func (retrieveResponseTranslator) Backward(ctx context.Context, res *gapi.DeviceRetrieveResponse) (apidevice.RetrieveResponse, error) {
	devices, err := devicepb.DevicesFromPB(ctx, res.Devices)
	if err != nil {
		return apidevice.RetrieveResponse{}, err
	}
	return apidevice.RetrieveResponse{Devices: devices}, nil
}

func (deleteRequestTranslator) Forward(_ context.Context, req apidevice.DeleteRequest) (*gapi.DeviceDeleteRequest, error) {
	return &gapi.DeviceDeleteRequest{Keys: req.Keys}, nil
}

func (deleteRequestTranslator) Backward(_ context.Context, req *gapi.DeviceDeleteRequest) (apidevice.DeleteRequest, error) {
	return apidevice.DeleteRequest{Keys: req.Keys}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &gapi.DeviceCreateService_ServiceDesc,
	}
	a.DeviceCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &gapi.DeviceRetrieveService_ServiceDesc,
	}
	a.DeviceRetrieve = retrieve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.DeviceDeleteService_ServiceDesc,
	}
	a.DeviceDelete = del

	return fgrpc.CompoundBindableTransport{
		create,
		retrieve,
		del,
	}
}
