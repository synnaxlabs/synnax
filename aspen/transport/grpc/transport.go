// Copyright 2023 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/alamos"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	aspenv1 "github.com/synnaxlabs/aspen/transport/grpc/gen/proto/go/v1"
	"github.com/synnaxlabs/freighter"
	"github.com/synnaxlabs/freighter/fgrpc"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
	"go/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"net"
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
	batchClient = fgrpc.UnaryClient[
		kv.TxRequest,
		*aspenv1.BatchRequest,
		kv.TxRequest,
		*aspenv1.BatchRequest,
	]
	batchServer = fgrpc.UnaryServer[
		kv.TxRequest,
		*aspenv1.BatchRequest,
		kv.TxRequest,
		*aspenv1.BatchRequest,
	]
	leaseClient = fgrpc.UnaryClient[
		kv.TxRequest,
		*aspenv1.BatchRequest,
		types.Nil,
		*emptypb.Empty,
	]
	leaseServer = fgrpc.UnaryServer[
		kv.TxRequest,
		*aspenv1.BatchRequest,
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
)

var (
	_ pledge.TransportServer             = (*pledgeServer)(nil)
	_ pledge.TransportClient             = (*pledgeClient)(nil)
	_ aspenv1.PledgeServiceServer        = (*pledgeServer)(nil)
	_ gossip.TransportClient             = (*clusterGossipClient)(nil)
	_ gossip.TransportServer             = (*clusterGossipServer)(nil)
	_ aspenv1.ClusterGossipServiceServer = (*clusterGossipServer)(nil)
	_ kv.BatchTransportClient            = (*batchClient)(nil)
	_ kv.BatchTransportServer            = (*batchServer)(nil)
	_ aspenv1.BatchServiceServer         = (*batchServer)(nil)
	_ kv.LeaseTransportClient            = (*leaseClient)(nil)
	_ kv.LeaseTransportServer            = (*leaseServer)(nil)
	_ aspenv1.LeaseServiceServer         = (*leaseServer)(nil)
	_ kv.FeedbackTransportClient         = (*feedbackClient)(nil)
	_ kv.FeedbackTransportServer         = (*feedbackServer)(nil)
	_ aspenv1.FeedbackServiceServer      = (*feedbackServer)(nil)
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
		batchClient: &batchClient{
			Pool:               pool,
			RequestTranslator:  batchTranslator{},
			ResponseTranslator: batchTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *aspenv1.BatchRequest,
			) (*aspenv1.BatchRequest, error) {
				return aspenv1.NewBatchServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &aspenv1.BatchService_ServiceDesc,
		},
		batchServer: &batchServer{
			Internal:           true,
			RequestTranslator:  batchTranslator{},
			ResponseTranslator: batchTranslator{},
			ServiceDesc:        &aspenv1.BatchService_ServiceDesc,
		},
		leaseClient: &leaseClient{
			Pool:               pool,
			RequestTranslator:  batchTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			Exec: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *aspenv1.BatchRequest,
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
	}
}

// Transport implements the aspen.Transport interface.
type Transport struct {
	pledgeServer   *pledgeServer
	pledgeClient   *pledgeClient
	gossipServer   *clusterGossipServer
	gossipClient   *clusterGossipClient
	batchServer    *batchServer
	batchClient    *batchClient
	leaseServer    *leaseServer
	leaseClient    *leaseClient
	feedbackServer *feedbackServer
	feedbackClient *feedbackClient
}

func (t Transport) PledgeServer() pledge.TransportServer { return t.pledgeServer }

func (t Transport) PledgeClient() pledge.TransportClient { return t.pledgeClient }

func (t Transport) GossipServer() gossip.TransportServer { return t.gossipServer }

func (t Transport) GossipClient() gossip.TransportClient { return t.gossipClient }

func (t Transport) BatchServer() kv.BatchTransportServer { return t.batchServer }

func (t Transport) BatchClient() kv.BatchTransportClient { return t.batchClient }

func (t Transport) LeaseServer() kv.LeaseTransportServer { return t.leaseServer }

func (t Transport) LeaseClient() kv.LeaseTransportClient { return t.leaseClient }

func (t Transport) FeedbackServer() kv.FeedbackTransportServer { return t.feedbackServer }

func (t Transport) FeedbackClient() kv.FeedbackTransportClient { return t.feedbackClient }

func (t Transport) BindTo(reg grpc.ServiceRegistrar) {
	t.pledgeServer.BindTo(reg)
	t.gossipServer.BindTo(reg)
	t.batchServer.BindTo(reg)
	t.leaseServer.BindTo(reg)
	t.feedbackServer.BindTo(reg)
}

func (t Transport) Use(middleware ...freighter.Middleware) {
	t.pledgeServer.Use(middleware...)
	t.pledgeClient.Use(middleware...)
	t.gossipServer.Use(middleware...)
	t.gossipClient.Use(middleware...)
	t.batchServer.Use(middleware...)
	t.batchClient.Use(middleware...)
	t.leaseServer.Use(middleware...)
	t.leaseClient.Use(middleware...)
	t.feedbackServer.Use(middleware...)
	t.feedbackClient.Use(middleware...)
}

func (t Transport) Report() alamos.Report {
	return t.pledgeServer.Report()
}

func (t Transport) Configure(ctx signal.Context, addr address.Address, external bool) error {
	if external {
		return nil
	}
	server := grpc.NewServer()
	t.BindTo(server)
	lis, err := net.Listen("tcp", addr.String())
	if err != nil {
		return err
	}
	ctx.Go(func(ctx context.Context) (err error) {
		go func() {
			err = server.Serve(lis)
		}()
		if err != nil {
			return err
		}
		defer server.Stop()
		<-ctx.Done()
		return ctx.Err()
	}, signal.CancelOnFail())
	return nil
}
