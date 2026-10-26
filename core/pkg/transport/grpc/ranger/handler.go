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
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/pb"
	rangerpb "github.com/synnaxlabs/synnax/pkg/service/ranger/pb"
	xpb "github.com/synnaxlabs/x/pb"
	"github.com/synnaxlabs/x/telem"
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
	renameServer = grpc.UnaryServer[
		ranger.RenameRequest,
		*RenameRequest,
		struct{},
		*emptypb.Empty,
	]
	setEndServer = grpc.UnaryServer[
		ranger.SetEndRequest,
		*SetEndRequest,
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
	renameRequestTranslator    struct{}
	setEndRequestTranslator    struct{}
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
	_ grpc.Translator[ranger.RenameRequest, *RenameRequest] = (*renameRequestTranslator)(
		nil,
	)
	_ grpc.Translator[ranger.SetEndRequest, *SetEndRequest] = (*setEndRequestTranslator)(
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

func (renameRequestTranslator) Forward(
	_ context.Context,
	r ranger.RenameRequest,
) (*RenameRequest, error) {
	return &RenameRequest{Key: r.Key.String(), Name: r.Name}, nil
}

func (renameRequestTranslator) Backward(
	_ context.Context,
	r *RenameRequest,
) (ranger.RenameRequest, error) {
	key, err := uuid.Parse(r.Key)
	if err != nil {
		return ranger.RenameRequest{}, err
	}
	return ranger.RenameRequest{Key: key, Name: r.Name}, nil
}

func (setEndRequestTranslator) Forward(
	_ context.Context,
	r ranger.SetEndRequest,
) (*SetEndRequest, error) {
	return &SetEndRequest{Key: r.Key.String(), End: int64(r.End)}, nil
}

func (setEndRequestTranslator) Backward(
	_ context.Context,
	r *SetEndRequest,
) (ranger.SetEndRequest, error) {
	key, err := uuid.Parse(r.Key)
	if err != nil {
		return ranger.SetEndRequest{}, err
	}
	return ranger.SetEndRequest{Key: key, End: telem.TimeStamp(r.End)}, nil
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
	rename := &renameServer{
		RequestTranslator:  renameRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &RangeRenameService_ServiceDesc,
	}
	t.RangeRename = rename
	setEnd := &setEndServer{
		RequestTranslator:  setEndRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &RangeSetEndService_ServiceDesc,
	}
	t.RangeSetEnd = setEnd
	return grpc.CompoundBindableTransport{
		create,
		retrieve,
		rangeDelete,
		rename,
		setEnd,
	}
}
