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

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	"github.com/synnaxlabs/synnax/pkg/service/status"
	xstatus "github.com/synnaxlabs/x/status"
	"github.com/synnaxlabs/x/telem"
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
	statusSetClient = fgrpc.UnaryClient[
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
	statusRetrieveClient = fgrpc.UnaryClient[
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
	statusDeleteClient = fgrpc.UnaryClient[
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

func translateStatusForward(s api.Status, _ int) *gapi.Status {
	return &gapi.Status{
		Key:         s.Key,
		Name:        s.Name,
		Variant:     string(s.Variant),
		Message:     s.Message,
		Description: s.Description,
		Time:        int64(s.Time),
	}
}

func translateStatusBackward(s *gapi.Status, _ int) api.Status {
	return api.Status{
		Status: status.Status(xstatus.Status[any]{
			Key:         s.Key,
			Name:        s.Name,
			Variant:     xstatus.Variant(s.Variant),
			Message:     s.Message,
			Description: s.Description,
			Time:        telem.TimeStamp(s.Time),
		}),
	}
}

func (t statusSetRequestTranslator) Forward(
	_ context.Context,
	msg api.StatusSetRequest,
) (*gapi.StatusSetRequest, error) {
	return &gapi.StatusSetRequest{
		Parent:   msg.Parent.String(),
		Statuses: lo.Map(msg.Statuses, translateStatusForward),
	}, nil
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
	return api.StatusSetRequest{
		Parent:   parent,
		Statuses: lo.Map(msg.Statuses, translateStatusBackward),
	}, nil
}

func (t statusSetResponseTranslator) Forward(
	_ context.Context,
	msg api.StatusSetResponse,
) (*gapi.StatusSetResponse, error) {
	return &gapi.StatusSetResponse{
		Statuses: lo.Map(msg.Statuses, translateStatusForward),
	}, nil
}

func (t statusSetResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.StatusSetResponse,
) (api.StatusSetResponse, error) {
	return api.StatusSetResponse{
		Statuses: lo.Map(msg.Statuses, translateStatusBackward),
	}, nil
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
	return api.StatusRetrieveRequest{
		Keys:          msg.Keys,
		SearchTerm:    msg.SearchTerm,
		Offset:        int(msg.Offset),
		Limit:         int(msg.Limit),
		IncludeLabels: msg.IncludeLabels,
	}, nil
}

func (t statusRetrieveResponseTranslator) Forward(
	_ context.Context,
	msg api.StatusRetrieveResponse,
) (*gapi.StatusRetrieveResponse, error) {
	return &gapi.StatusRetrieveResponse{
		Statuses: lo.Map(msg.Statuses, translateStatusForward),
	}, nil
}

func (t statusRetrieveResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.StatusRetrieveResponse,
) (api.StatusRetrieveResponse, error) {
	return api.StatusRetrieveResponse{
		Statuses: lo.Map(msg.Statuses, translateStatusBackward),
	}, nil
}

func (t statusDeleteRequestTranslator) Forward(
	_ context.Context,
	msg api.StatusDeleteRequest,
) (*gapi.StatusDeleteRequest, error) {
	return &gapi.StatusDeleteRequest{
		Keys: msg.Keys,
	}, nil
}

func (t statusDeleteRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.StatusDeleteRequest,
) (api.StatusDeleteRequest, error) {
	return api.StatusDeleteRequest{
		Keys: msg.Keys,
	}, nil
}

func newStatus(a *api.Transport) []fgrpc.BindableTransport {
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
	return []fgrpc.BindableTransport{s, r, d}
}
