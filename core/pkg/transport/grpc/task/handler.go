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
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/task"
	"github.com/synnaxlabs/synnax/pkg/api/task/pb"
	svctask "github.com/synnaxlabs/synnax/pkg/service/task"
	taskpb "github.com/synnaxlabs/synnax/pkg/service/task/pb"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	xpb "github.com/synnaxlabs/x/pb"
	"github.com/synnaxlabs/x/validate"
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
	dispatchServer = grpc.UnaryServer[
		task.DispatchRequest,
		*DispatchRequest,
		task.DispatchResponse,
		*emptypb.Empty,
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
	dispatchRequestTranslator  struct{}
	dispatchResponseTranslator struct{}
)

var (
	_ grpc.Translator[task.CreateRequest, *CreateRequest]          = createRequestTranslator{}
	_ grpc.Translator[task.CreateResponse, *CreateResponse]        = createResponseTranslator{}
	_ grpc.Translator[task.RetrieveRequest, *RetrieveRequest]      = retrieveRequestTranslator{}
	_ grpc.Translator[task.RetrieveResponse, *pb.RetrieveResponse] = retrieveResponseTranslator{}
	_ grpc.Translator[task.DeleteRequest, *DeleteRequest]          = deleteRequestTranslator{}
	_ grpc.Translator[task.CopyRequest, *CopyRequest]              = copyRequestTranslator{}
	_ grpc.Translator[task.CopyResponse, *CopyResponse]            = copyResponseTranslator{}
	_ grpc.Translator[task.DispatchRequest, *DispatchRequest]      = dispatchRequestTranslator{}
	_ grpc.Translator[task.DispatchResponse, *emptypb.Empty]       = dispatchResponseTranslator{}
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
	return &CopyRequest{Key: req.Key.String(), Name: req.Name}, nil
}

func (copyRequestTranslator) Backward(
	_ context.Context,
	req *CopyRequest,
) (task.CopyRequest, error) {
	key, err := uuid.Parse(req.Key)
	if err != nil {
		return task.CopyRequest{}, err
	}
	return task.CopyRequest{Key: key, Name: req.Name}, nil
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

func (dispatchRequestTranslator) Forward(
	ctx context.Context,
	req task.DispatchRequest,
) (*DispatchRequest, error) {
	acts := make([][]byte, len(req.Actions))
	for i, a := range req.Actions {
		var err error
		if acts[i], err = json.Codec.Encode(a); err != nil {
			return nil, err
		}
	}
	return &DispatchRequest{
		Key:         req.Key.String(),
		DispatchKey: req.DispatchKey,
		Actions:     acts,
	}, nil
}

func (dispatchRequestTranslator) Backward(
	ctx context.Context,
	req *DispatchRequest,
) (task.DispatchRequest, error) {
	key, err := uuid.Parse(req.Key)
	if err != nil {
		return task.DispatchRequest{}, err
	}
	acts := make([]svctask.Action, len(req.Actions))
	for i, b := range req.Actions {
		if err := json.Codec.Decode(b, &acts[i]); err != nil {
			return task.DispatchRequest{}, validate.PathedError(
				errors.Wrap(validate.ErrValidation, err.Error()),
				"actions",
				strconv.Itoa(i),
			)
		}
	}
	return task.DispatchRequest{
		Key:         key,
		DispatchKey: req.DispatchKey,
		Actions:     acts,
	}, nil
}

// Forward drops the actions that the Core computed, as a gRPC client holds no cached
// task to apply them to.
func (dispatchResponseTranslator) Forward(
	context.Context,
	task.DispatchResponse,
) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (dispatchResponseTranslator) Backward(
	context.Context,
	*emptypb.Empty,
) (task.DispatchResponse, error) {
	return task.DispatchResponse{}, nil
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
	dispatch := &dispatchServer{
		RequestTranslator:  dispatchRequestTranslator{},
		ResponseTranslator: dispatchResponseTranslator{},
		ServiceDesc:        &TaskDispatchService_ServiceDesc,
	}
	t.TaskDispatch = dispatch

	return grpc.CompoundBindableTransport{create, retrieve, del, cpy, dispatch}
}
