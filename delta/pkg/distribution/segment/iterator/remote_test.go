package iterator_test

import (
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/core/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/x/telem"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
)

var _ = Describe("Remote", Ordered, func() {
	var (
		log      *zap.Logger
		iter     iterator.Iterator
		builder  *mock.CoreBuilder
		services map[core.NodeID]serviceContainer
		nChan    int
	)
	BeforeAll(func() {
		log = zap.NewNop()
		builder, services = provisionNServices(3, log)
		dr := 1 * telem.Hz
		var channels []channel.Channel
		node1Channels, err := services[1].channel.NewCreate().
			WithName("SG02").
			WithRate(dr).
			WithDensity(telem.Float64).
			WithNodeID(1).
			ExecN(ctx, 1)
		Expect(err).ToNot(HaveOccurred())
		channels = append(channels, node1Channels...)
		node2Channels, err := services[2].channel.NewCreate().
			WithName("SG02").
			WithRate(dr).
			WithDensity(telem.Float64).
			WithNodeID(2).
			ExecN(ctx, 1)
		Expect(err).ToNot(HaveOccurred())
		channels = append(channels, node2Channels...)
		nChan = len(channels)
		keys := channel.KeysFromChannels(channels)
		writeMockData(builder, 10*telem.Second, 10, 10, channels...)

		Eventually(func(g Gomega) {
			g.Expect(services[3].channel.NewRetrieve().WhereKeys(keys...).Exists(ctx)).To(BeTrue())
		}).Should(Succeed())

		iter = openIter(3, services, builder, keys)
	})
	AfterAll(func() {
		Expect(iter.Close()).To(Succeed())
		_, ok := <-iter.Responses()
		Expect(ok).To(BeFalse())
		Expect(builder.Close()).To(Succeed())
	})
	Context("Behavioral Accuracy", func() {
		Describe("First", func() {
			It("Should return the first segment in the iterator", func() {
				Expect(iter.First()).To(BeTrue())
				Expect(assertResponse(nChan, 1, iter, sensibleTimeoutThreshold)).To(Succeed())
			})
		})
		Describe("SeekFirst + TraverseTo", func() {
			It("Should return the first segment in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next()).To(BeTrue())
				Expect(assertResponse(nChan, 1, iter, sensibleTimeoutThreshold)).To(Succeed())
			})
		})
		Describe("SeekLast + Prev", func() {
			It("Should return the last segment in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Prev()).To(BeTrue())
				Expect(assertResponse(nChan, 1, iter, sensibleTimeoutThreshold)).To(Succeed())
			})
		})
		Describe("NextSpan", func() {
			It("Should return the next span in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.NextSpan(20 * telem.Second)).To(BeTrue())
				Expect(assertResponse(nChan*2, 1, iter, sensibleTimeoutThreshold))
			})
		})
		Describe("PrevSpan", func() {
			It("Should return the previous span in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.PrevSpan(30 * telem.Second)).To(BeTrue())
				Expect(assertResponse(nChan*3, 1, iter, sensibleTimeoutThreshold)).To(Succeed())
			})
		})
		Describe("NextRange", func() {
			It("Should return the next range of data in the iterator", func() {
				Expect(iter.NextRange(telem.TimeRange{Start: 0, End: telem.TimeStamp(25 * telem.Second)})).To(BeTrue())
				Expect(assertResponse(nChan*3, 1, iter, sensibleTimeoutThreshold)).To(Succeed())
			})
		})
	})
})
