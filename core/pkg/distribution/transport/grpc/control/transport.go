// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package control

import (
	"context"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/freighter"
	fgrpc "github.com/synnaxlabs/freighter/grpc"
	"github.com/synnaxlabs/synnax/pkg/distribution/control"
	"github.com/synnaxlabs/synnax/pkg/distribution/control/pb"
	"google.golang.org/grpc"
)

type (
	retrieveClient = fgrpc.UnaryClient[
		control.RetrieveRequest,
		*pb.RetrieveRequest,
		control.RetrieveResponse,
		*pb.RetrieveResponse,
	]
	retrieveServer = fgrpc.UnaryServer[
		control.RetrieveRequest,
		*pb.RetrieveRequest,
		control.RetrieveResponse,
		*pb.RetrieveResponse,
	]
	subscribeClient = fgrpc.StreamClient[
		control.SubscribeRequest,
		*pb.SubscribeRequest,
		control.SubscribeResponse,
		*pb.SubscribeResponse,
	]
	subscribeServerCore = fgrpc.StreamServerCore[
		control.SubscribeRequest,
		*pb.SubscribeRequest,
		control.SubscribeResponse,
		*pb.SubscribeResponse,
	]
)

type subscribeServer struct{ subscribeServerCore }

func (s *subscribeServer) Exec(stream pb.SubscribeService_ExecServer) error {
	return s.Handler(stream.Context(), stream)
}

// Transport is a gRPC-backed implementation of the control.Transport interface.
type Transport struct {
	alamos.ReportProvider
	retrieveClient  *retrieveClient
	retrieveServer  *retrieveServer
	subscribeClient *subscribeClient
	subscribeServer *subscribeServer
}

var (
	_ control.Transport       = (*Transport)(nil)
	_ fgrpc.BindableTransport = (*Transport)(nil)
)

// New creates a new gRPC control Transport that opens connections from the given pool.
func New(pool *fgrpc.Pool) Transport {
	return Transport{
		ReportProvider: fgrpc.Reporter,
		retrieveClient: &retrieveClient{
			Pool:               pool,
			RequestTranslator:  pb.RetrieveRequestTranslator,
			ResponseTranslator: pb.RetrieveResponseTranslator,
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *pb.RetrieveRequest,
			) (*pb.RetrieveResponse, error) {
				return pb.NewRetrieveServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &pb.RetrieveService_ServiceDesc,
		},
		retrieveServer: &retrieveServer{
			Internal:           true,
			RequestTranslator:  pb.RetrieveRequestTranslator,
			ResponseTranslator: pb.RetrieveResponseTranslator,
			ServiceDesc:        &pb.RetrieveService_ServiceDesc,
		},
		subscribeClient: &subscribeClient{
			Pool:               pool,
			RequestTranslator:  pb.SubscribeRequestTranslator,
			ResponseTranslator: pb.SubscribeResponseTranslator,
			ClientFunc: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
			) (fgrpc.GRPCClientStream[*pb.SubscribeRequest, *pb.SubscribeResponse], error) {
				return pb.NewSubscribeServiceClient(conn).Exec(ctx)
			},
			ServiceDesc: &pb.SubscribeService_ServiceDesc,
		},
		subscribeServer: &subscribeServer{
			Internal:           true,
			RequestTranslator:  pb.SubscribeRequestTranslator,
			ResponseTranslator: pb.SubscribeResponseTranslator,
			ServiceDesc:        &pb.SubscribeService_ServiceDesc,
		},
	}
}

// RetrieveClient implements the control.Transport interface.
func (t Transport) RetrieveClient() control.RetrieveClient { return t.retrieveClient }

// RetrieveServer implements the control.Transport interface.
func (t Transport) RetrieveServer() control.RetrieveServer { return t.retrieveServer }

// SubscribeClient implements the control.Transport interface.
func (t Transport) SubscribeClient() control.SubscribeClient { return t.subscribeClient }

// SubscribeServer implements the control.Transport interface.
func (t Transport) SubscribeServer() control.SubscribeServer { return t.subscribeServer }

// BindTo implements the fgrpc.BindableTransport interface.
func (t Transport) BindTo(reg grpc.ServiceRegistrar) {
	t.retrieveServer.BindTo(reg)
	pb.RegisterSubscribeServiceServer(reg, t.subscribeServer)
}

// Use implements the freighter.Transport interface.
func (t Transport) Use(middleware ...freighter.Middleware) {
	t.retrieveClient.Use(middleware...)
	t.retrieveServer.Use(middleware...)
	t.subscribeClient.Use(middleware...)
	t.subscribeServer.Use(middleware...)
}
