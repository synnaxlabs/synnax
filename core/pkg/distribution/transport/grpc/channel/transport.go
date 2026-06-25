// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createClient = fgrpc.UnaryClient[
		channel.CreateMessage,
		*pb.CreateMessage,
		channel.CreateMessage,
		*pb.CreateMessage,
	]
	createServer = fgrpc.UnaryServer[
		channel.CreateMessage,
		*pb.CreateMessage,
		channel.CreateMessage,
		*pb.CreateMessage,
	]
	deleteClient = fgrpc.UnaryClient[
		channel.DeleteRequest,
		*pb.DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	deleteServer = fgrpc.UnaryServer[
		channel.DeleteRequest,
		*pb.DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	renameClient = fgrpc.UnaryClient[
		channel.RenameRequest,
		*pb.RenameRequest,
		types.Nil,
		*emptypb.Empty,
	]
	renameServer = fgrpc.UnaryServer[
		channel.RenameRequest,
		*pb.RenameRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

// Transport is a gRPC-backed implementation of the channel.Transport interface.
type Transport struct {
	// ReportProvider provides a report for the transport.
	alamos.ReportProvider
	createClient *createClient
	createServer *createServer
	deleteClient *deleteClient
	deleteServer *deleteServer
	renameClient *renameClient
	renameServer *renameServer
}

var (
	_ channel.Transport       = (*Transport)(nil)
	_ fgrpc.BindableTransport = (*Transport)(nil)
)

// New creates a new gRPC Transport that opens connections from the given pool.
func New(pool *fgrpc.Pool) Transport {
	return Transport{
		ReportProvider: fgrpc.Reporter,
		createClient: &createClient{
			Pool:               pool,
			RequestTranslator:  pb.CreateMessageTranslator{},
			ResponseTranslator: pb.CreateMessageTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *pb.CreateMessage,
			) (*pb.CreateMessage, error) {
				return pb.NewCreateServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &pb.CreateService_ServiceDesc,
		},
		createServer: &createServer{
			Internal:           true,
			RequestTranslator:  pb.CreateMessageTranslator{},
			ResponseTranslator: pb.CreateMessageTranslator{},
			ServiceDesc:        &pb.CreateService_ServiceDesc,
		},
		deleteClient: &deleteClient{
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
		deleteServer: &deleteServer{
			Internal:           true,
			RequestTranslator:  pb.DeleteRequestTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			ServiceDesc:        &pb.DeleteService_ServiceDesc,
		},
		renameClient: &renameClient{
			Pool:               pool,
			RequestTranslator:  pb.RenameMessageTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *pb.RenameRequest,
			) (*emptypb.Empty, error) {
				return pb.NewRenameServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &pb.RenameService_ServiceDesc,
		},
		renameServer: &renameServer{
			Internal:           true,
			RequestTranslator:  pb.RenameMessageTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			ServiceDesc:        &pb.RenameService_ServiceDesc,
		},
	}
}

// CreateClient implements the channel.Transport interface.
func (t Transport) CreateClient() channel.CreateClient { return t.createClient }

// CreateServer implements the channel.Transport interface.
func (t Transport) CreateServer() channel.CreateServer { return t.createServer }

// DeleteClient implements the channel.Transport interface.
func (t Transport) DeleteClient() channel.DeleteClient { return t.deleteClient }

// DeleteServer implements the channel.Transport interface.
func (t Transport) DeleteServer() channel.DeleteServer { return t.deleteServer }

// RenameClient implements the channel.Transport interface.
func (t Transport) RenameClient() channel.RenameClient { return t.renameClient }

// RenameServer implements the channel.Transport interface.
func (t Transport) RenameServer() channel.RenameServer { return t.renameServer }

// BindTo implements the fgrpc.BindableTransport interface.
func (t Transport) BindTo(reg grpc.ServiceRegistrar) {
	t.createServer.BindTo(reg)
	t.deleteServer.BindTo(reg)
	t.renameServer.BindTo(reg)
}

// Use implements the freighter.Transport interface.
func (t Transport) Use(middleware ...freighter.Middleware) {
	t.createClient.Use(middleware...)
	t.createServer.Use(middleware...)
	t.deleteClient.Use(middleware...)
	t.deleteServer.Use(middleware...)
	t.renameClient.Use(middleware...)
	t.renameServer.Use(middleware...)
}
