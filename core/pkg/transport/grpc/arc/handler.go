// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc

import (
	"context"
	"uuid"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/arc"
	"github.com/synnaxlabs/synnax/pkg/api/arc/pb"
	arcpb "github.com/synnaxlabs/synnax/pkg/service/arc/pb"
	xpb "github.com/synnaxlabs/x/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createRequestTranslator    struct{}
	createResponseTranslator   struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
	createServer               = grpc.UnaryServer[
		arc.CreateRequest,
		*CreateRequest,
		arc.CreateResponse,
		*CreateResponse,
	]
	retrieveServer = grpc.UnaryServer[
		arc.RetrieveRequest,
		*RetrieveRequest,
		arc.RetrieveResponse,
		*pb.RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		arc.DeleteRequest,
		*DeleteRequest,
		struct{},
		*emptypb.Empty,
	]
)

var (
	_ grpc.Translator[arc.CreateRequest, *CreateRequest] = (*createRequestTranslator)(
		nil,
	)
	_ grpc.Translator[arc.CreateResponse, *CreateResponse] = (*createResponseTranslator)(
		nil,
	)
	_ grpc.Translator[arc.RetrieveRequest, *RetrieveRequest] = (*retrieveRequestTranslator)(
		nil,
	)
	_ grpc.Translator[arc.RetrieveResponse, *pb.RetrieveResponse] = (*retrieveResponseTranslator)(
		nil,
	)
	_ grpc.Translator[arc.DeleteRequest, *DeleteRequest] = (*deleteRequestTranslator)(
		nil,
	)
)

func (createRequestTranslator) Forward(
	_ context.Context,
	msg arc.CreateRequest,
) (*CreateRequest, error) {
	arcs, err := arcpb.ArcsToPB(msg.Arcs)
	if err != nil {
		return nil, err
	}
	return &CreateRequest{Arcs: arcs}, nil
}

func (createRequestTranslator) Backward(
	_ context.Context,
	msg *CreateRequest,
) (arc.CreateRequest, error) {
	arcs, err := arcpb.ArcsFromPB(msg.Arcs)
	if err != nil {
		return arc.CreateRequest{}, err
	}
	return arc.CreateRequest{Arcs: arcs}, nil
}

func (createResponseTranslator) Forward(
	_ context.Context,
	msg arc.CreateResponse,
) (*CreateResponse, error) {
	pbArcs, err := arcpb.ArcsToPB(msg.Arcs)
	if err != nil {
		return nil, err
	}
	return &CreateResponse{Arcs: pbArcs}, nil
}

func (createResponseTranslator) Backward(
	_ context.Context,
	msg *CreateResponse,
) (arc.CreateResponse, error) {
	arcs, err := arcpb.ArcsFromPB(msg.Arcs)
	if err != nil {
		return arc.CreateResponse{}, err
	}
	return arc.CreateResponse{Arcs: arcs}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	r arc.RetrieveRequest,
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
) (arc.RetrieveRequest, error) {
	// A client older than the Core sends the reserved fields, which would otherwise
	// select every Arc module.
	if err := xpb.RejectUnknown(r); err != nil {
		return arc.RetrieveRequest{}, err
	}
	return pb.RetrieveRequestFromPB(r.Request)
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	res arc.RetrieveResponse,
) (*pb.RetrieveResponse, error) {
	return pb.RetrieveResponseToPB(res)
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	res *pb.RetrieveResponse,
) (arc.RetrieveResponse, error) {
	return pb.RetrieveResponseFromPB(res)
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	msg arc.DeleteRequest,
) (*DeleteRequest, error) {
	keys := lo.Map(msg.Keys, func(k arc.Key, _ int) string { return k.String() })
	return &DeleteRequest{Keys: keys}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	msg *DeleteRequest,
) (arc.DeleteRequest, error) {
	keys, err := lo.MapErr(msg.Keys, func(keyStr string, _ int) (arc.Key, error) {
		return uuid.Parse(keyStr)
	})
	if err != nil {
		return arc.DeleteRequest{}, err
	}
	return arc.DeleteRequest{Keys: keys}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
	c := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &ArcCreateService_ServiceDesc,
	}
	r := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &ArcRetrieveService_ServiceDesc,
	}
	d := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &ArcDeleteService_ServiceDesc,
	}
	t.ArcCreate = c
	t.ArcRetrieve = r
	t.ArcDelete = d
	return grpc.CompoundBindableTransport{c, r, d}
}
