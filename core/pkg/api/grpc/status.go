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
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xstatus "github.com/synnaxlabs/x/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	statusSetRequestTranslator       struct{}
	statusSetResponseTranslator      struct{}
	statusRetrieveRequestTranslator  struct{}
	statusRetrieveResponseTranslator struct{}
	statusDeleteRequestTranslator    struct{}
	statusSetServer                  = fgrpc.UnaryServer[
		api.StatusSetRequest,
		*gapi.StatusSetRequest,
		api.StatusSetResponse,
		*gapi.StatusSetResponse,
	]
	statusRetrieveServer = fgrpc.UnaryServer[
		api.StatusRetrieveRequest,
		*gapi.StatusRetrieveRequest,
		api.StatusRetrieveResponse,
		*gapi.StatusRetrieveResponse,
	]
	statusDeleteServer = fgrpc.UnaryServer[
		api.StatusDeleteRequest,
		*gapi.StatusDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[api.StatusSetRequest, *gapi.StatusSetRequest]             = (*statusSetRequestTranslator)(nil)
	_ fgrpc.Translator[api.StatusSetResponse, *gapi.StatusSetResponse]           = (*statusSetResponseTranslator)(nil)
	_ fgrpc.Translator[api.StatusRetrieveRequest, *gapi.StatusRetrieveRequest]   = (*statusRetrieveRequestTranslator)(nil)
	_ fgrpc.Translator[api.StatusRetrieveResponse, *gapi.StatusRetrieveResponse] = (*statusRetrieveResponseTranslator)(nil)
	_ fgrpc.Translator[api.StatusDeleteRequest, *gapi.StatusDeleteRequest]       = (*statusDeleteRequestTranslator)(nil)
)

func translateStatusesForward(s []api.Status) ([]*xstatus.PBStatus, error) {
	var err error
	out := make([]*xstatus.PBStatus, len(s))
	for i, stat := range s {
		out[i], err = xstatus.TranslateToPB[any](xstatus.Status[any](stat.Status))
		if err != nil {
			return nil, err
		}
	}
	return out, err
}

func translateStatusesBackward(s []*xstatus.PBStatus) ([]api.Status, error) {
	out := make([]api.Status, len(s))
	for i, stat := range s {
		os, err := xstatus.TranslateFromPB[any](stat)
		if err != nil {
			return nil, err
		}
		out[i] = api.Status{Status: status.Status[any](os)}
	}
	return out, nil
}

func (t statusSetRequestTranslator) Forward(
	_ context.Context,
	msg api.StatusSetRequest,
) (*gapi.StatusSetRequest, error) {
	statuses, err := translateStatusesForward(msg.Statuses)
	if err != nil {
		return nil, err
	}
	return &gapi.StatusSetRequest{Parent: msg.Parent.String(), Statuses: statuses}, nil
}

func (t statusSetRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.StatusSetRequest,
) (api.StatusSetRequest, error) {
	var parent ontology.ID
	if msg.Parent != "" {
		var err error
		parent, err = ontology.ParseID(msg.Parent)
		if err != nil {
			return api.StatusSetRequest{}, err
		}
	}
	statuses, err := translateStatusesBackward(msg.Statuses)
	if err != nil {
		return api.StatusSetRequest{}, err
	}
	return api.StatusSetRequest{Parent: parent, Statuses: statuses}, nil
}

func (t statusSetResponseTranslator) Forward(
	_ context.Context,
	msg api.StatusSetResponse,
) (*gapi.StatusSetResponse, error) {
	statuses, err := translateStatusesForward(msg.Statuses)
	if err != nil {
		return nil, err
	}
	return &gapi.StatusSetResponse{Statuses: statuses}, nil
}

func (t statusSetResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.StatusSetResponse,
) (api.StatusSetResponse, error) {
	statuses, err := translateStatusesBackward(msg.Statuses)
	if err != nil {
		return api.StatusSetResponse{}, err
	}
	return api.StatusSetResponse{Statuses: statuses}, nil
}

func (t statusRetrieveRequestTranslator) Forward(
	_ context.Context,
	msg api.StatusRetrieveRequest,
) (*gapi.StatusRetrieveRequest, error) {
	hasLabels := make([]string, len(msg.HasLabels))
	for i, label := range msg.HasLabels {
		hasLabels[i] = label.String()
	}
	return &gapi.StatusRetrieveRequest{
		Keys:          msg.Keys,
		SearchTerm:    msg.SearchTerm,
		Offset:        int32(msg.Offset),
		Limit:         int32(msg.Limit),
		IncludeLabels: msg.IncludeLabels,
		HasLabels:     hasLabels,
	}, nil
}

func (t statusRetrieveRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.StatusRetrieveRequest,
) (api.StatusRetrieveRequest, error) {
	var (
		err          error
		hasLabelKeys = make([]uuid.UUID, len(msg.HasLabels))
	)
	for i, label := range msg.HasLabels {
		hasLabelKeys[i], err = uuid.Parse(label)
		if err != nil {
			return api.StatusRetrieveRequest{}, err
		}
	}
	return api.StatusRetrieveRequest{
		Keys:          msg.Keys,
		SearchTerm:    msg.SearchTerm,
		Offset:        int(msg.Offset),
		Limit:         int(msg.Limit),
		HasLabels:     hasLabelKeys,
		IncludeLabels: msg.IncludeLabels,
	}, nil
}

func (t statusRetrieveResponseTranslator) Forward(
	_ context.Context,
	msg api.StatusRetrieveResponse,
) (*gapi.StatusRetrieveResponse, error) {
	statuses, err := translateStatusesForward(msg.Statuses)
	if err != nil {
		return nil, err
	}
	return &gapi.StatusRetrieveResponse{Statuses: statuses}, nil
}

func (t statusRetrieveResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.StatusRetrieveResponse,
) (api.StatusRetrieveResponse, error) {
	statuses, err := translateStatusesBackward(msg.Statuses)
	if err != nil {
		return api.StatusRetrieveResponse{}, err
	}
	return api.StatusRetrieveResponse{Statuses: statuses}, nil
}

func (t statusDeleteRequestTranslator) Forward(
	_ context.Context,
	msg api.StatusDeleteRequest,
) (*gapi.StatusDeleteRequest, error) {
	return &gapi.StatusDeleteRequest{Keys: msg.Keys}, nil
}

func (t statusDeleteRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.StatusDeleteRequest,
) (api.StatusDeleteRequest, error) {
	return api.StatusDeleteRequest{Keys: msg.Keys}, nil
}

func newStatus(a *api.Transport) fgrpc.BindableTransport {
	s := &statusSetServer{
		RequestTranslator:  statusSetRequestTranslator{},
		ResponseTranslator: statusSetResponseTranslator{},
		ServiceDesc:        &gapi.StatusSetService_ServiceDesc,
	}
	r := &statusRetrieveServer{
		RequestTranslator:  statusRetrieveRequestTranslator{},
		ResponseTranslator: statusRetrieveResponseTranslator{},
		ServiceDesc:        &gapi.StatusRetrieveService_ServiceDesc,
	}
	d := &statusDeleteServer{
		RequestTranslator:  statusDeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.StatusDeleteService_ServiceDesc,
	}
	a.StatusSet = s
	a.StatusRetrieve = r
	a.StatusDelete = d
	return fgrpc.CompoundBindableTransport{s, r, d}
}
