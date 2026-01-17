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

	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/api"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

type connectivityResponseTranslator struct{}

type (
	connectivityServer = fgrpc.UnaryServer[
		types.Nil,
		*emptypb.Empty,
		api.ConnectivityCheckResponse,
		*gapi.ConnectivityCheckResponse,
	]
)

var _ fgrpc.Translator[api.ConnectivityCheckResponse, *gapi.ConnectivityCheckResponse] = (*connectivityResponseTranslator)(nil)

func (c connectivityResponseTranslator) Forward(
	ctx context.Context,
	r api.ConnectivityCheckResponse,
) (*gapi.ConnectivityCheckResponse, error) {
	return &gapi.ConnectivityCheckResponse{
		ClusterKey:  r.ClusterKey,
		NodeVersion: r.NodeVersion,
	}, nil
}

func (c connectivityResponseTranslator) Backward(
	ctx context.Context,
	r *gapi.ConnectivityCheckResponse,
) (api.ConnectivityCheckResponse, error) {
	return api.ConnectivityCheckResponse{
		ClusterKey:  r.ClusterKey,
		NodeVersion: r.NodeVersion,
	}, nil
}

func newConnectivity(a *api.Transport) fgrpc.BindableTransport {
	s := &connectivityServer{
		RequestTranslator:  fgrpc.EmptyTranslator{},
		ResponseTranslator: connectivityResponseTranslator{},
		ServiceDesc:        &gapi.ConnectivityService_ServiceDesc,
	}
	a.ConnectivityCheck = s
	return s
}
