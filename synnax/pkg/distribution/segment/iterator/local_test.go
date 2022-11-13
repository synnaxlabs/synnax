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

var _ = Describe("Local", Ordered, func() {
	var (
		log      *zap.Logger
		iter     iterator.Iterator
		builder  *mock.CoreBuilder
		services map[core.NodeID]serviceContainer
	)
	BeforeAll(func() {
		log = zap.L()
		builder, services = provisionNServices(1, log)
		ch := channel.Channel{Name: "SG01", Rate: 25 * telem.Hz, DataType: telem.Float64, NodeID: 1}
		Expect(services[1].channel.Create(&ch)).To(Succeed())
		writeMockData(builder, 10*telem.Second, 10, 1, ch)
		iter = openIter(1, services, builder, channel.Keys{ch.Key()})
	})
	AfterAll(func() { Expect(iter.Close()).To(Succeed()) })
	Context("Behavioral Accuracy", func() {
		Describe("SeekFirst + Next", func() {
			It("Should return the next segment in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(10 * telem.Second)).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(1))
			})
		})
		Describe("SeekLast + Prev", func() {
			It("Should return the previous segment in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Prev(10 * telem.Second)).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(1))
			})
		})
		Describe("Next", func() {
			It("Should return the next span in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next(20 * telem.Second)).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(2))
			})
		})
		Describe("Prev", func() {
			It("Should return the previous span in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Prev(20 * telem.Second)).To(BeTrue())
				Expect(iter.Value()).To(HaveLen(2))
			})
		})
	})
})
