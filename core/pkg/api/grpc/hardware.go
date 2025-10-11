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
	"github.com/synnaxlabs/synnax/pkg/service/hardware/rack"
	"github.com/synnaxlabs/synnax/pkg/service/hardware/task"
	"github.com/synnaxlabs/x/unsafe"
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
	taskCreateServer = fgrpc.UnaryServer[
		api.HardwareCreateTaskRequest,
		*gapi.HardwareCreateTaskRequest,
		api.HardwareCreateTaskResponse,
		*gapi.HardwareCreateTaskResponse,
	]
	taskRetrieveServer = fgrpc.UnaryServer[
		api.HardwareRetrieveTaskRequest,
		*gapi.HardwareRetrieveTaskRequest,
		api.HardwareRetrieveTaskResponse,
		*gapi.HardwareRetrieveTaskResponse,
	]
	taskDeleteServer = fgrpc.UnaryServer[
		api.HardwareDeleteTaskRequest,
		*gapi.HardwareDeleteTaskRequest,
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
	taskCreateRequestTranslator      struct{}
	taskCreateResponseTranslator     struct{}
	taskRetrieveRequestTranslator    struct{}
	taskRetrieveResponseTranslator   struct{}
	taskDeleteRequestTranslator      struct{}
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
	_ fgrpc.Translator[api.HardwareCreateTaskRequest, *gapi.HardwareCreateTaskRequest]           = taskCreateRequestTranslator{}
	_ fgrpc.Translator[api.HardwareCreateTaskResponse, *gapi.HardwareCreateTaskResponse]         = taskCreateResponseTranslator{}
	_ fgrpc.Translator[api.HardwareRetrieveTaskRequest, *gapi.HardwareRetrieveTaskRequest]       = taskRetrieveRequestTranslator{}
	_ fgrpc.Translator[api.HardwareRetrieveTaskResponse, *gapi.HardwareRetrieveTaskResponse]     = taskRetrieveResponseTranslator{}
	_ fgrpc.Translator[api.HardwareDeleteTaskRequest, *gapi.HardwareDeleteTaskRequest]           = taskDeleteRequestTranslator{}
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
	return &gapi.HardwareRetrieveRackRequest{
		Keys:  unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys),
		Names: req.Names,
	}, nil
}

func (rackRetrieveRequestTranslator) Backward(_ context.Context, req *gapi.HardwareRetrieveRackRequest) (api.HardwareRetrieveRackRequest, error) {
	return api.HardwareRetrieveRackRequest{
		Keys:  unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys),
		Names: req.Names,
	}, nil
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

func translateTaskForward(m *api.Task) *gapi.Task {
	return &gapi.Task{
		Key:      uint64(m.Key),
		Name:     m.Name,
		Type:     m.Type,
		Config:   m.Config,
		Internal: m.Internal,
		Snapshot: m.Snapshot,
	}
}

func translateTaskBackward(m *gapi.Task) *api.Task {
	return &api.Task{
		Key:      task.Key(m.Key),
		Name:     m.Name,
		Type:     m.Type,
		Config:   m.Config,
		Internal: m.Internal,
		Snapshot: m.Snapshot,
	}
}

func translateTasksForward(ms []api.Task) []*gapi.Task {
	res := make([]*gapi.Task, len(ms))
	for i, m := range ms {
		res[i] = translateTaskForward(&m)
	}
	return res
}

func translateTasksBackward(ms []*gapi.Task) []api.Task {
	res := make([]api.Task, len(ms))
	for i, m := range ms {
		res[i] = *translateTaskBackward(m)
	}
	return res
}

func (taskCreateRequestTranslator) Forward(_ context.Context, req api.HardwareCreateTaskRequest) (*gapi.HardwareCreateTaskRequest, error) {
	return &gapi.HardwareCreateTaskRequest{Tasks: translateTasksForward(req.Tasks)}, nil
}

func (taskCreateRequestTranslator) Backward(_ context.Context, req *gapi.HardwareCreateTaskRequest) (api.HardwareCreateTaskRequest, error) {
	return api.HardwareCreateTaskRequest{Tasks: translateTasksBackward(req.Tasks)}, nil
}

func (taskCreateResponseTranslator) Forward(_ context.Context, res api.HardwareCreateTaskResponse) (*gapi.HardwareCreateTaskResponse, error) {
	return &gapi.HardwareCreateTaskResponse{Tasks: translateTasksForward(res.Tasks)}, nil
}

func (taskCreateResponseTranslator) Backward(_ context.Context, res *gapi.HardwareCreateTaskResponse) (api.HardwareCreateTaskResponse, error) {
	return api.HardwareCreateTaskResponse{Tasks: translateTasksBackward(res.Tasks)}, nil
}

func (taskRetrieveRequestTranslator) Forward(_ context.Context, req api.HardwareRetrieveTaskRequest) (*gapi.HardwareRetrieveTaskRequest, error) {
	return &gapi.HardwareRetrieveTaskRequest{
		Rack:  uint32(req.Rack),
		Keys:  unsafe.ReinterpretSlice[task.Key, uint64](req.Keys),
		Names: req.Names,
		Types: req.Types,
	}, nil
}

func (taskRetrieveRequestTranslator) Backward(_ context.Context, req *gapi.HardwareRetrieveTaskRequest) (api.HardwareRetrieveTaskRequest, error) {
	return api.HardwareRetrieveTaskRequest{
		Rack:  rack.Key(req.Rack),
		Keys:  unsafe.ReinterpretSlice[uint64, task.Key](req.Keys),
		Names: req.Names,
		Types: req.Types,
	}, nil
}

func (taskRetrieveResponseTranslator) Forward(_ context.Context, res api.HardwareRetrieveTaskResponse) (*gapi.HardwareRetrieveTaskResponse, error) {
	return &gapi.HardwareRetrieveTaskResponse{Tasks: translateTasksForward(res.Tasks)}, nil
}

func (taskRetrieveResponseTranslator) Backward(_ context.Context, res *gapi.HardwareRetrieveTaskResponse) (api.HardwareRetrieveTaskResponse, error) {
	return api.HardwareRetrieveTaskResponse{Tasks: translateTasksBackward(res.Tasks)}, nil
}

func (taskDeleteRequestTranslator) Forward(_ context.Context, req api.HardwareDeleteTaskRequest) (*gapi.HardwareDeleteTaskRequest, error) {
	return &gapi.HardwareDeleteTaskRequest{Keys: unsafe.ReinterpretSlice[task.Key, uint64](req.Keys)}, nil
}

func (taskDeleteRequestTranslator) Backward(_ context.Context, req *gapi.HardwareDeleteTaskRequest) (api.HardwareDeleteTaskRequest, error) {
	return api.HardwareDeleteTaskRequest{Keys: unsafe.ReinterpretSlice[uint64, task.Key](req.Keys)}, nil
}

func translateDeviceForward(d *api.Device) *gapi.Device {
	return &gapi.Device{
		Key:        d.Key,
		Name:       d.Name,
		Location:   d.Location,
		Rack:       uint32(d.Rack),
		Make:       d.Make,
		Model:      d.Model,
		Properties: d.Properties,
		Configured: d.Configured,
	}
}

func translateDeviceBackward(d *gapi.Device) *api.Device {
	return &api.Device{
		Key:        d.Key,
		Name:       d.Name,
		Rack:       rack.Key(d.Rack),
		Location:   d.Location,
		Make:       d.Make,
		Model:      d.Model,
		Properties: d.Properties,
		Configured: d.Configured,
	}
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
	ignoreNotFound := false
	if req.IgnoreNotFound != nil {
		ignoreNotFound = *req.IgnoreNotFound
	}
	return &gapi.HardwareRetrieveDeviceRequest{
		Keys:           req.Keys,
		Names:          req.Names,
		Makes:          req.Makes,
		Models:         req.Models,
		Locations:      req.Locations,
		Search:         req.SearchTerm,
		Racks:          unsafe.ReinterpretSlice[rack.Key, uint32](req.Racks),
		Limit:          uint32(req.Limit),
		Offset:         uint32(req.Offset),
		IgnoreNotFound: ignoreNotFound,
	}, nil
}

func (deviceRetrieveRequestTranslator) Backward(_ context.Context, req *gapi.HardwareRetrieveDeviceRequest) (api.HardwareRetrieveDeviceRequest, error) {
	ignoreNotFound := req.IgnoreNotFound
	return api.HardwareRetrieveDeviceRequest{
		Keys:           req.Keys,
		Names:          req.Names,
		Makes:          req.Makes,
		Models:         req.Models,
		Locations:      req.Locations,
		SearchTerm:     req.Search,
		Limit:          int(req.Limit),
		Racks:          unsafe.ReinterpretSlice[uint32, rack.Key](req.Racks),
		Offset:         int(req.Offset),
		IgnoreNotFound: &ignoreNotFound,
	}, nil
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
	createTask := &taskCreateServer{
		RequestTranslator:  taskCreateRequestTranslator{},
		ResponseTranslator: taskCreateResponseTranslator{},
		ServiceDesc:        &gapi.HardwareCreateTaskService_ServiceDesc,
	}
	a.HardwareCreateTask = createTask
	retrieveTask := &taskRetrieveServer{
		RequestTranslator:  taskRetrieveRequestTranslator{},
		ResponseTranslator: taskRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.HardwareRetrieveTaskService_ServiceDesc,
	}
	a.HardwareRetrieveTask = retrieveTask
	deleteTask := &taskDeleteServer{
		RequestTranslator:  taskDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.HardwareDeleteTaskService_ServiceDesc,
	}
	a.HardwareDeleteTask = deleteTask
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
		createTask,
		retrieveTask,
		deleteTask,
		createDevice,
		retrieveDevice,
		deleteDevice,
	}
}
