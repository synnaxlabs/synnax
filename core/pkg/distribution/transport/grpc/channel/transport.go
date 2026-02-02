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
	"github.com/synnaxlabs/freighter/fgrpc"
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

// Transport is a grpc backed implementation of the channel.Transport interface.
type Transport struct {
	alamos.ReportProvider
	createClient *createClient
	createServer *createServer
	deleteClient *deleteClient
	deleteServer *deleteServer
	renameClient *renameClient
	renameServer *renameServer
}

// CreateClient implements the channel.Transport interface.
func (t Transport) CreateClient() channel.CreateTransportClient { return t.createClient }

// CreateServer implements the channel.Transport interface.
func (t Transport) CreateServer() channel.CreateTransportServer { return t.createServer }

func (t Transport) DeleteClient() channel.DeleteTransportClient { return t.deleteClient }

func (t Transport) DeleteServer() channel.DeleteTransportServer { return t.deleteServer }

func (t Transport) RenameClient() channel.RenameTransportClient { return t.renameClient }

func (t Transport) RenameServer() channel.RenameTransportServer { return t.renameServer }

// BindTo implements the fgrpc.BindableTransport interface.
func (t Transport) BindTo(reg grpc.ServiceRegistrar) {
	t.createServer.BindTo(reg)
	t.deleteServer.BindTo(reg)
	t.renameServer.BindTo(reg)
}

var (
	_ channel.CreateTransportClient        = (*createClient)(nil)
	_ channel.CreateTransportServer        = (*createServer)(nil)
	_ channelpb.ChannelCreateServiceServer = (*createServer)(nil)
	_ channel.Transport                    = (*Transport)(nil)
	_ fgrpc.BindableTransport              = (*Transport)(nil)
)

// New creates a new grpc Transport that opens connections from the given pool.
func New(pool *fgrpc.Pool) Transport {
	createClient := &createClient{
		Pool:               pool,
		RequestTranslator:  channelpb.CreateMessageTranslator{},
		ResponseTranslator: channelpb.CreateMessageTranslator{},
		Exec: func(
			ctx context.Context,
			conn grpc.ClientConnInterface,
			req *channelpb.CreateMessage,
		) (*channelpb.CreateMessage, error) {
			return channelpb.NewChannelCreateServiceClient(conn).Exec(ctx, req)
		},
		ServiceDesc: &channelpb.ChannelCreateService_ServiceDesc,
	}
	createServer := &createServer{
		Internal:           true,
		RequestTranslator:  channelpb.CreateMessageTranslator{},
		ResponseTranslator: channelpb.CreateMessageTranslator{},
		ServiceDesc:        &channelpb.ChannelCreateService_ServiceDesc,
	}
	deleteClient := &deleteClient{
		Pool:               pool,
		RequestTranslator:  channelpb.DeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		Exec: func(
			ctx context.Context,
			conn grpc.ClientConnInterface,
			req *channelpb.DeleteRequest,
		) (*emptypb.Empty, error) {
			return channelpb.NewChannelDeleteServiceClient(conn).Exec(ctx, req)
		},
		ServiceDesc: &channelpb.ChannelDeleteService_ServiceDesc,
	}
	deleteServer := &deleteServer{
		RequestTranslator:  channelpb.DeleteRequestTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &channelpb.ChannelDeleteService_ServiceDesc,
	}
	renameClient := &renameClient{
		Pool:               pool,
		RequestTranslator:  channelpb.RenameMessageTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		Exec: func(
			ctx context.Context,
			conn grpc.ClientConnInterface,
			req *channelpb.RenameRequest,
		) (*emptypb.Empty, error) {
			return channelpb.NewChannelRenameServiceClient(conn).Exec(ctx, req)
		},
		ServiceDesc: &channelpb.ChannelRenameService_ServiceDesc,
	}
	renameServer := &renameServer{
		RequestTranslator:  channelpb.RenameMessageTranslator{},
		ResponseTranslator: fgrpc.EmptyTranslator{},
		ServiceDesc:        &channelpb.ChannelRenameService_ServiceDesc,
	}
	return Transport{
		ReportProvider: fgrpc.Reporter,
		createClient:   createClient,
		createServer:   createServer,
		deleteClient:   deleteClient,
		deleteServer:   deleteServer,
		renameClient:   renameClient,
		renameServer:   renameServer,
	}
}

func (t Transport) Use(middleware ...freighter.Middleware) {
	t.createClient.Use(middleware...)
	t.createServer.Use(middleware...)
	t.deleteClient.Use(middleware...)
	t.deleteServer.Use(middleware...)
}
