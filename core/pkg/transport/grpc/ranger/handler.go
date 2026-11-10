// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"
	"strconv"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/pb"
	svcranger "github.com/synnaxlabs/synnax/pkg/service/ranger"
	rangerpb "github.com/synnaxlabs/synnax/pkg/service/ranger/pb"
	"github.com/synnaxlabs/x/encoding/json"
	"github.com/synnaxlabs/x/errors"
	xpb "github.com/synnaxlabs/x/pb"
	"github.com/synnaxlabs/x/validate"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		ranger.CreateRequest,
		*CreateRequest,
		ranger.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		ranger.RetrieveRequest,
		*RetrieveRequest,
		ranger.RetrieveResponse,
		*pb.RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		ranger.DeleteRequest,
		*DeleteRequest,
		struct{},
		*emptypb.Empty,
	]
	dispatchServer = grpc.UnaryServer[
		ranger.DispatchRequest,
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
	_ grpc.Translator[ranger.CreateRequest, *CreateRequest] = (*createRequestTranslator)(
		nil,
	)
	_ grpc.Translator[ranger.CreateResponse, *CreateResponse] = (*createResponseTranslator)(
		nil,
	)
	_ grpc.Translator[ranger.RetrieveRequest, *RetrieveRequest] = (*retrieveRequestTranslator)(
		nil,
	)
	_ grpc.Translator[ranger.RetrieveResponse, *pb.RetrieveResponse] = (*retrieveResponseTranslator)(
		nil,
	)
	_ grpc.Translator[ranger.DeleteRequest, *DeleteRequest] = (*deleteRequestTranslator)(
		nil,
	)
	_ grpc.Translator[ranger.DispatchRequest, *DispatchRequest] = (*dispatchRequestTranslator)(
		nil,
	)
)

func (createRequestTranslator) Forward(
	_ context.Context,
	r ranger.CreateRequest,
) (*CreateRequest, error) {
	ranges, err := rangerpb.RangesToPB(r.Ranges)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Ranges: ranges}, nil
}

func (createRequestTranslator) Backward(
	_ context.Context,
	r *CreateRequest,
) (ranger.CreateRequest, error) {
	ranges, err := rangerpb.RangesFromPB(r.Ranges)
	if err != nil {
		return ranger.CreateRequest{}, err
	}
	return ranger.CreateRequest{Ranges: ranges}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	r ranger.CreateResponse,
) (*CreateResponse, error) {
	ranges, err := rangerpb.RangesToPB(r.Ranges)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Ranges: ranges}, nil
}

func (createResponseTranslator) Backward(
	_ context.Context,
	r *CreateResponse,
) (ranger.CreateResponse, error) {
	ranges, err := rangerpb.RangesFromPB(r.Ranges)
	if err != nil {
		return ranger.CreateResponse{}, err
	}
	return ranger.CreateResponse{Ranges: ranges}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	r ranger.RetrieveRequest,
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
) (ranger.RetrieveRequest, error) {
	// A client older than the Core sends the reserved keys and names fields, which
	// would otherwise select every range.
	if err := xpb.RejectUnknown(r); err != nil {
		return ranger.RetrieveRequest{}, err
	}
	return pb.RetrieveRequestFromPB(r.Request)
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	r ranger.RetrieveResponse,
) (*pb.RetrieveResponse, error) {
	return pb.RetrieveResponseToPB(r)
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	r *pb.RetrieveResponse,
) (ranger.RetrieveResponse, error) {
	return pb.RetrieveResponseFromPB(r)
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	r ranger.DeleteRequest,
) (*DeleteRequest, error) {
	keys := lo.Map(r.Keys, func(k ranger.Key, _ int) string { return k.String() })
	return &DeleteRequest{Keys: keys}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	r *DeleteRequest,
) (ranger.DeleteRequest, error) {
	keys, err := lo.MapErr(r.Keys, func(k string, _ int) (ranger.Key, error) {
		return uuid.Parse(k)
	})
	if err != nil {
		return ranger.DeleteRequest{}, err
	}
	return ranger.DeleteRequest{Keys: keys}, nil
}

func (dispatchRequestTranslator) Forward(
	ctx context.Context,
	r ranger.DispatchRequest,
) (*DispatchRequest, error) {
	acts := make([][]byte, len(r.Actions))
	for i, a := range r.Actions {
		var err error
		if acts[i], err = json.Codec.Encode(a); err != nil {
			return nil, err
		}
	}
	return &DispatchRequest{
		Key:         r.Key.String(),
		DispatchKey: r.DispatchKey,
		Actions:     acts,
	}, nil
}

func (dispatchRequestTranslator) Backward(
	ctx context.Context,
	r *DispatchRequest,
) (ranger.DispatchRequest, error) {
	key, err := uuid.Parse(r.Key)
	if err != nil {
		return ranger.DispatchRequest{}, err
	}
	acts := make([]svcranger.Action, len(r.Actions))
	for i, b := range r.Actions {
		if err := json.Codec.Decode(b, &acts[i]); err != nil {
			return ranger.DispatchRequest{}, validate.PathedError(
				errors.Wrap(validate.ErrValidation, err.Error()),
				"actions",
				strconv.Itoa(i),
			)
		}
	}
	return ranger.DispatchRequest{
		Key:         key,
		DispatchKey: r.DispatchKey,
		Actions:     acts,
	}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &RangeCreateService_ServiceDesc,
	}
	t.RangeCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &RangeRetrieveService_ServiceDesc,
	}
	t.RangeRetrieve = retrieve
	rangeDelete := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &RangeDeleteService_ServiceDesc,
	}
	t.RangeDelete = rangeDelete
	dispatch := &dispatchServer{
		RequestTranslator:  dispatchRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &RangeDispatchService_ServiceDesc,
	}
	t.RangeDispatch = dispatch
	return grpc.CompoundBindableTransport{
		create,
		retrieve,
		rangeDelete,
		dispatch,
	}
}
