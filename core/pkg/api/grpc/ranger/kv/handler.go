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
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apikv "github.com/synnaxlabs/synnax/pkg/api/ranger/kv"
	svckv "github.com/synnaxlabs/synnax/pkg/service/ranger/kv"
	kvpb "github.com/synnaxlabs/synnax/pkg/service/ranger/kv/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	getServer = fgrpc.UnaryServer[
		apikv.GetRequest,
		*GetRequest,
		apikv.GetResponse,
		*GetResponse,
	]
	setServer = fgrpc.UnaryServer[
		apikv.SetRequest,
		*SetRequest,
		types.Nil,
		*emptypb.Empty,
	]
	deleteServer = fgrpc.UnaryServer[
		apikv.DeleteRequest,
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
	_ fgrpc.Translator[apikv.GetRequest, *GetRequest]       = (*getRequestTranslator)(nil)
	_ fgrpc.Translator[apikv.GetResponse, *GetResponse]     = (*getResponseTranslator)(nil)
	_ fgrpc.Translator[apikv.SetRequest, *SetRequest]       = (*setRequestTranslator)(nil)
	_ fgrpc.Translator[apikv.DeleteRequest, *DeleteRequest] = (*deleteRequestTranslator)(nil)
)

func translatePairsForward(ctx context.Context, p []svckv.Pair) ([]*kvpb.Pair, error) {
	return kvpb.PairsToPB(ctx, p)
}

func translatePairsBackward(ctx context.Context, p []*kvpb.Pair) ([]svckv.Pair, error) {
	return kvpb.PairsFromPB(ctx, p)
}

func (t getRequestTranslator) Forward(
	_ context.Context,
	r apikv.GetRequest,
) (*GetRequest, error) {
	return &GetRequest{
		Range: r.Range.String(),
		Keys:  r.Keys,
	}, nil
}

func (t getRequestTranslator) Backward(
	_ context.Context,
	r *GetRequest,
) (apikv.GetRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apikv.GetRequest{
		Range: key,
		Keys:  r.Keys,
	}, err
}

func (t getResponseTranslator) Forward(
	ctx context.Context,
	r apikv.GetResponse,
) (*GetResponse, error) {
	pairs, err := translatePairsForward(ctx, r.Pairs)
	if err != nil {
		return nil, err
	}
	return &GetResponse{Pairs: pairs}, nil
}

func (t getResponseTranslator) Backward(
	ctx context.Context,
	r *GetResponse,
) (apikv.GetResponse, error) {
	pairs, err := translatePairsBackward(ctx, r.Pairs)
	if err != nil {
		return apikv.GetResponse{}, err
	}
	return apikv.GetResponse{Pairs: pairs}, nil
}

func (t setRequestTranslator) Forward(
	ctx context.Context,
	r apikv.SetRequest,
) (*SetRequest, error) {
	pairs, err := translatePairsForward(ctx, r.Pairs)
	if err != nil {
		return nil, err
	}
	return &SetRequest{Pairs: pairs}, nil
}

func (t setRequestTranslator) Backward(
	ctx context.Context,
	r *SetRequest,
) (apikv.SetRequest, error) {
	pairs, err := translatePairsBackward(ctx, r.Pairs)
	if err != nil {
		return apikv.SetRequest{}, err
	}
	return apikv.SetRequest{Pairs: pairs}, nil
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	r apikv.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{
		Range: r.Range.String(),
		Keys:  r.Keys,
	}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	r *DeleteRequest,
) (apikv.DeleteRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apikv.DeleteRequest{
		Range: key,
		Keys:  r.Keys,
	}, err
}

func New(a *api.Transport) fgrpc.BindableTransport {
	get := &getServer{
		RequestTranslator:  getRequestTranslator{},
		ResponseTranslator: getResponseTranslator{},
		ServiceDesc:        &KVGetService_ServiceDesc,
	}
	a.KVGet = get
	set := &setServer{
		RequestTranslator:  setRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &KVSetService_ServiceDesc,
	}
	a.KVSet = set
	del := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &KVDeleteService_ServiceDesc,
	}
	a.KVDelete = del
	return fgrpc.CompoundBindableTransport{get, set, del}
}
