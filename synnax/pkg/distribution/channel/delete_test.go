package channel_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Delete", Ordered, func() {
	var (
		services map[core.NodeKey]channel.Service
		builder  *mock.CoreBuilder
	)
	BeforeAll(func() { builder, services = provisionServices() })
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Context("Single Channel", func() {
		var ch channel.Channel
		JustBeforeEach(func() {
			var err error
			ch.Rate = 5 * telem.Hz
			ch.Name = "SG01"
			ch.DataType = telem.Float64T
			err = services[1].Create(ctx, &ch)
			Expect(err).ToNot(HaveOccurred())
		})
		Context("Node is local", func() {
			BeforeEach(func() { ch.Leaseholder = 1 })
			It("Should delete the channel without error", func() {
				Expect(services[1].Delete(ctx, ch.Key())).To(Succeed())
			})
			It("Should not be able to retrieve the channel after deletion", func() {
				Expect(services[1].Delete(ctx, ch.Key())).To(Succeed())
				exists, err := services[1].NewRetrieve().WhereKeys(ch.Key()).Exists(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())
			})
			It("Should not be able to retrieve the channel from the storage DB", func() {
				Expect(services[1].Delete(ctx, ch.Key())).To(Succeed())
				channels, err := builder.Cores[1].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())
				Expect(err).To(MatchError(cesium.ChannelNotFound))
				Expect(channels).To(BeEmpty())
			})
		})
		Context("Node is remote", func() {
			BeforeEach(func() { ch.Leaseholder = 2 })
			It("Should delete the channel without error", func() {
				Expect(services[2].Delete(ctx, ch.Key())).To(Succeed())
			})
			It("Should not be able to retrieve the channel after deletion", func() {
				Expect(services[2].Delete(ctx, ch.Key())).To(Succeed())
				exists, err := services[2].NewRetrieve().WhereKeys(ch.Key()).Exists(ctx, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(exists).To(BeFalse())
				Eventually(func(g Gomega) {
					exists, err = services[1].NewRetrieve().WhereKeys(ch.Key()).Exists(ctx, nil)
					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(exists).To(BeFalse())
				}).Should(Succeed())
			})
			It("Should not be able to retrieve the channel from the storage DB", func() {
				Expect(services[2].Delete(ctx, ch.Key())).To(Succeed())
				channels, err := builder.Cores[2].Storage.TS.RetrieveChannels(ctx, ch.Key().StorageKey())
				Expect(err).To(MatchError(cesium.ChannelNotFound))
				Expect(channels).To(BeEmpty())
			})
		})
	})
})
