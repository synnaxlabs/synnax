package writer_test

import (
	"context"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/testutil/seg"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	distribcore "github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/writer"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.uber.org/zap"
	"time"
)

var _ = Describe("Remote", Ordered, func() {
	var (
		log       *zap.Logger
		services  map[distribcore.NodeID]serviceContainer
		builder   *mock.CoreBuilder
		w         writer.Writer
		factory   seg.SequentialFactory
		wrapper   *core.StorageWrapper
		keys      channel.Keys
		newWriter func() (writer.Writer, error)
		channels  []channel.Channel
	)
	BeforeAll(func() {
		l := zap.NewNop()
		log = l
		builder, services = provisionNServices(3, log)
		dataFactory := &seg.RandomFloat64Factory{Cache: true}
		dr := 1 * telem.Hz
		store1Ch := channel.Channel{Name: "SG01", Rate: dr, DataType: telem.Float64T, NodeID: 1}
		Expect(services[1].channel.Create(&store1Ch)).To(Succeed())
		store2Ch := channel.Channel{Name: "SG02", Rate: dr, DataType: telem.Float64T, NodeID: 2}
		Expect(services[2].channel.Create(&store2Ch)).To(Succeed())
		var cesiumChannels []cesium.Channel
		channels = []channel.Channel{store1Ch, store2Ch}
		for _, c := range channels {
			cesiumChannels = append(cesiumChannels, c.Storage())
		}
		factory = seg.NewSequentialFactory(dataFactory, 10*telem.Second, cesiumChannels...)
		wrapper = &core.StorageWrapper{Host: 3}
		keys = channel.KeysFromChannels(channels)

		Eventually(func(g Gomega) {
			g.Expect(services[3].channel.NewRetrieve().WhereKeys(keys...).Exists(ctx)).To(BeTrue())
		}).Should(Succeed())

		newWriter = func() (writer.Writer, error) { return openWriter(3, services, builder, keys, log) }
	})
	BeforeEach(func() {
		routines := gleak.Goroutines()
		DeferCleanup(func() {
			Eventually(gleak.Goroutines).WithTimeout(time.Second).ShouldNot(gleak.HaveLeaked(routines))
		})
	})
	AfterAll(func() {
		Expect(builder.Close()).To(Succeed())
		Expect(builder.Cleanup()).To(Succeed())
	})
	Describe("Normal Operation", func() {
		BeforeEach(func() {
			var err error
			w, err = newWriter()
			Expect(err).ToNot(HaveOccurred())
		})
		Context("Behavioral Accuracy", func() {
			It("should write the segment to disk", func() {
				seg := wrapper.Wrap(factory.NextN(1))
				seg[0].ChannelKey = channels[0].Key()
				seg[1].ChannelKey = channels[1].Key()
				w.Write(seg)
				Expect(w.Close()).To(Succeed())
			})
		})
	})
	Describe("Err Handling", func() {
		Describe("Channel keys don't exist", func() {
			It("Should return an error", func() {
				_, err := writer.NewStream(
					ctx,
					writer.Config{
						TS:              builder.Cores[3].Storage.TS,
						ChannelService:  services[3].channel,
						HostResolver:    builder.Cores[3].Cluster,
						TransportServer: services[3].transport.writerServer,
						TransportClient: services[3].transport.writerClient,
						Keys:            channel.Keys{channel.NewKey(1, 5)},
						Logger:          log,
					},
				)
				Expect(err).To(HaveOccurredAs(query.NotFound))
			})
		})
		Describe("Context Cancellation", func() {
			It("Should immediately close the writerClient", func() {
				ctx, cancel := context.WithCancel(ctx)
				w, err := writer.New(
					ctx,
					writer.Config{
						TS:              builder.Cores[3].Storage.TS,
						ChannelService:  services[3].channel,
						HostResolver:    builder.Cores[3].Cluster,
						TransportServer: services[3].transport.writerServer,
						TransportClient: services[3].transport.writerClient,
						Keys:            keys,
						Logger:          log,
					},
				)
				Expect(err).ToNot(HaveOccurred())
				cancel()
				By("Exiting immediately")
				Expect(w.Close()).To(HaveOccurredAs(context.Canceled))
			})
		})
	})
})
