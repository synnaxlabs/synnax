// Copyright 2025 Synnax Labs, Inc.
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
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	server = fgrpc.UnaryServer[
		types.Nil,
		*emptypb.Empty,
		connectivity.CheckResponse,
		*gapi.ConnectivityCheckResponse,
	]
)

type responseTranslator struct{}

var _ fgrpc.Translator[connectivity.CheckResponse, *gapi.ConnectivityCheckResponse] = (*responseTranslator)(nil)

func (responseTranslator) Forward(
	_ context.Context,
	r connectivity.CheckResponse,
) (*gapi.ConnectivityCheckResponse, error) {
	return &gapi.ConnectivityCheckResponse{
		ClusterKey:  r.ClusterKey,
		NodeVersion: r.NodeVersion,
	}, nil
}

func (responseTranslator) Backward(
	_ context.Context,
	r *gapi.ConnectivityCheckResponse,
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
		ServiceDesc:        &gapi.ConnectivityService_ServiceDesc,
	}
	a.ConnectivityCheck = s
	return s
}
