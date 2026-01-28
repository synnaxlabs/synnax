// Copyright 2026 Synnax Labs, Inc.
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

	"github.com/google/uuid"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	rangeCreateServer = fgrpc.UnaryServer[
		api.RangeCreateRequest,
		*gapi.RangeCreateRequest,
		api.RangeCreateResponse,
		*gapi.RangeCreateResponse,
	]
	rangeRetrieveServer = fgrpc.UnaryServer[
		api.RangeRetrieveRequest,
		*gapi.RangeRetrieveRequest,
		api.RangeRetrieveResponse,
		*gapi.RangeRetrieveResponse,
	]
	rangeDeleteServer = fgrpc.UnaryServer[
		api.RangeDeleteRequest,
		*gapi.RangeDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	rangeKVGetServer = fgrpc.UnaryServer[
		api.RangeKVGetRequest,
		*gapi.RangeKVGetRequest,
		api.RangeKVGetResponse,
		*gapi.RangeKVGetResponse,
	]
	rangeKVSetServer = fgrpc.UnaryServer[
		api.RangeKVSetRequest,
		*gapi.RangeKVSetRequest,
		types.Nil,
		*emptypb.Empty,
	]
	rangeKVDeleteServer = fgrpc.UnaryServer[
		api.RangeKVDeleteRequest,
		*gapi.RangeKVDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	rangeAliasSetServer = fgrpc.UnaryServer[
		api.RangeAliasSetRequest,
		*gapi.RangeAliasSetRequest,
		types.Nil,
		*emptypb.Empty,
	]
	rangeAliasDeleteServer = fgrpc.UnaryServer[
		api.RangeAliasDeleteRequest,
		*gapi.RangeAliasDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	rangeAliasResolveServer = fgrpc.UnaryServer[
		api.RangeAliasResolveRequest,
		*gapi.RangeAliasResolveRequest,
		api.RangeAliasResolveResponse,
		*gapi.RangeAliasResolveResponse,
	]
	rangeAliasListServer = fgrpc.UnaryServer[
		api.RangeAliasListRequest,
		*gapi.RangeAliasListRequest,
		api.RangeAliasListResponse,
		*gapi.RangeAliasListResponse,
	]
)

type (
	rangeCreateRequestTranslator        struct{}
	rangeCreateResponseTranslator       struct{}
	rangeRetrieveRequestTranslator      struct{}
	rangeRetrieveResponseTranslator     struct{}
	rangeDeleteRequestTranslator        struct{}
	rangeKVGetRequestTranslator         struct{}
	rangeKVGetResponseTranslator        struct{}
	rangeKVSetRequestTranslator         struct{}
	rangeKVDeleteRequestTranslator      struct{}
	rangeAliasSetRequestTranslator      struct{}
	rangeAliasDeleteRequestTranslator   struct{}
	rangeAliasResolveRequestTranslator  struct{}
	rangeAliasResolveResponseTranslator struct{}
	rangeAliasListRequestTranslator     struct{}
	rangeAliasListResponseTranslator    struct{}
)

var (
	_ fgrpc.Translator[api.RangeCreateRequest, *gapi.RangeCreateRequest]               = (*rangeCreateRequestTranslator)(nil)
	_ fgrpc.Translator[api.RangeCreateResponse, *gapi.RangeCreateResponse]             = (*rangeCreateResponseTranslator)(nil)
	_ fgrpc.Translator[api.RangeRetrieveRequest, *gapi.RangeRetrieveRequest]           = (*rangeRetrieveRequestTranslator)(nil)
	_ fgrpc.Translator[api.RangeRetrieveResponse, *gapi.RangeRetrieveResponse]         = (*rangeRetrieveResponseTranslator)(nil)
	_ fgrpc.Translator[api.RangeKVGetRequest, *gapi.RangeKVGetRequest]                 = (*rangeKVGetRequestTranslator)(nil)
	_ fgrpc.Translator[api.RangeKVGetResponse, *gapi.RangeKVGetResponse]               = (*rangeKVGetResponseTranslator)(nil)
	_ fgrpc.Translator[api.RangeKVSetRequest, *gapi.RangeKVSetRequest]                 = (*rangeKVSetRequestTranslator)(nil)
	_ fgrpc.Translator[api.RangeKVDeleteRequest, *gapi.RangeKVDeleteRequest]           = (*rangeKVDeleteRequestTranslator)(nil)
	_ fgrpc.Translator[api.RangeAliasSetRequest, *gapi.RangeAliasSetRequest]           = (*rangeAliasSetRequestTranslator)(nil)
	_ fgrpc.Translator[api.RangeAliasDeleteRequest, *gapi.RangeAliasDeleteRequest]     = (*rangeAliasDeleteRequestTranslator)(nil)
	_ fgrpc.Translator[api.RangeAliasResolveRequest, *gapi.RangeAliasResolveRequest]   = (*rangeAliasResolveRequestTranslator)(nil)
	_ fgrpc.Translator[api.RangeAliasResolveResponse, *gapi.RangeAliasResolveResponse] = (*rangeAliasResolveResponseTranslator)(nil)
	_ fgrpc.Translator[api.RangeAliasListRequest, *gapi.RangeAliasListRequest]         = (*rangeAliasListRequestTranslator)(nil)
	_ fgrpc.Translator[api.RangeAliasListResponse, *gapi.RangeAliasListResponse]       = (*rangeAliasListResponseTranslator)(nil)
	_ fgrpc.Translator[api.RangeDeleteRequest, *gapi.RangeDeleteRequest]               = (*rangeDeleteRequestTranslator)(nil)
)

func translatePairForward(p api.RangeKVPair) *gapi.KVPair {
	return &gapi.KVPair{Key: p.Key, Value: p.Value}
}

func translatePairsForward(p []api.RangeKVPair) []*gapi.KVPair {
	pairs := make([]*gapi.KVPair, len(p))
	for i := range p {
		pairs[i] = translatePairForward(p[i])
	}
	return pairs
}

func translatePairBackward(p *gapi.KVPair) api.RangeKVPair {
	return api.RangeKVPair{Key: p.Key, Value: p.Value}
}

func translatePairsBackward(p []*gapi.KVPair) []api.RangeKVPair {
	pairs := make([]api.RangeKVPair, len(p))
	for i := range p {
		pairs[i] = translatePairBackward(p[i])
	}
	return pairs
}

func (t rangeCreateRequestTranslator) Forward(
	_ context.Context,
	r api.RangeCreateRequest,
) (*gapi.RangeCreateRequest, error) {
	return &gapi.RangeCreateRequest{Ranges: translateRangesForward(r.Ranges)}, nil
}

func (t rangeCreateRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeCreateRequest,
) (api.RangeCreateRequest, error) {
	ranges, err := translateRangesBackward(r.Ranges)
	return api.RangeCreateRequest{Ranges: ranges}, err
}

func (t rangeCreateResponseTranslator) Forward(
	_ context.Context,
	r api.RangeCreateResponse,
) (*gapi.RangeCreateResponse, error) {
	return &gapi.RangeCreateResponse{Ranges: translateRangesForward(r.Ranges)}, nil
}

func (t rangeCreateResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeCreateResponse,
) (api.RangeCreateResponse, error) {
	ranges, err := translateRangesBackward(r.Ranges)
	return api.RangeCreateResponse{Ranges: ranges}, err
}

func (t rangeRetrieveRequestTranslator) Forward(
	_ context.Context,
	r api.RangeRetrieveRequest,
) (*gapi.RangeRetrieveRequest, error) {
	keys := make([]string, len(r.Keys))
	for i := range r.Keys {
		keys[i] = r.Keys[i].String()
	}
	return &gapi.RangeRetrieveRequest{Keys: keys, Names: r.Names}, nil
}

func (t rangeRetrieveRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeRetrieveRequest,
) (api.RangeRetrieveRequest, error) {
	keys := make([]uuid.UUID, len(r.Keys))
	for i := range r.Keys {
		key, err := uuid.Parse(r.Keys[i])
		if err != nil {
			return api.RangeRetrieveRequest{}, err
		}
		keys[i] = key
	}
	return api.RangeRetrieveRequest{Keys: keys, Names: r.Names}, nil
}

func (t rangeRetrieveResponseTranslator) Forward(
	_ context.Context,
	r api.RangeRetrieveResponse,
) (*gapi.RangeRetrieveResponse, error) {
	return &gapi.RangeRetrieveResponse{Ranges: translateRangesForward(r.Ranges)}, nil
}

func (t rangeRetrieveResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeRetrieveResponse,
) (api.RangeRetrieveResponse, error) {
	ranges, err := translateRangesBackward(r.Ranges)
	return api.RangeRetrieveResponse{Ranges: ranges}, err
}

func (t rangeKVGetRequestTranslator) Forward(
	_ context.Context,
	r api.RangeKVGetRequest,
) (*gapi.RangeKVGetRequest, error) {
	return &gapi.RangeKVGetRequest{
		RangeKey: r.Range.String(),
		Keys:     r.Keys,
	}, nil
}

func (t rangeKVGetRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVGetRequest,
) (api.RangeKVGetRequest, error) {
	key, err := uuid.Parse(r.RangeKey)
	return api.RangeKVGetRequest{
		Range: key,
		Keys:  r.Keys,
	}, err
}

func (t rangeKVGetResponseTranslator) Forward(
	_ context.Context,
	r api.RangeKVGetResponse,
) (*gapi.RangeKVGetResponse, error) {
	return &gapi.RangeKVGetResponse{Pairs: translatePairsForward(r.Pairs)}, nil
}

func (t rangeKVGetResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVGetResponse,
) (api.RangeKVGetResponse, error) {
	return api.RangeKVGetResponse{Pairs: translatePairsBackward(r.Pairs)}, nil
}

func (t rangeKVSetRequestTranslator) Forward(
	_ context.Context,
	r api.RangeKVSetRequest,
) (*gapi.RangeKVSetRequest, error) {
	return &gapi.RangeKVSetRequest{
		RangeKey: r.Range.String(),
		Pairs:    translatePairsForward(r.Pairs),
	}, nil
}

func (t rangeKVSetRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVSetRequest,
) (api.RangeKVSetRequest, error) {
	key, err := uuid.Parse(r.RangeKey)
	return api.RangeKVSetRequest{
		Range: key,
		Pairs: translatePairsBackward(r.Pairs),
	}, err
}

func (t rangeKVDeleteRequestTranslator) Forward(
	_ context.Context,
	r api.RangeKVDeleteRequest,
) (*gapi.RangeKVDeleteRequest, error) {
	return &gapi.RangeKVDeleteRequest{
		RangeKey: r.Range.String(),
		Keys:     r.Keys,
	}, nil
}

func (t rangeKVDeleteRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVDeleteRequest,
) (api.RangeKVDeleteRequest, error) {
	key, err := uuid.Parse(r.RangeKey)
	return api.RangeKVDeleteRequest{
		Range: key,
		Keys:  r.Keys,
	}, err
}

func (t rangeAliasSetRequestTranslator) Forward(
	_ context.Context,
	r api.RangeAliasSetRequest,
) (*gapi.RangeAliasSetRequest, error) {
	return &gapi.RangeAliasSetRequest{
		Range:   r.Range.String(),
		Aliases: unsafe.ReinterpretMapKeys[channel.Key, uint32, string](r.Aliases),
	}, nil
}

func (t rangeAliasSetRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasSetRequest,
) (api.RangeAliasSetRequest, error) {
	key, err := uuid.Parse(r.Range)
	return api.RangeAliasSetRequest{
		Range:   key,
		Aliases: unsafe.ReinterpretMapKeys[uint32, channel.Key, string](r.Aliases),
	}, err
}

func (t rangeAliasDeleteRequestTranslator) Forward(
	_ context.Context,
	r api.RangeAliasDeleteRequest,
) (*gapi.RangeAliasDeleteRequest, error) {
	return &gapi.RangeAliasDeleteRequest{
		Range:    r.Range.String(),
		Channels: unsafe.ReinterpretSlice[channel.Key, uint32](r.Channels),
	}, nil
}

func (t rangeAliasDeleteRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasDeleteRequest,
) (api.RangeAliasDeleteRequest, error) {
	key, err := uuid.Parse(r.Range)
	return api.RangeAliasDeleteRequest{
		Range:    key,
		Channels: unsafe.ReinterpretSlice[uint32, channel.Key](r.Channels),
	}, err
}

func (t rangeAliasResolveRequestTranslator) Forward(
	_ context.Context,
	r api.RangeAliasResolveRequest,
) (*gapi.RangeAliasResolveRequest, error) {
	return &gapi.RangeAliasResolveRequest{
		Range:   r.Range.String(),
		Aliases: r.Aliases,
	}, nil
}

func (t rangeAliasResolveRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasResolveRequest,
) (api.RangeAliasResolveRequest, error) {
	key, err := uuid.Parse(r.Range)
	return api.RangeAliasResolveRequest{
		Range:   key,
		Aliases: r.Aliases,
	}, err
}

func (t rangeAliasListRequestTranslator) Forward(
	_ context.Context,
	r api.RangeAliasListRequest,
) (*gapi.RangeAliasListRequest, error) {
	return &gapi.RangeAliasListRequest{
		Range: r.Range.String(),
	}, nil
}

func (t rangeAliasListRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasListRequest,
) (api.RangeAliasListRequest, error) {
	key, err := uuid.Parse(r.Range)
	return api.RangeAliasListRequest{
		Range: key,
	}, err
}

func (t rangeAliasResolveResponseTranslator) Forward(
	_ context.Context,
	r api.RangeAliasResolveResponse,
) (*gapi.RangeAliasResolveResponse, error) {
	return &gapi.RangeAliasResolveResponse{
		Aliases: unsafe.ReinterpretMapValues[string, channel.Key, uint32](r.Aliases),
	}, nil
}

func (t rangeAliasResolveResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasResolveResponse,
) (api.RangeAliasResolveResponse, error) {
	return api.RangeAliasResolveResponse{
		Aliases: unsafe.ReinterpretMapValues[string, uint32, channel.Key](r.Aliases),
	}, nil
}

func (t rangeAliasListResponseTranslator) Forward(
	_ context.Context,
	r api.RangeAliasListResponse,
) (*gapi.RangeAliasListResponse, error) {
	return &gapi.RangeAliasListResponse{
		Aliases: unsafe.ReinterpretMapKeys[channel.Key, uint32, string](r.Aliases),
	}, nil
}

func (t rangeAliasListResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasListResponse,
) (api.RangeAliasListResponse, error) {
	return api.RangeAliasListResponse{
		Aliases: unsafe.ReinterpretMapKeys[uint32, channel.Key, string](r.Aliases),
	}, nil
}

func (t rangeDeleteRequestTranslator) Forward(
	_ context.Context,
	r api.RangeDeleteRequest,
) (*gapi.RangeDeleteRequest, error) {
	keys := make([]string, len(r.Keys))
	for i, k := range r.Keys {
		keys[i] = k.String()
	}
	return &gapi.RangeDeleteRequest{Keys: keys}, nil
}

func (t rangeDeleteRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeDeleteRequest,
) (api.RangeDeleteRequest, error) {
	keys := make([]uuid.UUID, len(r.Keys))
	for i := range r.Keys {
		key, err := uuid.Parse(r.Keys[i])
		if err != nil {
			return api.RangeDeleteRequest{}, err
		}
		keys[i] = key
	}
	return api.RangeDeleteRequest{Keys: keys}, nil
}

func translateRangeForward(r api.Range) *gapi.Range {
	return &gapi.Range{
		Key:       r.Key.String(),
		Name:      r.Name,
		TimeRange: telem.TranslateTimeRangeForward(r.TimeRange),
	}
}

func translateRangesForward(r []api.Range) []*gapi.Range {
	ranges := make([]*gapi.Range, len(r))
	for i := range r {
		ranges[i] = translateRangeForward(r[i])
	}
	return ranges
}

func translateRangeBackward(r *gapi.Range) (or api.Range, err error) {
	if r.Key != "" {
		or.Key, err = uuid.Parse(r.Key)
		if err != nil {
			return api.Range{}, err
		}
	}
	or.Name = r.Name
	or.TimeRange = telem.TranslateTimeRangeBackward(r.TimeRange)
	return
}

func translateRangesBackward(r []*gapi.Range) ([]api.Range, error) {
	ranges := make([]api.Range, len(r))
	var err error
	for i := range r {
		ranges[i], err = translateRangeBackward(r[i])
		if err != nil {
			return nil, err
		}
	}
	return ranges, nil
}

func newRanger(a *api.Transport) fgrpc.BindableTransport {
	create := &rangeCreateServer{
		RequestTranslator:  rangeCreateRequestTranslator{},
		ResponseTranslator: rangeCreateResponseTranslator{},
		ServiceDesc:        &gapi.RangeCreateService_ServiceDesc,
	}
	a.RangeCreate = create
	retrieve := &rangeRetrieveServer{
		RequestTranslator:  rangeRetrieveRequestTranslator{},
		ResponseTranslator: rangeRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.RangeRetrieveService_ServiceDesc,
	}
	a.RangeRetrieve = retrieve
	rangeDelete := &rangeDeleteServer{
		RequestTranslator:  rangeDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeDeleteService_ServiceDesc,
	}
	a.RangeDelete = rangeDelete
	kvGet := &rangeKVGetServer{
		RequestTranslator:  rangeKVGetRequestTranslator{},
		ResponseTranslator: rangeKVGetResponseTranslator{},
		ServiceDesc:        &gapi.RangeKVGetService_ServiceDesc,
	}
	a.RangeKVGet = kvGet
	kvSet := &rangeKVSetServer{
		RequestTranslator:  rangeKVSetRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeKVSetService_ServiceDesc,
	}
	a.RangeKVSet = kvSet
	kvDelete := &rangeKVDeleteServer{
		RequestTranslator:  rangeKVDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeKVDeleteService_ServiceDesc,
	}
	a.RangeKVDelete = kvDelete
	rangeAliasSet := &rangeAliasSetServer{
		RequestTranslator:  rangeAliasSetRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeAliasSetService_ServiceDesc,
	}
	a.RangeAliasSet = rangeAliasSet
	rangeAliasDelete := &rangeAliasDeleteServer{
		RequestTranslator:  rangeAliasDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeAliasDeleteService_ServiceDesc,
	}
	a.RangeAliasDelete = rangeAliasDelete
	rangeAliasResolve := &rangeAliasResolveServer{
		RequestTranslator:  rangeAliasResolveRequestTranslator{},
		ResponseTranslator: rangeAliasResolveResponseTranslator{},
		ServiceDesc:        &gapi.RangeAliasResolveService_ServiceDesc,
	}
	a.RangeAliasResolve = rangeAliasResolve
	rangeAliasList := &rangeAliasListServer{
		RequestTranslator:  rangeAliasListRequestTranslator{},
		ResponseTranslator: rangeAliasListResponseTranslator{},
		ServiceDesc:        &gapi.RangeAliasListService_ServiceDesc,
	}
	a.RangeAliasList = rangeAliasList
	return fgrpc.CompoundBindableTransport{
		create,
		retrieve,
		kvGet,
		kvSet,
		kvDelete,
		rangeAliasSet,
		rangeAliasDelete,
		rangeAliasResolve,
		rangeAliasList,
		rangeDelete,
	}
}
