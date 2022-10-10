package cluster_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/freighter/fmock"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/alamos"
	"github.com/synnaxlabs/x/binary"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
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
				gossipT1 := gossipNet.UnaryServer("")
				pledgeT1 := pledgeNet.UnaryServer(gossipT1.Address)
				clusterOne, err := cluster.Join(
					clusterCtx,
					cluster.Config{
						HostAddress: gossipT1.Address,
						Logger:      logger,
						Pledge: pledge.Config{
							Peers:           []address.Address{},
							Logger:          logger,
							TransportClient: pledgeNet.UnaryClient(),
							TransportServer: pledgeT1,
						},
						Gossip: gossip.Config{
							Logger:          logger,
							TransportClient: gossipNet.UnaryClient(),
							TransportServer: gossipT1,
							Interval:        100 * time.Millisecond,
						},
						Experiment: alamos.New("cluster-join-test"),
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterOne.Host().ID).To(Equal(node.ID(1)))

				By("Pledging a new node to the cluster")
				gossipT2 := gossipNet.UnaryServer("")
				pledgeT2 := pledgeNet.UnaryServer(gossipT2.Address)
				clusterTwo, err := cluster.Join(
					clusterCtx,
					cluster.Config{
						HostAddress: gossipT2.Address,
						Logger:      logger,
						Pledge: pledge.Config{
							Peers:           []address.Address{gossipT1.Address},
							Logger:          logger,
							TransportServer: pledgeT2,
							TransportClient: pledgeNet.UnaryClient(),
						},
						Gossip: gossip.Config{
							Logger:          logger,
							TransportServer: gossipT2,
							TransportClient: gossipNet.UnaryClient(),
							Interval:        100 * time.Millisecond,
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

				gossipT1 := gossipNet.UnaryServer("")
				pledgeT1 := pledgeNet.UnaryServer(gossipT1.Address)
				clusterOne, err := cluster.Join(
					clusterCtx,
					cluster.Config{
						HostAddress: gossipT1.Address,
						Logger:      logger,
						Pledge: pledge.Config{
							Peers:           []address.Address{},
							Logger:          logger,
							TransportClient: pledgeNet.UnaryClient(),
							TransportServer: pledgeT1,
						},
						Gossip: gossip.Config{
							Logger:          logger,
							TransportClient: gossipNet.UnaryClient(),
							TransportServer: gossipT1,
							Interval:        100 * time.Millisecond,
						},
					},
				)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterOne.Host().ID).To(Equal(node.ID(1)))

				sCtxTwo, cancelTwo := signal.TODO()
				kvDB := memkv.New()
				gossipT2 := gossipNet.UnaryServer("")
				pledgeT2 := pledgeNet.UnaryServer(gossipT2.Address)

				clusterTwoConfig := cluster.Config{
					HostAddress: gossipT2.Address,
					Logger:      logger,
					Pledge: pledge.Config{
						Peers:           []address.Address{gossipT1.Address},
						Logger:          logger,
						TransportClient: pledgeNet.UnaryClient(),
						TransportServer: pledgeT2,
					},
					Gossip: gossip.Config{
						Logger:          logger,
						TransportClient: gossipNet.UnaryClient(),
						TransportServer: gossipT2,
						Interval:        100 * time.Millisecond,
					},
					StorageKey:           []byte("cluster-join-test-storage"),
					Storage:              kvDB,
					StorageFlushInterval: cluster.FlushOnEvery,
					EncoderDecoder:       &binary.MsgPackEncoderDecoder{},
				}
				clusterTwo, err := cluster.Join(sCtxTwo, clusterTwoConfig)
				Expect(err).ToNot(HaveOccurred())
				Expect(clusterTwo.Host().ID).To(Equal(node.ID(2)))
				cancelTwo()
				Expect(sCtxTwo.Wait()).To(HaveOccurredAs(context.Canceled))

				clusterTwoAgain, err := cluster.Join(clusterCtx, clusterTwoConfig)
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
