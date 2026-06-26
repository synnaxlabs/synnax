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
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	client = fgrpc.UnaryClient[
		deleter.Request,
		*pb.DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	server = fgrpc.UnaryServer[
		deleter.Request,
		*pb.DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

// Transport is a gRPC-backed implementation of the deleter.Transport interface.
type Transport struct {
	client *client
	server *server
}

var _ deleter.Transport = Transport{}

// New creates a new gRPC deleter Transport that opens connections from the given pool.
func New(pool *fgrpc.Pool) Transport {
	return Transport{
		server: &server{
			Internal:           true,
			RequestTranslator:  pb.DeleteRequestTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			ServiceDesc:        &pb.DeleteService_ServiceDesc,
		},
		client: &client{
			Pool:               pool,
			RequestTranslator:  pb.DeleteRequestTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *pb.DeleteRequest,
			) (*emptypb.Empty, error) {
				return pb.NewDeleteServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &pb.DeleteService_ServiceDesc,
		},
	}
}

// Client implements the deleter.Transport interface.
func (t Transport) Client() deleter.Client { return t.client }

// Server implements the deleter.Transport interface.
func (t Transport) Server() deleter.Server { return t.server }

// BindTo registers the transport's server with the given gRPC service registrar.
func (t Transport) BindTo(reg grpc.ServiceRegistrar) { t.server.BindTo(reg) }

// Use binds the given middleware to both the client and server endpoints.
func (t Transport) Use(middleware ...freighter.Middleware) {
	t.client.Use(middleware...)
	t.server.Use(middleware...)
}
