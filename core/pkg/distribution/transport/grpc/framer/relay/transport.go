// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package relay

import (
	"context"

	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	framerpb "github.com/synnaxlabs/synnax/pkg/distribution/framer/pb"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"google.golang.org/grpc"
)

type (
	client = fgrpc.StreamClient[
		relay.Request,
		*framerpb.RelayRequest,
		relay.Response,
		*framerpb.RelayResponse,
	]
	serverCore = fgrpc.StreamServerCore[
		relay.Request,
		*framerpb.RelayRequest,
		relay.Response,
		*framerpb.RelayResponse,
	]
)

var (
	_ relay.Server                = (*server)(nil)
	_ relay.Client                = (*client)(nil)
	_ framerpb.RelayServiceServer = (*server)(nil)
	_ relay.Transport             = Transport{}
)

type server struct{ serverCore }

// Relay implements the framerpb.RelayServiceServer interface, dispatching the gRPC
// stream to the registered freighter handler.
func (s *server) Relay(stream framerpb.RelayService_RelayServer) error {
	return s.Handler(stream.Context(), stream)
}

// Transport is a gRPC-backed implementation of the relay.Transport interface.
type Transport struct {
	client *client
	server *server
}

// New creates a new gRPC relay Transport that opens connections from the given pool.
func New(pool *fgrpc.Pool) Transport {
	return Transport{
		client: &client{
			Pool:               pool,
			RequestTranslator:  framerpb.RelayRequestTranslator{},
			ResponseTranslator: framerpb.RelayResponseTranslator{},
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
			) (fgrpc.GRPCClientStream[*framerpb.RelayRequest, *framerpb.RelayResponse], error) {
				return framerpb.NewRelayServiceClient(conn).Relay(ctx)
			},
			ServiceDesc: &framerpb.RelayService_ServiceDesc,
		},
		server: &server{serverCore: serverCore{
			Internal:           true,
			RequestTranslator:  framerpb.RelayRequestTranslator{},
			ResponseTranslator: framerpb.RelayResponseTranslator{},
			ServiceDesc:        &framerpb.RelayService_ServiceDesc,
		}},
	}
}

// Client implements the relay.Transport interface.
func (t Transport) Client() relay.Client { return t.client }

// Server implements the relay.Transport interface.
func (t Transport) Server() relay.Server { return t.server }

// BindTo registers the transport's server with the given gRPC service registrar.
func (t Transport) BindTo(reg grpc.ServiceRegistrar) {
	framerpb.RegisterRelayServiceServer(reg, t.server)
}

// Use binds the given middleware to both the client and server endpoints.
func (t Transport) Use(middleware ...freighter.Middleware) {
	t.client.Use(middleware...)
	t.server.Use(middleware...)
}
