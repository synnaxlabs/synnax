// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package iterator

import (
	"context"

	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	framerpb "github.com/synnaxlabs/synnax/pkg/distribution/framer/pb"
	"google.golang.org/grpc"
)

type (
	client = fgrpc.StreamClient[
		iterator.Request,
		*framerpb.IteratorRequest,
		iterator.Response,
		*framerpb.IteratorResponse,
	]
	serverCore = fgrpc.StreamServerCore[
		iterator.Request,
		*framerpb.IteratorRequest,
		iterator.Response,
		*framerpb.IteratorResponse,
	]
)

var (
	_ iterator.StreamServer          = (*server)(nil)
	_ iterator.StreamClient          = (*client)(nil)
	_ framerpb.IteratorServiceServer = (*server)(nil)
	_ iterator.Transport             = Transport{}
)

type server struct{ serverCore }

// Iterate implements the framerpb.IteratorServiceServer interface, dispatching the gRPC
// stream to the registered freighter handler.
func (s *server) Iterate(stream framerpb.IteratorService_IterateServer) error {
	return s.Handler(stream.Context(), stream)
}

// Transport is a gRPC-backed implementation of the iterator.Transport interface.
type Transport struct {
	client *client
	server *server
}

// New creates a new gRPC iterator Transport that opens connections from the given pool.
func New(pool *fgrpc.Pool) Transport {
	return Transport{
		client: &client{
			Pool:               pool,
			RequestTranslator:  framerpb.IteratorRequestTranslator{},
			ResponseTranslator: framerpb.IteratorResponseTranslator{},
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
			) (fgrpc.GRPCClientStream[*framerpb.IteratorRequest, *framerpb.IteratorResponse], error) {
				return framerpb.NewIteratorServiceClient(conn).Iterate(ctx)
			},
			ServiceDesc: &framerpb.IteratorService_ServiceDesc,
		},
		server: &server{serverCore: serverCore{
			Internal:           true,
			RequestTranslator:  framerpb.IteratorRequestTranslator{},
			ResponseTranslator: framerpb.IteratorResponseTranslator{},
			ServiceDesc:        &framerpb.IteratorService_ServiceDesc,
		}},
	}
}

// Client implements the iterator.Transport interface.
func (t Transport) Client() iterator.StreamClient { return t.client }

// Server implements the iterator.Transport interface.
func (t Transport) Server() iterator.StreamServer { return t.server }

// BindTo registers the transport's server with the given gRPC service registrar.
func (t Transport) BindTo(reg grpc.ServiceRegistrar) {
	framerpb.RegisterIteratorServiceServer(reg, t.server)
}

// Use binds the given middleware to both the client and server endpoints.
func (t Transport) Use(middleware ...freighter.Middleware) {
	t.client.Use(middleware...)
	t.server.Use(middleware...)
}
