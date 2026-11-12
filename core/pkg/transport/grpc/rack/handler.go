// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package rack

import (
	"context"
	"strconv"

	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/rack"
	"github.com/synnaxlabs/synnax/pkg/api/rack/pb"
	svcrack "github.com/synnaxlabs/synnax/pkg/service/rack"
	rackpb "github.com/synnaxlabs/synnax/pkg/service/rack/pb"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	xpb "github.com/synnaxlabs/x/pb"
	"github.com/synnaxlabs/x/unsafe"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		rack.CreateRequest,
		*CreateRequest,
		rack.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		rack.RetrieveRequest,
		*RetrieveRequest,
		rack.RetrieveResponse,
		*pb.RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		rack.DeleteRequest,
		*DeleteRequest,
		struct{},
		*emptypb.Empty,
	]
	dispatchServer = grpc.UnaryServer[
		rack.DispatchRequest,
		*DispatchRequest,
		struct{},
		*emptypb.Empty,
	]
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
	dispatchRequestTranslator  struct{}
)

var (
	_ grpc.Translator[rack.CreateRequest, *CreateRequest]          = createRequestTranslator{}
	_ grpc.Translator[rack.CreateResponse, *CreateResponse]        = createResponseTranslator{}
	_ grpc.Translator[rack.RetrieveRequest, *RetrieveRequest]      = retrieveRequestTranslator{}
	_ grpc.Translator[rack.RetrieveResponse, *pb.RetrieveResponse] = retrieveResponseTranslator{}
	_ grpc.Translator[rack.DeleteRequest, *DeleteRequest]          = deleteRequestTranslator{}
	_ grpc.Translator[rack.DispatchRequest, *DispatchRequest]      = dispatchRequestTranslator{}
)

func (createRequestTranslator) Forward(
	_ context.Context,
	req rack.CreateRequest,
) (*CreateRequest, error) {
	racks, err := rackpb.RacksToPB(req.Racks)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Racks: racks}, nil
}

func (createRequestTranslator) Backward(
	_ context.Context,
	req *CreateRequest,
) (rack.CreateRequest, error) {
	racks, err := rackpb.RacksFromPB(req.Racks)
	if err != nil {
		return rack.CreateRequest{}, err
	}
	return rack.CreateRequest{Racks: racks}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	res rack.CreateResponse,
) (*CreateResponse, error) {
	racks, err := rackpb.RacksToPB(res.Racks)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Racks: racks}, nil
}

func (createResponseTranslator) Backward(
	_ context.Context,
	res *CreateResponse,
) (rack.CreateResponse, error) {
	racks, err := rackpb.RacksFromPB(res.Racks)
	if err != nil {
		return rack.CreateResponse{}, err
	}
	return rack.CreateResponse{Racks: racks}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	r rack.RetrieveRequest,
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
) (rack.RetrieveRequest, error) {
	// A client older than the Core sends the reserved fields, which would otherwise
	// select every rack.
	if err := xpb.RejectUnknown(r); err != nil {
		return rack.RetrieveRequest{}, err
	}
	return pb.RetrieveRequestFromPB(r.Request)
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	res rack.RetrieveResponse,
) (*pb.RetrieveResponse, error) {
	return pb.RetrieveResponseToPB(res)
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	res *pb.RetrieveResponse,
) (rack.RetrieveResponse, error) {
	return pb.RetrieveResponseFromPB(res)
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	req rack.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{
		Keys: unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys),
	}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	req *DeleteRequest,
) (rack.DeleteRequest, error) {
	return rack.DeleteRequest{
		Keys: unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys),
	}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &RackCreateService_ServiceDesc,
	}
	t.RackCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &RackRetrieveService_ServiceDesc,
	}
	t.RackRetrieve = retrieve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &RackDeleteService_ServiceDesc,
	}
	t.RackDelete = del
	dispatch := &dispatchServer{
		RequestTranslator:  dispatchRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &RackDispatchService_ServiceDesc,
	}
	t.RackDispatch = dispatch

	return grpc.CompoundBindableTransport{create, retrieve, del, dispatch}
}

func (dispatchRequestTranslator) Forward(
	ctx context.Context,
	req rack.DispatchRequest,
) (*DispatchRequest, error) {
	acts := make([][]byte, len(req.Actions))
	for i, a := range req.Actions {
		var err error
		if acts[i], err = json.Codec.Encode(a); err != nil {
			return nil, err
		}
	}
	return &DispatchRequest{
		Key:         uint32(req.Key),
		DispatchKey: req.DispatchKey,
		Actions:     acts,
	}, nil
}

func (dispatchRequestTranslator) Backward(
	ctx context.Context,
	req *DispatchRequest,
) (rack.DispatchRequest, error) {
	acts := make([]svcrack.Action, len(req.Actions))
	for i, b := range req.Actions {
		if err := json.Codec.Decode(b, &acts[i]); err != nil {
			return rack.DispatchRequest{}, validate.PathedError(
				errors.Wrap(validate.ErrValidation, err.Error()),
				"actions",
				strconv.Itoa(i),
			)
		}
	}
	return rack.DispatchRequest{
		Key:         svcrack.Key(req.Key),
		DispatchKey: req.DispatchKey,
		Actions:     acts,
	}, nil
}
