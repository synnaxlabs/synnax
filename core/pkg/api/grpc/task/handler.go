// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package task

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apitask "github.com/synnaxlabs/synnax/pkg/api/task"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task"
	taskpb "github.com/synnaxlabs/synnax/pkg/service/task/pb"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = fgrpc.UnaryServer[
		apitask.CreateRequest,
		*gapi.TaskCreateRequest,
		apitask.CreateResponse,
		*gapi.TaskCreateResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apitask.RetrieveRequest,
		*gapi.TaskRetrieveRequest,
		apitask.RetrieveResponse,
		*gapi.TaskRetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apitask.DeleteRequest,
		*gapi.TaskDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	copyServer = fgrpc.UnaryServer[
		apitask.CopyRequest,
		*gapi.TaskCopyRequest,
		apitask.CopyResponse,
		*gapi.TaskCopyResponse,
	]
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
	copyRequestTranslator      struct{}
	copyResponseTranslator     struct{}
)

var (
	_ fgrpc.Translator[apitask.CreateRequest, *gapi.TaskCreateRequest]       = createRequestTranslator{}
	_ fgrpc.Translator[apitask.CreateResponse, *gapi.TaskCreateResponse]     = createResponseTranslator{}
	_ fgrpc.Translator[apitask.RetrieveRequest, *gapi.TaskRetrieveRequest]   = retrieveRequestTranslator{}
	_ fgrpc.Translator[apitask.RetrieveResponse, *gapi.TaskRetrieveResponse] = retrieveResponseTranslator{}
	_ fgrpc.Translator[apitask.DeleteRequest, *gapi.TaskDeleteRequest]       = deleteRequestTranslator{}
	_ fgrpc.Translator[apitask.CopyRequest, *gapi.TaskCopyRequest]           = copyRequestTranslator{}
	_ fgrpc.Translator[apitask.CopyResponse, *gapi.TaskCopyResponse]         = copyResponseTranslator{}
)

func (createRequestTranslator) Forward(ctx context.Context, req apitask.CreateRequest) (*gapi.TaskCreateRequest, error) {
	tasks, err := taskpb.TasksToPB(ctx, req.Tasks)
	if err != nil {
		return nil, err
	}
	return &gapi.TaskCreateRequest{Tasks: tasks}, nil
}

func (createRequestTranslator) Backward(ctx context.Context, req *gapi.TaskCreateRequest) (apitask.CreateRequest, error) {
	tasks, err := taskpb.TasksFromPB(ctx, req.Tasks)
	if err != nil {
		return apitask.CreateRequest{}, err
	}
	return apitask.CreateRequest{Tasks: tasks}, nil
}

func (createResponseTranslator) Forward(ctx context.Context, res apitask.CreateResponse) (*gapi.TaskCreateResponse, error) {
	tasks, err := taskpb.TasksToPB(ctx, res.Tasks)
	if err != nil {
		return nil, err
	}
	return &gapi.TaskCreateResponse{Tasks: tasks}, nil
}

func (createResponseTranslator) Backward(ctx context.Context, res *gapi.TaskCreateResponse) (apitask.CreateResponse, error) {
	tasks, err := taskpb.TasksFromPB(ctx, res.Tasks)
	if err != nil {
		return apitask.CreateResponse{}, err
	}
	return apitask.CreateResponse{Tasks: tasks}, nil
}

func (retrieveRequestTranslator) Forward(_ context.Context, req apitask.RetrieveRequest) (*gapi.TaskRetrieveRequest, error) {
	return &gapi.TaskRetrieveRequest{
		Rack:          uint32(req.Rack),
		Keys:          unsafe.ReinterpretSlice[task.Key, uint64](req.Keys),
		Names:         req.Names,
		Types:         req.Types,
		IncludeStatus: req.IncludeStatus,
	}, nil
}

func (retrieveRequestTranslator) Backward(_ context.Context, req *gapi.TaskRetrieveRequest) (apitask.RetrieveRequest, error) {
	return apitask.RetrieveRequest{
		Rack:          rack.Key(req.Rack),
		Keys:          unsafe.ReinterpretSlice[uint64, task.Key](req.Keys),
		Names:         req.Names,
		Types:         req.Types,
		IncludeStatus: req.IncludeStatus,
	}, nil
}

func (retrieveResponseTranslator) Forward(ctx context.Context, res apitask.RetrieveResponse) (*gapi.TaskRetrieveResponse, error) {
	tasks, err := taskpb.TasksToPB(ctx, res.Tasks)
	if err != nil {
		return nil, err
	}
	return &gapi.TaskRetrieveResponse{Tasks: tasks}, nil
}

func (retrieveResponseTranslator) Backward(ctx context.Context, res *gapi.TaskRetrieveResponse) (apitask.RetrieveResponse, error) {
	tasks, err := taskpb.TasksFromPB(ctx, res.Tasks)
	if err != nil {
		return apitask.RetrieveResponse{}, err
	}
	return apitask.RetrieveResponse{Tasks: tasks}, nil
}

func (deleteRequestTranslator) Forward(_ context.Context, req apitask.DeleteRequest) (*gapi.TaskDeleteRequest, error) {
	return &gapi.TaskDeleteRequest{Keys: unsafe.ReinterpretSlice[task.Key, uint64](req.Keys)}, nil
}

func (deleteRequestTranslator) Backward(_ context.Context, req *gapi.TaskDeleteRequest) (apitask.DeleteRequest, error) {
	return apitask.DeleteRequest{Keys: unsafe.ReinterpretSlice[uint64, task.Key](req.Keys)}, nil
}

func (copyRequestTranslator) Forward(_ context.Context, req apitask.CopyRequest) (*gapi.TaskCopyRequest, error) {
	return &gapi.TaskCopyRequest{
		Key:      uint64(req.Key),
		Name:     req.Name,
		Snapshot: req.Snapshot,
	}, nil
}

func (copyRequestTranslator) Backward(_ context.Context, req *gapi.TaskCopyRequest) (apitask.CopyRequest, error) {
	return apitask.CopyRequest{
		Key:      task.Key(req.Key),
		Name:     req.Name,
		Snapshot: req.Snapshot,
	}, nil
}

func (copyResponseTranslator) Forward(ctx context.Context, res apitask.CopyResponse) (*gapi.TaskCopyResponse, error) {
	t, err := taskpb.TaskToPB(ctx, res.Task)
	if err != nil {
		return nil, err
	}
	return &gapi.TaskCopyResponse{Task: t}, nil
}

func (copyResponseTranslator) Backward(ctx context.Context, res *gapi.TaskCopyResponse) (apitask.CopyResponse, error) {
	t, err := taskpb.TaskFromPB(ctx, res.Task)
	if err != nil {
		return apitask.CopyResponse{}, err
	}
	return apitask.CopyResponse{Task: t}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &gapi.TaskCreateService_ServiceDesc,
	}
	a.TaskCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &gapi.TaskRetrieveService_ServiceDesc,
	}
	a.TaskRetrieve = retrieve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.TaskDeleteService_ServiceDesc,
	}
	a.TaskDelete = del
	cpy := &copyServer{
		RequestTranslator:  copyRequestTranslator{},
		ResponseTranslator: copyResponseTranslator{},
		ServiceDesc:        &gapi.TaskCopyService_ServiceDesc,
	}
	a.TaskCopy = cpy

	return fgrpc.CompoundBindableTransport{
		create,
		retrieve,
		del,
		cpy,
	}
}
