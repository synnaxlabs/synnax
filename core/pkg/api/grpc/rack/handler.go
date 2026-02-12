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
	"go/types"

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	apirack "github.com/synnaxlabs/synnax/pkg/api/rack"
	svcrack "github.com/synnaxlabs/synnax/pkg/service/rack"
	"github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = fgrpc.UnaryServer[
		apirack.CreateRequest,
		*gapi.RackCreateRequest,
		apirack.CreateResponse,
		*gapi.RackCreateResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apirack.RetrieveRequest,
		*gapi.RackRetrieveRequest,
		apirack.RetrieveResponse,
		*gapi.RackRetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apirack.DeleteRequest,
		*gapi.RackDeleteRequest,
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
)

var (
	_ fgrpc.Translator[apirack.CreateRequest, *gapi.RackCreateRequest]       = createRequestTranslator{}
	_ fgrpc.Translator[apirack.CreateResponse, *gapi.RackCreateResponse]     = createResponseTranslator{}
	_ fgrpc.Translator[apirack.RetrieveRequest, *gapi.RackRetrieveRequest]   = retrieveRequestTranslator{}
	_ fgrpc.Translator[apirack.RetrieveResponse, *gapi.RackRetrieveResponse] = retrieveResponseTranslator{}
	_ fgrpc.Translator[apirack.DeleteRequest, *gapi.RackDeleteRequest]       = deleteRequestTranslator{}
)

func translateForward(r *apirack.Rack) (*gapi.Rack, error) {
	gr := &gapi.Rack{Key: uint32(r.Key), Name: r.Name}
	if r.Status != nil {
		var err error
		gr.Status, err = status.TranslateToPB[svcrack.StatusDetails](status.Status[svcrack.StatusDetails](*r.Status))
		if err != nil {
			return nil, err
		}
	}
	return gr, nil
}

func translateBackward(r *gapi.Rack) (*apirack.Rack, error) {
	ar := &apirack.Rack{Key: svcrack.Key(r.Key), Name: r.Name}
	if r.Status != nil {
		s, err := status.TranslateFromPB[svcrack.StatusDetails](r.Status)
		if err != nil {
			return nil, err
		}
		rs := svcrack.Status(s)
		ar.Status = &rs
	}
	return ar, nil
}

func translateManyForward(rs []apirack.Rack) ([]*gapi.Rack, error) {
	res := make([]*gapi.Rack, len(rs))
	for i, r := range rs {
		var err error
		res[i], err = translateForward(&r)
		if err != nil {
			return nil, err
		}
	}
	return res, nil
}

func translateManyBackward(rs []*gapi.Rack) ([]apirack.Rack, error) {
	res := make([]apirack.Rack, len(rs))
	for i, r := range rs {
		rr, err := translateBackward(r)
		if err != nil {
			return nil, err
		}
		res[i] = *rr
	}
	return res, nil
}

func (createRequestTranslator) Forward(_ context.Context, req apirack.CreateRequest) (*gapi.RackCreateRequest, error) {
	racks, err := translateManyForward(req.Racks)
	if err != nil {
		return nil, err
	}
	return &gapi.RackCreateRequest{Racks: racks}, nil
}

func (createRequestTranslator) Backward(_ context.Context, req *gapi.RackCreateRequest) (apirack.CreateRequest, error) {
	racks, err := translateManyBackward(req.Racks)
	if err != nil {
		return apirack.CreateRequest{}, err
	}
	return apirack.CreateRequest{Racks: racks}, nil
}

func (createResponseTranslator) Forward(_ context.Context, res apirack.CreateResponse) (*gapi.RackCreateResponse, error) {
	racks, err := translateManyForward(res.Racks)
	if err != nil {
		return nil, err
	}
	return &gapi.RackCreateResponse{Racks: racks}, nil
}

func (createResponseTranslator) Backward(_ context.Context, res *gapi.RackCreateResponse) (apirack.CreateResponse, error) {
	racks, err := translateManyBackward(res.Racks)
	if err != nil {
		return apirack.CreateResponse{}, err
	}
	return apirack.CreateResponse{Racks: racks}, nil
}

func (retrieveRequestTranslator) Forward(_ context.Context, req apirack.RetrieveRequest) (*gapi.RackRetrieveRequest, error) {
	return &gapi.RackRetrieveRequest{
		Keys:  unsafe.ReinterpretSlice[svcrack.Key, uint32](req.Keys),
		Names: req.Names,
	}, nil
}

func (retrieveRequestTranslator) Backward(_ context.Context, req *gapi.RackRetrieveRequest) (apirack.RetrieveRequest, error) {
	return apirack.RetrieveRequest{
		Keys:  unsafe.ReinterpretSlice[uint32, svcrack.Key](req.Keys),
		Names: req.Names,
	}, nil
}

func (retrieveResponseTranslator) Forward(_ context.Context, res apirack.RetrieveResponse) (*gapi.RackRetrieveResponse, error) {
	racks, err := translateManyForward(res.Racks)
	if err != nil {
		return nil, err
	}
	return &gapi.RackRetrieveResponse{Racks: racks}, nil
}

func (retrieveResponseTranslator) Backward(_ context.Context, res *gapi.RackRetrieveResponse) (apirack.RetrieveResponse, error) {
	racks, err := translateManyBackward(res.Racks)
	if err != nil {
		return apirack.RetrieveResponse{}, err
	}
	return apirack.RetrieveResponse{Racks: racks}, nil
}

func (deleteRequestTranslator) Forward(_ context.Context, req apirack.DeleteRequest) (*gapi.RackDeleteRequest, error) {
	return &gapi.RackDeleteRequest{Keys: unsafe.ReinterpretSlice[svcrack.Key, uint32](req.Keys)}, nil
}

func (deleteRequestTranslator) Backward(_ context.Context, req *gapi.RackDeleteRequest) (apirack.DeleteRequest, error) {
	return apirack.DeleteRequest{Keys: unsafe.ReinterpretSlice[uint32, svcrack.Key](req.Keys)}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	createRack := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &gapi.RackCreateService_ServiceDesc,
	}
	a.RackCreate = createRack
	retrieveRack := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &gapi.RackRetrieveService_ServiceDesc,
	}
	a.RackRetrieve = retrieveRack
	deleteRack := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
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
