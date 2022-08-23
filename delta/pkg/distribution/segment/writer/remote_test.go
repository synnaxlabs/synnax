package writer_test

import (
	"context"
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/cesium/testutil/seg"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	distribcore "github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/core/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/core"
	"github.com/arya-analytics/delta/pkg/distribution/segment/writer"
	"github.com/arya-analytics/delta/pkg/storage"
	"github.com/arya-analytics/x/query"
	"github.com/arya-analytics/x/telem"
	. "github.com/arya-analytics/x/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gleak"
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
		store1Channels, err := services[1].channel.NewCreate().
			WithName("SG02").
			WithRate(dr).
			WithDensity(telem.Float64).
			WithNodeID(1).
			ExecN(ctx, 1)
		Expect(err).ToNot(HaveOccurred())
		channels = append(channels, store1Channels...)
		store2Channels, err := services[2].channel.NewCreate().
			WithName("SG02").
			WithRate(dr).
			WithDensity(telem.Float64).
			WithNodeID(2).
			ExecN(ctx, 1)
		Expect(err).ToNot(HaveOccurred())
		channels = append(channels, store2Channels...)
		var cesiumChannels []cesium.Channel
		for _, c := range channels {
			cesiumChannels = append(cesiumChannels, c.Channel)
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
				w.Requests() <- writer.Request{Segments: seg}
				close(w.Requests())
				for res := range w.Responses() {
					Expect(res.Error).ToNot(HaveOccurred())
				}
				Expect(w.Close()).To(Succeed())
			})
		})
	})
	Describe("Error Handling", func() {
		Describe("channel keys don't exist", func() {
			It("Should return an error", func() {
				_, err := writer.New(
					ctx,
					builder.Cores[3].Storage.TS,
					services[3].channel,
					builder.Cores[3].Cluster,
					services[3].transport.writer,
					channel.Keys{channel.NewKey(1, 5)},
					log,
				)
				Expect(err).To(HaveOccurredAs(query.NotFound))
			})
		})
		Describe("Context Cancellation", func() {
			It("Should immediately close the writer", func() {
				ctx, cancel := context.WithCancel(ctx)
				w, err := writer.New(
					ctx,
					builder.Cores[3].Storage.TS,
					services[3].channel,
					builder.Cores[3].Cluster,
					services[3].transport.writer,
					keys,
					log,
				)
				Expect(err).ToNot(HaveOccurred())
				cancel()
				By("Exiting immediately")
				Expect(w.Close()).To(HaveOccurredAs(context.Canceled))
				By("Keeping the request channel open")
				close(w.Requests())
			})
		})
		Describe("Writing to an unspecified channel", func() {
			Describe("Node not in the cluster", func() {
				It("Should return a query error", func() {
					w, err := newWriter()
					Expect(err).ToNot(HaveOccurred())
					w.Requests() <- writer.Request{Segments: []core.Segment{
						{
							ChannelKey: channel.NewKey(5, 5),
							Segment: storage.Segment{
								Start: 0,
								Data:  []byte{1, 2, 3, 4, 5},
							},
						},
					}}
					close(w.Requests())
					res, ok := <-w.Responses()
					Expect(ok).To(BeTrue())
					Expect(res.Error).To(HaveOccurredAs(query.NotFound))
					_, ok = <-w.Responses()
					Expect(ok).To(BeFalse())
					Expect(w.Close()).To(Succeed())
				})
			})
		})
	})
})
