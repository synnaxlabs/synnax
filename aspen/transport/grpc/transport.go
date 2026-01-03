// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package grpc

import (
	"context"
	"go/types"
	"net"
	"time"

	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/aspen/transport"
	aspenv1 "github.com/synnaxlabs/aspen/transport/grpc/v1"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

type (
	pledgeClient = fgrpc.UnaryClient[
		pledge.Request,
		*aspenv1.ClusterPledge,
		pledge.Response,
		*aspenv1.ClusterPledge,
	]
	pledgeServer = fgrpc.UnaryServer[
		pledge.Request,
		*aspenv1.ClusterPledge,
		pledge.Response,
		*aspenv1.ClusterPledge,
	]
	clusterGossipClient = fgrpc.UnaryClient[
		gossip.Message,
		*aspenv1.ClusterGossip,
		gossip.Message,
		*aspenv1.ClusterGossip,
	]
	clusterGossipServer = fgrpc.UnaryServer[
		gossip.Message,
		*aspenv1.ClusterGossip,
		gossip.Message,
		*aspenv1.ClusterGossip,
	]
	txClient = fgrpc.UnaryClient[
		kv.TxRequest,
		*aspenv1.TxRequest,
		kv.TxRequest,
		*aspenv1.TxRequest,
	]
	txServer = fgrpc.UnaryServer[
		kv.TxRequest,
		*aspenv1.TxRequest,
		kv.TxRequest,
		*aspenv1.TxRequest,
	]
	leaseClient = fgrpc.UnaryClient[
		kv.TxRequest,
		*aspenv1.TxRequest,
		types.Nil,
		*emptypb.Empty,
	]
	leaseServer = fgrpc.UnaryServer[
		kv.TxRequest,
		*aspenv1.TxRequest,
		types.Nil,
		*emptypb.Empty,
	]
	feedbackClient = fgrpc.UnaryClient[
		kv.FeedbackMessage,
		*aspenv1.FeedbackMessage,
		types.Nil,
		*emptypb.Empty,
	]
	feedbackServer = fgrpc.UnaryServer[
		kv.FeedbackMessage,
		*aspenv1.FeedbackMessage,
		types.Nil,
		*emptypb.Empty,
	]
	recoveryClient = fgrpc.StreamClient[
		kv.RecoveryRequest,
		*aspenv1.RecoveryRequest,
		kv.RecoveryResponse,
		*aspenv1.RecoveryResponse,
	]
	recoveryServerCore = fgrpc.StreamServerCore[
		kv.RecoveryRequest,
		*aspenv1.RecoveryRequest,
		kv.RecoveryResponse,
		*aspenv1.RecoveryResponse,
	]
)

type recoveryServer struct {
	recoveryServerCore
}

func (w *recoveryServer) Exec(server aspenv1.RecoveryService_ExecServer) error {
	return w.Handler(server.Context(), server)
}

func (w *recoveryServer) BindTo(reg grpc.ServiceRegistrar) {
	aspenv1.RegisterRecoveryServiceServer(reg, w)
}

var (
	_ pledge.TransportServer             = (*pledgeServer)(nil)
	_ pledge.TransportClient             = (*pledgeClient)(nil)
	_ aspenv1.PledgeServiceServer        = (*pledgeServer)(nil)
	_ gossip.TransportClient             = (*clusterGossipClient)(nil)
	_ gossip.TransportServer             = (*clusterGossipServer)(nil)
	_ aspenv1.ClusterGossipServiceServer = (*clusterGossipServer)(nil)
	_ kv.TxTransportClient               = (*txClient)(nil)
	_ kv.TxTransportServer               = (*txServer)(nil)
	_ aspenv1.TxServiceServer            = (*txServer)(nil)
	_ kv.LeaseTransportClient            = (*leaseClient)(nil)
	_ kv.LeaseTransportServer            = (*leaseServer)(nil)
	_ aspenv1.LeaseServiceServer         = (*leaseServer)(nil)
	_ kv.FeedbackTransportClient         = (*feedbackClient)(nil)
	_ kv.FeedbackTransportServer         = (*feedbackServer)(nil)
	_ aspenv1.FeedbackServiceServer      = (*feedbackServer)(nil)
	_ kv.RecoveryTransportClient         = (*recoveryClient)(nil)
	_ kv.RecoveryTransportServer         = (*recoveryServerCore)(nil)
	_ aspenv1.RecoveryServiceServer      = (*recoveryServer)(nil)
	_ fgrpc.BindableTransport            = (*Transport)(nil)
	_ freighter.Transport                = (*Transport)(nil)
)

func New(pool *fgrpc.Pool) *Transport {
	return &Transport{
		pledgeClient: &pledgeClient{
			Pool:               pool,
			RequestTranslator:  pledgeTranslator{},
			ResponseTranslator: pledgeTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *aspenv1.ClusterPledge,
			) (*aspenv1.ClusterPledge, error) {
				return aspenv1.NewPledgeServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &aspenv1.PledgeService_ServiceDesc,
		},
		pledgeServer: &pledgeServer{
			Internal:           true,
			RequestTranslator:  pledgeTranslator{},
			ResponseTranslator: pledgeTranslator{},
			ServiceDesc:        &aspenv1.PledgeService_ServiceDesc,
		},
		gossipClient: &clusterGossipClient{
			Pool:               pool,
			RequestTranslator:  clusterGossipTranslator{},
			ResponseTranslator: clusterGossipTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *aspenv1.ClusterGossip,
			) (*aspenv1.ClusterGossip, error) {
				return aspenv1.NewClusterGossipServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &aspenv1.ClusterGossipService_ServiceDesc,
		},
		gossipServer: &clusterGossipServer{
			Internal:           true,
			RequestTranslator:  clusterGossipTranslator{},
			ResponseTranslator: clusterGossipTranslator{},
			ServiceDesc:        &aspenv1.ClusterGossipService_ServiceDesc,
		},
		txClient: &txClient{
			Pool:               pool,
			RequestTranslator:  batchTranslator{},
			ResponseTranslator: batchTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *aspenv1.TxRequest,
			) (*aspenv1.TxRequest, error) {
				return aspenv1.NewTxServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &aspenv1.TxService_ServiceDesc,
		},
		txServer: &txServer{
			Internal:           true,
			RequestTranslator:  batchTranslator{},
			ResponseTranslator: batchTranslator{},
			ServiceDesc:        &aspenv1.TxService_ServiceDesc,
		},
		leaseClient: &leaseClient{
			Pool:               pool,
			RequestTranslator:  batchTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *aspenv1.TxRequest,
			) (*emptypb.Empty, error) {
				return aspenv1.NewLeaseServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &aspenv1.LeaseService_ServiceDesc,
		},
		leaseServer: &leaseServer{
			Internal:           true,
			RequestTranslator:  batchTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			ServiceDesc:        &aspenv1.LeaseService_ServiceDesc,
		},
		feedbackClient: &feedbackClient{
			Pool:               pool,
			RequestTranslator:  feedbackTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *aspenv1.FeedbackMessage,
			) (*emptypb.Empty, error) {
				return aspenv1.NewFeedbackServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &aspenv1.FeedbackService_ServiceDesc,
		},
		feedbackServer: &feedbackServer{
			Internal:           true,
			RequestTranslator:  feedbackTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			ServiceDesc:        &aspenv1.FeedbackService_ServiceDesc,
		},
		recServer: &recoveryServer{
			recoveryServerCore: recoveryServerCore{
				RequestTranslator:  recoveryRequestTranslator{},
				ResponseTranslator: recoveryResponseTranslator{},
				ServiceDesc:        &aspenv1.RecoveryService_ServiceDesc,
			},
		},
		recClient: &recoveryClient{
			Pool:               pool,
			RequestTranslator:  recoveryRequestTranslator{},
			ResponseTranslator: recoveryResponseTranslator{},
			ClientFunc: func(ctx context.Context, connInterface grpc.ClientConnInterface) (fgrpc.GRPCClientStream[*aspenv1.RecoveryRequest, *aspenv1.RecoveryResponse], error) {
				return aspenv1.NewRecoveryServiceClient(connInterface).Exec(ctx)
			},
			ServiceDesc: &aspenv1.RecoveryService_ServiceDesc,
		},
	}
}

// Transport implements the aspen.Transport interface.
type Transport struct {
	pledgeServer   *pledgeServer
	pledgeClient   *pledgeClient
	gossipServer   *clusterGossipServer
	gossipClient   *clusterGossipClient
	txServer       *txServer
	txClient       *txClient
	leaseServer    *leaseServer
	leaseClient    *leaseClient
	feedbackServer *feedbackServer
	feedbackClient *feedbackClient
	recServer      *recoveryServer
	recClient      *recoveryClient
}

var _ transport.Transport = (*Transport)(nil)

func (t Transport) PledgeServer() pledge.TransportServer { return t.pledgeServer }

func (t Transport) PledgeClient() pledge.TransportClient { return t.pledgeClient }

func (t Transport) GossipServer() gossip.TransportServer { return t.gossipServer }

func (t Transport) GossipClient() gossip.TransportClient { return t.gossipClient }

func (t Transport) TxServer() kv.TxTransportServer { return t.txServer }

func (t Transport) TxClient() kv.TxTransportClient { return t.txClient }

func (t Transport) LeaseServer() kv.LeaseTransportServer { return t.leaseServer }

func (t Transport) LeaseClient() kv.LeaseTransportClient { return t.leaseClient }

func (t Transport) FeedbackServer() kv.FeedbackTransportServer { return t.feedbackServer }

func (t Transport) FeedbackClient() kv.FeedbackTransportClient { return t.feedbackClient }

func (t Transport) RecoveryServer() kv.RecoveryTransportServer { return t.recServer }

func (t Transport) RecoveryClient() kv.RecoveryTransportClient { return t.recClient }

func (t Transport) BindTo(reg grpc.ServiceRegistrar) {
	t.pledgeServer.BindTo(reg)
	t.gossipServer.BindTo(reg)
	t.txServer.BindTo(reg)
	t.leaseServer.BindTo(reg)
	t.feedbackServer.BindTo(reg)
	t.recServer.BindTo(reg)
}

func (t Transport) Use(middleware ...freighter.Middleware) {
	t.pledgeServer.Use(middleware...)
	t.pledgeClient.Use(middleware...)
	t.gossipServer.Use(middleware...)
	t.gossipClient.Use(middleware...)
	t.txServer.Use(middleware...)
	t.txClient.Use(middleware...)
	t.leaseServer.Use(middleware...)
	t.leaseClient.Use(middleware...)
	t.feedbackServer.Use(middleware...)
	t.feedbackClient.Use(middleware...)
	t.recServer.Use(middleware...)
	t.recClient.Use(middleware...)
}

func (t Transport) Report() alamos.Report {
	return t.pledgeServer.Report()
}

func (t Transport) Configure(sCtx signal.Context, addr address.Address, external bool) error {
	if external {
		return nil
	}
	server := grpc.NewServer()
	t.BindTo(server)
	lis, err := net.Listen("tcp", addr.String())
	if err != nil {
		return err
	}
	sCtx.Go(func(ctx context.Context) (err error) {
		go func() {
			err = server.Serve(lis)
		}()
		if err != nil {
			return err
		}
		defer server.Stop()
		<-ctx.Done()
		return ctx.Err()
	},
		signal.CancelOnFail(),
		signal.WithRetryOnPanic(),
		signal.WithBaseRetryInterval(200*time.Millisecond),
		signal.WithRetryScale(1.05),
	)
	return nil
}
