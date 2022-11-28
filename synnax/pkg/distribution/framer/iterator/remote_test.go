package iterator_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/distribution/core"
	"github.com/synnaxlabs/synnax/pkg/distribution/core/mock"
	"github.com/synnaxlabs/synnax/pkg/distribution/framer/iterator"
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
		node1Ch := channel.Channel{Name: "SG01", Rate: dr, DataType: telem.Float64T, NodeID: 1}
		node2Ch := channel.Channel{Name: "SG02", Rate: dr, DataType: telem.Float64T, NodeID: 2}
		Expect(services[1].channel.Create(&node1Ch)).To(Succeed())
		Expect(services[2].channel.Create(&node2Ch)).To(Succeed())
		channels := []channel.Channel{node1Ch, node2Ch}
		keys = channel.KeysFromChannels(channels)
		nChan = len(channels)
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
