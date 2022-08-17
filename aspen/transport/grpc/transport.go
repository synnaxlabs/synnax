package grpc

import (
	"context"
	"github.com/arya-analytics/aspen/internal/cluster/gossip"
	"github.com/arya-analytics/aspen/internal/cluster/pledge"
	"github.com/arya-analytics/aspen/internal/kv"
	"github.com/arya-analytics/aspen/internal/node"
	aspenv1 "github.com/arya-analytics/aspen/transport/grpc/gen/proto/go/v1"
	"github.com/arya-analytics/freighter/fgrpc"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/signal"
	"go/types"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
	"net"
)

type (
	pledgeTransport = fgrpc.UnaryTransport[
		node.ID,
		*aspenv1.ClusterPledge,
		node.ID,
		*aspenv1.ClusterPledge,
	]
	clusterGossipTransport = fgrpc.UnaryTransport[
		gossip.Message,
		*aspenv1.ClusterGossip,
		gossip.Message,
		*aspenv1.ClusterGossip,
	]
	batchTransport = fgrpc.UnaryTransport[
		kv.BatchRequest,
		*aspenv1.BatchRequest,
		kv.BatchRequest,
		*aspenv1.BatchRequest,
	]
	leaseTransport = fgrpc.UnaryTransport[
		kv.BatchRequest,
		*aspenv1.BatchRequest,
		types.Nil,
		*emptypb.Empty,
	]
	feedbackTransport = fgrpc.UnaryTransport[
		kv.FeedbackMessage,
		*aspenv1.FeedbackMessage,
		types.Nil,
		*emptypb.Empty,
	]
)

var (
	_ pledge.Transport                   = (*pledgeTransport)(nil)
	_ aspenv1.PledgeServiceServer        = (*pledgeTransport)(nil)
	_ gossip.Transport                   = (*clusterGossipTransport)(nil)
	_ aspenv1.ClusterGossipServiceServer = (*clusterGossipTransport)(nil)
	_ kv.BatchTransport                  = (*batchTransport)(nil)
	_ aspenv1.BatchServiceServer         = (*batchTransport)(nil)
	_ kv.LeaseTransport                  = (*leaseTransport)(nil)
	_ aspenv1.LeaseServiceServer         = (*leaseTransport)(nil)
	_ kv.FeedbackTransport               = (*feedbackTransport)(nil)
	_ aspenv1.FeedbackServiceServer      = (*feedbackTransport)(nil)
)

func New(pool *fgrpc.Pool) *transport {
	return &transport{
		pledge: &pledgeTransport{
			Pool:               pool,
			RequestTranslator:  pledgeTranslator{},
			ResponseTranslator: pledgeTranslator{},
			Client: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *aspenv1.ClusterPledge,
			) (*aspenv1.ClusterPledge, error) {
				return aspenv1.NewPledgeServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &aspenv1.PledgeService_ServiceDesc,
		},
		clusterGossip: &clusterGossipTransport{
			Pool:               pool,
			RequestTranslator:  clusterGossipTranslator{},
			ResponseTranslator: clusterGossipTranslator{},
			Client: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *aspenv1.ClusterGossip,
			) (*aspenv1.ClusterGossip, error) {
				return aspenv1.NewClusterGossipServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &aspenv1.ClusterGossipService_ServiceDesc,
		},
		batch: &batchTransport{
			Pool:               pool,
			RequestTranslator:  batchTranslator{},
			ResponseTranslator: batchTranslator{},
			Client: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *aspenv1.BatchRequest,
			) (*aspenv1.BatchRequest, error) {
				return aspenv1.NewBatchServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &aspenv1.BatchService_ServiceDesc,
		},
		lease: &leaseTransport{
			Pool:               pool,
			RequestTranslator:  batchTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			Client: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *aspenv1.BatchRequest,
			) (*emptypb.Empty, error) {
				return aspenv1.NewLeaseServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &aspenv1.LeaseService_ServiceDesc,
		},
		feedback: &feedbackTransport{
			Pool:               pool,
			RequestTranslator:  feedbackTranslator{},
			ResponseTranslator: fgrpc.EmptyTranslator{},
			Client: func(
				ctx context.Context,
				conn grpc.ClientConnInterface,
				req *aspenv1.FeedbackMessage,
			) (*emptypb.Empty, error) {
				return aspenv1.NewFeedbackServiceClient(conn).Exec(ctx, req)
			},
			ServiceDesc: &aspenv1.FeedbackService_ServiceDesc,
		},
	}
}

// transport implements the aspen.Transport interface.
type transport struct {
	pledge        *pledgeTransport
	clusterGossip *clusterGossipTransport
	batch         *batchTransport
	lease         *leaseTransport
	feedback      *feedbackTransport
}

func (t *transport) Pledge() pledge.Transport { return t.pledge }

func (t *transport) Cluster() gossip.Transport { return t.clusterGossip }

func (t *transport) Operations() kv.BatchTransport { return t.batch }

func (t *transport) Lease() kv.LeaseTransport { return t.lease }

func (t *transport) Feedback() kv.FeedbackTransport { return t.feedback }

func (t *transport) BindTo(reg grpc.ServiceRegistrar) {
	t.pledge.BindTo(reg)
	t.clusterGossip.BindTo(reg)
	t.batch.BindTo(reg)
	t.lease.BindTo(reg)
	t.feedback.BindTo(reg)
}

func (t *transport) Configure(ctx signal.Context, addr address.Address, external bool) error {
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
		go func() { err = server.Serve(lis) }()
		if err != nil {
			return err
		}
		defer server.Stop()
		<-ctx.Done()
		return ctx.Err()
	})
	return nil
}
