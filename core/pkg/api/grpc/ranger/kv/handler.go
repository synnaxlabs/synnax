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
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	apikv "github.com/synnaxlabs/synnax/pkg/api/ranger/kv"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	getRequestTranslator    struct{}
	getResponseTranslator   struct{}
	setRequestTranslator    struct{}
	deleteRequestTranslator struct{}
	getServer               = fgrpc.UnaryServer[
		apikv.GetRequest,
		*gapi.RangeKVGetRequest,
		apikv.GetResponse,
		*gapi.RangeKVGetResponse,
	]
	setServer = fgrpc.UnaryServer[
		apikv.SetRequest,
		*gapi.RangeKVSetRequest,
		types.Nil,
		*emptypb.Empty,
	]
	deleteServer = fgrpc.UnaryServer[
		apikv.DeleteRequest,
		*gapi.RangeKVDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[apikv.GetRequest, *gapi.RangeKVGetRequest]       = (*getRequestTranslator)(nil)
	_ fgrpc.Translator[apikv.GetResponse, *gapi.RangeKVGetResponse]     = (*getResponseTranslator)(nil)
	_ fgrpc.Translator[apikv.SetRequest, *gapi.RangeKVSetRequest]       = (*setRequestTranslator)(nil)
	_ fgrpc.Translator[apikv.DeleteRequest, *gapi.RangeKVDeleteRequest] = (*deleteRequestTranslator)(nil)
)

func translatePairForward(p apikv.Pair) *gapi.KVPair {
	return &gapi.KVPair{Key: p.Key, Value: p.Value}
}

func translatePairsForward(p []apikv.Pair) []*gapi.KVPair {
	pairs := make([]*gapi.KVPair, len(p))
	for i := range p {
		pairs[i] = translatePairForward(p[i])
	}
	return pairs
}

func translatePairBackward(p *gapi.KVPair) apikv.Pair {
	return apikv.Pair{Key: p.Key, Value: p.Value}
}

func translatePairsBackward(p []*gapi.KVPair) []apikv.Pair {
	pairs := make([]apikv.Pair, len(p))
	for i := range p {
		pairs[i] = translatePairBackward(p[i])
	}
	return pairs
}

func (t getRequestTranslator) Forward(
	_ context.Context,
	r apikv.GetRequest,
) (*gapi.RangeKVGetRequest, error) {
	return &gapi.RangeKVGetRequest{
		RangeKey: r.Range.String(),
		Keys:     r.Keys,
	}, nil
}

func (t getRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVGetRequest,
) (apikv.GetRequest, error) {
	key, err := uuid.Parse(r.RangeKey)
	return apikv.GetRequest{
		Range: key,
		Keys:  r.Keys,
	}, err
}

func (t getResponseTranslator) Forward(
	_ context.Context,
	r apikv.GetResponse,
) (*gapi.RangeKVGetResponse, error) {
	return &gapi.RangeKVGetResponse{Pairs: translatePairsForward(r.Pairs)}, nil
}

func (t getResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVGetResponse,
) (apikv.GetResponse, error) {
	return apikv.GetResponse{Pairs: translatePairsBackward(r.Pairs)}, nil
}

func (t setRequestTranslator) Forward(
	_ context.Context,
	r apikv.SetRequest,
) (*gapi.RangeKVSetRequest, error) {
	return &gapi.RangeKVSetRequest{
		RangeKey: r.Range.String(),
		Pairs:    translatePairsForward(r.Pairs),
	}, nil
}

func (t setRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVSetRequest,
) (apikv.SetRequest, error) {
	key, err := uuid.Parse(r.RangeKey)
	return apikv.SetRequest{
		Range: key,
		Pairs: translatePairsBackward(r.Pairs),
	}, err
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	r apikv.DeleteRequest,
) (*gapi.RangeKVDeleteRequest, error) {
	return &gapi.RangeKVDeleteRequest{
		RangeKey: r.Range.String(),
		Keys:     r.Keys,
	}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVDeleteRequest,
) (apikv.DeleteRequest, error) {
	key, err := uuid.Parse(r.RangeKey)
	return apikv.DeleteRequest{
		Range: key,
		Keys:  r.Keys,
	}, err
}

func New(a *api.Transport) fgrpc.BindableTransport {
	kvGet := &getServer{
		RequestTranslator:  getRequestTranslator{},
		ResponseTranslator: getResponseTranslator{},
		ServiceDesc:        &gapi.RangeKVGetService_ServiceDesc,
	}
	a.KVGet = kvGet
	kvSet := &setServer{
		RequestTranslator:  setRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeKVSetService_ServiceDesc,
	}
	a.KVSet = kvSet
	kvDelete := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeKVDeleteService_ServiceDesc,
	}
	a.KVDelete = kvDelete
	return fgrpc.CompoundBindableTransport{
		kvGet,
		kvSet,
		kvDelete,
	}
}
