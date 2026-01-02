// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ranger

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	apiranger "github.com/synnaxlabs/synnax/pkg/api/ranger"
	rangepb "github.com/synnaxlabs/synnax/pkg/api/ranger/pb"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/unsafe"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createServer = fgrpc.UnaryServer[
		apiranger.CreateRequest,
		*gapi.RangeCreateRequest,
		apiranger.CreateResponse,
		*gapi.RangeCreateResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apiranger.RetrieveRequest,
		*gapi.RangeRetrieveRequest,
		apiranger.RetrieveResponse,
		*gapi.RangeRetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apiranger.DeleteRequest,
		*gapi.RangeDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	renameServer = fgrpc.UnaryServer[
		apiranger.RenameRequest,
		*gapi.RangeRenameRequest,
		types.Nil,
		*emptypb.Empty,
	]
	kvGetServer = fgrpc.UnaryServer[
		apiranger.KVGetRequest,
		*gapi.RangeKVGetRequest,
		apiranger.KVGetResponse,
		*gapi.RangeKVGetResponse,
	]
	kvSetServer = fgrpc.UnaryServer[
		apiranger.KVSetRequest,
		*gapi.RangeKVSetRequest,
		types.Nil,
		*emptypb.Empty,
	]
	kvDeleteServer = fgrpc.UnaryServer[
		apiranger.KVDeleteRequest,
		*gapi.RangeKVDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	aliasSetServer = fgrpc.UnaryServer[
		apiranger.AliasSetRequest,
		*gapi.RangeAliasSetRequest,
		types.Nil,
		*emptypb.Empty,
	]
	aliasDeleteServer = fgrpc.UnaryServer[
		apiranger.AliasDeleteRequest,
		*gapi.RangeAliasDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	aliasResolveServer = fgrpc.UnaryServer[
		apiranger.AliasResolveRequest,
		*gapi.RangeAliasResolveRequest,
		apiranger.AliasResolveResponse,
		*gapi.RangeAliasResolveResponse,
	]
	aliasListServer = fgrpc.UnaryServer[
		apiranger.AliasListRequest,
		*gapi.RangeAliasListRequest,
		apiranger.AliasListResponse,
		*gapi.RangeAliasListResponse,
	]
	aliasRetrieveServer = fgrpc.UnaryServer[
		apiranger.AliasRetrieveRequest,
		*gapi.RangeAliasRetrieveRequest,
		apiranger.AliasRetrieveResponse,
		*gapi.RangeAliasRetrieveResponse,
	]
)

type (
	createRequestTranslator         struct{}
	createResponseTranslator        struct{}
	retrieveRequestTranslator       struct{}
	retrieveResponseTranslator      struct{}
	deleteRequestTranslator         struct{}
	renameRequestTranslator         struct{}
	kvGetRequestTranslator          struct{}
	kvGetResponseTranslator         struct{}
	kvSetRequestTranslator          struct{}
	kvDeleteRequestTranslator       struct{}
	aliasSetRequestTranslator       struct{}
	aliasDeleteRequestTranslator    struct{}
	aliasResolveRequestTranslator   struct{}
	aliasResolveResponseTranslator  struct{}
	aliasListRequestTranslator      struct{}
	aliasListResponseTranslator     struct{}
	aliasRetrieveRequestTranslator  struct{}
	aliasRetrieveResponseTranslator struct{}
)

var (
	_ fgrpc.Translator[apiranger.CreateRequest, *gapi.RangeCreateRequest]                 = (*createRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.CreateResponse, *gapi.RangeCreateResponse]               = (*createResponseTranslator)(nil)
	_ fgrpc.Translator[apiranger.RetrieveRequest, *gapi.RangeRetrieveRequest]             = (*retrieveRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.RetrieveResponse, *gapi.RangeRetrieveResponse]           = (*retrieveResponseTranslator)(nil)
	_ fgrpc.Translator[apiranger.DeleteRequest, *gapi.RangeDeleteRequest]                 = (*deleteRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.RenameRequest, *gapi.RangeRenameRequest]                 = (*renameRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.KVGetRequest, *gapi.RangeKVGetRequest]                   = (*kvGetRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.KVGetResponse, *gapi.RangeKVGetResponse]                 = (*kvGetResponseTranslator)(nil)
	_ fgrpc.Translator[apiranger.KVSetRequest, *gapi.RangeKVSetRequest]                   = (*kvSetRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.KVDeleteRequest, *gapi.RangeKVDeleteRequest]             = (*kvDeleteRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.AliasSetRequest, *gapi.RangeAliasSetRequest]             = (*aliasSetRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.AliasDeleteRequest, *gapi.RangeAliasDeleteRequest]       = (*aliasDeleteRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.AliasResolveRequest, *gapi.RangeAliasResolveRequest]     = (*aliasResolveRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.AliasResolveResponse, *gapi.RangeAliasResolveResponse]   = (*aliasResolveResponseTranslator)(nil)
	_ fgrpc.Translator[apiranger.AliasListRequest, *gapi.RangeAliasListRequest]           = (*aliasListRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.AliasListResponse, *gapi.RangeAliasListResponse]         = (*aliasListResponseTranslator)(nil)
	_ fgrpc.Translator[apiranger.AliasRetrieveRequest, *gapi.RangeAliasRetrieveRequest]   = (*aliasRetrieveRequestTranslator)(nil)
	_ fgrpc.Translator[apiranger.AliasRetrieveResponse, *gapi.RangeAliasRetrieveResponse] = (*aliasRetrieveResponseTranslator)(nil)
)

func translatePairForward(p apiranger.KVPair) *gapi.KVPair {
	return &gapi.KVPair{Key: p.Key, Value: p.Value}
}

func translatePairsForward(p []apiranger.KVPair) []*gapi.KVPair {
	pairs := make([]*gapi.KVPair, len(p))
	for i := range p {
		pairs[i] = translatePairForward(p[i])
	}
	return pairs
}

func translatePairBackward(p *gapi.KVPair) apiranger.KVPair {
	return apiranger.KVPair{Key: p.Key, Value: p.Value}
}

func translatePairsBackward(p []*gapi.KVPair) []apiranger.KVPair {
	pairs := make([]apiranger.KVPair, len(p))
	for i := range p {
		pairs[i] = translatePairBackward(p[i])
	}
	return pairs
}

func (t createRequestTranslator) Forward(
	_ context.Context,
	r apiranger.CreateRequest,
) (*gapi.RangeCreateRequest, error) {
	return &gapi.RangeCreateRequest{Ranges: translateRangesForward(r.Ranges)}, nil
}

func (t createRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeCreateRequest,
) (apiranger.CreateRequest, error) {
	ranges, err := translateRangesBackward(r.Ranges)
	return apiranger.CreateRequest{Ranges: ranges}, err
}

func (t createResponseTranslator) Forward(
	_ context.Context,
	r apiranger.CreateResponse,
) (*gapi.RangeCreateResponse, error) {
	return &gapi.RangeCreateResponse{Ranges: translateRangesForward(r.Ranges)}, nil
}

func (t createResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeCreateResponse,
) (apiranger.CreateResponse, error) {
	ranges, err := translateRangesBackward(r.Ranges)
	return apiranger.CreateResponse{Ranges: ranges}, err
}

func (t retrieveRequestTranslator) Forward(
	_ context.Context,
	r apiranger.RetrieveRequest,
) (*gapi.RangeRetrieveRequest, error) {
	keys := make([]string, len(r.Keys))
	for i := range r.Keys {
		keys[i] = r.Keys[i].String()
	}
	return &gapi.RangeRetrieveRequest{Keys: keys, Names: r.Names}, nil
}

func (t retrieveRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeRetrieveRequest,
) (apiranger.RetrieveRequest, error) {
	keys := make([]uuid.UUID, len(r.Keys))
	for i := range r.Keys {
		key, err := uuid.Parse(r.Keys[i])
		if err != nil {
			return apiranger.RetrieveRequest{}, err
		}
		keys[i] = key
	}
	return apiranger.RetrieveRequest{Keys: keys, Names: r.Names}, nil
}

func (t retrieveResponseTranslator) Forward(
	_ context.Context,
	r apiranger.RetrieveResponse,
) (*gapi.RangeRetrieveResponse, error) {
	return &gapi.RangeRetrieveResponse{Ranges: translateRangesForward(r.Ranges)}, nil
}

func (t retrieveResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeRetrieveResponse,
) (apiranger.RetrieveResponse, error) {
	ranges, err := translateRangesBackward(r.Ranges)
	return apiranger.RetrieveResponse{Ranges: ranges}, err
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	r apiranger.DeleteRequest,
) (*gapi.RangeDeleteRequest, error) {
	keys := make([]string, len(r.Keys))
	for i, k := range r.Keys {
		keys[i] = k.String()
	}
	return &gapi.RangeDeleteRequest{Keys: keys}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeDeleteRequest,
) (apiranger.DeleteRequest, error) {
	keys := make([]uuid.UUID, len(r.Keys))
	for i := range r.Keys {
		key, err := uuid.Parse(r.Keys[i])
		if err != nil {
			return apiranger.DeleteRequest{}, err
		}
		keys[i] = key
	}
	return apiranger.DeleteRequest{Keys: keys}, nil
}

func (t renameRequestTranslator) Forward(
	_ context.Context,
	r apiranger.RenameRequest,
) (*gapi.RangeRenameRequest, error) {
	return &gapi.RangeRenameRequest{
		Key:  r.Key.String(),
		Name: r.Name,
	}, nil
}

func (t renameRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeRenameRequest,
) (apiranger.RenameRequest, error) {
	key, err := uuid.Parse(r.Key)
	return apiranger.RenameRequest{
		Key:  key,
		Name: r.Name,
	}, err
}

func (t kvGetRequestTranslator) Forward(
	_ context.Context,
	r apiranger.KVGetRequest,
) (*gapi.RangeKVGetRequest, error) {
	return &gapi.RangeKVGetRequest{
		RangeKey: r.Range.String(),
		Keys:     r.Keys,
	}, nil
}

func (t kvGetRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVGetRequest,
) (apiranger.KVGetRequest, error) {
	key, err := uuid.Parse(r.RangeKey)
	return apiranger.KVGetRequest{
		Range: key,
		Keys:  r.Keys,
	}, err
}

func (t kvGetResponseTranslator) Forward(
	_ context.Context,
	r apiranger.KVGetResponse,
) (*gapi.RangeKVGetResponse, error) {
	return &gapi.RangeKVGetResponse{Pairs: translatePairsForward(r.Pairs)}, nil
}

func (t kvGetResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVGetResponse,
) (apiranger.KVGetResponse, error) {
	return apiranger.KVGetResponse{Pairs: translatePairsBackward(r.Pairs)}, nil
}

func (t kvSetRequestTranslator) Forward(
	_ context.Context,
	r apiranger.KVSetRequest,
) (*gapi.RangeKVSetRequest, error) {
	pairs, err := rangepb.KVPairsToPB(r.Pairs)
	if err != nil {
		return nil, err
	}
	return &gapi.RangeKVSetRequest{
		RangeKey: r.Range.String(),
		Pairs:    pairs,
	}, nil
}

func (t kvSetRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVSetRequest,
) (apiranger.KVSetRequest, error) {
	key, err := uuid.Parse(r.RangeKey)
	return apiranger.KVSetRequest{
		Range: key,
		Pairs: translatePairsBackward(r.Pairs),
	}, err
}

func (t kvDeleteRequestTranslator) Forward(
	_ context.Context,
	r apiranger.KVDeleteRequest,
) (*gapi.RangeKVDeleteRequest, error) {
	return &gapi.RangeKVDeleteRequest{
		RangeKey: r.Range.String(),
		Keys:     r.Keys,
	}, nil
}

func (t kvDeleteRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVDeleteRequest,
) (apiranger.KVDeleteRequest, error) {
	key, err := uuid.Parse(r.RangeKey)
	return apiranger.KVDeleteRequest{
		Range: key,
		Keys:  r.Keys,
	}, err
}

func (t aliasSetRequestTranslator) Forward(
	_ context.Context,
	r apiranger.AliasSetRequest,
) (*gapi.RangeAliasSetRequest, error) {
	return &gapi.RangeAliasSetRequest{
		Range:   r.Range.String(),
		Aliases: unsafe.ReinterpretMapKeys[channel.Key, uint32, string](r.Aliases),
	}, nil
}

func (t aliasSetRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasSetRequest,
) (apiranger.AliasSetRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apiranger.AliasSetRequest{
		Range:   key,
		Aliases: unsafe.ReinterpretMapKeys[uint32, channel.Key, string](r.Aliases),
	}, err
}

func (t aliasDeleteRequestTranslator) Forward(
	_ context.Context,
	r apiranger.AliasDeleteRequest,
) (*gapi.RangeAliasDeleteRequest, error) {
	return &gapi.RangeAliasDeleteRequest{
		Range:    r.Range.String(),
		Channels: unsafe.ReinterpretSlice[channel.Key, uint32](r.Channels),
	}, nil
}

func (t aliasDeleteRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasDeleteRequest,
) (apiranger.AliasDeleteRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apiranger.AliasDeleteRequest{
		Range:    key,
		Channels: unsafe.ReinterpretSlice[uint32, channel.Key](r.Channels),
	}, err
}

func (t aliasResolveRequestTranslator) Forward(
	_ context.Context,
	r apiranger.AliasResolveRequest,
) (*gapi.RangeAliasResolveRequest, error) {
	return &gapi.RangeAliasResolveRequest{
		Range:   r.Range.String(),
		Aliases: r.Aliases,
	}, nil
}

func (t aliasResolveRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasResolveRequest,
) (apiranger.AliasResolveRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apiranger.AliasResolveRequest{
		Range:   key,
		Aliases: r.Aliases,
	}, err
}

func (t aliasListRequestTranslator) Forward(
	_ context.Context,
	r apiranger.AliasListRequest,
) (*gapi.RangeAliasListRequest, error) {
	return &gapi.RangeAliasListRequest{
		Range: r.Range.String(),
	}, nil
}

func (t aliasListRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasListRequest,
) (apiranger.AliasListRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apiranger.AliasListRequest{
		Range: key,
	}, err
}

func (t aliasResolveResponseTranslator) Forward(
	_ context.Context,
	r apiranger.AliasResolveResponse,
) (*gapi.RangeAliasResolveResponse, error) {
	return &gapi.RangeAliasResolveResponse{
		Aliases: unsafe.ReinterpretMapValues[string, channel.Key, uint32](r.Aliases),
	}, nil
}

func (t aliasResolveResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasResolveResponse,
) (apiranger.AliasResolveResponse, error) {
	return apiranger.AliasResolveResponse{
		Aliases: unsafe.ReinterpretMapValues[string, uint32, channel.Key](r.Aliases),
	}, nil
}

func (t aliasListResponseTranslator) Forward(
	_ context.Context,
	r apiranger.AliasListResponse,
) (*gapi.RangeAliasListResponse, error) {
	return &gapi.RangeAliasListResponse{
		Aliases: unsafe.ReinterpretMapKeys[channel.Key, uint32, string](r.Aliases),
	}, nil
}

func (t aliasListResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasListResponse,
) (apiranger.AliasListResponse, error) {
	return apiranger.AliasListResponse{
		Aliases: unsafe.ReinterpretMapKeys[uint32, channel.Key, string](r.Aliases),
	}, nil
}

func (t aliasRetrieveRequestTranslator) Forward(
	_ context.Context,
	r apiranger.AliasRetrieveRequest,
) (*gapi.RangeAliasRetrieveRequest, error) {
	return &gapi.RangeAliasRetrieveRequest{
		Range:    r.Range.String(),
		Channels: unsafe.ReinterpretSlice[channel.Key, uint32](r.Channels),
	}, nil
}

func (t aliasRetrieveRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasRetrieveRequest,
) (apiranger.AliasRetrieveRequest, error) {
	key, err := uuid.Parse(r.Range)
	return apiranger.AliasRetrieveRequest{
		Range:    key,
		Channels: unsafe.ReinterpretSlice[uint32, channel.Key](r.Channels),
	}, err
}

func (t aliasRetrieveResponseTranslator) Forward(
	_ context.Context,
	r apiranger.AliasRetrieveResponse,
) (*gapi.RangeAliasRetrieveResponse, error) {
	return &gapi.RangeAliasRetrieveResponse{
		Aliases: unsafe.ReinterpretMapKeys[channel.Key, uint32, string](r.Aliases),
	}, nil
}

func (t aliasRetrieveResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeAliasRetrieveResponse,
) (apiranger.AliasRetrieveResponse, error) {
	return apiranger.AliasRetrieveResponse{
		Aliases: unsafe.ReinterpretMapKeys[uint32, channel.Key, string](r.Aliases),
	}, nil
}

func translateRangeForward(r apiranger.Range) *gapi.Range {
	return &gapi.Range{
		Key:       r.Key.String(),
		Name:      r.Name,
		TimeRange: telem.TranslateTimeRangeForward(r.TimeRange),
	}
}

func translateRangesForward(r []apiranger.Range) []*gapi.Range {
	ranges := make([]*gapi.Range, len(r))
	for i := range r {
		ranges[i] = translateRangeForward(r[i])
	}
	return ranges
}

func translateRangeBackward(r *gapi.Range) (or apiranger.Range, err error) {
	if r.Key != "" {
		or.Key, err = uuid.Parse(r.Key)
		if err != nil {
			return apiranger.Range{}, err
		}
	}
	or.Name = r.Name
	or.TimeRange = telem.TranslateTimeRangeBackward(r.TimeRange)
	return
}

func translateRangesBackward(r []*gapi.Range) ([]apiranger.Range, error) {
	ranges := make([]apiranger.Range, len(r))
	var err error
	for i := range r {
		ranges[i], err = translateRangeBackward(r[i])
		if err != nil {
			return nil, err
		}
	}
	return ranges, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	create := &createServer{
		RequestTranslator:  createRequestTranslator{},
		ResponseTranslator: createResponseTranslator{},
		ServiceDesc:        &gapi.RangeCreateService_ServiceDesc,
	}
	a.RangeCreate = create
	retrieve := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &gapi.RangeRetrieveService_ServiceDesc,
	}
	a.RangeRetrieve = retrieve
	rangeDelete := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeDeleteService_ServiceDesc,
	}
	a.RangeDelete = rangeDelete
	rename := &renameServer{
		RequestTranslator:  renameRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeRenameService_ServiceDesc,
	}
	a.RangeRename = rename
	kvGet := &kvGetServer{
		RequestTranslator:  kvGetRequestTranslator{},
		ResponseTranslator: kvGetResponseTranslator{},
		ServiceDesc:        &gapi.RangeKVGetService_ServiceDesc,
	}
	a.RangeKVGet = kvGet
	kvSet := &kvSetServer{
		RequestTranslator:  kvSetRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeKVSetService_ServiceDesc,
	}
	a.RangeKVSet = kvSet
	kvDelete := &kvDeleteServer{
		RequestTranslator:  kvDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeKVDeleteService_ServiceDesc,
	}
	a.RangeKVDelete = kvDelete
	aliasSet := &aliasSetServer{
		RequestTranslator:  aliasSetRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeAliasSetService_ServiceDesc,
	}
	a.RangeAliasSet = aliasSet
	aliasDelete := &aliasDeleteServer{
		RequestTranslator:  aliasDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.RangeAliasDeleteService_ServiceDesc,
	}
	a.RangeAliasDelete = aliasDelete
	aliasResolve := &aliasResolveServer{
		RequestTranslator:  aliasResolveRequestTranslator{},
		ResponseTranslator: aliasResolveResponseTranslator{},
		ServiceDesc:        &gapi.RangeAliasResolveService_ServiceDesc,
	}
	a.RangeAliasResolve = aliasResolve
	aliasList := &aliasListServer{
		RequestTranslator:  aliasListRequestTranslator{},
		ResponseTranslator: aliasListResponseTranslator{},
		ServiceDesc:        &gapi.RangeAliasListService_ServiceDesc,
	}
	a.RangeAliasList = aliasList
	aliasRetrieve := &aliasRetrieveServer{
		RequestTranslator:  aliasRetrieveRequestTranslator{},
		ResponseTranslator: aliasRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.RangeAliasRetrieveService_ServiceDesc,
	}
	a.RangeAliasRetrieve = aliasRetrieve
	return fgrpc.CompoundBindableTransport{
		create,
		retrieve,
		kvGet,
		kvSet,
		kvDelete,
		aliasSet,
		aliasDelete,
		aliasResolve,
		aliasList,
		aliasRetrieve,
		rangeDelete,
		rename,
	}
}
