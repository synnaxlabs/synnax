/*
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
	"github.com/synnaxlabs/synnax/pkg/device/module"
	"github.com/synnaxlabs/synnax/pkg/device/rack"
	"github.com/synnaxlabs/x/unsafe"
	"go/types"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	rackCreateServer = fgrpc.UnaryServer[
		api.DeviceCreateRackRequest,
		*gapi.DeviceCreateRackRequest,
		api.DeviceCreateRackResponse,
		*gapi.DeviceCreateRackResponse,
	]
	rackRetrieveServer = fgrpc.UnaryServer[
		api.DeviceRetrieveRackRequest,
		*gapi.DeviceRetrieveRackRequest,
		api.DeviceRetrieveRackResponse,
		*gapi.DeviceRetrieveRackResponse,
	]
	rackDeleteServer = fgrpc.UnaryServer[
		api.DeviceDeleteRackRequest,
		*gapi.DeviceDeleteRackRequest,
		types.Nil,
		*emptypb.Empty,
	]
	moduleCreateServer = fgrpc.UnaryServer[
		api.DeviceCreateModuleRequest,
		*gapi.DeviceCreateModuleRequest,
		api.DeviceCreateModuleResponse,
		*gapi.DeviceCreateModuleResponse,
	]
	moduleRetrieveServer = fgrpc.UnaryServer[
		api.DeviceRetrieveModuleRequest,
		*gapi.DeviceRetrieveModuleRequest,
		api.DeviceRetrieveModuleResponse,
		*gapi.DeviceRetrieveModuleResponse,
	]
	moduleDeleteServer = fgrpc.UnaryServer[
		api.DeviceDeleteModuleRequest,
		*gapi.DeviceDeleteModuleRequest,
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
	rackDeleteResponseTranslator     struct{}
	moduleCreateRequestTranslator    struct{}
	moduleCreateResponseTranslator   struct{}
	moduleRetrieveRequestTranslator  struct{}
	moduleRetrieveResponseTranslator struct{}
	moduleDeleteRequestTranslator    struct{}
	moduleDeleteResponseTranslator   struct{}
)

var (
	_ fgrpc.Translator[api.DeviceCreateRackRequest, *gapi.DeviceCreateRackRequest]           = rackCreateRequestTranslator{}
	_ fgrpc.Translator[api.DeviceCreateRackResponse, *gapi.DeviceCreateRackResponse]         = rackCreateResponseTranslator{}
	_ fgrpc.Translator[api.DeviceRetrieveRackRequest, *gapi.DeviceRetrieveRackRequest]       = rackRetrieveRequestTranslator{}
	_ fgrpc.Translator[api.DeviceRetrieveRackResponse, *gapi.DeviceRetrieveRackResponse]     = rackRetrieveResponseTranslator{}
	_ fgrpc.Translator[api.DeviceDeleteRackRequest, *gapi.DeviceDeleteRackRequest]           = rackDeleteRequestTranslator{}
	_ fgrpc.Translator[api.DeviceCreateModuleRequest, *gapi.DeviceCreateModuleRequest]       = moduleCreateRequestTranslator{}
	_ fgrpc.Translator[api.DeviceCreateModuleResponse, *gapi.DeviceCreateModuleResponse]     = moduleCreateResponseTranslator{}
	_ fgrpc.Translator[api.DeviceRetrieveModuleRequest, *gapi.DeviceRetrieveModuleRequest]   = moduleRetrieveRequestTranslator{}
	_ fgrpc.Translator[api.DeviceRetrieveModuleResponse, *gapi.DeviceRetrieveModuleResponse] = moduleRetrieveResponseTranslator{}
	_ fgrpc.Translator[api.DeviceDeleteModuleRequest, *gapi.DeviceDeleteModuleRequest]       = moduleDeleteRequestTranslator{}
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

func (rackCreateRequestTranslator) Forward(_ context.Context, req api.DeviceCreateRackRequest) (*gapi.DeviceCreateRackRequest, error) {
	return &gapi.DeviceCreateRackRequest{Racks: translateRacksForward(req.Racks)}, nil
}

func (rackCreateRequestTranslator) Backward(_ context.Context, req *gapi.DeviceCreateRackRequest) (api.DeviceCreateRackRequest, error) {
	return api.DeviceCreateRackRequest{Racks: translateRacksBackward(req.Racks)}, nil
}

func (rackCreateResponseTranslator) Forward(_ context.Context, res api.DeviceCreateRackResponse) (*gapi.DeviceCreateRackResponse, error) {
	return &gapi.DeviceCreateRackResponse{Racks: translateRacksForward(res.Racks)}, nil
}

func (rackCreateResponseTranslator) Backward(_ context.Context, res *gapi.DeviceCreateRackResponse) (api.DeviceCreateRackResponse, error) {
	return api.DeviceCreateRackResponse{Racks: translateRacksBackward(res.Racks)}, nil
}

func (rackRetrieveRequestTranslator) Forward(_ context.Context, req api.DeviceRetrieveRackRequest) (*gapi.DeviceRetrieveRackRequest, error) {
	return &gapi.DeviceRetrieveRackRequest{Keys: unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys)}, nil
}

func (rackRetrieveRequestTranslator) Backward(_ context.Context, req *gapi.DeviceRetrieveRackRequest) (api.DeviceRetrieveRackRequest, error) {
	return api.DeviceRetrieveRackRequest{Keys: unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys)}, nil
}

func (rackRetrieveResponseTranslator) Forward(_ context.Context, res api.DeviceRetrieveRackResponse) (*gapi.DeviceRetrieveRackResponse, error) {
	return &gapi.DeviceRetrieveRackResponse{Racks: translateRacksForward(res.Racks)}, nil
}

func (rackRetrieveResponseTranslator) Backward(_ context.Context, res *gapi.DeviceRetrieveRackResponse) (api.DeviceRetrieveRackResponse, error) {
	return api.DeviceRetrieveRackResponse{Racks: translateRacksBackward(res.Racks)}, nil
}

func (rackDeleteRequestTranslator) Forward(_ context.Context, req api.DeviceDeleteRackRequest) (*gapi.DeviceDeleteRackRequest, error) {
	return &gapi.DeviceDeleteRackRequest{Keys: unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys)}, nil
}

func (rackDeleteRequestTranslator) Backward(_ context.Context, req *gapi.DeviceDeleteRackRequest) (api.DeviceDeleteRackRequest, error) {
	return api.DeviceDeleteRackRequest{Keys: unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys)}, nil
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

func (moduleCreateRequestTranslator) Forward(_ context.Context, req api.DeviceCreateModuleRequest) (*gapi.DeviceCreateModuleRequest, error) {
	return &gapi.DeviceCreateModuleRequest{Modules: translateModulesForward(req.Modules)}, nil
}

func (moduleCreateRequestTranslator) Backward(_ context.Context, req *gapi.DeviceCreateModuleRequest) (api.DeviceCreateModuleRequest, error) {
	return api.DeviceCreateModuleRequest{Modules: translateModulesBackward(req.Modules)}, nil
}

func (moduleCreateResponseTranslator) Forward(_ context.Context, res api.DeviceCreateModuleResponse) (*gapi.DeviceCreateModuleResponse, error) {
	return &gapi.DeviceCreateModuleResponse{Modules: translateModulesForward(res.Modules)}, nil
}

func (moduleCreateResponseTranslator) Backward(_ context.Context, res *gapi.DeviceCreateModuleResponse) (api.DeviceCreateModuleResponse, error) {
	return api.DeviceCreateModuleResponse{Modules: translateModulesBackward(res.Modules)}, nil
}

func (moduleRetrieveRequestTranslator) Forward(_ context.Context, req api.DeviceRetrieveModuleRequest) (*gapi.DeviceRetrieveModuleRequest, error) {
	return &gapi.DeviceRetrieveModuleRequest{Rack: uint32(req.Rack), Keys: unsafe.ReinterpretSlice[module.Key, uint64](req.Keys)}, nil
}

func (moduleRetrieveRequestTranslator) Backward(_ context.Context, req *gapi.DeviceRetrieveModuleRequest) (api.DeviceRetrieveModuleRequest, error) {
	return api.DeviceRetrieveModuleRequest{Rack: rack.Key(req.Rack), Keys: unsafe.ReinterpretSlice[uint64, module.Key](req.Keys)}, nil
}

func (moduleRetrieveResponseTranslator) Forward(_ context.Context, res api.DeviceRetrieveModuleResponse) (*gapi.DeviceRetrieveModuleResponse, error) {
	return &gapi.DeviceRetrieveModuleResponse{Modules: translateModulesForward(res.Modules)}, nil
}

func (moduleRetrieveResponseTranslator) Backward(_ context.Context, res *gapi.DeviceRetrieveModuleResponse) (api.DeviceRetrieveModuleResponse, error) {
	return api.DeviceRetrieveModuleResponse{Modules: translateModulesBackward(res.Modules)}, nil
}

func (moduleDeleteRequestTranslator) Forward(_ context.Context, req api.DeviceDeleteModuleRequest) (*gapi.DeviceDeleteModuleRequest, error) {
	return &gapi.DeviceDeleteModuleRequest{Keys: unsafe.ReinterpretSlice[module.Key, uint64](req.Keys)}, nil
}

func (moduleDeleteRequestTranslator) Backward(_ context.Context, req *gapi.DeviceDeleteModuleRequest) (api.DeviceDeleteModuleRequest, error) {
	return api.DeviceDeleteModuleRequest{Keys: unsafe.ReinterpretSlice[uint64, module.Key](req.Keys)}, nil
}

func newDevice(a *api.Transport) fgrpc.BindableTransport {
	createRack := &rackCreateServer{
		RequestTranslator:  rackCreateRequestTranslator{},
		ResponseTranslator: rackCreateResponseTranslator{},
		ServiceDesc:        &gapi.DeviceCreateRackService_ServiceDesc,
	}
	a.DeviceCreateRack = createRack
	retrieveRack := &rackRetrieveServer{
		RequestTranslator:  rackRetrieveRequestTranslator{},
		ResponseTranslator: rackRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.DeviceRetrieveRackService_ServiceDesc,
	}
	a.DeviceRetrieveRack = retrieveRack
	deleteRack := &rackDeleteServer{
		RequestTranslator:  rackDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.DeviceDeleteRackService_ServiceDesc,
	}
	a.DeviceDeleteRack = deleteRack
	createModule := &moduleCreateServer{
		RequestTranslator:  moduleCreateRequestTranslator{},
		ResponseTranslator: moduleCreateResponseTranslator{},
		ServiceDesc:        &gapi.DeviceCreateModuleService_ServiceDesc,
	}
	a.DeviceCreateModule = createModule
	retrieveModule := &moduleRetrieveServer{
		RequestTranslator:  moduleRetrieveRequestTranslator{},
		ResponseTranslator: moduleRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.DeviceRetrieveModuleService_ServiceDesc,
	}
	a.DeviceRetrieveModule = retrieveModule
	deleteModule := &moduleDeleteServer{
		RequestTranslator:  moduleDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.DeviceDeleteModuleService_ServiceDesc,
	}
	a.DeviceDeleteModule = deleteModule
	return fgrpc.CompoundBindableTransport{
		createRack,
		retrieveRack,
		deleteRack,
		createModule,
		retrieveModule,
		deleteModule,
	}
}
