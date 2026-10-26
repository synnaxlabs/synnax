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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/task"
	"github.com/synnaxlabs/synnax/pkg/api/task/pb"
	taskpb "github.com/synnaxlabs/synnax/pkg/service/task/pb"
	xpb "github.com/synnaxlabs/x/pb"
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
		*pb.RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		task.DeleteRequest,
		*DeleteRequest,
		struct{},
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
	_ grpc.Translator[task.CreateRequest, *CreateRequest]          = createRequestTranslator{}
	_ grpc.Translator[task.CreateResponse, *CreateResponse]        = createResponseTranslator{}
	_ grpc.Translator[task.RetrieveRequest, *RetrieveRequest]      = retrieveRequestTranslator{}
	_ grpc.Translator[task.RetrieveResponse, *pb.RetrieveResponse] = retrieveResponseTranslator{}
	_ grpc.Translator[task.DeleteRequest, *DeleteRequest]          = deleteRequestTranslator{}
	_ grpc.Translator[task.CopyRequest, *CopyRequest]              = copyRequestTranslator{}
	_ grpc.Translator[task.CopyResponse, *CopyResponse]            = copyResponseTranslator{}
)

func (createRequestTranslator) Forward(
	_ context.Context,
	req task.CreateRequest,
) (*CreateRequest, error) {
	tasks, err := taskpb.TasksToPB(req.Tasks)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Tasks: tasks}, nil
}

func (createRequestTranslator) Backward(
	_ context.Context,
	req *CreateRequest,
) (task.CreateRequest, error) {
	tasks, err := taskpb.TasksFromPB(req.Tasks)
	if err != nil {
		return task.CreateRequest{}, err
	}
	return task.CreateRequest{Tasks: tasks}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	res task.CreateResponse,
) (*CreateResponse, error) {
	tasks, err := taskpb.TasksToPB(res.Tasks)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Tasks: tasks}, nil
}

func (createResponseTranslator) Backward(
	_ context.Context,
	res *CreateResponse,
) (task.CreateResponse, error) {
	tasks, err := taskpb.TasksFromPB(res.Tasks)
	if err != nil {
		return task.CreateResponse{}, err
	}
	return task.CreateResponse{Tasks: tasks}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	r task.RetrieveRequest,
) (*RetrieveRequest, error) {
	req, err := pb.RetrieveRequestToPB(r)
	if err != nil {
		return nil, err
	}
	return &RetrieveRequest{Request: req}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	r *RetrieveRequest,
) (task.RetrieveRequest, error) {
	// A client older than the Core sends the reserved fields, which would otherwise
	// select every task.
	if err := xpb.RejectUnknown(r); err != nil {
		return task.RetrieveRequest{}, err
	}
	return pb.RetrieveRequestFromPB(r.Request)
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	res task.RetrieveResponse,
) (*pb.RetrieveResponse, error) {
	return pb.RetrieveResponseToPB(res)
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	res *pb.RetrieveResponse,
) (task.RetrieveResponse, error) {
	return pb.RetrieveResponseFromPB(res)
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	req task.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{
		Keys: lo.Map(req.Keys, func(k task.Key, _ int) string { return k.String() }),
	}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	req *DeleteRequest,
) (task.DeleteRequest, error) {
	keys, err := lo.MapErr(req.Keys, func(k string, _ int) (task.Key, error) {
		return uuid.Parse(k)
	})
	if err != nil {
		return task.DeleteRequest{}, err
	}
	return task.DeleteRequest{Keys: keys}, nil
}

func (copyRequestTranslator) Forward(
	_ context.Context,
	req task.CopyRequest,
) (*CopyRequest, error) {
	return &CopyRequest{
		Key:      req.Key.String(),
		Name:     req.Name,
		Snapshot: req.Snapshot,
	}, nil
}

func (copyRequestTranslator) Backward(
	_ context.Context,
	req *CopyRequest,
) (task.CopyRequest, error) {
	key, err := uuid.Parse(req.Key)
	if err != nil {
		return task.CopyRequest{}, err
	}
	return task.CopyRequest{
		Key:      key,
		Name:     req.Name,
		Snapshot: req.Snapshot,
	}, nil
}

func (copyResponseTranslator) Forward(
	_ context.Context,
	res task.CopyResponse,
) (*CopyResponse, error) {
	t, err := taskpb.TaskToPB(res.Task)
	if err != nil {
		return nil, err
	}
	return &CopyResponse{Task: t}, nil
}

func (copyResponseTranslator) Backward(
	_ context.Context,
	res *CopyResponse,
) (task.CopyResponse, error) {
	t, err := taskpb.TaskFromPB(res.Task)
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
