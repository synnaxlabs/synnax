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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/pb"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"google.golang.org/grpc"
)

type (
	client = fgrpc.StreamClient[
		relay.Request,
		*pb.RelayRequest,
		relay.Response,
		*pb.RelayResponse,
	]
	serverCore = fgrpc.StreamServerCore[
		relay.Request,
		*pb.RelayRequest,
		relay.Response,
		*pb.RelayResponse,
	]
)

type server struct{ serverCore }

func (s *server) Relay(stream pb.RelayService_RelayServer) error {
	return s.Handler(stream.Context(), stream)
}

// Transport is a gRPC-backed implementation of the relay.Transport interface.
type Transport struct {
	client *client
	server *server
}

var _ relay.Transport = Transport{}

// New creates a new gRPC relay Transport that opens connections from the given pool.
func New(pool *fgrpc.Pool) Transport {
	return Transport{
		client: &client{
			Pool:               pool,
			RequestTranslator:  pb.RelayRequestTranslator{},
			ResponseTranslator: pb.RelayResponseTranslator{},
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
			) (fgrpc.GRPCClientStream[*pb.RelayRequest, *pb.RelayResponse], error) {
				return pb.NewRelayServiceClient(conn).Relay(ctx)
			},
			ServiceDesc: &pb.RelayService_ServiceDesc,
		},
		server: &server{serverCore: serverCore{
			Internal:           true,
			RequestTranslator:  pb.RelayRequestTranslator{},
			ResponseTranslator: pb.RelayResponseTranslator{},
			ServiceDesc:        &pb.RelayService_ServiceDesc,
		}},
	}
}

// Client implements the relay.Transport interface.
func (t Transport) Client() relay.Client { return t.client }

// Server implements the relay.Transport interface.
func (t Transport) Server() relay.Server { return t.server }

// BindTo registers the transport's server with the given gRPC service registrar.
func (t Transport) BindTo(reg grpc.ServiceRegistrar) {
	pb.RegisterRelayServiceServer(reg, t.server)
}

// Use binds the given middleware to both the client and server endpoints.
func (t Transport) Use(middleware ...freighter.Middleware) {
	t.client.Use(middleware...)
	t.server.Use(middleware...)
}
