package cluster_test

import (
	"context"
	"github.com/arya-analytics/aspen/internal/cluster"
	"github.com/arya-analytics/aspen/internal/cluster/clustermock"
	"github.com/arya-analytics/aspen/internal/cluster/gossip"
	"github.com/arya-analytics/x/address"
	"github.com/arya-analytics/x/signal"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"time"
)

var _ = Describe("Cluster", func() {
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
		})
	})

	AfterEach(func() {
		shutdown()
		Expect(errors.Is(clusterCtx.Err(), context.Canceled)).To(BeTrue())
	})

	Describe("Node", func() {

		It("Should return a node by its ID", func() {
			c1, err := builder.New(clusterCtx, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			c2, err := builder.New(clusterCtx, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			time.Sleep(10 * time.Millisecond)
			Expect(c2.Node(c1.HostID())).To(Equal(c1.Host()))
			Expect(c1.Node(c2.HostID())).To(Equal(c2.Host()))
		})

	})

	Describe("Resolve", func() {

		It("Should resolve the address of a node by its ID", func() {
			c1, err := builder.New(clusterCtx, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			c2, err := builder.New(clusterCtx, cluster.Config{})
			Expect(err).ToNot(HaveOccurred())
			time.Sleep(10 * time.Millisecond)
			Expect(c2.Resolve(c1.HostID())).To(Equal(address.Address("localhost:0")))
			Expect(c1.Resolve(c2.HostID())).To(Equal(address.Address("localhost:1")))
		})

	})

	Describe("Config", func() {

		It("Should return the cluster configuration", func() {
			c1, err := builder.New(clusterCtx, cluster.Config{StorageKey: []byte("crazy")})
			Expect(err).ToNot(HaveOccurred())
			Expect(c1.Config().StorageKey).To(Equal([]byte("crazy")))
		})

	})

})
