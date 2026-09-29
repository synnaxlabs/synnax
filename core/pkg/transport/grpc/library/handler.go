// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package library

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/library"
	"github.com/synnaxlabs/synnax/pkg/service/library/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		library.CreateRequest,
		*CreateRequest,
		library.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		library.RetrieveRequest,
		*RetrieveRequest,
		library.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		library.DeleteRequest,
		*DeleteRequest,
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
)

var (
	_ grpc.Translator[
		library.CreateRequest,
		*CreateRequest,
	] = createRequestTranslator{}
	_ grpc.Translator[
		library.CreateResponse,
		*CreateResponse,
	] = createResponseTranslator{}
	_ grpc.Translator[
		library.RetrieveRequest,
		*RetrieveRequest,
	] = retrieveRequestTranslator{}
	_ grpc.Translator[
		library.RetrieveResponse,
		*RetrieveResponse,
	] = retrieveResponseTranslator{}
	_ grpc.Translator[
		library.DeleteRequest,
		*DeleteRequest,
	] = deleteRequestTranslator{}
)

func (createRequestTranslator) Forward(
	_ context.Context,
	req library.CreateRequest,
) (*CreateRequest, error) {
	libs, err := pb.LibrariesToPB(req.Libraries)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Libraries: libs}, nil
}

func (createRequestTranslator) Backward(
	_ context.Context,
	req *CreateRequest,
) (library.CreateRequest, error) {
	libs, err := pb.LibrariesFromPB(req.Libraries)
	if err != nil {
		return library.CreateRequest{}, err
	}
	return library.CreateRequest{Libraries: libs}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	res library.CreateResponse,
) (*CreateResponse, error) {
	libs, err := pb.LibrariesToPB(res.Libraries)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Libraries: libs}, nil
}

func (createResponseTranslator) Backward(
	_ context.Context,
	res *CreateResponse,
) (library.CreateResponse, error) {
	libs, err := pb.LibrariesFromPB(res.Libraries)
	if err != nil {
		return library.CreateResponse{}, err
	}
	return library.CreateResponse{Libraries: libs}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	req library.RetrieveRequest,
) (*RetrieveRequest, error) {
	keys := lo.Map(req.Keys, func(k library.Key, _ int) string { return k.String() })
	return &RetrieveRequest{
		Keys:       keys,
		SearchTerm: req.SearchTerm,
		Limit:      int32(req.Limit),
		Offset:     int32(req.Offset),
	}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	req *RetrieveRequest,
) (library.RetrieveRequest, error) {
	keys, err := lo.MapErr(req.Keys, func(keyStr string, _ int) (library.Key, error) {
		return uuid.Parse(keyStr)
	})
	if err != nil {
		return library.RetrieveRequest{}, err
	}
	return library.RetrieveRequest{
		Keys:       keys,
		SearchTerm: req.SearchTerm,
		Limit:      int(req.Limit),
		Offset:     int(req.Offset),
	}, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	res library.RetrieveResponse,
) (*RetrieveResponse, error) {
	libs, err := pb.LibrariesToPB(res.Libraries)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Libraries: libs}, nil
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	res *RetrieveResponse,
) (library.RetrieveResponse, error) {
	libs, err := pb.LibrariesFromPB(res.Libraries)
	if err != nil {
		return library.RetrieveResponse{}, err
	}
	return library.RetrieveResponse{Libraries: libs}, nil
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	req library.DeleteRequest,
) (*DeleteRequest, error) {
	keys := lo.Map(req.Keys, func(k library.Key, _ int) string { return k.String() })
	return &DeleteRequest{Keys: keys}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	req *DeleteRequest,
) (library.DeleteRequest, error) {
	keys, err := lo.MapErr(req.Keys, func(k string, _ int) (library.Key, error) {
		return uuid.Parse(k)
	})
	if err != nil {
		return library.DeleteRequest{}, err
	}
	return library.DeleteRequest{Keys: keys}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &LibraryCreateService_ServiceDesc,
	}
	t.LibraryCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &LibraryRetrieveService_ServiceDesc,
	}
	t.LibraryRetrieve = retrieve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &LibraryDeleteService_ServiceDesc,
	}
	t.LibraryDelete = del

	return grpc.CompoundBindableTransport{create, retrieve, del}
}
