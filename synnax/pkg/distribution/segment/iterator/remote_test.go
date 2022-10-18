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

var _ = Describe("Remote", Ordered, Serial, func() {
	var (
		log      *zap.Logger
		iter     iterator.Iterator
		builder  *mock.CoreBuilder
		services map[core.NodeID]serviceContainer
		nChan    int
		keys     channel.Keys
	)
	BeforeAll(func() {
		log = zap.NewNop()
		builder, services = provisionNServices(3, log)
		dr := 1 * telem.Hz
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
		keys = channel.KeysFromChannels(channels)
		writeMockData(builder, 10*telem.Second, 10, 10, channels...)

		Eventually(func(g Gomega) {
			g.Expect(services[3].channel.NewRetrieve().WhereKeys(keys...).Exists(ctx)).To(BeTrue())
		}).Should(Succeed())

	})
	BeforeEach(func() { iter = openIter(3, services, builder, keys) })
	AfterEach(func() { Expect(iter.Close()).To(Succeed()) })
	AfterAll(func() { Expect(builder.Close()).To(Succeed()) })
	Context("Behavioral Accuracy", func() {
		Describe("SeekFirst + Next", func() {
			It("Should return the first segment in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(10 * telem.Second)).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(nChan))
			})
		})
		Describe("SeekLast + Prev", func() {
			It("Should return the last segment in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Prev(10 * telem.Second)).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(nChan))
			})
		})
		Describe("Next", func() {
			It("Should return the next span in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(20 * telem.Second)).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(nChan * 2))
			})
		})
		Describe("Prev", func() {
			It("Should return the previous span in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Prev(30 * telem.Second)).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(nChan * 3))
			})
		})
	})
})
