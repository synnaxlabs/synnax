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
	apiconnectivity "github.com/synnaxlabs/synnax/pkg/api/connectivity"
	gapi "github.com/synnaxlabs/synnax/pkg/api/grpc/v1"
	"google.golang.org/protobuf/types/known/emptypb"
)

type responseTranslator struct{}

type (
	server = fgrpc.UnaryServer[
		types.Nil,
		*emptypb.Empty,
		apiconnectivity.CheckResponse,
		*gapi.ConnectivityCheckResponse,
	]
)

var _ fgrpc.Translator[apiconnectivity.CheckResponse, *gapi.ConnectivityCheckResponse] = (*responseTranslator)(nil)

func (c responseTranslator) Forward(
	ctx context.Context,
	r apiconnectivity.CheckResponse,
) (*gapi.ConnectivityCheckResponse, error) {
	return &gapi.ConnectivityCheckResponse{
		ClusterKey:  r.ClusterKey,
		NodeVersion: r.NodeVersion,
	}, nil
}

func (c responseTranslator) Backward(
	ctx context.Context,
	r *gapi.ConnectivityCheckResponse,
) (apiconnectivity.CheckResponse, error) {
	return apiconnectivity.CheckResponse{
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
