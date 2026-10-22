// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package icd

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/icd"
	"github.com/synnaxlabs/synnax/pkg/service/icd/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = grpc.UnaryServer[
		icd.CreateRequest,
		*CreateRequest,
		icd.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		icd.RetrieveRequest,
		*RetrieveRequest,
		icd.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		icd.DeleteRequest,
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
		icd.CreateRequest,
		*CreateRequest,
	] = createRequestTranslator{}
	_ grpc.Translator[
		icd.CreateResponse,
		*CreateResponse,
	] = createResponseTranslator{}
	_ grpc.Translator[
		icd.RetrieveRequest,
		*RetrieveRequest,
	] = retrieveRequestTranslator{}
	_ grpc.Translator[
		icd.RetrieveResponse,
		*RetrieveResponse,
	] = retrieveResponseTranslator{}
	_ grpc.Translator[
		icd.DeleteRequest,
		*DeleteRequest,
	] = deleteRequestTranslator{}
)

func (createRequestTranslator) Forward(
	_ context.Context,
	req icd.CreateRequest,
) (*CreateRequest, error) {
	ics, err := pb.ICDSToPB(req.ICDs)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Icds: ics}, nil
}

func (createRequestTranslator) Backward(
	_ context.Context,
	req *CreateRequest,
) (icd.CreateRequest, error) {
	ics, err := pb.ICDSFromPB(req.Icds)
	if err != nil {
		return icd.CreateRequest{}, err
	}
	return icd.CreateRequest{ICDs: ics}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	res icd.CreateResponse,
) (*CreateResponse, error) {
	ics, err := pb.ICDSToPB(res.ICDs)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Icds: ics}, nil
}

func (createResponseTranslator) Backward(
	_ context.Context,
	res *CreateResponse,
) (icd.CreateResponse, error) {
	ics, err := pb.ICDSFromPB(res.Icds)
	if err != nil {
		return icd.CreateResponse{}, err
	}
	return icd.CreateResponse{ICDs: ics}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	req icd.RetrieveRequest,
) (*RetrieveRequest, error) {
	keys := lo.Map(req.Keys, func(k icd.Key, _ int) string { return k.String() })
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
) (icd.RetrieveRequest, error) {
	keys, err := lo.MapErr(req.Keys, func(keyStr string, _ int) (icd.Key, error) {
		return uuid.Parse(keyStr)
	})
	if err != nil {
		return icd.RetrieveRequest{}, err
	}
	return icd.RetrieveRequest{
		Keys:       keys,
		SearchTerm: req.SearchTerm,
		Limit:      int(req.Limit),
		Offset:     int(req.Offset),
	}, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	res icd.RetrieveResponse,
) (*RetrieveResponse, error) {
	ics, err := pb.ICDSToPB(res.ICDs)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Icds: ics}, nil
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	res *RetrieveResponse,
) (icd.RetrieveResponse, error) {
	ics, err := pb.ICDSFromPB(res.Icds)
	if err != nil {
		return icd.RetrieveResponse{}, err
	}
	return icd.RetrieveResponse{ICDs: ics}, nil
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	req icd.DeleteRequest,
) (*DeleteRequest, error) {
	keys := lo.Map(req.Keys, func(k icd.Key, _ int) string { return k.String() })
	return &DeleteRequest{Keys: keys}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	req *DeleteRequest,
) (icd.DeleteRequest, error) {
	keys, err := lo.MapErr(req.Keys, func(k string, _ int) (icd.Key, error) {
		return uuid.Parse(k)
	})
	if err != nil {
		return icd.DeleteRequest{}, err
	}
	return icd.DeleteRequest{Keys: keys}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &ICDCreateService_ServiceDesc,
	}
	t.ICDCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &ICDRetrieveService_ServiceDesc,
	}
	t.ICDRetrieve = retrieve
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &ICDDeleteService_ServiceDesc,
	}
	t.ICDDelete = del

	return grpc.CompoundBindableTransport{create, retrieve, del}
}
