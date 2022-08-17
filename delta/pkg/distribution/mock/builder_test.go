package mock_test

import (
	"github.com/arya-analytics/delta/pkg/distribution"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/mock"
	"github.com/arya-analytics/x/telem"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("ProviderBuilder", func() {
	Describe("New", func() {
		It("Should open a three node memory backed distribution layer", func() {

			builder := mock.NewBuilder()

			coreOne := builder.New()
			coreTwo := builder.New()
			coreThree := builder.New()

			Expect(coreOne.Cluster.HostID()).To(Equal(core.NodeID(1)))
			Expect(coreTwo.Cluster.HostID()).To(Equal(core.NodeID(2)))
			Expect(coreThree.Cluster.HostID()).To(Equal(core.NodeID(3)))

			ch, err := coreOne.Channel.NewCreate().
				WithName("SG_01").
				WithDataType(telem.Float64).
				WithDataRate(25 * telem.Hz).
				WithNodeID(1).
				Exec(ctx)
			Expect(err).To(BeNil())
			Expect(ch.Key().NodeID()).To(Equal(distribution.NodeID(1)))

			time.Sleep(100 * time.Millisecond)

			var resCH channel.Channel

			Expect(coreThree.Channel.NewRetrieve().
				WhereKeys(ch.Key()).
				Entry(&resCH).
				Exec(ctx)).To(Succeed())

			Expect(resCH.Key()).To(Equal(ch.Key()))
		})
	})

})
