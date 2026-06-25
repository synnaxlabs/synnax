// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package deleter

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	framerpb "github.com/synnaxlabs/synnax/pkg/distribution/framer/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	client = fgrpc.UnaryClient[
		deleter.Request,
		*framerpb.DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	server = fgrpc.UnaryServer[
		deleter.Request,
		*framerpb.DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var _ deleter.Transport = Transport{}

// Transport is a gRPC-backed implementation of the deleter.Transport interface.
type Transport struct {
	client *client
	server *server
}

// New creates a new gRPC deleter Transport that opens connections from the given pool.
func New(pool *fgrpc.Pool) Transport {
	return Transport{
		server: &server{
			Internal:           true,
			RequestTranslator:  framerpb.DeleteRequestTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			ServiceDesc:        &framerpb.DeleteService_ServiceDesc,
		},
		client: &client{
			Pool:               pool,
			RequestTranslator:  framerpb.DeleteRequestTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *framerpb.DeleteRequest,
			) (*emptypb.Empty, error) {
				return framerpb.NewDeleteServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &framerpb.DeleteService_ServiceDesc,
		},
	}
}

// Client implements the deleter.Transport interface.
func (t Transport) Client() deleter.TransportClient { return t.client }

// Server implements the deleter.Transport interface.
func (t Transport) Server() deleter.TransportServer { return t.server }

// BindTo registers the transport's server with the given gRPC service registrar.
func (t Transport) BindTo(reg grpc.ServiceRegistrar) { t.server.BindTo(reg) }

// Use binds the given middleware to both the client and server endpoints.
func (t Transport) Use(middleware ...freighter.Middleware) {
	t.client.Use(middleware...)
	t.server.Use(middleware...)
}
