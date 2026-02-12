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

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	apitask "github.com/synnaxlabs/synnax/pkg/api/task"
	svcrack "github.com/synnaxlabs/synnax/pkg/service/rack"
	svctask "github.com/synnaxlabs/synnax/pkg/service/task"
	"github.com/synnaxlabs/x/status"
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

func translateForward(m *apitask.Task) (*gapi.Task, error) {
	gt := &gapi.Task{
		Key:      uint64(m.Key),
		Name:     m.Name,
		Type:     m.Type,
		Config:   m.Config,
		Internal: m.Internal,
		Snapshot: m.Snapshot,
	}
	if m.Status != nil {
		var err error
		gt.Status, err = status.TranslateToPB[svctask.StatusDetails](status.Status[svctask.StatusDetails](*m.Status))
		if err != nil {
			return nil, err
		}
	}
	return gt, nil
}

func translateBackward(m *gapi.Task) (*apitask.Task, error) {
	at := &apitask.Task{
		Key:      svctask.Key(m.Key),
		Name:     m.Name,
		Type:     m.Type,
		Config:   m.Config,
		Internal: m.Internal,
		Snapshot: m.Snapshot,
	}
	if m.Status != nil {
		s, err := status.TranslateFromPB[svctask.StatusDetails](m.Status)
		if err != nil {
			return nil, err
		}
		ts := svctask.Status(s)
		at.Status = &ts
	}
	return at, nil
}

func translateManyForward(ms []apitask.Task) ([]*gapi.Task, error) {
	res := make([]*gapi.Task, len(ms))
	for i, m := range ms {
		var err error
		res[i], err = translateForward(&m)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func translateManyBackward(ms []*gapi.Task) ([]apitask.Task, error) {
	res := make([]apitask.Task, len(ms))
	for i, m := range ms {
		tt, err := translateBackward(m)
		if err != nil {
			return nil, err
		}
		res[i] = *tt
	}
	return res, nil
}

func (createRequestTranslator) Forward(_ context.Context, req apitask.CreateRequest) (*gapi.TaskCreateRequest, error) {
	tasks, err := translateManyForward(req.Tasks)
	if err != nil {
		return nil, err
	}
	return &gapi.TaskCreateRequest{Tasks: tasks}, nil
}

func (createRequestTranslator) Backward(_ context.Context, req *gapi.TaskCreateRequest) (apitask.CreateRequest, error) {
	tasks, err := translateManyBackward(req.Tasks)
	if err != nil {
		return apitask.CreateRequest{}, err
	}
	return apitask.CreateRequest{Tasks: tasks}, nil
}

func (createResponseTranslator) Forward(_ context.Context, res apitask.CreateResponse) (*gapi.TaskCreateResponse, error) {
	tasks, err := translateManyForward(res.Tasks)
	if err != nil {
		return nil, err
	}
	return &gapi.TaskCreateResponse{Tasks: tasks}, nil
}

func (createResponseTranslator) Backward(_ context.Context, res *gapi.TaskCreateResponse) (apitask.CreateResponse, error) {
	tasks, err := translateManyBackward(res.Tasks)
	if err != nil {
		return apitask.CreateResponse{}, err
	}
	return apitask.CreateResponse{Tasks: tasks}, nil
}

func (retrieveRequestTranslator) Forward(_ context.Context, req apitask.RetrieveRequest) (*gapi.TaskRetrieveRequest, error) {
	return &gapi.TaskRetrieveRequest{
		Rack:          uint32(req.Rack),
		Keys:          unsafe.ReinterpretSlice[svctask.Key, uint64](req.Keys),
		Names:         req.Names,
		Types:         req.Types,
		IncludeStatus: req.IncludeStatus,
	}, nil
}

func (retrieveRequestTranslator) Backward(_ context.Context, req *gapi.TaskRetrieveRequest) (apitask.RetrieveRequest, error) {
	return apitask.RetrieveRequest{
		Rack:          svcrack.Key(req.Rack),
		Keys:          unsafe.ReinterpretSlice[uint64, svctask.Key](req.Keys),
		Names:         req.Names,
		Types:         req.Types,
		IncludeStatus: req.IncludeStatus,
	}, nil
}

func (retrieveResponseTranslator) Forward(_ context.Context, res apitask.RetrieveResponse) (*gapi.TaskRetrieveResponse, error) {
	tasks, err := translateManyForward(res.Tasks)
	if err != nil {
		return nil, err
	}
	return &gapi.TaskRetrieveResponse{Tasks: tasks}, nil
}

func (retrieveResponseTranslator) Backward(_ context.Context, res *gapi.TaskRetrieveResponse) (apitask.RetrieveResponse, error) {
	tasks, err := translateManyBackward(res.Tasks)
	if err != nil {
		return apitask.RetrieveResponse{}, err
	}
	return apitask.RetrieveResponse{Tasks: tasks}, nil
}

func (deleteRequestTranslator) Forward(_ context.Context, req apitask.DeleteRequest) (*gapi.TaskDeleteRequest, error) {
	return &gapi.TaskDeleteRequest{Keys: unsafe.ReinterpretSlice[svctask.Key, uint64](req.Keys)}, nil
}

func (deleteRequestTranslator) Backward(_ context.Context, req *gapi.TaskDeleteRequest) (apitask.DeleteRequest, error) {
	return apitask.DeleteRequest{Keys: unsafe.ReinterpretSlice[uint64, svctask.Key](req.Keys)}, nil
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
		Key:      svctask.Key(req.Key),
		Name:     req.Name,
		Snapshot: req.Snapshot,
	}, nil
}

func (copyResponseTranslator) Forward(_ context.Context, res apitask.CopyResponse) (*gapi.TaskCopyResponse, error) {
	t, err := translateForward(&res.Task)
	if err != nil {
		return nil, err
	}
	return &gapi.TaskCopyResponse{Task: t}, nil
}

func (copyResponseTranslator) Backward(_ context.Context, res *gapi.TaskCopyResponse) (apitask.CopyResponse, error) {
	t, err := translateBackward(res.Task)
	if err != nil {
		return apitask.CopyResponse{}, err
	}
	return apitask.CopyResponse{Task: *t}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	createTask := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &gapi.TaskCreateService_ServiceDesc,
	}
	a.TaskCreate = createTask
	retrieveTask := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &gapi.TaskRetrieveService_ServiceDesc,
	}
	a.TaskRetrieve = retrieveTask
	deleteTask := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.TaskDeleteService_ServiceDesc,
	}
	a.TaskDelete = deleteTask
	copyTask := &copyServer{
		RequestTranslator:  copyRequestTranslator{},
		ResponseTranslator: copyResponseTranslator{},
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
