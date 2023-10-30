package grpc

import (
	"context"
	"github.com/google/uuid"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/gen/go/v1"
	"github.com/synnaxlabs/x/telem/telempb"
	"go/types"
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
)

type (
	rangeCreateRequestTranslator    struct{}
	rangeCreateResponseTranslator   struct{}
	rangeRetrieveRequestTranslator  struct{}
	rangeRetrieveResponseTranslator struct{}
	rangeKVGetRequestTranslator     struct{}
	rangeKVGetResponseTranslator    struct{}
	rangeKVSetRequestTranslator     struct{}
	rangeKVDeleteRequestTranslator  struct{}
)

var (
	_ fgrpc.Translator[api.RangeCreateRequest, *gapi.RangeCreateRequest]       = (*rangeCreateRequestTranslator)(nil)
	_ fgrpc.Translator[api.RangeCreateResponse, *gapi.RangeCreateResponse]     = (*rangeCreateResponseTranslator)(nil)
	_ fgrpc.Translator[api.RangeRetrieveRequest, *gapi.RangeRetrieveRequest]   = (*rangeRetrieveRequestTranslator)(nil)
	_ fgrpc.Translator[api.RangeRetrieveResponse, *gapi.RangeRetrieveResponse] = (*rangeRetrieveResponseTranslator)(nil)
	_ fgrpc.Translator[api.RangeKVGetRequest, *gapi.RangeKVGetRequest]         = (*rangeKVGetRequestTranslator)(nil)
	_ fgrpc.Translator[api.RangeKVGetResponse, *gapi.RangeKVGetResponse]       = (*rangeKVGetResponseTranslator)(nil)
	_ fgrpc.Translator[api.RangeKVSetRequest, *gapi.RangeKVSetRequest]         = (*rangeKVSetRequestTranslator)(nil)
	_ fgrpc.Translator[api.RangeKVDeleteRequest, *gapi.RangeKVDeleteRequest]   = (*rangeKVDeleteRequestTranslator)(nil)
)

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
	return &gapi.RangeKVGetResponse{Pairs: r.Pairs}, nil
}

func (t rangeKVGetResponseTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVGetResponse,
) (api.RangeKVGetResponse, error) {
	return api.RangeKVGetResponse{Pairs: r.Pairs}, nil
}

func (t rangeKVSetRequestTranslator) Forward(
	_ context.Context,
	r api.RangeKVSetRequest,
) (*gapi.RangeKVSetRequest, error) {
	return &gapi.RangeKVSetRequest{
		RangeKey: r.Range.String(),
		Pairs:    r.Pairs,
	}, nil
}

func (t rangeKVSetRequestTranslator) Backward(
	_ context.Context,
	r *gapi.RangeKVSetRequest,
) (api.RangeKVSetRequest, error) {
	key, err := uuid.Parse(r.RangeKey)
	return api.RangeKVSetRequest{
		Range: key,
		Pairs: r.Pairs,
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

func translateRangeForward(r api.Range) *gapi.Range {
	return &gapi.Range{
		Key:       r.Key.String(),
		Name:      r.Name,
		TimeRange: telempb.TranslateTimeRangeForward(r.TimeRange),
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
	or.TimeRange = telempb.TranslateTimeRangeBackward(r.TimeRange)
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
	return fgrpc.CompoundBindableTransport{
		create,
		retrieve,
		kvGet,
		kvSet,
		kvDelete,
	}
}
