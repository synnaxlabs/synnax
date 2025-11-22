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
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	taskCreateServer = fgrpc.UnaryServer[
		api.TaskCreateRequest,
		*gapi.TaskCreateRequest,
		api.TaskCreateResponse,
		*gapi.TaskCreateResponse,
	]
	taskRetrieveServer = fgrpc.UnaryServer[
		api.TaskRetrieveRequest,
		*gapi.TaskRetrieveRequest,
		api.TaskRetrieveResponse,
		*gapi.TaskRetrieveResponse,
	]
	taskDeleteServer = fgrpc.UnaryServer[
		api.TaskDeleteRequest,
		*gapi.TaskDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	taskCopyServer = fgrpc.UnaryServer[
		api.TaskCopyRequest,
		*gapi.TaskCopyRequest,
		api.TaskCopyResponse,
		*gapi.TaskCopyResponse,
	]
)

type (
	taskCreateRequestTranslator    struct{}
	taskCreateResponseTranslator   struct{}
	taskRetrieveRequestTranslator  struct{}
	taskRetrieveResponseTranslator struct{}
	taskDeleteRequestTranslator    struct{}
	taskCopyRequestTranslator      struct{}
	taskCopyResponseTranslator     struct{}
)

var (
	_ fgrpc.Translator[api.TaskCreateRequest, *gapi.TaskCreateRequest]       = taskCreateRequestTranslator{}
	_ fgrpc.Translator[api.TaskCreateResponse, *gapi.TaskCreateResponse]     = taskCreateResponseTranslator{}
	_ fgrpc.Translator[api.TaskRetrieveRequest, *gapi.TaskRetrieveRequest]   = taskRetrieveRequestTranslator{}
	_ fgrpc.Translator[api.TaskRetrieveResponse, *gapi.TaskRetrieveResponse] = taskRetrieveResponseTranslator{}
	_ fgrpc.Translator[api.TaskDeleteRequest, *gapi.TaskDeleteRequest]       = taskDeleteRequestTranslator{}
	_ fgrpc.Translator[api.TaskCopyRequest, *gapi.TaskCopyRequest]           = taskCopyRequestTranslator{}
	_ fgrpc.Translator[api.TaskCopyResponse, *gapi.TaskCopyResponse]         = taskCopyResponseTranslator{}
)

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

func (taskCreateRequestTranslator) Forward(_ context.Context, req api.TaskCreateRequest) (*gapi.TaskCreateRequest, error) {
	return &gapi.TaskCreateRequest{Tasks: translateTasksForward(req.Tasks)}, nil
}

func (taskCreateRequestTranslator) Backward(_ context.Context, req *gapi.TaskCreateRequest) (api.TaskCreateRequest, error) {
	return api.TaskCreateRequest{Tasks: translateTasksBackward(req.Tasks)}, nil
}

func (taskCreateResponseTranslator) Forward(_ context.Context, res api.TaskCreateResponse) (*gapi.TaskCreateResponse, error) {
	return &gapi.TaskCreateResponse{Tasks: translateTasksForward(res.Tasks)}, nil
}

func (taskCreateResponseTranslator) Backward(_ context.Context, res *gapi.TaskCreateResponse) (api.TaskCreateResponse, error) {
	return api.TaskCreateResponse{Tasks: translateTasksBackward(res.Tasks)}, nil
}

func (taskRetrieveRequestTranslator) Forward(_ context.Context, req api.TaskRetrieveRequest) (*gapi.TaskRetrieveRequest, error) {
	return &gapi.TaskRetrieveRequest{
		Rack:  uint32(req.Rack),
		Keys:  unsafe.ReinterpretSlice[task.Key, uint64](req.Keys),
		Names: req.Names,
		Types: req.Types,
	}, nil
}

func (taskRetrieveRequestTranslator) Backward(_ context.Context, req *gapi.TaskRetrieveRequest) (api.TaskRetrieveRequest, error) {
	return api.TaskRetrieveRequest{
		Rack:  rack.Key(req.Rack),
		Keys:  unsafe.ReinterpretSlice[uint64, task.Key](req.Keys),
		Names: req.Names,
		Types: req.Types,
	}, nil
}

func (taskRetrieveResponseTranslator) Forward(_ context.Context, res api.TaskRetrieveResponse) (*gapi.TaskRetrieveResponse, error) {
	return &gapi.TaskRetrieveResponse{Tasks: translateTasksForward(res.Tasks)}, nil
}

func (taskRetrieveResponseTranslator) Backward(_ context.Context, res *gapi.TaskRetrieveResponse) (api.TaskRetrieveResponse, error) {
	return api.TaskRetrieveResponse{Tasks: translateTasksBackward(res.Tasks)}, nil
}

func (taskDeleteRequestTranslator) Forward(_ context.Context, req api.TaskDeleteRequest) (*gapi.TaskDeleteRequest, error) {
	return &gapi.TaskDeleteRequest{Keys: unsafe.ReinterpretSlice[task.Key, uint64](req.Keys)}, nil
}

func (taskDeleteRequestTranslator) Backward(_ context.Context, req *gapi.TaskDeleteRequest) (api.TaskDeleteRequest, error) {
	return api.TaskDeleteRequest{Keys: unsafe.ReinterpretSlice[uint64, task.Key](req.Keys)}, nil
}

func (taskCopyRequestTranslator) Forward(_ context.Context, req api.TaskCopyRequest) (*gapi.TaskCopyRequest, error) {
	return &gapi.TaskCopyRequest{
		Key:      uint64(req.Key),
		Name:     req.Name,
		Snapshot: req.Snapshot,
	}, nil
}

func (taskCopyRequestTranslator) Backward(_ context.Context, req *gapi.TaskCopyRequest) (api.TaskCopyRequest, error) {
	return api.TaskCopyRequest{
		Key:      task.Key(req.Key),
		Name:     req.Name,
		Snapshot: req.Snapshot,
	}, nil
}

func (taskCopyResponseTranslator) Forward(_ context.Context, res api.TaskCopyResponse) (*gapi.TaskCopyResponse, error) {
	return &gapi.TaskCopyResponse{Task: translateTaskForward(&res.Task)}, nil
}

func (taskCopyResponseTranslator) Backward(_ context.Context, res *gapi.TaskCopyResponse) (api.TaskCopyResponse, error) {
	return api.TaskCopyResponse{Task: *translateTaskBackward(res.Task)}, nil
}

func newTask(a *api.Transport) fgrpc.BindableTransport {
	createTask := &taskCreateServer{
		RequestTranslator:  taskCreateRequestTranslator{},
		ResponseTranslator: taskCreateResponseTranslator{},
		ServiceDesc:        &gapi.TaskCreateService_ServiceDesc,
	}
	a.TaskCreate = createTask
	retrieveTask := &taskRetrieveServer{
		RequestTranslator:  taskRetrieveRequestTranslator{},
		ResponseTranslator: taskRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.TaskRetrieveService_ServiceDesc,
	}
	a.TaskRetrieve = retrieveTask
	deleteTask := &taskDeleteServer{
		RequestTranslator:  taskDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.TaskDeleteService_ServiceDesc,
	}
	a.TaskDelete = deleteTask
	copyTask := &taskCopyServer{
		RequestTranslator:  taskCopyRequestTranslator{},
		ResponseTranslator: taskCopyResponseTranslator{},
		ServiceDesc:        &gapi.TaskCopyService_ServiceDesc,
	}
	a.TaskCopy = copyTask

	return fgrpc.CompoundBindableTransport{
		createTask,
		retrieveTask,
		deleteTask,
		copyTask,
	}
}
