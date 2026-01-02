// Copyright 2025 Synnax Labs, Inc.
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
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apiranger "github.com/synnaxlabs/synnax/pkg/api/ranger"
	pb "github.com/synnaxlabs/synnax/pkg/api/ranger/pb"
	"github.com/synnaxlabs/x/telem"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = fgrpc.UnaryServer[
		apiranger.CreateRequest,
		*CreateRequest,
		apiranger.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apiranger.RetrieveRequest,
		*RetrieveRequest,
		apiranger.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apiranger.DeleteRequest,
		*DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	renameServer = fgrpc.UnaryServer[
		apiranger.RenameRequest,
		*RenameRequest,
		types.Nil,
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
)

var (
	_ fgrpc.Translator[apiranger.CreateRequest, *CreateRequest]       = (*createRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.CreateResponse, *CreateResponse]     = (*createResponseTranslator)(nil)
	_ fgrpc.Translator[apiranger.RetrieveRequest, *RetrieveRequest]   = (*retrieveRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.RetrieveResponse, *RetrieveResponse] = (*retrieveResponseTranslator)(nil)
	_ fgrpc.Translator[apiranger.DeleteRequest, *DeleteRequest]       = (*deleteRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.RenameRequest, *RenameRequest]       = (*renameRequestTranslator)(nil)
)

func (t createRequestTranslator) Forward(
	_ context.Context,
	r apiranger.CreateRequest,
) (*CreateRequest, error) {
	return &CreateRequest{Ranges: translateRangesForward(r.Ranges)}, nil
}

func (t createRequestTranslator) Backward(
	_ context.Context,
	r *CreateRequest,
) (apiranger.CreateRequest, error) {
	ranges, err := translateRangesBackward(r.Ranges)
	return apiranger.CreateRequest{Ranges: ranges}, err
}

func (t createResponseTranslator) Forward(
	_ context.Context,
	r apiranger.CreateResponse,
) (*CreateResponse, error) {
	return &CreateResponse{Ranges: translateRangesForward(r.Ranges)}, nil
}

func (t createResponseTranslator) Backward(
	_ context.Context,
	r *CreateResponse,
) (apiranger.CreateResponse, error) {
	ranges, err := translateRangesBackward(r.Ranges)
	return apiranger.CreateResponse{Ranges: ranges}, err
}

func (t retrieveRequestTranslator) Forward(
	_ context.Context,
	r apiranger.RetrieveRequest,
) (*RetrieveRequest, error) {
	keys := make([]string, len(r.Keys))
	for i := range r.Keys {
		keys[i] = r.Keys[i].String()
	}
	return &RetrieveRequest{Keys: keys, Names: r.Names}, nil
}

func (t retrieveRequestTranslator) Backward(
	_ context.Context,
	r *RetrieveRequest,
) (apiranger.RetrieveRequest, error) {
	keys := make([]uuid.UUID, len(r.Keys))
	for i := range r.Keys {
		key, err := uuid.Parse(r.Keys[i])
		if err != nil {
			return apiranger.RetrieveRequest{}, err
		}
		keys[i] = key
	}
	return apiranger.RetrieveRequest{Keys: keys, Names: r.Names}, nil
}

func (t retrieveResponseTranslator) Forward(
	_ context.Context,
	r apiranger.RetrieveResponse,
) (*RetrieveResponse, error) {
	return &RetrieveResponse{Ranges: translateRangesForward(r.Ranges)}, nil
}

func (t retrieveResponseTranslator) Backward(
	_ context.Context,
	r *RetrieveResponse,
) (apiranger.RetrieveResponse, error) {
	ranges, err := translateRangesBackward(r.Ranges)
	return apiranger.RetrieveResponse{Ranges: ranges}, err
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	r apiranger.DeleteRequest,
) (*DeleteRequest, error) {
	keys := make([]string, len(r.Keys))
	for i, k := range r.Keys {
		keys[i] = k.String()
	}
	return &DeleteRequest{Keys: keys}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	r *DeleteRequest,
) (apiranger.DeleteRequest, error) {
	keys := make([]uuid.UUID, len(r.Keys))
	for i := range r.Keys {
		key, err := uuid.Parse(r.Keys[i])
		if err != nil {
			return apiranger.DeleteRequest{}, err
		}
		keys[i] = key
	}
	return apiranger.DeleteRequest{Keys: keys}, nil
}

func (t renameRequestTranslator) Forward(
	_ context.Context,
	r apiranger.RenameRequest,
) (*RenameRequest, error) {
	return &RenameRequest{
		Key:  r.Key.String(),
		Name: r.Name,
	}, nil
}

func (t renameRequestTranslator) Backward(
	_ context.Context,
	r *RenameRequest,
) (apiranger.RenameRequest, error) {
	key, err := uuid.Parse(r.Key)
	return apiranger.RenameRequest{
		Key:  key,
		Name: r.Name,
	}, err
}

func translateRangeForward(r apiranger.Range) *pb.Range {
	return &pb.Range{
		Key:       r.Key.String(),
		Name:      r.Name,
		TimeRange: telem.TranslateTimeRangeForward(r.TimeRange),
	}
}

func translateRangesForward(r []apiranger.Range) []*pb.Range {
	ranges := make([]*pb.Range, len(r))
	for i := range r {
		ranges[i] = translateRangeForward(r[i])
	}
	return ranges
}

func translateRangeBackward(r *pb.Range) (or apiranger.Range, err error) {
	if r.Key != "" {
		or.Key, err = uuid.Parse(r.Key)
		if err != nil {
			return apiranger.Range{}, err
		}
	}
	or.Name = r.Name
	or.TimeRange = telem.TranslateTimeRangeBackward(r.TimeRange)
	return
}

func translateRangesBackward(r []*pb.Range) ([]apiranger.Range, error) {
	ranges := make([]apiranger.Range, len(r))
	var err error
	for i := range r {
		ranges[i], err = translateRangeBackward(r[i])
		if err != nil {
			return nil, err
		}
	}
	return ranges, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &RangeCreateService_ServiceDesc,
	}
	a.RangeCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &RangeRetrieveService_ServiceDesc,
	}
	a.RangeRetrieve = retrieve
	rangeDelete := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &RangeDeleteService_ServiceDesc,
	}
	a.RangeDelete = rangeDelete
	rename := &renameServer{
		RequestTranslator:  renameRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &RangeRenameService_ServiceDesc,
	}
	a.RangeRename = rename
	return fgrpc.CompoundBindableTransport{
		create,
		retrieve,
		rangeDelete,
		rename,
	}
}
