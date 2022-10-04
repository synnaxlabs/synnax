package kv_test

import (
	"context"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/kv"
	"github.com/synnaxlabs/aspen/internal/kv/kvmock"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/signal"
	"go.uber.org/zap"
	"time"
)

var _ = Describe("txn", func() {
	var (
		logger   *zap.SugaredLogger
		builder  *kvmock.Builder
		kvCtx    signal.Context
		shutdown context.CancelFunc
	)

	BeforeEach(func() {
		kvCtx, shutdown = signal.WithCancel(ctx)
		logger = zap.NewNop().Sugar()
		builder = kvmock.NewBuilder(
			kv.Config{
				Logger:            logger,
				RecoveryThreshold: 12,
				GossipInterval:    100 * time.Millisecond,
			},
			cluster.Config{
				Gossip: gossip.Config{Interval: 50 * time.Millisecond},
				Pledge: pledge.Config{RetryInterval: 50 * time.Millisecond},
			},
		)
	})

	AfterEach(func() {
		shutdown()
		Expect(errors.Is(kvCtx.Wait(), context.Canceled)).To(BeTrue())
	})

	Describe("StreamServer", func() {

		It("Should open a new txn storeSink without error", func() {
			kv, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv).ToNot(BeNil())
		})

	})

	Describe("Set", func() {

		Describe("Local Leaseholder", func() {

			It("Should commit the operation to storage", func() {
				kv, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				Expect(kv).ToNot(BeNil())
				Expect(kv.Set([]byte("key"), []byte("value"))).To(Succeed())
				v, err := kv.Get([]byte("key"))
				Expect(err).ToNot(HaveOccurred())
				Expect(v).To(Equal([]byte("value")))
			})

			It("Should propagate the operation to other members of the cluster",
				func() {
					kv1, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
					Expect(err).ToNot(HaveOccurred())
					kv2, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
					Expect(err).ToNot(HaveOccurred())
					Expect(kv1.Set([]byte("key"), []byte("value"))).To(Succeed())
					Eventually(func(g Gomega) {
						v, err := kv2.Get([]byte("key"))
						g.Expect(err).ToNot(HaveOccurred())
						g.Expect(v).To(Equal([]byte("value")))
					}).Should(Succeed())
				})

			It("Should forward an update to the Leaseholder", func() {
				kv1, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				kv2, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				Expect(kv1.Set([]byte("key"), []byte("value"))).To(Succeed())
				Eventually(func(g Gomega) {
					v, err := kv2.Get([]byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value")))
					g.Expect(kv2.Set([]byte("key"), []byte("value2"))).To(Succeed())
				}).Should(Succeed())
				Expect(func(g Gomega) {
					v, err := kv1.Get([]byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value2")))
					v, err = kv2.Get([]byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value2")))
				})
			})

			It("Should return an error when attempting to transfer the leaseAlloc",
				func() {
					kv1, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
					Expect(err).ToNot(HaveOccurred())
					_, err = builder.New(kvCtx, kv.Config{}, cluster.Config{})
					Expect(err).ToNot(HaveOccurred())
					Expect(kv1.Set([]byte("key"), []byte("value"))).To(Succeed())
					err = kv1.Set([]byte("key"), []byte("value2"), node.ID(2))
					Expect(err).To(HaveOccurred())
					Expect(errors.Is(err, kv.ErrLeaseNotTransferable)).To(BeTrue())
				})

		})

		Describe("Remote Leaseholder", func() {

			It("Should commit the operation to storage", func() {
				kv1, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				kv2, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				waitForClusterStateToConverge(builder)
				Expect(kv1.Set([]byte("key"), []byte("value"), node.ID(2))).To(Succeed())
				Eventually(func(g Gomega) {
					v, err := kv2.Get([]byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value")))
				}).Should(Succeed())
			})

			It("Should return an error if the lease option is not a node ID", func() {
				kv, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				Expect(kv.Set([]byte("key"), []byte("value"), "2")).To(HaveOccurred())
			})

		})

	})

	Describe("Batch", func() {
		It("Should execute a batch of operations", func() {
			kv, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv).ToNot(BeNil())
			b := kv.NewBatch()
			Expect(b.Set([]byte("key"), []byte("value"))).To(Succeed())
			Expect(b.Set([]byte("key2"), []byte("value2"))).To(Succeed())
			Expect(b.Commit()).To(Succeed())
			v, err := kv.Get([]byte("key"))
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal([]byte("value")))
			v, err = kv.Get([]byte("key2"))
			Expect(err).ToNot(HaveOccurred())
			Expect(v).To(Equal([]byte("value2")))
		})

	})

	Describe("Delete", func() {

		Describe("Local Leaseholder", func() {

			It("Should applyToAndCommit the operation to storage", func() {
				kv, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				Expect(kv).ToNot(BeNil())
				Expect(kv.Set([]byte("key"), []byte("value"))).To(Succeed())
				v, err := kv.Get([]byte("key"))
				Expect(err).ToNot(HaveOccurred())
				Expect(v).To(Equal([]byte("value")))
				Expect(kv.Delete([]byte("key"))).To(Succeed())
				v, err = kv.Get([]byte("key"))
				Expect(err).To(HaveOccurred())
				Expect(v).To(BeNil())
			})

		})

		Describe("Remote Leaseholder", func() {

			It("Should apply the operation to storage", func() {
				kv1, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				kv2, err := builder.New(kvCtx, kv.Config{}, cluster.Config{})
				Expect(err).ToNot(HaveOccurred())
				waitForClusterStateToConverge(builder)
				Expect(kv1.Set([]byte("key"), []byte("value"), node.ID(2))).To(Succeed())
				Eventually(func(g Gomega) {
					v, err := kv2.Get([]byte("key"))
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(v).To(Equal([]byte("value")))
				}).Should(Succeed())
			})

		})

	})

	Describe("Request Recovery", func() {

		It("Should stop propagating an operation after a set threshold of"+
			" redundant broadcasts", func() {
			kv1, err := builder.New(kvCtx, kv.Config{
				GossipInterval:    20 * time.Millisecond,
				RecoveryThreshold: 2,
			}, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			_, err = builder.New(kvCtx, kv.Config{
				GossipInterval:    20 * time.Millisecond,
				RecoveryThreshold: 2,
			}, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			Expect(kv1.Set([]byte("key"), []byte("value"))).To(Succeed())
			Eventually(func() int {
				return len(builder.OpNet.Entries)
			}).
				WithPolling(250 * time.Millisecond).
				WithTimeout(500 * time.Millisecond).
				Should(BeElementOf([]int{5, 7}))
		})

	})

})

func waitForClusterStateToConverge(builder *kvmock.Builder) {
	Eventually(func(g Gomega) {
		_, err := builder.ClusterAPIs[1].Resolve(2)
		g.Expect(err).ToNot(HaveOccurred())
	}).Should(Succeed())
}
