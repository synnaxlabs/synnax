// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package framer

import (
	"context"
	"go/types"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	framerv1 "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	writerClient = fgrpc.StreamClient[
		writer.Request,
		*framerv1.WriterRequest,
		writer.Response,
		*framerv1.WriterResponse,
	]
	writerServerCore = fgrpc.StreamServerCore[
		writer.Request,
		*framerv1.WriterRequest,
		writer.Response,
		*framerv1.WriterResponse,
	]
	iteratorClient = fgrpc.StreamClient[
		iterator.Request,
		*framerv1.IteratorRequest,
		iterator.Response,
		*framerv1.IteratorResponse,
	]
	iteratorServerCore = fgrpc.StreamServerCore[
		iterator.Request,
		*framerv1.IteratorRequest,
		iterator.Response,
		*framerv1.IteratorResponse,
	]
	relayClient = fgrpc.StreamClient[
		relay.Request,
		*framerv1.RelayRequest,
		relay.Response,
		*framerv1.RelayResponse,
	]
	relayServerCore = fgrpc.StreamServerCore[
		relay.Request,
		*framerv1.RelayRequest,
		relay.Response,
		*framerv1.RelayResponse,
	]
	deleteClient = fgrpc.UnaryClient[
		deleter.Request,
		*framerv1.DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	deleteServer = fgrpc.UnaryServer[
		deleter.Request,
		*framerv1.DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ framerv1.WriterServiceServer   = (*writerServer)(nil)
	_ writer.TransportServer         = (*writerServer)(nil)
	_ writer.TransportClient         = (*writerClient)(nil)
	_ framerv1.IteratorServiceServer = (*iteratorServer)(nil)
	_ iterator.TransportServer       = (*iteratorServer)(nil)
	_ iterator.TransportClient       = (*iteratorClient)(nil)
	_ relay.TransportServer          = (*relayServer)(nil)
	_ relay.TransportClient          = (*relayClient)(nil)
	_ framer.Transport               = Transport{}
	_ fgrpc.BindableTransport        = Transport{}
)

// New creates a new grpc Transport that opens connections from the given pool.
func New(pool *fgrpc.Pool) Transport {
	return Transport{
		ReportProvider: fgrpc.Reporter,
		writer: writerTransport{
			client: &writerClient{
				Pool:               pool,
				RequestTranslator:  writerRequestTranslator{},
				ResponseTranslator: writerResponseTranslator{},
				ClientFunc: func(
					ctx context.Context,
					conn grpc.ClientConnInterface,
				) (fgrpc.GRPCClientStream[*framerv1.WriterRequest, *framerv1.WriterResponse], error) {
					return framerv1.NewWriterServiceClient(conn).Write(ctx)
				},
				ServiceDesc: &framerv1.WriterService_ServiceDesc,
			},
			server: &writerServer{
				writerServerCore: writerServerCore{
					RequestTranslator:  writerRequestTranslator{},
					ResponseTranslator: writerResponseTranslator{},
					ServiceDesc:        &framerv1.WriterService_ServiceDesc,
				}},
		},
		iterator: iteratorTransport{
			server: &iteratorServer{iteratorServerCore: iteratorServerCore{
				RequestTranslator:  iteratorRequestTranslator{},
				ResponseTranslator: iteratorResponseTranslator{},
				ServiceDesc:        &framerv1.IteratorService_ServiceDesc,
			}},
			client: &iteratorClient{
				Pool:               pool,
				RequestTranslator:  iteratorRequestTranslator{},
				ResponseTranslator: iteratorResponseTranslator{},
				ClientFunc: func(
					ctx context.Context,
					conn grpc.ClientConnInterface,
				) (fgrpc.GRPCClientStream[*framerv1.IteratorRequest, *framerv1.IteratorResponse], error) {
					return framerv1.NewIteratorServiceClient(conn).Iterate(ctx)
				},
				ServiceDesc: &framerv1.IteratorService_ServiceDesc,
			},
		},
		relay: relayTransport{
			server: &relayServer{relayServerCore: relayServerCore{
				RequestTranslator:  relayRequestTranslator{},
				ResponseTranslator: relayResponseTranslator{},
				ServiceDesc:        &framerv1.RelayService_ServiceDesc,
			}},
			client: &relayClient{
				Pool:               pool,
				RequestTranslator:  relayRequestTranslator{},
				ResponseTranslator: relayResponseTranslator{},
				ClientFunc: func(
					ctx context.Context,
					conn grpc.ClientConnInterface,
				) (fgrpc.GRPCClientStream[*framerv1.RelayRequest, *framerv1.RelayResponse], error) {
					return framerv1.NewRelayServiceClient(conn).Relay(ctx)
				},
				ServiceDesc: &framerv1.RelayService_ServiceDesc,
			},
		},
		deleter: deleteTransport{
			server: &deleteServer{
				RequestTranslator:  deleteRequestTranslator{},
				ResponseTranslator: fgrpc.EmptyTranslator{},
				ServiceDesc:        &framerv1.DeleteService_ServiceDesc,
			},
			client: &deleteClient{
				Pool:               pool,
				RequestTranslator:  deleteRequestTranslator{},
				ResponseTranslator: fgrpc.EmptyTranslator{},
				ServiceDesc:        &framerv1.DeleteService_ServiceDesc,
			},
		},
	}
}

type writerServer struct{ writerServerCore }

func (w *writerServer) Write(server framerv1.WriterService_WriteServer) error {
	return w.Handler(server.Context(), server)
}

type iteratorServer struct{ iteratorServerCore }

func (t *iteratorServer) Iterate(server framerv1.IteratorService_IterateServer) error {
	return t.Handler(server.Context(), server)
}

// Transport is a grpc backed implementation of the framer.Transport interface.
type Transport struct {
	alamos.ReportProvider
	writer   writerTransport
	iterator iteratorTransport
	relay    relayTransport
	deleter  deleteTransport
}

// Writer implements the framer.Transport interface.
func (t Transport) Writer() writer.Transport { return t.writer }

// Iterator implements the framer.Transport interface.
func (t Transport) Iterator() iterator.Transport { return t.iterator }

// Relay implements the framer.Transport interface.
func (t Transport) Relay() relay.Transport { return t.relay }

// Deleter implements the framer.Transport interface
func (t Transport) Deleter() deleter.Transport { return t.deleter }

// BindTo implements the fgrpc.BindableTransport interface.
func (t Transport) BindTo(server grpc.ServiceRegistrar) {
	framerv1.RegisterWriterServiceServer(server, t.writer.server)
	framerv1.RegisterIteratorServiceServer(server, t.iterator.server)
	framerv1.RegisterRelayServiceServer(server, t.relay.server)
}

func (t Transport) Use(middleware ...freighter.Middleware) {
	t.writer.client.Use(middleware...)
	t.iterator.client.Use(middleware...)
	t.relay.client.Use(middleware...)
}

type writerTransport struct {
	client *writerClient
	server *writerServer
}

// Client implements the writer.Transport interface.
func (t writerTransport) Client() writer.TransportClient { return t.client }

// Server implements the writer.Transport interface.
func (t writerTransport) Server() writer.TransportServer { return t.server }

type iteratorTransport struct {
	client *iteratorClient
	server *iteratorServer
}

// Client implements the iterator.Transport interface.
func (t iteratorTransport) Client() iterator.TransportClient { return t.client }

// Server implements the iterator.Transport interface.
func (t iteratorTransport) Server() iterator.TransportServer { return t.server }

type relayServer struct{ relayServerCore }

func (t *relayServer) Relay(server framerv1.RelayService_RelayServer) error {
	return t.Handler(server.Context(), server)
}

type relayTransport struct {
	client *relayClient
	server *relayServer
}

// Client implements the framer.Transport interface.
func (t relayTransport) Client() relay.TransportClient { return t.client }

// Server implements the framer.Transport interface.
func (t relayTransport) Server() relay.TransportServer { return t.server }

type deleteTransport struct {
	client *deleteClient
	server *deleteServer
}

// Client implements the framer.Transport interface.
func (t deleteTransport) Client() deleter.TransportClient { return t.client }

// Server implements the framer.Transport interface.
func (t deleteTransport) Server() deleter.TransportServer { return t.server }
