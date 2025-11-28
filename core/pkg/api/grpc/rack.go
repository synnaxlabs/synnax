// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	rackCreateServer = fgrpc.UnaryServer[
		api.RackCreateRequest,
		*gapi.RackCreateRequest,
		api.RackCreateResponse,
		*gapi.RackCreateResponse,
	]
	rackRetrieveServer = fgrpc.UnaryServer[
		api.RackRetrieveRequest,
		*gapi.RackRetrieveRequest,
		api.RackRetrieveResponse,
		*gapi.RackRetrieveResponse,
	]
	rackDeleteServer = fgrpc.UnaryServer[
		api.RackDeleteRequest,
		*gapi.RackDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

type (
	rackCreateRequestTranslator    struct{}
	rackCreateResponseTranslator   struct{}
	rackRetrieveRequestTranslator  struct{}
	rackRetrieveResponseTranslator struct{}
	rackDeleteRequestTranslator    struct{}
)

var (
	_ fgrpc.Translator[api.RackCreateRequest, *gapi.RackCreateRequest]       = rackCreateRequestTranslator{}
	_ fgrpc.Translator[api.RackCreateResponse, *gapi.RackCreateResponse]     = rackCreateResponseTranslator{}
	_ fgrpc.Translator[api.RackRetrieveRequest, *gapi.RackRetrieveRequest]   = rackRetrieveRequestTranslator{}
	_ fgrpc.Translator[api.RackRetrieveResponse, *gapi.RackRetrieveResponse] = rackRetrieveResponseTranslator{}
	_ fgrpc.Translator[api.RackDeleteRequest, *gapi.RackDeleteRequest]       = rackDeleteRequestTranslator{}
)

func translateRackForward(r *api.Rack) *gapi.Rack {
	gr := &gapi.Rack{Key: uint32(r.Key), Name: r.Name}
	if r.Status != nil {
		gr.Status, _ = status.TranslateToPB[rack.StatusDetails](status.Status[rack.StatusDetails](*r.Status))
	}
	return gr
}

func translateRackBackward(r *gapi.Rack) *api.Rack {
	ar := &api.Rack{Key: rack.Key(r.Key), Name: r.Name}
	if r.Status != nil {
		s, _ := status.TranslateFromPB[rack.StatusDetails](r.Status)
		rs := rack.Status(s)
		ar.Status = &rs
	}
	return ar
}

func translateRacksForward(rs []api.Rack) []*gapi.Rack {
	res := make([]*gapi.Rack, len(rs))
	for i, r := range rs {
		res[i] = translateRackForward(&r)
	}
	return res
}

func translateRacksBackward(rs []*gapi.Rack) []api.Rack {
	res := make([]api.Rack, len(rs))
	for i, r := range rs {
		res[i] = *translateRackBackward(r)
	}
	return res
}

func (rackCreateRequestTranslator) Forward(_ context.Context, req api.RackCreateRequest) (*gapi.RackCreateRequest, error) {
	return &gapi.RackCreateRequest{Racks: translateRacksForward(req.Racks)}, nil
}

func (rackCreateRequestTranslator) Backward(_ context.Context, req *gapi.RackCreateRequest) (api.RackCreateRequest, error) {
	return api.RackCreateRequest{Racks: translateRacksBackward(req.Racks)}, nil
}

func (rackCreateResponseTranslator) Forward(_ context.Context, res api.RackCreateResponse) (*gapi.RackCreateResponse, error) {
	return &gapi.RackCreateResponse{Racks: translateRacksForward(res.Racks)}, nil
}

func (rackCreateResponseTranslator) Backward(_ context.Context, res *gapi.RackCreateResponse) (api.RackCreateResponse, error) {
	return api.RackCreateResponse{Racks: translateRacksBackward(res.Racks)}, nil
}

func (rackRetrieveRequestTranslator) Forward(_ context.Context, req api.RackRetrieveRequest) (*gapi.RackRetrieveRequest, error) {
	return &gapi.RackRetrieveRequest{
		Keys:  unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys),
		Names: req.Names,
	}, nil
}

func (rackRetrieveRequestTranslator) Backward(_ context.Context, req *gapi.RackRetrieveRequest) (api.RackRetrieveRequest, error) {
	return api.RackRetrieveRequest{
		Keys:  unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys),
		Names: req.Names,
	}, nil
}

func (rackRetrieveResponseTranslator) Forward(_ context.Context, res api.RackRetrieveResponse) (*gapi.RackRetrieveResponse, error) {
	return &gapi.RackRetrieveResponse{Racks: translateRacksForward(res.Racks)}, nil
}

func (rackRetrieveResponseTranslator) Backward(_ context.Context, res *gapi.RackRetrieveResponse) (api.RackRetrieveResponse, error) {
	return api.RackRetrieveResponse{Racks: translateRacksBackward(res.Racks)}, nil
}

func (rackDeleteRequestTranslator) Forward(_ context.Context, req api.RackDeleteRequest) (*gapi.RackDeleteRequest, error) {
	return &gapi.RackDeleteRequest{Keys: unsafe.ReinterpretSlice[rack.Key, uint32](req.Keys)}, nil
}

func (rackDeleteRequestTranslator) Backward(_ context.Context, req *gapi.RackDeleteRequest) (api.RackDeleteRequest, error) {
	return api.RackDeleteRequest{Keys: unsafe.ReinterpretSlice[uint32, rack.Key](req.Keys)}, nil
}

func newRack(a *api.Transport) fgrpc.BindableTransport {
	createRack := &rackCreateServer{
		RequestTranslator:  rackCreateRequestTranslator{},
		ResponseTranslator: rackCreateResponseTranslator{},
		ServiceDesc:        &gapi.RackCreateService_ServiceDesc,
	}
	a.RackCreate = createRack
	retrieveRack := &rackRetrieveServer{
		RequestTranslator:  rackRetrieveRequestTranslator{},
		ResponseTranslator: rackRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.RackRetrieveService_ServiceDesc,
	}
	a.RackRetrieve = retrieveRack
	deleteRack := &rackDeleteServer{
		RequestTranslator:  rackDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RackDeleteService_ServiceDesc,
	}
	a.RackDelete = deleteRack

	return fgrpc.CompoundBindableTransport{
		createRack,
		retrieveRack,
		deleteRack,
	}
}
