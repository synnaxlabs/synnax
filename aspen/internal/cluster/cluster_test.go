package cluster_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/clustermock"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/cluster/pledge"
	"github.com/synnaxlabs/aspen/internal/node"
	"github.com/synnaxlabs/x/address"
	"github.com/synnaxlabs/x/signal"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"time"
)

var _ = Describe("cluster", func() {
	var (
		builder    *clustermock.Builder
		clusterCtx signal.Context
		shutdown   context.CancelFunc
		log        *zap.Logger
	)
	BeforeEach(func() {
		log = zap.NewNop()
		clusterCtx, shutdown = signal.WithCancel(ctx)
		builder = clustermock.NewBuilder(cluster.Config{
			Gossip: gossip.Config{Interval: 5 * time.Millisecond},
			Logger: log.Sugar(),
			Pledge: pledge.Config{RetryInterval: 1 * time.Millisecond},
		})
	})

	AfterEach(func() {
		shutdown()
		Expect(clusterCtx.Err()).To(HaveOccurredAs(context.Canceled))
	})

	Describe("Node", func() {

		It("Should return a node by its ID", func() {
			c1, err := builder.New(clusterCtx, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			c2, err := builder.New(clusterCtx, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() node.ID {
				n, _ := c2.Node(c1.HostID())
				return n.ID
			}).Should(Equal(c1.HostID()))
			Eventually(func() node.ID {
				n, _ := c1.Node(c2.HostID())
				return n.ID
			}).Should(Equal(c2.HostID()))
		})

	})

	Describe("Resolve", func() {

		It("Should resolve the address of a node by its ID", func() {
			c1, err := builder.New(clusterCtx, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			c2, err := builder.New(clusterCtx, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			Eventually(func() address.Address {
				addr, _ := c1.Resolve(c2.HostID())
				return addr
			}).Should(Equal(address.Address("localhost:1")))
			Eventually(func() address.Address {
				addr, _ := c2.Resolve(c1.HostID())
				return addr
			}).Should(Equal(address.Address("localhost:0")))
		})

	})

})
