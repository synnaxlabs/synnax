package cluster_test

import (
	"context"
	"github.com/arya-analytics/aspen/internal/cluster"
	"github.com/arya-analytics/aspen/internal/cluster/gossip"
	"github.com/arya-analytics/aspen/internal/cluster/pledge"
	"github.com/arya-analytics/aspen/internal/node"
	"github.com/arya-analytics/freighter/fmock"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/kv/memkv"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"time"
)

var _ = Describe("Join", func() {

	var (
		gossipNet  *fmock.Network[gossip.Message, gossip.Message]
		pledgeNet  *fmock.Network[node.ID, node.ID]
		logger     *zap.SugaredLogger
		clusterCtx signal.Context
		shutdown   context.CancelFunc
	)

	BeforeEach(func() {
		clusterCtx, shutdown = signal.WithCancel(ctx)
		gossipNet = fmock.NewNetwork[gossip.Message, gossip.Message]()
		pledgeNet = fmock.NewNetwork[node.ID, node.ID]()
		logger = zap.NewNop().Sugar()
	})

	Context("New cluster", func() {

		It("Should correctly join the cluster", func() {

			By("Initializing the cluster correctly")
			gossipT1 := gossipNet.RouteUnary("")
			pledgeT1 := pledgeNet.RouteUnary(gossipT1.Address)
			clusterOne, err := cluster.Join(
				clusterCtx,
				gossipT1.Address,
				[]address.Address{},
				cluster.Config{
					Logger: logger,
					Pledge: pledge.Config{
						Logger:    logger,
						Transport: pledgeT1,
					},
					Gossip: gossip.Config{
						Logger:    logger,
						Transport: gossipT1,
						Interval:  100 * time.Millisecond,
					},
					Storage: memkv.New(),
				},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(clusterOne.Host().ID).To(Equal(node.ID(1)))

			By("Pledging a new node to the cluster")
			gossipT2 := gossipNet.RouteUnary("")
			pledgeT2 := pledgeNet.RouteUnary(gossipT2.Address)
			clusterTwo, err := cluster.Join(
				clusterCtx,
				gossipT2.Address,
				[]address.Address{gossipT1.Address},
				cluster.Config{
					Logger: logger,
					Pledge: pledge.Config{
						Logger:    logger,
						Transport: pledgeT2,
					},
					Gossip: gossip.Config{
						Logger:    logger,
						Transport: gossipT2,
						Interval:  100 * time.Millisecond,
					},
					Storage: memkv.New(),
				},
			)
			Expect(err).ToNot(HaveOccurred())
			Expect(clusterTwo.Host().ID).To(Equal(node.ID(2)))

			By("Converging cluster state through gossip")
			time.Sleep(300 * time.Millisecond)
			shutdown()
			Expect(errors.Is(clusterCtx.Wait(), context.Canceled)).To(BeTrue())
			Expect(clusterOne.Nodes()).To(HaveLen(2))
			Expect(clusterTwo.Nodes()).To(HaveLen(2))
		})

	})
})
