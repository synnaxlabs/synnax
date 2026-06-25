// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package writer

import (
	"context"

	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	framerpb "github.com/synnaxlabs/synnax/pkg/distribution/framer/pb"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"google.golang.org/grpc"
)

type (
	client = fgrpc.StreamClient[
		writer.Request,
		*framerpb.WriterRequest,
		writer.Response,
		*framerpb.WriterResponse,
	]
	serverCore = fgrpc.StreamServerCore[
		writer.Request,
		*framerpb.WriterRequest,
		writer.Response,
		*framerpb.WriterResponse,
	]
)

var (
	_ writer.StreamServer          = (*server)(nil)
	_ writer.StreamClient          = (*client)(nil)
	_ framerpb.WriterServiceServer = (*server)(nil)
	_ writer.Transport             = Transport{}
)

type server struct{ serverCore }

// Write implements the framerpb.WriterServiceServer interface, dispatching the gRPC
// stream to the registered freighter handler.
func (s *server) Write(stream framerpb.WriterService_WriteServer) error {
	return s.Handler(stream.Context(), stream)
}

// Transport is a gRPC-backed implementation of the writer.Transport interface.
type Transport struct {
	client *client
	server *server
}

// New creates a new gRPC writer Transport that opens connections from the given pool.
func New(pool *fgrpc.Pool) Transport {
	return Transport{
		client: &client{
			Pool:               pool,
			RequestTranslator:  framerpb.WriterRequestTranslator{},
			ResponseTranslator: framerpb.WriterResponseTranslator{},
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
			) (fgrpc.GRPCClientStream[*framerpb.WriterRequest, *framerpb.WriterResponse], error) {
				return framerpb.NewWriterServiceClient(conn).Write(ctx)
			},
			ServiceDesc: &framerpb.WriterService_ServiceDesc,
		},
		server: &server{serverCore: serverCore{
			Internal:           true,
			RequestTranslator:  framerpb.WriterRequestTranslator{},
			ResponseTranslator: framerpb.WriterResponseTranslator{},
			ServiceDesc:        &framerpb.WriterService_ServiceDesc,
		}},
	}
}

// Client implements the writer.Transport interface.
func (t Transport) Client() writer.StreamClient { return t.client }

// Server implements the writer.Transport interface.
func (t Transport) Server() writer.StreamServer { return t.server }

// BindTo registers the transport's server with the given gRPC service registrar.
func (t Transport) BindTo(reg grpc.ServiceRegistrar) {
	framerpb.RegisterWriterServiceServer(reg, t.server)
}

// Use binds the given middleware to both the client and server endpoints.
func (t Transport) Use(middleware ...freighter.Middleware) {
	t.client.Use(middleware...)
	t.server.Use(middleware...)
}
