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
	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/status"
	"github.com/synnaxlabs/synnax/pkg/service/label"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	svcstatus "github.com/synnaxlabs/synnax/pkg/service/status"
	"github.com/synnaxlabs/synnax/pkg/service/status/pb"
	xpb "github.com/synnaxlabs/x/pb"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	setRequestTranslator             struct{}
	setResponseTranslator            struct{}
	retrieveRequestTranslator        struct{}
	retrieveResponseTranslator       struct{}
	deleteRequestTranslator          struct{}
	setByKeyOrNameRequestTranslator  struct{}
	setByKeyOrNameResponseTranslator struct{}
	setServer                        = grpc.UnaryServer[
		status.SetRequest,
		*SetRequest,
		status.SetResponse,
		*SetResponse,
	]
	retrieveServer = grpc.UnaryServer[
		status.RetrieveRequest,
		*RetrieveRequest,
		status.RetrieveResponse,
		*RetrieveResponse,
	]
	deleteServer = grpc.UnaryServer[
		status.DeleteRequest,
		*DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	setByKeyOrNameServer = grpc.UnaryServer[
		status.SetByKeyOrNameRequest,
		*SetByKeyOrNameRequest,
		status.SetByKeyOrNameResponse,
		*SetByKeyOrNameResponse,
	]
)

var (
	_ grpc.Translator[status.SetRequest, *SetRequest] = (*setRequestTranslator)(
		nil,
	)
	_ grpc.Translator[status.SetResponse, *SetResponse] = (*setResponseTranslator)(
		nil,
	)
	_ grpc.Translator[status.RetrieveRequest, *RetrieveRequest] = (*retrieveRequestTranslator)(
		nil,
	)
	_ grpc.Translator[status.RetrieveResponse, *RetrieveResponse] = (*retrieveResponseTranslator)(
		nil,
	)
	_ grpc.Translator[status.DeleteRequest, *DeleteRequest] = (*deleteRequestTranslator)(
		nil,
	)
	_ grpc.Translator[status.SetByKeyOrNameRequest, *SetByKeyOrNameRequest] = (*setByKeyOrNameRequestTranslator)(
		nil,
	)
	_ grpc.Translator[status.SetByKeyOrNameResponse, *SetByKeyOrNameResponse] = (*setByKeyOrNameResponseTranslator)(
		nil,
	)
)

func (setRequestTranslator) Forward(
	_ context.Context,
	msg status.SetRequest,
) (*SetRequest, error) {
	statuses, err := pb.StatusesToPB(msg.Statuses, xpb.AnyToPBAny)
	if err != nil {
		return nil, err
	}
	return &SetRequest{Parent: msg.Parent.String(), Statuses: statuses}, nil
}

func (setRequestTranslator) Backward(
	_ context.Context,
	msg *SetRequest,
) (status.SetRequest, error) {
	var parent ontology.ID
	if msg.Parent != "" {
		var err error
		parent, err = ontology.ParseID(msg.Parent)
		if err != nil {
			return status.SetRequest{}, err
		}
	}
	statuses, err := pb.StatusesFromPB(msg.Statuses, xpb.AnyFromPBAny)
	if err != nil {
		return status.SetRequest{}, err
	}
	return status.SetRequest{Parent: parent, Statuses: statuses}, nil
}

func (setResponseTranslator) Forward(
	_ context.Context,
	msg status.SetResponse,
) (*SetResponse, error) {
	statuses, err := pb.StatusesToPB(msg.Statuses, xpb.AnyToPBAny)
	if err != nil {
		return nil, err
	}
	return &SetResponse{Statuses: statuses}, nil
}

func (setResponseTranslator) Backward(
	_ context.Context,
	msg *SetResponse,
) (status.SetResponse, error) {
	statuses, err := pb.StatusesFromPB(msg.Statuses, xpb.AnyFromPBAny)
	if err != nil {
		return status.SetResponse{}, err
	}
	return status.SetResponse{Statuses: statuses}, nil
}

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	msg status.RetrieveRequest,
) (*RetrieveRequest, error) {
	hasLabels := lo.Map(msg.HasLabels, func(k label.Key, _ int) string {
		return k.String()
	})
	variants := lo.Map(msg.Variants, func(v svcstatus.Variant, _ int) string {
		return string(v)
	})
	return &RetrieveRequest{
		Keys:          msg.Keys,
		SearchTerm:    msg.SearchTerm,
		Offset:        int32(msg.Offset),
		Limit:         int32(msg.Limit),
		IncludeLabels: msg.IncludeLabels,
		HasLabels:     hasLabels,
		Variants:      variants,
	}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	msg *RetrieveRequest,
) (status.RetrieveRequest, error) {
	hasLabelKeys, err := lo.MapErr(
		msg.HasLabels,
		func(k string, _ int) (label.Key, error) {
			return uuid.Parse(k)
		},
	)
	if err != nil {
		return status.RetrieveRequest{}, err
	}
	variants := lo.Map(msg.Variants, func(v string, _ int) svcstatus.Variant {
		return svcstatus.Variant(v)
	})
	return status.RetrieveRequest{
		Keys:          msg.Keys,
		SearchTerm:    msg.SearchTerm,
		Offset:        int(msg.Offset),
		Limit:         int(msg.Limit),
		HasLabels:     hasLabelKeys,
		Variants:      variants,
		IncludeLabels: msg.IncludeLabels,
	}, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	msg status.RetrieveResponse,
) (*RetrieveResponse, error) {
	statuses, err := pb.StatusesToPB(msg.Statuses, xpb.AnyToPBAny)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{Statuses: statuses}, nil
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	msg *RetrieveResponse,
) (status.RetrieveResponse, error) {
	statuses, err := pb.StatusesFromPB(msg.Statuses, xpb.AnyFromPBAny)
	if err != nil {
		return status.RetrieveResponse{}, err
	}
	return status.RetrieveResponse{Statuses: statuses}, nil
}

func (deleteRequestTranslator) Forward(
	_ context.Context,
	msg status.DeleteRequest,
) (*DeleteRequest, error) {
	return &DeleteRequest{Keys: msg.Keys}, nil
}

func (deleteRequestTranslator) Backward(
	_ context.Context,
	msg *DeleteRequest,
) (status.DeleteRequest, error) {
	return status.DeleteRequest{Keys: msg.Keys}, nil
}

func (setByKeyOrNameRequestTranslator) Forward(
	_ context.Context,
	msg status.SetByKeyOrNameRequest,
) (*SetByKeyOrNameRequest, error) {
	return &SetByKeyOrNameRequest{
		KeyOrName: msg.KeyOrName,
		Message:   msg.Message,
		Variant:   string(msg.Variant),
	}, nil
}

func (setByKeyOrNameRequestTranslator) Backward(
	_ context.Context,
	msg *SetByKeyOrNameRequest,
) (status.SetByKeyOrNameRequest, error) {
	return status.SetByKeyOrNameRequest{
		KeyOrName: msg.KeyOrName,
		Message:   msg.Message,
		Variant:   svcstatus.Variant(msg.Variant),
	}, nil
}

func (setByKeyOrNameResponseTranslator) Forward(
	_ context.Context,
	msg status.SetByKeyOrNameResponse,
) (*SetByKeyOrNameResponse, error) {
	return &SetByKeyOrNameResponse{
		Key:             msg.Key,
		MultipleMatches: msg.MultipleMatches,
	}, nil
}

func (setByKeyOrNameResponseTranslator) Backward(
	_ context.Context,
	msg *SetByKeyOrNameResponse,
) (status.SetByKeyOrNameResponse, error) {
	return status.SetByKeyOrNameResponse{
		Key:             msg.Key,
		MultipleMatches: msg.MultipleMatches,
	}, nil
}

func New(t *api.Transport) grpc.BindableTransport {
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
		ResponseTranslator: grpc.EmptyTranslator{},
		ServiceDesc:        &StatusDeleteService_ServiceDesc,
	}
	sbkn := &setByKeyOrNameServer{
		RequestTranslator:  setByKeyOrNameRequestTranslator{},
		ResponseTranslator: setByKeyOrNameResponseTranslator{},
		ServiceDesc:        &StatusSetByKeyOrNameService_ServiceDesc,
	}
	t.StatusSet = s
	t.StatusRetrieve = r
	t.StatusDelete = d
	t.StatusSetByKeyOrName = sbkn
	return grpc.CompoundBindableTransport{s, r, d, sbkn}
}
