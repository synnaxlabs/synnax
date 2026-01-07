// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package connectivity

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	"github.com/synnaxlabs/synnax/pkg/api/connectivity"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	server = fgrpc.UnaryServer[
		types.Nil,
		*emptypb.Empty,
		connectivity.CheckResponse,
		*CheckResponse,
	]
)

type responseTranslator struct{}

var _ fgrpc.Translator[connectivity.CheckResponse, *CheckResponse] = (*responseTranslator)(nil)

func (responseTranslator) Forward(
	_ context.Context,
	r connectivity.CheckResponse,
) (*CheckResponse, error) {
	return &CheckResponse{
		ClusterKey:  r.ClusterKey,
		NodeVersion: r.NodeVersion,
	}, nil
}

func (responseTranslator) Backward(
	_ context.Context,
	r *CheckResponse,
) (connectivity.CheckResponse, error) {
	return connectivity.CheckResponse{
		ClusterKey:  r.ClusterKey,
		NodeVersion: r.NodeVersion,
	}, nil
}

func New(a *api.Transport) fgrpc.BindableTransport {
	s := &server{
		RequestTranslator:  fgrpc.EmptyTranslator{},
		ResponseTranslator: responseTranslator{},
		ServiceDesc:        &ConnectivityService_ServiceDesc,
	}
	a.ConnectivityCheck = s
	return s
}
