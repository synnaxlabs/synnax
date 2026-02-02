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
	framerpb "github.com/synnaxlabs/synnax/pkg/distribution/framer/pb"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	writerClient = fgrpc.StreamClient[
		writer.Request,
		*framerpb.WriterRequest,
		writer.Response,
		*framerpb.WriterResponse,
	]
	writerServerCore = fgrpc.StreamServerCore[
		writer.Request,
		*framerpb.WriterRequest,
		writer.Response,
		*framerpb.WriterResponse,
	]
	iteratorClient = fgrpc.StreamClient[
		iterator.Request,
		*framerpb.IteratorRequest,
		iterator.Response,
		*framerpb.IteratorResponse,
	]
	iteratorServerCore = fgrpc.StreamServerCore[
		iterator.Request,
		*framerpb.IteratorRequest,
		iterator.Response,
		*framerpb.IteratorResponse,
	]
	relayClient = fgrpc.StreamClient[
		relay.Request,
		*framerpb.RelayRequest,
		relay.Response,
		*framerpb.RelayResponse,
	]
	relayServerCore = fgrpc.StreamServerCore[
		relay.Request,
		*framerpb.RelayRequest,
		relay.Response,
		*framerpb.RelayResponse,
	]
	deleteClient = fgrpc.UnaryClient[
		deleter.Request,
		*framerpb.DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
	deleteServer = fgrpc.UnaryServer[
		deleter.Request,
		*framerpb.DeleteRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ framerpb.WriterServiceServer   = (*writerServer)(nil)
	_ writer.TransportServer         = (*writerServer)(nil)
	_ writer.TransportClient         = (*writerClient)(nil)
	_ framerpb.IteratorServiceServer = (*iteratorServer)(nil)
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
			server: &writerServer{
				writerServerCore: writerServerCore{
					RequestTranslator:  framerpb.WriterRequestTranslator{},
					ResponseTranslator: framerpb.WriterResponseTranslator{},
					ServiceDesc:        &framerpb.WriterService_ServiceDesc,
				}},
		},
		iterator: iteratorTransport{
			server: &iteratorServer{iteratorServerCore: iteratorServerCore{
				RequestTranslator:  framerpb.IteratorRequestTranslator{},
				ResponseTranslator: framerpb.IteratorResponseTranslator{},
				ServiceDesc:        &framerpb.IteratorService_ServiceDesc,
			}},
			client: &iteratorClient{
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
		},
		relay: relayTransport{
			server: &relayServer{relayServerCore: relayServerCore{
				RequestTranslator:  framerpb.RelayRequestTranslator{},
				ResponseTranslator: framerpb.RelayResponseTranslator{},
				ServiceDesc:        &framerpb.RelayService_ServiceDesc,
			}},
			client: &relayClient{
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
		},
		deleter: deleteTransport{
			server: &deleteServer{
				RequestTranslator:  framerpb.DeleteRequestTranslator{},
				ResponseTranslator: fgrpc.EmptyTranslator{},
				ServiceDesc:        &framerpb.DeleteService_ServiceDesc,
			},
			client: &deleteClient{
				Pool:               pool,
				RequestTranslator:  framerpb.DeleteRequestTranslator{},
				ResponseTranslator: fgrpc.EmptyTranslator{},
				ServiceDesc:        &framerpb.DeleteService_ServiceDesc,
			},
		},
	}
}

type writerServer struct{ writerServerCore }

func (w *writerServer) Write(server framerpb.WriterService_WriteServer) error {
	return w.Handler(server.Context(), server)
}

type iteratorServer struct{ iteratorServerCore }

func (t *iteratorServer) Iterate(server framerpb.IteratorService_IterateServer) error {
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
	framerpb.RegisterWriterServiceServer(server, t.writer.server)
	framerpb.RegisterIteratorServiceServer(server, t.iterator.server)
	framerpb.RegisterRelayServiceServer(server, t.relay.server)
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

func (t *relayServer) Relay(server framerpb.RelayService_RelayServer) error {
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
