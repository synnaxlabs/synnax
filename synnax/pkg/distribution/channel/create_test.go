package channel_test

import (
	"github.com/arya-analytics/aspen"
	"github.com/arya-analytics/cesium"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/arya-analytics/x/telem"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Create", Ordered, func() {
	var (
		services map[core.NodeID]*channel.Service
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
	Context("Single channel", func() {
		var (
			channelLeaseNodeID aspen.NodeID
			ch                 channel.Channel
		)
		JustBeforeEach(func() {
			var err error
			ch, err = services[1].NewCreate().
				WithRate(5 * telem.Hz).
				WithDataType(telem.Float64).
				WithName("SG01").
				WithNodeID(channelLeaseNodeID).
				Exec(ctx)
			Expect(err).ToNot(HaveOccurred())
		})
		Context("Node is local", func() {
			BeforeEach(func() { channelLeaseNodeID = 1 })
			It("Should create the channel without error", func() {
				Expect(ch.Key().NodeID()).To(Equal(aspen.NodeID(1)))
				Expect(ch.Key().StorageKey()).To(Equal(cesium.ChannelKey(1)))
			})
			It("Should create the channel in the cesium gorpDB", func() {
				channels, err := builder.Cores[1].Storage.TS.RetrieveChannel(ch.Key().StorageKey())
				Expect(err).ToNot(HaveOccurred())
				Expect(channels).To(HaveLen(1))
				cesiumCH := channels[0]
				Expect(cesiumCH.Key).To(Equal(ch.Key().StorageKey()))
				Expect(cesiumCH.Density).To(Equal(telem.Bit64))
				Expect(cesiumCH.Rate).To(Equal(5 * telem.Hz))
			})
		})
		Context("Node is remote", func() {
			BeforeEach(func() { channelLeaseNodeID = 2 })
			It("Should create the channel without error", func() {
				Expect(ch.Key().NodeID()).To(Equal(aspen.NodeID(2)))
				Expect(ch.Key().StorageKey()).To(Equal(cesium.ChannelKey(1)))
			})
			It("Should create the channel in the cesium gorpDB", func() {
				channels, err := builder.Cores[2].Storage.TS.RetrieveChannel(ch.Key().StorageKey())
				Expect(err).ToNot(HaveOccurred())
				Expect(channels).To(HaveLen(1))
				cesiumCH := channels[0]
				Expect(cesiumCH.Key).To(Equal(ch.Key().StorageKey()))
				Expect(cesiumCH.Density).To(Equal(telem.Bit64))
				Expect(cesiumCH.Rate).To(Equal(5 * telem.Hz))
			})
			It("Should not create the channel on another node's ceisum gorpDB", func() {
				channels, err := builder.Cores[1].Storage.TS.RetrieveChannel(ch.Key().StorageKey())
				Expect(err).To(HaveOccurred())
				Expect(channels).To(HaveLen(0))
			})
			It("Should assign a sequential key to the channels on each node",
				func() {
					ch2, err := services[1].NewCreate().
						WithRate(5 * telem.Hz).
						WithDataType(telem.Float64).
						WithName("SG01").
						WithNodeID(1).
						Exec(ctx)
					Expect(err).To(BeNil())
					Expect(ch2.Key().NodeID()).To(Equal(aspen.NodeID(1)))
					Expect(ch2.Key().StorageKey()).To(Equal(cesium.ChannelKey(3)))
				})
		})
	})
})
