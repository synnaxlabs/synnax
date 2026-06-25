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
	channelpb "github.com/synnaxlabs/synnax/pkg/distribution/channel/pb"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	createClient = fgrpc.UnaryClient[
		channel.CreateMessage,
		*channelpb.CreateMessage,
		channel.CreateMessage,
		*channelpb.CreateMessage,
	]
	createServer = fgrpc.UnaryServer[
		channel.CreateMessage,
		*channelpb.CreateMessage,
		channel.CreateMessage,
		*channelpb.CreateMessage,
	]
	deleteClient = fgrpc.UnaryClient[
		channel.DeleteRequest,
		*channelpb.DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	deleteServer = fgrpc.UnaryServer[
		channel.DeleteRequest,
		*channelpb.DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	renameClient = fgrpc.UnaryClient[
		channel.RenameRequest,
		*channelpb.RenameRequest,
		types.Nil,
		*emptypb.Empty,
	]
	renameServer = fgrpc.UnaryServer[
		channel.RenameRequest,
		*channelpb.RenameRequest,
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
			RequestTranslator:  channelpb.CreateMessageTranslator{},
			ResponseTranslator: channelpb.CreateMessageTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *channelpb.CreateMessage,
			) (*channelpb.CreateMessage, error) {
				return channelpb.NewCreateServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &channelpb.CreateService_ServiceDesc,
		},
		createServer: &createServer{
			Internal:           true,
			RequestTranslator:  channelpb.CreateMessageTranslator{},
			ResponseTranslator: channelpb.CreateMessageTranslator{},
			ServiceDesc:        &channelpb.CreateService_ServiceDesc,
		},
		deleteClient: &deleteClient{
			Pool:               pool,
			RequestTranslator:  channelpb.DeleteRequestTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *channelpb.DeleteRequest,
			) (*emptypb.Empty, error) {
				return channelpb.NewDeleteServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &channelpb.DeleteService_ServiceDesc,
		},
		deleteServer: &deleteServer{
			Internal:           true,
			RequestTranslator:  channelpb.DeleteRequestTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			ServiceDesc:        &channelpb.DeleteService_ServiceDesc,
		},
		renameClient: &renameClient{
			Pool:               pool,
			RequestTranslator:  channelpb.RenameMessageTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *channelpb.RenameRequest,
			) (*emptypb.Empty, error) {
				return channelpb.NewRenameServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &channelpb.RenameService_ServiceDesc,
		},
		renameServer: &renameServer{
			Internal:           true,
			RequestTranslator:  channelpb.RenameMessageTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			ServiceDesc:        &channelpb.RenameService_ServiceDesc,
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
