// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/control"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	xcontrol "github.com/synnaxlabs/x/control"
	controlpb "github.com/synnaxlabs/x/control/pb"
	"github.com/synnaxlabs/x/unsafe"
)

type (
	retrieveRequestTranslator  struct{}
	retrieveResponseTranslator struct{}
	retrieveServer             = grpc.UnaryServer[
		control.RetrieveRequest,
		*RetrieveRequest,
		control.RetrieveResponse,
		*RetrieveResponse,
	]
)

var (
	_ grpc.Translator[control.RetrieveRequest, *RetrieveRequest] = (*retrieveRequestTranslator)(
		nil,
	)
	_ grpc.Translator[control.RetrieveResponse, *RetrieveResponse] = (*retrieveResponseTranslator)(
		nil,
	)
)

func (retrieveRequestTranslator) Forward(
	_ context.Context,
	msg control.RetrieveRequest,
) (*RetrieveRequest, error) {
	return &RetrieveRequest{
		Keys: unsafe.ReinterpretSlice[channel.Key, uint32](msg.Keys),
	}, nil
}

func (retrieveRequestTranslator) Backward(
	_ context.Context,
	msg *RetrieveRequest,
) (control.RetrieveRequest, error) {
	return control.RetrieveRequest{
		Keys: unsafe.ReinterpretSlice[uint32, channel.Key](msg.Keys),
	}, nil
}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	msg control.RetrieveResponse,
) (*RetrieveResponse, error) {
	states, err := lo.MapErr(
		msg.States,
		func(s control.State, _ int) (*State, error) {
			subject, err := controlpb.SubjectToPB(s.Subject)
			if err != nil {
				return nil, err
			}
			return &State{
				Subject:   subject,
				Resource:  uint32(s.Resource),
				Authority: uint32(s.Authority),
			}, nil
		},
	)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{States: states}, nil
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	msg *RetrieveResponse,
) (control.RetrieveResponse, error) {
	states, err := lo.MapErr(msg.States, func(s *State, _ int) (control.State, error) {
		subject, err := controlpb.SubjectFromPB(s.Subject)
		if err != nil {
			return control.State{}, err
		}
		return control.State{
			Subject:   subject,
			Resource:  channel.Key(s.Resource),
			Authority: xcontrol.Authority(s.Authority),
		}, nil
	})
	if err != nil {
		return control.RetrieveResponse{}, err
	}
	return control.RetrieveResponse{States: states}, nil
}

// New binds the control retrieve server onto t and returns it for gRPC registration.
func New(t *api.Transport) grpc.BindableTransport {
	r := &retrieveServer{
		RequestTranslator:  retrieveRequestTranslator{},
		ResponseTranslator: retrieveResponseTranslator{},
		ServiceDesc:        &ControlRetrieveService_ServiceDesc,
	}
	t.ControlRetrieve = r
	return r
}
