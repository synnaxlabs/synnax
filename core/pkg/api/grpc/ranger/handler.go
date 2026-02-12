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
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	apiranger "github.com/synnaxlabs/synnax/pkg/api/ranger"
	"github.com/synnaxlabs/x/telem"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
	createServer               = fgrpc.UnaryServer[
		apiranger.CreateRequest,
		*gapi.RangeCreateRequest,
		apiranger.CreateResponse,
		*gapi.RangeCreateResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apiranger.RetrieveRequest,
		*gapi.RangeRetrieveRequest,
		apiranger.RetrieveResponse,
		*gapi.RangeRetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apiranger.DeleteRequest,
		*gapi.RangeDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[apiranger.CreateRequest, *gapi.RangeCreateRequest]       = (*createRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.CreateResponse, *gapi.RangeCreateResponse]     = (*createResponseTranslator)(nil)
	_ fgrpc.Translator[apiranger.RetrieveRequest, *gapi.RangeRetrieveRequest]   = (*retrieveRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.RetrieveResponse, *gapi.RangeRetrieveResponse] = (*retrieveResponseTranslator)(nil)
	_ fgrpc.Translator[apiranger.DeleteRequest, *gapi.RangeDeleteRequest]       = (*deleteRequestTranslator)(nil)
)

func (t createRequestTranslator) Forward(
	_ context.Context,
	r apiranger.CreateRequest,
) (*gapi.RangeCreateRequest, error) {
	return &gapi.RangeCreateRequest{Ranges: translateManyForward(r.Ranges)}, nil
}

func (t createRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeCreateRequest,
) (apiranger.CreateRequest, error) {
	ranges, err := translateManyBackward(r.Ranges)
	return apiranger.CreateRequest{Ranges: ranges}, err
}

func (t createResponseTranslator) Forward(
	_ context.Context,
	r apiranger.CreateResponse,
) (*gapi.RangeCreateResponse, error) {
	return &gapi.RangeCreateResponse{Ranges: translateManyForward(r.Ranges)}, nil
}

func (t createResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeCreateResponse,
) (apiranger.CreateResponse, error) {
	ranges, err := translateManyBackward(r.Ranges)
	return apiranger.CreateResponse{Ranges: ranges}, err
}

func (t retrieveRequestTranslator) Forward(
	_ context.Context,
	r apiranger.RetrieveRequest,
) (*gapi.RangeRetrieveRequest, error) {
	keys := make([]string, len(r.Keys))
	for i := range r.Keys {
		keys[i] = r.Keys[i].String()
	}
	return &gapi.RangeRetrieveRequest{Keys: keys, Names: r.Names}, nil
}

func (t retrieveRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeRetrieveRequest,
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
) (*gapi.RangeRetrieveResponse, error) {
	return &gapi.RangeRetrieveResponse{Ranges: translateManyForward(r.Ranges)}, nil
}

func (t retrieveResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeRetrieveResponse,
) (apiranger.RetrieveResponse, error) {
	ranges, err := translateManyBackward(r.Ranges)
	return apiranger.RetrieveResponse{Ranges: ranges}, err
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	r apiranger.DeleteRequest,
) (*gapi.RangeDeleteRequest, error) {
	keys := make([]string, len(r.Keys))
	for i, k := range r.Keys {
		keys[i] = k.String()
	}
	return &gapi.RangeDeleteRequest{Keys: keys}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeDeleteRequest,
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

func translateForward(r apiranger.Range) *gapi.Range {
	return &gapi.Range{
		Key:       r.Key.String(),
		Name:      r.Name,
		TimeRange: telem.TranslateTimeRangeForward(r.TimeRange),
	}
}

func translateManyForward(r []apiranger.Range) []*gapi.Range {
	ranges := make([]*gapi.Range, len(r))
	for i := range r {
		ranges[i] = translateForward(r[i])
	}
	return ranges
}

func translateBackward(r *gapi.Range) (or apiranger.Range, err error) {
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

func translateManyBackward(r []*gapi.Range) ([]apiranger.Range, error) {
	ranges := make([]apiranger.Range, len(r))
	var err error
	for i := range r {
		ranges[i], err = translateBackward(r[i])
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
		ServiceDesc:        &gapi.RangeCreateService_ServiceDesc,
	}
	a.RangeCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &gapi.RangeRetrieveService_ServiceDesc,
	}
	a.RangeRetrieve = retrieve
	rangeDelete := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeDeleteService_ServiceDesc,
	}
	a.RangeDelete = rangeDelete
	return fgrpc.CompoundBindableTransport{
		create,
		retrieve,
		rangeDelete,
	}
}
