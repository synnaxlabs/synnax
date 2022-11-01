package cesium_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/cesium"
	"github.com/synnaxlabs/cesium/internal/core"
	"github.com/synnaxlabs/cesium/testutil/seg"
	"github.com/synnaxlabs/x/telem"
	"sync"
)

var _ = Describe("GoRead", func() {
	var db cesium.DB
	BeforeEach(func() {
		var err error
		db, err = cesium.Open("", cesium.MemBacked(), cesium.WithLogger(logger))
		Expect(err).ToNot(HaveOccurred())
	})
	Context("Single channel", func() {
		var (
			channel *cesium.Channel
			factory seg.SequentialFactory
		)
		BeforeEach(func() {
			channel = &cesium.Channel{Rate: 1 * telem.Hz, Density: telem.Bit64}
			Expect(db.CreateChannel(channel)).To(Succeed())
			factory = seg.NewSequentialFactory(&seg.RandomFloat64Factory{}, 10*telem.Second, *channel)
		})
		AfterEach(func() { Expect(db.Close()).To(Succeed()) })
		It("Should read the segments correctly", func() {
			Expect(db.Write(factory.NextN(20))).To(Succeed())
			var segments []cesium.Segment
			segments, err := db.Read(telem.TimeRangeMax, channel.Key)
			Expect(err).ToNot(HaveOccurred())
			Expect(segments).To(HaveLen(20))
		})
		It("It should support multiple concurrent read requests", func() {
			Expect(db.Write(factory.NextN(20))).To(Succeed())
			const nRequests = 10
			var wg sync.WaitGroup
			wg.Add(nRequests)
			for i := 0; i < nRequests; i++ {
				go func() {
					defer GinkgoRecover()
					defer wg.Done()
					segments, err := db.Read(telem.TimeRangeMax, channel.Key)
					Expect(err).ToNot(HaveOccurred())
					Expect(segments).To(HaveLen(20))
				}()
			}
			wg.Wait()
		})
	})
	Context("Multi channel", func() {
		var (
			channels     []cesium.Channel
			channelCount = 10
		)
		BeforeEach(func() {
			var err error
			db, err = cesium.Open("testdata", cesium.MemBacked())
			Expect(err).ToNot(HaveOccurred())
			for i := 0; i < channelCount; i++ {
				c := cesium.Channel{Rate: 1 * telem.Hz, Density: telem.Bit64}
				Expect(db.CreateChannel(&c)).To(Succeed())
				Expect(err).ToNot(HaveOccurred())
				channels = append(channels, c)
			}
		})
		AfterEach(func() {
			Expect(db.Close()).To(Succeed())
		})
		It("Should support reading data from multiple Channels", func() {
			for i := 0; i < channelCount; i++ {
				factory := seg.NewSequentialFactory(&seg.RandomFloat64Factory{}, 10*telem.Second, channels[i])
				Expect(db.Write(factory.NextN(20))).To(Succeed())
			}
			segments, err := db.Read(telem.TimeRangeMax, core.ChannelKeys(channels)...)
			Expect(err).ToNot(HaveOccurred())
			Expect(segments).To(HaveLen(200))
		})
	})
})
