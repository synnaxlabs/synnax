package clustermock_test

import (
	"github.com/arya-analytics/x/signal"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen/internal/cluster"
	"github.com/synnaxlabs/aspen/internal/cluster/clustermock"
	"github.com/synnaxlabs/aspen/internal/cluster/gossip"
	"github.com/synnaxlabs/aspen/internal/node"
	"time"
)

var _ = Describe("Clustermock", func() {
	Describe("Builder", func() {
		It("Should provision a set of cluster ClusterAPIs correctly", func() {
			cfg := cluster.Config{Gossip: gossip.Config{Interval: 50 * time.Millisecond}}
			builder := clustermock.NewBuilder(cfg)
			ctx, cancel := signal.TODO()
			defer cancel()
			c1, err := builder.New(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(c1.HostID()).To(Equal(node.ID(1)))
			c2, err := builder.New(ctx, cfg)
			Expect(err).ToNot(HaveOccurred())
			Expect(c2.HostID()).To(Equal(node.ID(2)))
			Expect(c2.Nodes()).To(HaveLen(2))
		})
	})
})
