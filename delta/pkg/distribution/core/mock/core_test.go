package mock_test

import (
	"github.com/arya-analytics/delta/pkg/distribution"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/core/mock"
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/x/config"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Logging", func() {
	DescribeTable("New", func(cfg ...distribution.Config) {
		builder := mock.NewCoreBuilder(cfg...)
		coreOne := builder.New()
		coreTwo := builder.New()
		coreThree := builder.New()

		Expect(coreOne.Cluster.HostID()).To(Equal(core.NodeID(1)))
		Expect(coreTwo.Cluster.HostID()).To(Equal(core.NodeID(2)))
		Expect(coreThree.Cluster.HostID()).To(Equal(core.NodeID(3)))

		Expect(coreOne.Storage.KV.Set([]byte("foo"), []byte("bar"))).To(Succeed())

		Eventually(func(g Gomega) {
			v, err := coreOne.Storage.KV.Get([]byte("foo"))
			g.Expect(err).To(Succeed())
			g.Expect(v).To(Equal([]byte("bar")))
		}).Should(Succeed())

		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	},
		Entry("Should open a three node memory backed distribution core"),
		Entry("Should open a three node file-system backed distribution core", distribution.Config{
			Storage: storage.Config{MemBacked: config.BoolPointer(false), Dirname: "./tmp"},
		}),
	)
})
