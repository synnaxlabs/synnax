// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/task"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/synnax/pkg/service/task/pb"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		task.CreateRequest,
		*CreateRequest,
		task.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		task.RetrieveRequest,
		*RetrieveRequest,
		task.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		task.DeleteRequest,
		*DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	copyServer = grpc.UnaryServer[
		task.CopyRequest,
		*CopyRequest,
		task.CopyResponse,
		*CopyResponse,
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
	_ grpc.Translator[task.CreateRequest, *CreateRequest]       = createRequestTranslator{}
	_ grpc.Translator[task.CreateResponse, *CreateResponse]     = createResponseTranslator{}
	_ grpc.Translator[task.RetrieveRequest, *RetrieveRequest]   = retrieveRequestTranslator{}
	_ grpc.Translator[task.RetrieveResponse, *RetrieveResponse] = retrieveResponseTranslator{}
	_ grpc.Translator[task.DeleteRequest, *DeleteRequest]       = deleteRequestTranslator{}
	_ grpc.Translator[task.CopyRequest, *CopyRequest]           = copyRequestTranslator{}
	_ grpc.Translator[task.CopyResponse, *CopyResponse]         = copyResponseTranslator{}
)

func (createRequestTranslator) Forward(
	_ context.Context,
	req task.CreateRequest,
) (*CreateRequest, error) {
	tasks, err := pb.TasksToPB(req.Tasks)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Tasks: tasks}, nil
}

func (createRequestTranslator) Backward(
	_ context.Context,
	req *CreateRequest,
) (task.CreateRequest, error) {
	tasks, err := pb.TasksFromPB(req.Tasks)
	if err != nil {
		return task.CreateRequest{}, err
	}
	return task.CreateRequest{Tasks: tasks}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	res task.CreateResponse,
) (*CreateResponse, error) {
	tasks, err := pb.TasksToPB(res.Tasks)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Tasks: tasks}, nil
}

func (createResponseTranslator) Backward(
	_ context.Context,
	res *CreateResponse,
) (task.CreateResponse, error) {
	tasks, err := pb.TasksFromPB(res.Tasks)
	if err != nil {
		return task.CreateResponse{}, err
	}
	return task.CreateResponse{Tasks: tasks}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	req task.RetrieveRequest,
) (*RetrieveRequest, error) {
	return &RetrieveRequest{
		Rack:          uint32(req.Rack),
		Keys:          unsafe.ReinterpretSlice[task.Key, uint64](req.Keys),
		Names:         req.Names,
		Types:         req.Types,
		IncludeStatus: req.IncludeStatus,
	}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	req *RetrieveRequest,
) (task.RetrieveRequest, error) {
	return task.RetrieveRequest{
		Rack:          rack.Key(req.Rack),
		Keys:          unsafe.ReinterpretSlice[uint64, task.Key](req.Keys),
		Names:         req.Names,
		Types:         req.Types,
		IncludeStatus: req.IncludeStatus,
	}, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	res task.RetrieveResponse,
) (*RetrieveResponse, error) {
	tasks, err := pb.TasksToPB(res.Tasks)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Tasks: tasks}, nil
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	res *RetrieveResponse,
) (task.RetrieveResponse, error) {
	tasks, err := pb.TasksFromPB(res.Tasks)
	if err != nil {
		return task.RetrieveResponse{}, err
	}
	return task.RetrieveResponse{Tasks: tasks}, nil
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	req task.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{
		Keys: unsafe.ReinterpretSlice[task.Key, uint64](req.Keys),
	}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	req *DeleteRequest,
) (task.DeleteRequest, error) {
	return task.DeleteRequest{
		Keys: unsafe.ReinterpretSlice[uint64, task.Key](req.Keys),
	}, nil
}

func (copyRequestTranslator) Forward(
	_ context.Context,
	req task.CopyRequest,
) (*CopyRequest, error) {
	return &CopyRequest{
		Key:      uint64(req.Key),
		Name:     req.Name,
		Snapshot: req.Snapshot,
	}, nil
}

func (copyRequestTranslator) Backward(
	_ context.Context,
	req *CopyRequest,
) (task.CopyRequest, error) {
	return task.CopyRequest{
		Key:      task.Key(req.Key),
		Name:     req.Name,
		Snapshot: req.Snapshot,
	}, nil
}

func (copyResponseTranslator) Forward(
	_ context.Context,
	res task.CopyResponse,
) (*CopyResponse, error) {
	t, err := pb.TaskToPB(res.Task)
	if err != nil {
		return nil, err
	}
	return &CopyResponse{Task: t}, nil
}

func (copyResponseTranslator) Backward(
	_ context.Context,
	res *CopyResponse,
) (task.CopyResponse, error) {
	t, err := pb.TaskFromPB(res.Task)
	if err != nil {
		return task.CopyResponse{}, err
	}
	return task.CopyResponse{Task: t}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &TaskCreateService_ServiceDesc,
	}
	t.TaskCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &TaskRetrieveService_ServiceDesc,
	}
	t.TaskRetrieve = retrieve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &TaskDeleteService_ServiceDesc,
	}
	t.TaskDelete = del
	cpy := &copyServer{
		RequestTranslator:  copyRequestTranslator{},
		ResponseTranslator: copyResponseTranslator{},
		ServiceDesc:        &TaskCopyService_ServiceDesc,
	}
	t.TaskCopy = cpy

	return grpc.CompoundBindableTransport{create, retrieve, del, cpy}
}
