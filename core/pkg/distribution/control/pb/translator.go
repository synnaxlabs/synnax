// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package pb

import (
	"context"

	"github.com/samber/lo"
	"github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/control"
	xcontrol "github.com/synnaxlabs/x/control"
	controlpb "github.com/synnaxlabs/x/control/pb"
	"github.com/synnaxlabs/x/unsafe"
)

var (
	RetrieveRequestTranslator   grpc.Translator[control.RetrieveRequest, *RetrieveRequest]     = retrieveRequestTranslator{}
	RetrieveResponseTranslator  grpc.Translator[control.RetrieveResponse, *RetrieveResponse]   = retrieveResponseTranslator{}
	SubscribeRequestTranslator  grpc.Translator[control.SubscribeRequest, *SubscribeRequest]   = subscribeRequestTranslator{}
	SubscribeResponseTranslator grpc.Translator[control.SubscribeResponse, *SubscribeResponse] = subscribeResponseTranslator{}
)

func stateToPB(s control.State) (*State, error) {
	subject, err := controlpb.SubjectToPB(s.Subject)
	if err != nil {
		return nil, err
	}
	return &State{
		Subject:   subject,
		Resource:  uint32(s.Resource),
		Authority: uint32(s.Authority),
	}, nil
}

func stateFromPB(s *State) (control.State, error) {
	if s == nil {
		return control.State{}, nil
	}
	subject, err := controlpb.SubjectFromPB(s.Subject)
	if err != nil {
		return control.State{}, err
	}
	return control.State{
		Subject:   subject,
		Resource:  channel.Key(s.Resource),
		Authority: xcontrol.Authority(s.Authority),
	}, nil
}

func statesToPB(states []control.State) ([]*State, error) {
	return lo.MapErr(states, func(s control.State, _ int) (*State, error) {
		return stateToPB(s)
	})
}

func statesFromPB(states []*State) ([]control.State, error) {
	return lo.MapErr(states, func(s *State, _ int) (control.State, error) {
		return stateFromPB(s)
	})
}

type retrieveRequestTranslator struct{}

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

type retrieveResponseTranslator struct{}

func (retrieveResponseTranslator) Forward(
	_ context.Context,
	msg control.RetrieveResponse,
) (*RetrieveResponse, error) {
	states, err := statesToPB(msg.States)
	if err != nil {
		return nil, err
	}
	return &RetrieveResponse{States: states}, nil
}

func (retrieveResponseTranslator) Backward(
	_ context.Context,
	msg *RetrieveResponse,
) (control.RetrieveResponse, error) {
	states, err := statesFromPB(msg.States)
	if err != nil {
		return control.RetrieveResponse{}, err
	}
	return control.RetrieveResponse{States: states}, nil
}

type subscribeRequestTranslator struct{}

func (subscribeRequestTranslator) Forward(
	_ context.Context,
	_ control.SubscribeRequest,
) (*SubscribeRequest, error) {
	return &SubscribeRequest{}, nil
}

func (subscribeRequestTranslator) Backward(
	_ context.Context,
	_ *SubscribeRequest,
) (control.SubscribeRequest, error) {
	return control.SubscribeRequest{}, nil
}

type subscribeResponseTranslator struct{}

func (subscribeResponseTranslator) Forward(
	_ context.Context,
	msg control.SubscribeResponse,
) (*SubscribeResponse, error) {
	states, err := statesToPB(msg.States)
	if err != nil {
		return nil, err
	}
	return &SubscribeResponse{States: states}, nil
}

func (subscribeResponseTranslator) Backward(
	_ context.Context,
	msg *SubscribeResponse,
) (control.SubscribeResponse, error) {
	states, err := statesFromPB(msg.States)
	if err != nil {
		return control.SubscribeResponse{}, err
	}
	return control.SubscribeResponse{States: states}, nil
}
