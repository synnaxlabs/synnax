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
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	apistatus "github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/distribution/ontology"
	svcstatus "github.com/synnaxlabs/synnax/pkg/service/status"
	xstatus "github.com/synnaxlabs/x/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	setRequestTranslator       struct{}
	setResponseTranslator      struct{}
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	deleteRequestTranslator    struct{}
	setServer                  = fgrpc.UnaryServer[
		apistatus.SetRequest,
		*gapi.StatusSetRequest,
		apistatus.SetResponse,
		*gapi.StatusSetResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		apistatus.RetrieveRequest,
		*gapi.StatusRetrieveRequest,
		apistatus.RetrieveResponse,
		*gapi.StatusRetrieveResponse,
	]
	deleteServer = fgrpc.UnaryServer[
		apistatus.DeleteRequest,
		*gapi.StatusDeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ fgrpc.Translator[apistatus.SetRequest, *gapi.StatusSetRequest]             = (*setRequestTranslator)(nil)
	_ fgrpc.Translator[apistatus.SetResponse, *gapi.StatusSetResponse]           = (*setResponseTranslator)(nil)
	_ fgrpc.Translator[apistatus.RetrieveRequest, *gapi.StatusRetrieveRequest]   = (*retrieveRequestTranslator)(nil)
	_ fgrpc.Translator[apistatus.RetrieveResponse, *gapi.StatusRetrieveResponse] = (*retrieveResponseTranslator)(nil)
	_ fgrpc.Translator[apistatus.DeleteRequest, *gapi.StatusDeleteRequest]       = (*deleteRequestTranslator)(nil)
)

func translateManyForward(s []apistatus.Status) ([]*xstatus.PBStatus, error) {
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

func translateManyBackward(s []*xstatus.PBStatus) ([]apistatus.Status, error) {
	out := make([]apistatus.Status, len(s))
	for i, stat := range s {
		os, err := xstatus.TranslateFromPB[any](stat)
		if err != nil {
			return nil, err
		}
		out[i] = apistatus.Status{Status: svcstatus.Status[any](os)}
	}
	return out, nil
}

func (t setRequestTranslator) Forward(
	_ context.Context,
	msg apistatus.SetRequest,
) (*gapi.StatusSetRequest, error) {
	statuses, err := translateManyForward(msg.Statuses)
	if err != nil {
		return nil, err
	}
	return &gapi.StatusSetRequest{Parent: msg.Parent.String(), Statuses: statuses}, nil
}

func (t setRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.StatusSetRequest,
) (apistatus.SetRequest, error) {
	var parent ontology.ID
	if msg.Parent != "" {
		var err error
		parent, err = ontology.ParseID(msg.Parent)
		if err != nil {
			return apistatus.SetRequest{}, err
		}
	}
	statuses, err := translateManyBackward(msg.Statuses)
	if err != nil {
		return apistatus.SetRequest{}, err
	}
	return apistatus.SetRequest{Parent: parent, Statuses: statuses}, nil
}

func (t setResponseTranslator) Forward(
	_ context.Context,
	msg apistatus.SetResponse,
) (*gapi.StatusSetResponse, error) {
	statuses, err := translateManyForward(msg.Statuses)
	if err != nil {
		return nil, err
	}
	return &gapi.StatusSetResponse{Statuses: statuses}, nil
}

func (t setResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.StatusSetResponse,
) (apistatus.SetResponse, error) {
	statuses, err := translateManyBackward(msg.Statuses)
	if err != nil {
		return apistatus.SetResponse{}, err
	}
	return apistatus.SetResponse{Statuses: statuses}, nil
}

func (t retrieveRequestTranslator) Forward(
	_ context.Context,
	msg apistatus.RetrieveRequest,
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

func (t retrieveRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.StatusRetrieveRequest,
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
	_ context.Context,
	msg apistatus.RetrieveResponse,
) (*gapi.StatusRetrieveResponse, error) {
	statuses, err := translateManyForward(msg.Statuses)
	if err != nil {
		return nil, err
	}
	return &gapi.StatusRetrieveResponse{Statuses: statuses}, nil
}

func (t retrieveResponseTranslator) Backward(
	_ context.Context,
	msg *gapi.StatusRetrieveResponse,
) (apistatus.RetrieveResponse, error) {
	statuses, err := translateManyBackward(msg.Statuses)
	if err != nil {
		return apistatus.RetrieveResponse{}, err
	}
	return apistatus.RetrieveResponse{Statuses: statuses}, nil
}

func (t deleteRequestTranslator) Forward(
	_ context.Context,
	msg apistatus.DeleteRequest,
) (*gapi.StatusDeleteRequest, error) {
	return &gapi.StatusDeleteRequest{Keys: msg.Keys}, nil
}

func (t deleteRequestTranslator) Backward(
	_ context.Context,
	msg *gapi.StatusDeleteRequest,
) (apistatus.DeleteRequest, error) {
	return apistatus.DeleteRequest{Keys: msg.Keys}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	s := &setServer{
		RequestTranslator:  setRequestTranslator{},
		ResponseTranslator: setResponseTranslator{},
		ServiceDesc:        &gapi.StatusSetService_ServiceDesc,
	}
	r := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &gapi.StatusRetrieveService_ServiceDesc,
	}
	d := &deleteServer{
		RequestTranslator:  deleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &gapi.StatusDeleteService_ServiceDesc,
	}
	a.StatusSet = s
	a.StatusRetrieve = r
	a.StatusDelete = d
	return fgrpc.CompoundBindableTransport{s, r, d}
}
