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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/pb"
	"google.golang.org/grpc"
)

type (
	client = fgrpc.StreamClient[
		iterator.Request,
		*pb.IteratorRequest,
		iterator.Response,
		*pb.IteratorResponse,
	]
	serverCore = fgrpc.StreamServerCore[
		iterator.Request,
		*pb.IteratorRequest,
		iterator.Response,
		*pb.IteratorResponse,
	]
)

var (
	_ iterator.Server          = (*server)(nil)
	_ iterator.Client          = (*client)(nil)
	_ pb.IteratorServiceServer = (*server)(nil)
	_ iterator.Transport       = Transport{}
)

type server struct{ serverCore }

// Iterate implements the pb.IteratorServiceServer interface, dispatching the gRPC
// stream to the registered freighter handler.
func (s *server) Iterate(stream pb.IteratorService_IterateServer) error {
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
			RequestTranslator:  pb.IteratorRequestTranslator{},
			ResponseTranslator: pb.IteratorResponseTranslator{},
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
			) (fgrpc.GRPCClientStream[*pb.IteratorRequest, *pb.IteratorResponse], error) {
				return pb.NewIteratorServiceClient(conn).Iterate(ctx)
			},
			ServiceDesc: &pb.IteratorService_ServiceDesc,
		},
		server: &server{serverCore: serverCore{
			Internal:           true,
			RequestTranslator:  pb.IteratorRequestTranslator{},
			ResponseTranslator: pb.IteratorResponseTranslator{},
			ServiceDesc:        &pb.IteratorService_ServiceDesc,
		}},
	}
}

// Client implements the iterator.Transport interface.
func (t Transport) Client() iterator.Client { return t.client }

// Server implements the iterator.Transport interface.
func (t Transport) Server() iterator.Server { return t.server }

// BindTo registers the transport's server with the given gRPC service registrar.
func (t Transport) BindTo(reg grpc.ServiceRegistrar) {
	pb.RegisterIteratorServiceServer(reg, t.server)
}

// Use binds the given middleware to both the client and server endpoints.
func (t Transport) Use(middleware ...freighter.Middleware) {
	t.client.Use(middleware...)
	t.server.Use(middleware...)
}
