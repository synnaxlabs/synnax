// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package kv

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/ranger/kv"
	"github.com/synnaxlabs/synnax/pkg/service/ranger/kv/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	getServer = grpc.UnaryServer[
		kv.GetRequest,
		*GetRequest,
		kv.GetResponse,
		*GetResponse,
	]
	setServer = grpc.UnaryServer[
		kv.SetRequest,
		*SetRequest,
		types.Nil,
		*emptypb.Empty,
	]
	deleteServer = grpc.UnaryServer[
		kv.DeleteRequest,
		*DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

type (
	getRequestTranslator    struct{}
	getResponseTranslator   struct{}
	setRequestTranslator    struct{}
	deleteRequestTranslator struct{}
)

var (
	_ grpc.Translator[kv.GetRequest, *GetRequest]       = (*getRequestTranslator)(nil)
	_ grpc.Translator[kv.GetResponse, *GetResponse]     = (*getResponseTranslator)(nil)
	_ grpc.Translator[kv.SetRequest, *SetRequest]       = (*setRequestTranslator)(nil)
	_ grpc.Translator[kv.DeleteRequest, *DeleteRequest] = (*deleteRequestTranslator)(
		nil,
	)
)

func (getRequestTranslator) Forward(
	_ context.Context,
	r kv.GetRequest,
) (*GetRequest, error) {
	return &GetRequest{
		Range: r.Range.String(), Keys: r.Keys,
	}, nil
}

func (getRequestTranslator) Backward(
	_ context.Context,
	r *GetRequest,
) (kv.GetRequest, error) {
	key, err := uuid.Parse(r.Range)
	if err != nil {
		return kv.GetRequest{}, err
	}
	return kv.GetRequest{Range: key, Keys: r.Keys}, nil
}

func (getResponseTranslator) Forward(
	_ context.Context,
	r kv.GetResponse,
) (*GetResponse, error) {
	pairs, err := pb.PairsToPB(r.Pairs)
	if err != nil {
		return nil, err
	}
	return &GetResponse{Pairs: pairs}, nil
}

func (getResponseTranslator) Backward(
	_ context.Context,
	r *GetResponse,
) (kv.GetResponse, error) {
	pairs, err := pb.PairsFromPB(r.Pairs)
	if err != nil {
		return kv.GetResponse{}, err
	}
	return kv.GetResponse{Pairs: pairs}, nil
}

func (setRequestTranslator) Forward(
	_ context.Context,
	r kv.SetRequest,
) (*SetRequest, error) {
	pairs, err := pb.PairsToPB(r.Pairs)
	if err != nil {
		return nil, err
	}
	return &SetRequest{Pairs: pairs}, nil
}

func (setRequestTranslator) Backward(
	_ context.Context,
	r *SetRequest,
) (kv.SetRequest, error) {
	pairs, err := pb.PairsFromPB(r.Pairs)
	if err != nil {
		return kv.SetRequest{}, err
	}
	return kv.SetRequest{Pairs: pairs}, nil
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	r kv.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{Range: r.Range.String(), Keys: r.Keys}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	r *DeleteRequest,
) (kv.DeleteRequest, error) {
	key, err := uuid.Parse(r.Range)
	if err != nil {
		return kv.DeleteRequest{}, err
	}
	return kv.DeleteRequest{Range: key, Keys: r.Keys}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
	get := &getServer{
		RequestTranslator:  getRequestTranslator{},
		ResponseTranslator: getResponseTranslator{},
		ServiceDesc:        &KVGetService_ServiceDesc,
	}
	t.KVGet = get
	set := &setServer{
		RequestTranslator:  setRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &KVSetService_ServiceDesc,
	}
	t.KVSet = set
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &KVDeleteService_ServiceDesc,
	}
	t.KVDelete = del
	return grpc.CompoundBindableTransport{get, set, del}
}
