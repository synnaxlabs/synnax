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

	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/icd"
	"github.com/synnaxlabs/synnax/pkg/api/icd/pb"
	icdpb "github.com/synnaxlabs/synnax/pkg/service/icd/pb"
	"github.com/synnaxlabs/x/errors"
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
		*pb.RetrieveRequest,
		icd.RetrieveResponse,
		*pb.RetrieveResponse,
	]
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
)

var (
	_ grpc.BackwardTranslator[icd.CreateRequest, *CreateRequest]         = createRequestTranslator{}
	_ grpc.ForwardTranslator[icd.CreateResponse, *CreateResponse]        = createResponseTranslator{}
	_ grpc.BackwardTranslator[icd.RetrieveRequest, *pb.RetrieveRequest]  = retrieveRequestTranslator{}
	_ grpc.ForwardTranslator[icd.RetrieveResponse, *pb.RetrieveResponse] = retrieveResponseTranslator{}
)

func (createRequestTranslator) Backward(
	_ context.Context,
	req *CreateRequest,
) (icd.CreateRequest, error) {
	icds, err := icdpb.ICDSFromPB(req.Icds)
	if err != nil {
		return icd.CreateRequest{}, err
	}
	return icd.CreateRequest{ICDs: icds}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	res icd.CreateResponse,
) (*CreateResponse, error) {
	icds, err := icdpb.ICDSToPB(res.ICDs)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Icds: icds}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	req *pb.RetrieveRequest,
) (icd.RetrieveRequest, error) {
	return pb.RetrieveRequestFromPB(req)
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	res icd.RetrieveResponse,
) (*pb.RetrieveResponse, error) {
	return pb.RetrieveResponseToPB(res)
}

// New binds the ICD create and retrieve routes to t and returns their gRPC transport.
func New(t *api.Transport, errs *errors.Codec) grpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &ICDCreateService_ServiceDesc,
		Errors:             errs,
	}
	t.ICDCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &ICDRetrieveService_ServiceDesc,
		Errors:             errs,
	}
	t.ICDRetrieve = retrieve
	return grpc.CompoundBindableTransport{create, retrieve}
}
