/*
	Hello myt name is emi
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
*/

package grpc

import (
	"context"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/gen/go/v1"
	"github.com/synnaxlabs/synnax/pkg/hardware/module"
	"github.com/synnaxlabs/synnax/pkg/hardware/rack"
	"github.com/synnaxlabs/x/unsafe"
	"go/types"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	rackCreateServer = fgrpc.UnaryServer[
		api.HardwareCreateRackRequest,
		*gapi.HardwareCreateRackRequest,
		api.HardwareCreateRackResponse,
		*gapi.HardwareCreateRackResponse,
	]
	rackRetrieveServer = fgrpc.UnaryServer[
		api.HardwareRetrieveRackRequest,
		*gapi.HardwareRetrieveRackRequest,
		api.HardwareRetrieveRackResponse,
		*gapi.HardwareRetrieveRackResponse,
	]
	rackDeleteServer = fgrpc.UnaryServer[
		api.HardwareDeleteRackRequest,
		*gapi.HardwareDeleteRackRequest,
		types.Nil,
		*emptypb.Empty,
	]
	moduleCreateServer = fgrpc.UnaryServer[
		api.HardwareCreateModuleRequest,
		*gapi.HardwareCreateModuleRequest,
		api.HardwareCreateModuleResponse,
		*gapi.HardwareCreateModuleResponse,
	]
	moduleRetrieveServer = fgrpc.UnaryServer[
		api.HardwareRetrieveModuleRequest,
		*gapi.HardwareRetrieveModuleRequest,
		api.HardwareRetrieveModuleResponse,
		*gapi.HardwareRetrieveModuleResponse,
	]
	moduleDeleteServer = fgrpc.UnaryServer[
		api.HardwareDeleteModuleRequest,
		*gapi.HardwareDeleteModuleRequest,
		types.Nil,
		*emptypb.Empty,
	]
	deviceCreateServer = fgrpc.UnaryServer[
		api.HardwareCreateDeviceRequest,
		*gapi.HardwareCreateDeviceRequest,
		api.HardwareCreateDeviceResponse,
		*gapi.HardwareCreateDeviceResponse,
	]
	deviceRetrieveServer = fgrpc.UnaryServer[
		api.HardwareRetrieveDeviceRequest,
		*gapi.HardwareRetrieveDeviceRequest,
		api.HardwareRetrieveDeviceResponse,
		*gapi.HardwareRetrieveDeviceResponse,
	]
	deviceDeleteServer = fgrpc.UnaryServer[
		api.HardwareDeleteDeviceRequest,
		*gapi.HardwareDeleteDeviceRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

type (
	rackCreateRequestTranslator      struct{}
	rackCreateResponseTranslator     struct{}
	rackRetrieveRequestTranslator    struct{}
	rackRetrieveResponseTranslator   struct{}
	rackDeleteRequestTranslator      struct{}
	moduleCreateRequestTranslator    struct{}
	moduleCreateResponseTranslator   struct{}
	moduleRetrieveRequestTranslator  struct{}
	moduleRetrieveResponseTranslator struct{}
	moduleDeleteRequestTranslator    struct{}
	deviceCreateRequestTranslator    struct{}
	deviceCreateResponseTranslator   struct{}
	deviceRetrieveRequestTranslator  struct{}
	deviceRetrieveResponseTranslator struct{}
	deviceDeleteRequestTranslator    struct{}
)

var (
	_ fgrpc.Translator[api.HardwareCreateRackRequest, *gapi.HardwareCreateRackRequest]           = rackCreateRequestTranslator{}
	_ fgrpc.Translator[api.HardwareCreateRackResponse, *gapi.HardwareCreateRackResponse]         = rackCreateResponseTranslator{}
	_ fgrpc.Translator[api.HardwareRetrieveRackRequest, *gapi.HardwareRetrieveRackRequest]       = rackRetrieveRequestTranslator{}
	_ fgrpc.Translator[api.HardwareRetrieveRackResponse, *gapi.HardwareRetrieveRackResponse]     = rackRetrieveResponseTranslator{}
	_ fgrpc.Translator[api.HardwareDeleteRackRequest, *gapi.HardwareDeleteRackRequest]           = rackDeleteRequestTranslator{}
	_ fgrpc.Translator[api.HardwareCreateModuleRequest, *gapi.HardwareCreateModuleRequest]       = moduleCreateRequestTranslator{}
	_ fgrpc.Translator[api.HardwareCreateModuleResponse, *gapi.HardwareCreateModuleResponse]     = moduleCreateResponseTranslator{}
	_ fgrpc.Translator[api.HardwareRetrieveModuleRequest, *gapi.HardwareRetrieveModuleRequest]   = moduleRetrieveRequestTranslator{}
	_ fgrpc.Translator[api.HardwareRetrieveModuleResponse, *gapi.HardwareRetrieveModuleResponse] = moduleRetrieveResponseTranslator{}
	_ fgrpc.Translator[api.HardwareDeleteModuleRequest, *gapi.HardwareDeleteModuleRequest]       = moduleDeleteRequestTranslator{}
	_ fgrpc.Translator[api.HardwareCreateDeviceRequest, *gapi.HardwareCreateDeviceRequest]       = deviceCreateRequestTranslator{}
	_ fgrpc.Translator[api.HardwareCreateDeviceResponse, *gapi.HardwareCreateDeviceResponse]     = deviceCreateResponseTranslator{}
	_ fgrpc.Translator[api.HardwareRetrieveDeviceRequest, *gapi.HardwareRetrieveDeviceRequest]   = deviceRetrieveRequestTranslator{}
	_ fgrpc.Translator[api.HardwareRetrieveDeviceResponse, *gapi.HardwareRetrieveDeviceResponse] = deviceRetrieveResponseTranslator{}
	_ fgrpc.Translator[api.HardwareDeleteDeviceRequest, *gapi.HardwareDeleteDeviceRequest]       = deviceDeleteRequestTranslator{}
)

func translateRackForward(r *api.Rack) *gapi.Rack {
	return &gapi.Rack{Key: uint32(r.Key), Name: r.Name}
}

func translateRackBackward(r *gapi.Rack) *api.Rack {
	return &api.Rack{Key: rack.Key(r.Key), Name: r.Name}
}

func translateRacksForward(rs []api.Rack) []*gapi.Rack {
	res := make([]*gapi.Rack, len(rs))
	for i, r := range rs {
		res[i] = translateRackForward(&r)
	}
	return res
}

func translateRacksBackward(rs []*gapi.Rack) []api.Rack {
	res := make([]api.Rack, len(rs))
	for i, r := range rs {
		res[i] = *translateRackBackward(r)
	}
	return res
}

func (rackCreateRequestTranslator) Forward(_ context.Context, req api.HardwareCreateRackRequest) (*gapi.HardwareCreateRackRequest, error) {
	return &gapi.HardwareCreateRackRequest{Racks: translateRacksForward(req.Racks)}, nil
}

func (rackCreateRequestTranslator) Backward(_ context.Context, req *gapi.HardwareCreateRackRequest) (api.HardwareCreateRackRequest, error) {
	return api.HardwareCreateRackRequest{Racks: translateRacksBackward(req.Racks)}, nil
}

func (rackCreateResponseTranslator) Forward(_ context.Context, res api.HardwareCreateRackResponse) (*gapi.HardwareCreateRackResponse, error) {
	return &gapi.HardwareCreateRackResponse{Racks: translateRacksForward(res.Racks)}, nil
}

func (rackCreateResponseTranslator) Backward(_ context.Context, res *gapi.HardwareCreateRackResponse) (api.HardwareCreateRackResponse, error) {
	return api.HardwareCreateRackResponse{Racks: translateRacksBackward(res.Racks)}, nil
}

func (rackRetrieveRequestTranslator) Forward(_ context.Context, req api.HardwareRetrieveRackRequest) (*gapi.HardwareRetrieveRackRequest, error) {
	return &gapi.HardwareRetrieveRackRequest{Keys: unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys)}, nil
}

func (rackRetrieveRequestTranslator) Backward(_ context.Context, req *gapi.HardwareRetrieveRackRequest) (api.HardwareRetrieveRackRequest, error) {
	return api.HardwareRetrieveRackRequest{Keys: unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys)}, nil
}

func (rackRetrieveResponseTranslator) Forward(_ context.Context, res api.HardwareRetrieveRackResponse) (*gapi.HardwareRetrieveRackResponse, error) {
	return &gapi.HardwareRetrieveRackResponse{Racks: translateRacksForward(res.Racks)}, nil
}

func (rackRetrieveResponseTranslator) Backward(_ context.Context, res *gapi.HardwareRetrieveRackResponse) (api.HardwareRetrieveRackResponse, error) {
	return api.HardwareRetrieveRackResponse{Racks: translateRacksBackward(res.Racks)}, nil
}

func (rackDeleteRequestTranslator) Forward(_ context.Context, req api.HardwareDeleteRackRequest) (*gapi.HardwareDeleteRackRequest, error) {
	return &gapi.HardwareDeleteRackRequest{Keys: unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys)}, nil
}

func (rackDeleteRequestTranslator) Backward(_ context.Context, req *gapi.HardwareDeleteRackRequest) (api.HardwareDeleteRackRequest, error) {
	return api.HardwareDeleteRackRequest{Keys: unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys)}, nil
}

func translateModuleForward(m *api.Module) *gapi.Module {
	return &gapi.Module{Key: uint64(m.Key), Name: m.Name, Type: m.Type, Config: m.Config}
}

func translateModuleBackward(m *gapi.Module) *api.Module {
	return &api.Module{Key: module.Key(m.Key), Name: m.Name, Type: m.Type, Config: m.Config}
}

func translateModulesForward(ms []api.Module) []*gapi.Module {
	res := make([]*gapi.Module, len(ms))
	for i, m := range ms {
		res[i] = translateModuleForward(&m)
	}
	return res
}

func translateModulesBackward(ms []*gapi.Module) []api.Module {
	res := make([]api.Module, len(ms))
	for i, m := range ms {
		res[i] = *translateModuleBackward(m)
	}
	return res
}

func (moduleCreateRequestTranslator) Forward(_ context.Context, req api.HardwareCreateModuleRequest) (*gapi.HardwareCreateModuleRequest, error) {
	return &gapi.HardwareCreateModuleRequest{Modules: translateModulesForward(req.Modules)}, nil
}

func (moduleCreateRequestTranslator) Backward(_ context.Context, req *gapi.HardwareCreateModuleRequest) (api.HardwareCreateModuleRequest, error) {
	return api.HardwareCreateModuleRequest{Modules: translateModulesBackward(req.Modules)}, nil
}

func (moduleCreateResponseTranslator) Forward(_ context.Context, res api.HardwareCreateModuleResponse) (*gapi.HardwareCreateModuleResponse, error) {
	return &gapi.HardwareCreateModuleResponse{Modules: translateModulesForward(res.Modules)}, nil
}

func (moduleCreateResponseTranslator) Backward(_ context.Context, res *gapi.HardwareCreateModuleResponse) (api.HardwareCreateModuleResponse, error) {
	return api.HardwareCreateModuleResponse{Modules: translateModulesBackward(res.Modules)}, nil
}

func (moduleRetrieveRequestTranslator) Forward(_ context.Context, req api.HardwareRetrieveModuleRequest) (*gapi.HardwareRetrieveModuleRequest, error) {
	return &gapi.HardwareRetrieveModuleRequest{Rack: uint32(req.Rack), Keys: unsafe.ReinterpretSlice[module.Key, uint64](req.Keys)}, nil
}

func (moduleRetrieveRequestTranslator) Backward(_ context.Context, req *gapi.HardwareRetrieveModuleRequest) (api.HardwareRetrieveModuleRequest, error) {
	return api.HardwareRetrieveModuleRequest{Rack: rack.Key(req.Rack), Keys: unsafe.ReinterpretSlice[uint64, module.Key](req.Keys)}, nil
}

func (moduleRetrieveResponseTranslator) Forward(_ context.Context, res api.HardwareRetrieveModuleResponse) (*gapi.HardwareRetrieveModuleResponse, error) {
	return &gapi.HardwareRetrieveModuleResponse{Modules: translateModulesForward(res.Modules)}, nil
}

func (moduleRetrieveResponseTranslator) Backward(_ context.Context, res *gapi.HardwareRetrieveModuleResponse) (api.HardwareRetrieveModuleResponse, error) {
	return api.HardwareRetrieveModuleResponse{Modules: translateModulesBackward(res.Modules)}, nil
}

func (moduleDeleteRequestTranslator) Forward(_ context.Context, req api.HardwareDeleteModuleRequest) (*gapi.HardwareDeleteModuleRequest, error) {
	return &gapi.HardwareDeleteModuleRequest{Keys: unsafe.ReinterpretSlice[module.Key, uint64](req.Keys)}, nil
}

func (moduleDeleteRequestTranslator) Backward(_ context.Context, req *gapi.HardwareDeleteModuleRequest) (api.HardwareDeleteModuleRequest, error) {
	return api.HardwareDeleteModuleRequest{Keys: unsafe.ReinterpretSlice[uint64, module.Key](req.Keys)}, nil
}

func translateDeviceForward(d *api.Device) *gapi.Device {
	return &gapi.Device{Key: d.Key, Name: d.Name, Make: d.Make, Model: d.Model, Properties: d.Properties}
}

func translateDeviceBackward(d *gapi.Device) *api.Device {
	return &api.Device{Key: d.Key, Name: d.Name, Make: d.Make, Model: d.Model, Properties: d.Properties}
}

func translateDevicesForward(ds []api.Device) []*gapi.Device {
	res := make([]*gapi.Device, len(ds))
	for i, d := range ds {
		res[i] = translateDeviceForward(&d)
	}
	return res
}

func translateDevicesBackward(ds []*gapi.Device) []api.Device {
	res := make([]api.Device, len(ds))
	for i, d := range ds {
		res[i] = *translateDeviceBackward(d)
	}
	return res
}

func (deviceCreateRequestTranslator) Forward(_ context.Context, req api.HardwareCreateDeviceRequest) (*gapi.HardwareCreateDeviceRequest, error) {
	return &gapi.HardwareCreateDeviceRequest{Devices: translateDevicesForward(req.Devices)}, nil
}

func (deviceCreateRequestTranslator) Backward(_ context.Context, req *gapi.HardwareCreateDeviceRequest) (api.HardwareCreateDeviceRequest, error) {
	return api.HardwareCreateDeviceRequest{Devices: translateDevicesBackward(req.Devices)}, nil
}

func (deviceCreateResponseTranslator) Forward(_ context.Context, res api.HardwareCreateDeviceResponse) (*gapi.HardwareCreateDeviceResponse, error) {
	return &gapi.HardwareCreateDeviceResponse{Devices: translateDevicesForward(res.Devices)}, nil
}

func (deviceCreateResponseTranslator) Backward(_ context.Context, res *gapi.HardwareCreateDeviceResponse) (api.HardwareCreateDeviceResponse, error) {
	return api.HardwareCreateDeviceResponse{Devices: translateDevicesBackward(res.Devices)}, nil
}

func (deviceRetrieveRequestTranslator) Forward(_ context.Context, req api.HardwareRetrieveDeviceRequest) (*gapi.HardwareRetrieveDeviceRequest, error) {
	return &gapi.HardwareRetrieveDeviceRequest{Keys: req.Keys}, nil
}

func (deviceRetrieveRequestTranslator) Backward(_ context.Context, req *gapi.HardwareRetrieveDeviceRequest) (api.HardwareRetrieveDeviceRequest, error) {
	return api.HardwareRetrieveDeviceRequest{Keys: req.Keys}, nil
}

func (deviceRetrieveResponseTranslator) Forward(_ context.Context, res api.HardwareRetrieveDeviceResponse) (*gapi.HardwareRetrieveDeviceResponse, error) {
	return &gapi.HardwareRetrieveDeviceResponse{Devices: translateDevicesForward(res.Devices)}, nil
}

func (deviceRetrieveResponseTranslator) Backward(_ context.Context, res *gapi.HardwareRetrieveDeviceResponse) (api.HardwareRetrieveDeviceResponse, error) {
	return api.HardwareRetrieveDeviceResponse{Devices: translateDevicesBackward(res.Devices)}, nil
}

func (deviceDeleteRequestTranslator) Forward(_ context.Context, req api.HardwareDeleteDeviceRequest) (*gapi.HardwareDeleteDeviceRequest, error) {
	return &gapi.HardwareDeleteDeviceRequest{Keys: req.Keys}, nil
}

func (deviceDeleteRequestTranslator) Backward(_ context.Context, req *gapi.HardwareDeleteDeviceRequest) (api.HardwareDeleteDeviceRequest, error) {
	return api.HardwareDeleteDeviceRequest{Keys: req.Keys}, nil
}

func newHardware(a *api.Transport) fgrpc.BindableTransport {
	createRack := &rackCreateServer{
		RequestTranslator:  rackCreateRequestTranslator{},
		ResponseTranslator: rackCreateResponseTranslator{},
		ServiceDesc:        &gapi.HardwareCreateRackService_ServiceDesc,
	}
	a.HardwareCreateRack = createRack
	retrieveRack := &rackRetrieveServer{
		RequestTranslator:  rackRetrieveRequestTranslator{},
		ResponseTranslator: rackRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.HardwareRetrieveRackService_ServiceDesc,
	}
	a.HardwareRetrieveRack = retrieveRack
	deleteRack := &rackDeleteServer{
		RequestTranslator:  rackDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.HardwareDeleteRackService_ServiceDesc,
	}
	a.HardwareDeleteRack = deleteRack
	createModule := &moduleCreateServer{
		RequestTranslator:  moduleCreateRequestTranslator{},
		ResponseTranslator: moduleCreateResponseTranslator{},
		ServiceDesc:        &gapi.HardwareCreateModuleService_ServiceDesc,
	}
	a.HardwareCreateModule = createModule
	retrieveModule := &moduleRetrieveServer{
		RequestTranslator:  moduleRetrieveRequestTranslator{},
		ResponseTranslator: moduleRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.HardwareRetrieveModuleService_ServiceDesc,
	}
	a.HardwareRetrieveModule = retrieveModule
	deleteModule := &moduleDeleteServer{
		RequestTranslator:  moduleDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.HardwareDeleteModuleService_ServiceDesc,
	}
	a.HardwareDeleteModule = deleteModule
	createDevice := &deviceCreateServer{
		RequestTranslator:  deviceCreateRequestTranslator{},
		ResponseTranslator: deviceCreateResponseTranslator{},
		ServiceDesc:        &gapi.HardwareCreateDeviceService_ServiceDesc,
	}
	a.HardwareCreateDevice = createDevice
	retrieveDevice := &deviceRetrieveServer{
		RequestTranslator:  deviceRetrieveRequestTranslator{},
		ResponseTranslator: deviceRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.HardwareRetrieveDeviceService_ServiceDesc,
	}
	a.HardwareRetrieveDevice = retrieveDevice
	deleteDevice := &deviceDeleteServer{
		RequestTranslator:  deviceDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.HardwareDeleteDeviceService_ServiceDesc,
	}
	a.HardwareDeleteDevice = deleteDevice

	return fgrpc.CompoundBindableTransport{
		createRack,
		retrieveRack,
		deleteRack,
		createModule,
		retrieveModule,
		deleteModule,
		createDevice,
		retrieveDevice,
		deleteDevice,
	}
}
