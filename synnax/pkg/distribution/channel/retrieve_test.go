package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/aspen"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

var _ = Describe("getAttributes", Ordered, func() {
	var (
		services map[aspen.NodeID]*channel.Service
		builder  *mock.CoreBuilder
		log      *zap.Logger
	)
	BeforeAll(func() {
		log = zap.NewNop()
		builder, services = provisionServices(log)
	})
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Describe("RetrieveP", func() {

		It("Should correctly retrieve a set of channels", func() {
			created, err := services[1].NewCreate().
				WithName("SG02").
				WithRate(25*telem.KHz).
				WithDataType(telem.Float32).
				WithNodeID(1).
				ExecN(ctx, 10)
			Expect(err).ToNot(HaveOccurred())

			var resChannels []channel.Channel

			err = services[1].
				NewRetrieve().
				WhereNodeID(1).
				Entries(&resChannels).
				Exec(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(len(created)))

			Eventually(func(g Gomega) {
				var resChannelsTwo []channel.Channel

				err = services[2].
					NewRetrieve().
					WhereNodeID(1).
					Entries(&resChannelsTwo).
					Exec(ctx)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(resChannelsTwo).To(HaveLen(len(created)))
			})

		})
		It("Should correctly retrieve a channel by its key", func() {
			created, err := services[1].NewCreate().
				WithName("SG02").
				WithRate(25*telem.KHz).
				WithDataType(telem.Float32).
				WithNodeID(1).
				ExecN(ctx, 10)
			Expect(err).ToNot(HaveOccurred())

			var resChannels []channel.Channel

			err = services[1].
				NewRetrieve().
				WhereKeys(created[0].Key()).
				Entries(&resChannels).
				Exec(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(resChannels).To(HaveLen(1))
			Expect(resChannels[0].Key()).To(Equal(created[0].Key()))
		})
	})
	Describe("Exists", func() {
		It("Should return true if a channel exists", func() {
			_, err := services[1].NewCreate().
				WithName("SG02").
				WithRate(25*telem.KHz).
				WithDataType(telem.Float32).
				WithNodeID(1).
				ExecN(ctx, 10)
			Expect(err).ToNot(HaveOccurred())

			key, err := channel.ParseKey("1-21")
			Expect(err).ToNot(HaveOccurred())

			exists, err := services[1].
				NewRetrieve().
				WhereKeys(key).
				Exists(ctx)
			Expect(err).ToNot(HaveOccurred())
			Expect(exists).To(BeTrue())
		})
	})
})
