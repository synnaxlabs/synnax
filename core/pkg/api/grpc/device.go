// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/service/device"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	deviceCreateServer = fgrpc.UnaryServer[
		api.DeviceCreateRequest,
		*gapi.DeviceCreateRequest,
		api.DeviceCreateResponse,
		*gapi.DeviceCreateResponse,
	]
	deviceRetrieveServer = fgrpc.UnaryServer[
		api.DeviceRetrieveRequest,
		*gapi.DeviceRetrieveRequest,
		api.DeviceRetrieveResponse,
		*gapi.DeviceRetrieveResponse,
	]
	deviceDeleteServer = fgrpc.UnaryServer[
		api.DeviceDeleteRequest,
		*gapi.DeviceDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

type (
	deviceCreateRequestTranslator    struct{}
	deviceCreateResponseTranslator   struct{}
	deviceRetrieveRequestTranslator  struct{}
	deviceRetrieveResponseTranslator struct{}
	deviceDeleteRequestTranslator    struct{}
)

var (
	_ fgrpc.Translator[api.DeviceCreateRequest, *gapi.DeviceCreateRequest]       = deviceCreateRequestTranslator{}
	_ fgrpc.Translator[api.DeviceCreateResponse, *gapi.DeviceCreateResponse]     = deviceCreateResponseTranslator{}
	_ fgrpc.Translator[api.DeviceRetrieveRequest, *gapi.DeviceRetrieveRequest]   = deviceRetrieveRequestTranslator{}
	_ fgrpc.Translator[api.DeviceRetrieveResponse, *gapi.DeviceRetrieveResponse] = deviceRetrieveResponseTranslator{}
	_ fgrpc.Translator[api.DeviceDeleteRequest, *gapi.DeviceDeleteRequest]       = deviceDeleteRequestTranslator{}
)

func translateDeviceForward(d *api.Device) (*gapi.Device, error) {
	gd := &gapi.Device{
		Key:        d.Key,
		Name:       d.Name,
		Location:   d.Location,
		Rack:       uint32(d.Rack),
		Make:       d.Make,
		Model:      d.Model,
		Properties: d.Properties,
		Configured: d.Configured,
	}
	if d.Status != nil {
		var err error
		gd.Status, err = status.TranslateToPB[device.StatusDetails](status.Status[device.StatusDetails](*d.Status))
		if err != nil {
			return nil, err
		}
	}
	return gd, nil
}

func translateDeviceBackward(d *gapi.Device) (*api.Device, error) {
	ad := &api.Device{
		Key:        d.Key,
		Name:       d.Name,
		Rack:       rack.Key(d.Rack),
		Location:   d.Location,
		Make:       d.Make,
		Model:      d.Model,
		Properties: d.Properties,
		Configured: d.Configured,
	}
	if d.Status != nil {
		s, err := status.TranslateFromPB[device.StatusDetails](d.Status)
		if err != nil {
			return nil, err
		}
		ds := device.Status(s)
		ad.Status = &ds
	}
	return ad, nil
}

func translateDevicesForward(ds []api.Device) ([]*gapi.Device, error) {
	res := make([]*gapi.Device, len(ds))
	for i, d := range ds {
		var err error
		res[i], err = translateDeviceForward(&d)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func translateDevicesBackward(ds []*gapi.Device) ([]api.Device, error) {
	res := make([]api.Device, len(ds))
	for i, d := range ds {
		dd, err := translateDeviceBackward(d)
		if err != nil {
			return nil, err
		}
		res[i] = *dd
	}
	return res, nil
}

func (deviceCreateRequestTranslator) Forward(_ context.Context, req api.DeviceCreateRequest) (*gapi.DeviceCreateRequest, error) {
	devices, err := translateDevicesForward(req.Devices)
	if err != nil {
		return nil, err
	}
	return &gapi.DeviceCreateRequest{Devices: devices}, nil
}

func (deviceCreateRequestTranslator) Backward(_ context.Context, req *gapi.DeviceCreateRequest) (api.DeviceCreateRequest, error) {
	devices, err := translateDevicesBackward(req.Devices)
	if err != nil {
		return api.DeviceCreateRequest{}, err
	}
	return api.DeviceCreateRequest{Devices: devices}, nil
}

func (deviceCreateResponseTranslator) Forward(_ context.Context, res api.DeviceCreateResponse) (*gapi.DeviceCreateResponse, error) {
	devices, err := translateDevicesForward(res.Devices)
	if err != nil {
		return nil, err
	}
	return &gapi.DeviceCreateResponse{Devices: devices}, nil
}

func (deviceCreateResponseTranslator) Backward(_ context.Context, res *gapi.DeviceCreateResponse) (api.DeviceCreateResponse, error) {
	devices, err := translateDevicesBackward(res.Devices)
	if err != nil {
		return api.DeviceCreateResponse{}, err
	}
	return api.DeviceCreateResponse{Devices: devices}, nil
}

func (deviceRetrieveRequestTranslator) Forward(_ context.Context, req api.DeviceRetrieveRequest) (*gapi.DeviceRetrieveRequest, error) {
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

func (deviceRetrieveRequestTranslator) Backward(_ context.Context, req *gapi.DeviceRetrieveRequest) (api.DeviceRetrieveRequest, error) {
	return api.DeviceRetrieveRequest{
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

func (deviceRetrieveResponseTranslator) Forward(_ context.Context, res api.DeviceRetrieveResponse) (*gapi.DeviceRetrieveResponse, error) {
	devices, err := translateDevicesForward(res.Devices)
	if err != nil {
		return nil, err
	}
	return &gapi.DeviceRetrieveResponse{Devices: devices}, nil
}

func (deviceRetrieveResponseTranslator) Backward(_ context.Context, res *gapi.DeviceRetrieveResponse) (api.DeviceRetrieveResponse, error) {
	devices, err := translateDevicesBackward(res.Devices)
	if err != nil {
		return api.DeviceRetrieveResponse{}, err
	}
	return api.DeviceRetrieveResponse{Devices: devices}, nil
}

func (deviceDeleteRequestTranslator) Forward(_ context.Context, req api.DeviceDeleteRequest) (*gapi.DeviceDeleteRequest, error) {
	return &gapi.DeviceDeleteRequest{Keys: req.Keys}, nil
}

func (deviceDeleteRequestTranslator) Backward(_ context.Context, req *gapi.DeviceDeleteRequest) (api.DeviceDeleteRequest, error) {
	return api.DeviceDeleteRequest{Keys: req.Keys}, nil
}

func newDevice(a *api.Transport) fgrpc.BindableTransport {
	createDevice := &deviceCreateServer{
		RequestTranslator:  deviceCreateRequestTranslator{},
		ResponseTranslator: deviceCreateResponseTranslator{},
		ServiceDesc:        &gapi.DeviceCreateService_ServiceDesc,
	}
	a.DeviceCreate = createDevice
	retrieveDevice := &deviceRetrieveServer{
		RequestTranslator:  deviceRetrieveRequestTranslator{},
		ResponseTranslator: deviceRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.DeviceRetrieveService_ServiceDesc,
	}
	a.DeviceRetrieve = retrieveDevice
	deleteDevice := &deviceDeleteServer{
		RequestTranslator:  deviceDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.DeviceDeleteService_ServiceDesc,
	}
	a.DeviceDelete = deleteDevice

	return fgrpc.CompoundBindableTransport{
		createDevice,
		retrieveDevice,
		deleteDevice,
	}
}
