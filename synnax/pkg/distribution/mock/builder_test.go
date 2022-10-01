package mock_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/mock"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Builder", func() {
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
				WithRate(25 * telem.Hz).
				WithNodeID(1).
				Exec(ctx)
			Expect(err).To(BeNil())
			Expect(ch.Key().NodeID()).To(Equal(distribution.NodeID(1)))

			Eventually(func(g Gomega) {
				var resCH channel.Channel
				g.Expect(coreThree.Channel.NewRetrieve().
					WhereKeys(ch.Key()).
					Entry(&resCH).
					Exec(ctx)).To(Succeed())

				g.Expect(resCH.Key()).To(Equal(ch.Key()))
			}).Should(Succeed())

		})
	})

})
