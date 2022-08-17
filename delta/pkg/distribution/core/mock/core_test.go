package mock_test

import (
	"github.com/arya-analytics/delta/pkg/distribution"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/core/mock"
	"github.com/arya-analytics/delta/pkg/storage"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Core", func() {
	DescribeTable("New", func(cfg ...distribution.Config) {
		builder := mock.NewCoreBuilder(cfg...)
		coreOne := builder.New()
		coreTwo := builder.New()
		coreThree := builder.New()

		Expect(coreOne.Cluster.HostID()).To(Equal(core.NodeID(1)))
		Expect(coreTwo.Cluster.HostID()).To(Equal(core.NodeID(2)))
		Expect(coreThree.Cluster.HostID()).To(Equal(core.NodeID(3)))

		Expect(coreOne.Storage.KV.Set([]byte("foo"), []byte("bar"))).To(Succeed())

		time.Sleep(100 * time.Millisecond)

		v, err := coreOne.Storage.KV.Get([]byte("foo"))
		Expect(err).To(Succeed())
		Expect(v).To(Equal([]byte("bar")))

		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	},
		Entry("Should open a three node memory backed distribution core"),
		Entry("Should open a three node file-system backed distribution core", distribution.Config{
			Storage: storage.Config{MemBacked: false, Dirname: "./tmp"},
		}),
	)
})
