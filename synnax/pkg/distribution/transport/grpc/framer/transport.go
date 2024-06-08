// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/deleter"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/relay"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	tsv1 "github.com/synnaxlabs/synnax/pkg/distribution/transport/grpc/framer/v1"
	"go/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	writerClient = fgrpc.StreamClientCore[
		writer.Request,
		*tsv1.WriterRequest,
		writer.Response,
		*tsv1.WriterResponse,
	]
	writerServerCore = fgrpc.StreamServerCore[
		writer.Request,
		*tsv1.WriterRequest,
		writer.Response,
		*tsv1.WriterResponse,
	]
	iteratorClient = fgrpc.StreamClientCore[
		iterator.Request,
		*tsv1.IteratorRequest,
		iterator.Response,
		*tsv1.IteratorResponse,
	]
	iteratorServerCore = fgrpc.StreamServerCore[
		iterator.Request,
		*tsv1.IteratorRequest,
		iterator.Response,
		*tsv1.IteratorResponse,
	]
	relayClient = fgrpc.StreamClientCore[
		relay.Request,
		*tsv1.RelayRequest,
		relay.Response,
		*tsv1.RelayResponse,
	]
	relayServerCore = fgrpc.StreamServerCore[
		relay.Request,
		*tsv1.RelayRequest,
		relay.Response,
		*tsv1.RelayResponse,
	]
	deleterClient = fgrpc.UnaryClient[
		deleter.Request,
		*tsv1.DeleterRequest,
		types.Nil,
		*emptypb.Empty,
	]
	deleterServer = fgrpc.UnaryServer[
		deleter.Request,
		*tsv1.DeleterRequest,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ tsv1.WriterServiceServer   = (*writerServer)(nil)
	_ writer.TransportServer     = (*writerServer)(nil)
	_ writer.TransportClient     = (*writerClient)(nil)
	_ tsv1.IteratorServiceServer = (*iteratorServer)(nil)
	_ iterator.TransportServer   = (*iteratorServer)(nil)
	_ iterator.TransportClient   = (*iteratorClient)(nil)
	_ relay.TransportServer      = (*relayServer)(nil)
	_ relay.TransportClient      = (*relayClient)(nil)
	_ framer.Transport           = Transport{}
	_ fgrpc.BindableTransport    = Transport{}
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
				) (fgrpc.GRPCClientStream[*tsv1.WriterRequest, *tsv1.WriterResponse], error) {
					return tsv1.NewWriterServiceClient(conn).Write(ctx)
				},
			},
			server: &writerServer{
				writerServerCore: writerServerCore{
					RequestTranslator:  writerRequestTranslator{},
					ResponseTranslator: writerResponseTranslator{},
					ServiceDesc:        &tsv1.WriterService_ServiceDesc,
				}},
		},
		iterator: iteratorTransport{
			server: &iteratorServer{iteratorServerCore: iteratorServerCore{
				RequestTranslator:  iteratorRequestTranslator{},
				ResponseTranslator: iteratorResponseTranslator{},
				ServiceDesc:        &tsv1.IteratorService_ServiceDesc,
			}},
			client: &iteratorClient{
				Pool:               pool,
				RequestTranslator:  iteratorRequestTranslator{},
				ResponseTranslator: iteratorResponseTranslator{},
				ClientFunc: func(
					ctx context.Context,
					conn grpc.ClientConnInterface,
				) (fgrpc.GRPCClientStream[*tsv1.IteratorRequest, *tsv1.IteratorResponse], error) {
					return tsv1.NewIteratorServiceClient(conn).Iterate(ctx)
				},
			},
		},
		relay: relayTransport{
			server: &relayServer{relayServerCore: relayServerCore{
				RequestTranslator:  relayRequestTranslator{},
				ResponseTranslator: relayResponseTranslator{},
				ServiceDesc:        &tsv1.RelayService_ServiceDesc,
			}},
			client: &relayClient{
				Pool:               pool,
				RequestTranslator:  relayRequestTranslator{},
				ResponseTranslator: relayResponseTranslator{},
				ClientFunc: func(
					ctx context.Context,
					conn grpc.ClientConnInterface,
				) (fgrpc.GRPCClientStream[*tsv1.RelayRequest, *tsv1.RelayResponse], error) {
					return tsv1.NewRelayServiceClient(conn).Relay(ctx)
				},
			},
		},
		deleter: deleterTransport{
			server: &deleterServer{
				RequestTranslator:  deleterRequestTranslator{},
				ResponseTranslator: fgrpc.EmptyTranslator{},
				ServiceDesc:        &tsv1.DeleterService_ServiceDesc,
			},
			client: &deleterClient{
				Pool:               pool,
				RequestTranslator:  deleterRequestTranslator{},
				ResponseTranslator: fgrpc.EmptyTranslator{},
				ServiceDesc:        &tsv1.DeleterService_ServiceDesc,
			},
		},
	}
}

type writerServer struct{ writerServerCore }

func (w *writerServer) Write(server tsv1.WriterService_WriteServer) error {
	return w.Handler(server.Context(), server)
}

type iteratorServer struct{ iteratorServerCore }

func (t *iteratorServer) Iterate(server tsv1.IteratorService_IterateServer) error {
	return t.Handler(server.Context(), server)
}

// Transport is a grpc backed implementation of the framer.Transport interface.
type Transport struct {
	alamos.ReportProvider
	writer   writerTransport
	iterator iteratorTransport
	relay    relayTransport
	deleter  deleterTransport
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
	tsv1.RegisterWriterServiceServer(server, t.writer.server)
	tsv1.RegisterIteratorServiceServer(server, t.iterator.server)
}

func (t Transport) Use(middleware ...freighter.Middleware) {
	t.writer.client.Use(middleware...)
	t.iterator.client.Use(middleware...)
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

func (t *relayServer) Relay(server tsv1.RelayService_RelayServer) error {
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

type deleterTransport struct {
	client *deleterClient
	server *deleterServer
}

// Client implements the framer.Transport interface.
func (t deleterTransport) Client() deleter.TransportClient { return t.client }

// Server implements the framer.Transport interface.
func (t deleterTransport) Server() deleter.TransportServer { return t.server }
