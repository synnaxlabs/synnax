package iterator_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/segment/iterator"
	"github.com/synnaxlabs/x/telem"
	"go.uber.org/zap"
)

var _ = Describe("Compound", Ordered, func() {
	var (
		log      *zap.Logger
		iter     iterator.Iterator
		builder  *mock.CoreBuilder
		services map[core.NodeID]serviceContainer
		nChan    int
	)
	BeforeAll(func() {
		log = zap.NewNop()
		dr := 25 * telem.Hz
		builder, services = provisionNServices(2, log)
		var channels []channel.Channel
		node1Channels, err := services[1].channel.NewCreate().
			WithName("SG02").
			WithRate(dr).
			WithDataType(telem.Float64).
			WithNodeID(1).
			ExecN(ctx, 1)
		Expect(err).ToNot(HaveOccurred())
		channels = append(channels, node1Channels...)
		node2Channels, err := services[2].channel.NewCreate().
			WithName("SG02").
			WithRate(dr).
			WithDataType(telem.Float64).
			WithNodeID(2).
			ExecN(ctx, 1)
		Expect(err).ToNot(HaveOccurred())
		channels = append(channels, node2Channels...)
		nChan = len(channels)
		writeMockData(builder, 10*telem.Second, 10, 10, channels...)

		Eventually(func(g Gomega) {
			g.Expect(services[2].channel.NewRetrieve().WhereKeys(channel.KeysFromChannels(channels)...).Exists(ctx)).To(BeTrue())
		}).Should(Succeed())

		iter = openIter(2, services, builder, channel.KeysFromChannels(channels))
	})
	AfterAll(func() {
		Expect(iter.Close()).To(Succeed())
		Expect(builder.Close()).To(Succeed())
	})
	Context("Behavioral Accuracy", func() {
		Describe("First", func() {
			It("Should return the first segment in the iterator", func() {
				Expect(iter.First()).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(nChan))
			})
		})
		Describe("SeekFirst + TraverseTo", func() {
			It("Should return the first segment in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next()).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(nChan))
			})
		})
		Describe("SeekLast + Prev", func() {
			It("Should return the last segment in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Prev()).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(nChan))
			})
		})
		Describe("NextSpan", func() {
			It("Should return the next span in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.NextSpan(20 * telem.Second)).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(nChan * 2))
			})
		})
		Describe("PrevSpan", func() {
			It("Should return the previous span in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.PrevSpan(20 * telem.Second)).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(nChan * 2))
			})
		})
		Describe("ReadView", func() {
			It("Should return the next range of data in the iterator", func() {
				Expect(iter.Range(telem.TimeRange{Start: 0, End: telem.TimeStamp(30 * telem.Second)})).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(nChan * 3))
			})
		})
	})
})
