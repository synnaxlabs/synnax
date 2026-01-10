// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package status

import (
	"context"
	"go/types"

	"github.com/google/uuid"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	apistatus "github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	svcstatus "github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/x/status"
	statuspb "github.com/synnaxlabs/x/status/pb"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/structpb"
)

type (
	setRequestTranslator       struct{}
	setResponseTranslator      struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
	setServer                  = fgrpc.UnaryServer[
		apistatus.SetRequest,
		*SetRequest,
		apistatus.SetResponse,
		*SetResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apistatus.RetrieveRequest,
		*RetrieveRequest,
		apistatus.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apistatus.DeleteRequest,
		*DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[apistatus.SetRequest, *SetRequest]             = (*setRequestTranslator)(nil)
	_ fgrpc.Translator[apistatus.SetResponse, *SetResponse]           = (*setResponseTranslator)(nil)
	_ fgrpc.Translator[apistatus.RetrieveRequest, *RetrieveRequest]   = (*retrieveRequestTranslator)(nil)
	_ fgrpc.Translator[apistatus.RetrieveResponse, *RetrieveResponse] = (*retrieveResponseTranslator)(nil)
	_ fgrpc.Translator[apistatus.DeleteRequest, *DeleteRequest]       = (*deleteRequestTranslator)(nil)
)

func translateDetailsToAny(ctx context.Context, details any) (*anypb.Any, error) {
	if details == nil {
		return nil, nil
	}
	// Try to convert to proto message if possible
	if pm, ok := details.(proto.Message); ok {
		return anypb.New(pm)
	}
	// Otherwise wrap in structpb.Value
	val, err := structpb.NewValue(details)
	if err != nil {
		return nil, err
	}
	return anypb.New(val)
}

func translateDetailsFromAny(ctx context.Context, a *anypb.Any) (any, error) {
	if a == nil {
		return nil, nil
	}
	// Try to unmarshal as structpb.Value (common case for any)
	val := &structpb.Value{}
	if err := a.UnmarshalTo(val); err == nil {
		return val.AsInterface(), nil
	}
	return nil, nil
}

func translateStatusesForward(ctx context.Context, s []apistatus.Status) ([]*statuspb.Status, error) {
	out := make([]*statuspb.Status, len(s))
	for i, stat := range s {
		// Convert service/status.Status[any] to x/status.Status[any]
		xstat := status.Status[any](stat.Status)
		pb, err := statuspb.StatusToPB(ctx, xstat, translateDetailsToAny)
		if err != nil {
			return nil, err
		}
		out[i] = pb
	}
	return out, nil
}

func translateStatusesBackward(ctx context.Context, s []*statuspb.Status) ([]apistatus.Status, error) {
	out := make([]apistatus.Status, len(s))
	for i, pb := range s {
		xstat, err := statuspb.StatusFromPB(ctx, pb, translateDetailsFromAny)
		if err != nil {
			return nil, err
		}
		// Convert x/status.Status[any] to service/status.Status[any]
		out[i] = apistatus.Status{Status: svcstatus.Status[any](xstat)}
	}
	return out, nil
}

func (t setRequestTranslator) Forward(
	ctx context.Context,
	msg apistatus.SetRequest,
) (*SetRequest, error) {
	statuses, err := translateStatusesForward(ctx, msg.Statuses)
	if err != nil {
		return nil, err
	}
	return &SetRequest{Parent: msg.Parent.String(), Statuses: statuses}, nil
}

func (t setRequestTranslator) Backward(
	ctx context.Context,
	msg *SetRequest,
) (apistatus.SetRequest, error) {
	var parent ontology.ID
	if msg.Parent != "" {
		var err error
		parent, err = ontology.ParseID(msg.Parent)
		if err != nil {
			return apistatus.SetRequest{}, err
		}
	}
	statuses, err := translateStatusesBackward(ctx, msg.Statuses)
	if err != nil {
		return apistatus.SetRequest{}, err
	}
	return apistatus.SetRequest{Parent: parent, Statuses: statuses}, nil
}

func (t setResponseTranslator) Forward(
	ctx context.Context,
	msg apistatus.SetResponse,
) (*SetResponse, error) {
	statuses, err := translateStatusesForward(ctx, msg.Statuses)
	if err != nil {
		return nil, err
	}
	return &SetResponse{Statuses: statuses}, nil
}

func (t setResponseTranslator) Backward(
	ctx context.Context,
	msg *SetResponse,
) (apistatus.SetResponse, error) {
	statuses, err := translateStatusesBackward(ctx, msg.Statuses)
	if err != nil {
		return apistatus.SetResponse{}, err
	}
	return apistatus.SetResponse{Statuses: statuses}, nil
}

func (t retrieveRequestTranslator) Forward(
	_ context.Context,
	msg apistatus.RetrieveRequest,
) (*RetrieveRequest, error) {
	hasLabels := make([]string, len(msg.HasLabels))
	for i, label := range msg.HasLabels {
		hasLabels[i] = label.String()
	}
	return &RetrieveRequest{
		Keys:          msg.Keys,
		SearchTerm:    msg.SearchTerm,
		Offset:        int32(msg.Offset),
		Limit:         int32(msg.Limit),
		IncludeLabels: msg.IncludeLabels,
		HasLabels:     hasLabels,
	}, nil
}

func (t retrieveRequestTranslator) Backward(
	_ context.Context,
	msg *RetrieveRequest,
) (apistatus.RetrieveRequest, error) {
	var (
		err          error
		hasLabelKeys = make([]uuid.UUID, len(msg.HasLabels))
	)
	for i, label := range msg.HasLabels {
		hasLabelKeys[i], err = uuid.Parse(label)
		if err != nil {
			return apistatus.RetrieveRequest{}, err
		}
	}
	return apistatus.RetrieveRequest{
		Keys:          msg.Keys,
		SearchTerm:    msg.SearchTerm,
		Offset:        int(msg.Offset),
		Limit:         int(msg.Limit),
		HasLabels:     hasLabelKeys,
		IncludeLabels: msg.IncludeLabels,
	}, nil
}

func (t retrieveResponseTranslator) Forward(
	ctx context.Context,
	msg apistatus.RetrieveResponse,
) (*RetrieveResponse, error) {
	statuses, err := translateStatusesForward(ctx, msg.Statuses)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Statuses: statuses}, nil
}

func (t retrieveResponseTranslator) Backward(
	ctx context.Context,
	msg *RetrieveResponse,
) (apistatus.RetrieveResponse, error) {
	statuses, err := translateStatusesBackward(ctx, msg.Statuses)
	if err != nil {
		return apistatus.RetrieveResponse{}, err
	}
	return apistatus.RetrieveResponse{Statuses: statuses}, nil
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	msg apistatus.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{Keys: msg.Keys}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	msg *DeleteRequest,
) (apistatus.DeleteRequest, error) {
	return apistatus.DeleteRequest{Keys: msg.Keys}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	s := &setServer{
		RequestTranslator:  setRequestTranslator{},
		ResponseTranslator: setResponseTranslator{},
		ServiceDesc:        &StatusSetService_ServiceDesc,
	}
	r := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &StatusRetrieveService_ServiceDesc,
	}
	d := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &StatusDeleteService_ServiceDesc,
	}
	a.StatusSet = s
	a.StatusRetrieve = r
	a.StatusDelete = d
	return fgrpc.CompoundBindableTransport{s, r, d}
}
