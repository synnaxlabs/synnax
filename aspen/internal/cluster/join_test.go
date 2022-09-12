package cluster_test

import (
	"context"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/node"
	"go.uber.org/zap"
	"time"
)

var _ = Describe("Join", func() {
	Context("Valid Configuration", func() {

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

		AfterEach(func() {
			shutdown()
			Expect(clusterCtx.Wait()).To(HaveOccurredAs(context.Canceled))
		})

		Context("New cluster", func() {

			It("Should correctly join the cluster", func() {

				By("Initializing the cluster correctly")
				gossipT1 := gossipNet.RouteUnary("")
				pledgeT1 := pledgeNet.RouteUnary(gossipT1.Address)
				clusterOne, err := cluster.Join(
					clusterCtx,
					cluster.Config{
						HostAddress: gossipT1.Address,
						Logger:      logger,
						Pledge: pledge.Config{
							Peers:     []address.Address{},
							Logger:    logger,
							Transport: pledgeT1,
						},
						Gossip: gossip.Config{
							Logger:    logger,
							Transport: gossipT1,
							Interval:  100 * time.Millisecond,
						},
						Experiment: alamos.New("cluster-join-test"),
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterOne.Host().ID).To(Equal(node.ID(1)))

				By("Pledging a new node to the cluster")
				gossipT2 := gossipNet.RouteUnary("")
				pledgeT2 := pledgeNet.RouteUnary(gossipT2.Address)
				clusterTwo, err := cluster.Join(
					clusterCtx,
					cluster.Config{
						HostAddress: gossipT2.Address,
						Logger:      logger,
						Pledge: pledge.Config{
							Peers:     []address.Address{gossipT1.Address},
							Logger:    logger,
							Transport: pledgeT2,
						},
						Gossip: gossip.Config{
							Logger:    logger,
							Transport: gossipT2,
							Interval:  100 * time.Millisecond,
						},
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterTwo.Host().ID).To(Equal(node.ID(2)))
				By("Converging cluster state through gossip")
				Eventually(clusterOne.Nodes).Should(HaveLen(2))
				Eventually(clusterTwo.Nodes).Should(HaveLen(2))
			})

		})

		Context("Existing Cluster in Storage", func() {

			It("Should restart cluster activities using the persisted state", func() {

				gossipT1 := gossipNet.RouteUnary("")
				pledgeT1 := pledgeNet.RouteUnary(gossipT1.Address)
				clusterOne, err := cluster.Join(
					clusterCtx,
					cluster.Config{
						HostAddress: gossipT1.Address,
						Logger:      logger,
						Pledge: pledge.Config{
							Peers:     []address.Address{},
							Logger:    logger,
							Transport: pledgeT1,
						},
						Gossip: gossip.Config{
							Logger:    logger,
							Transport: gossipT1,
							Interval:  100 * time.Millisecond,
						},
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterOne.Host().ID).To(Equal(node.ID(1)))

				sCtxTwo, cancelTwo := signal.TODO()
				kvDB := memkv.New()
				gossipT2 := gossipNet.RouteUnary("")
				pledgeT2 := pledgeNet.RouteUnary(gossipT2.Address)
				clusterTwo, err := cluster.Join(
					sCtxTwo,
					cluster.Config{
						HostAddress: gossipT2.Address,
						Logger:      logger,
						Pledge: pledge.Config{
							Peers:     []address.Address{gossipT1.Address},
							Logger:    logger,
							Transport: pledgeT2,
						},
						Gossip: gossip.Config{
							Logger:    logger,
							Transport: gossipT2,
							Interval:  100 * time.Millisecond,
						},
						StorageKey:     []byte("cluster-join-test-storage"),
						Storage:        kvDB,
						EncoderDecoder: &binary.MsgPackEncoderDecoder{},
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterTwo.Host().ID).To(Equal(node.ID(2)))
				cancelTwo()
				Expect(sCtxTwo.Wait()).To(HaveOccurredAs(context.Canceled))

				clusterTwoAgain, err := cluster.Join(
					clusterCtx,
					cluster.Config{
						HostAddress: gossipT2.Address,
						Logger:      logger,
						Pledge: pledge.Config{
							Logger:    logger,
							Transport: pledgeT2,
						},
						Gossip: gossip.Config{
							Logger:    logger,
							Transport: gossipT2,
							Interval:  100 * time.Millisecond,
						},
						Storage:              kvDB,
						StorageFlushInterval: cluster.FlushOnEvery,
						StorageKey:           []byte("cluster-join-test-storage"),
						EncoderDecoder:       &binary.MsgPackEncoderDecoder{},
					})
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterTwoAgain.Host().ID).To(Equal(node.ID(2)))
				Expect(clusterTwoAgain.Nodes()).To(HaveLen(2))

				shutdown()
				Expect(clusterCtx.Wait()).To(HaveOccurredAs(context.Canceled))
				Expect(kvDB.Close()).To(Succeed())

			})

		})

	})

	Context("Invalid Configuration", func() {
		It("Should return an error ", func() {
			cfg := cluster.Config{}
			ctx, cancel := signal.TODO()
			defer cancel()
			_, err := cluster.Join(ctx, cfg)
			Expect(err).To(HaveOccurred())
		})
	})
})
