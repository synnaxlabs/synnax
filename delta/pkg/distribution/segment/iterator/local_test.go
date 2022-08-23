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

var _ = Describe("Local", Ordered, func() {
	var (
		log      *zap.Logger
		iter     iterator.Iterator
		builder  *mock.CoreBuilder
		services map[core.NodeID]serviceContainer
	)
	BeforeAll(func() {
		log = zap.NewNop()
		builder, services = provisionNServices(1, log)
		channels, err := services[1].channel.NewCreate().
			WithName("SG02").
			WithRate(25*telem.Hz).
			WithDataType(telem.Float64).
			WithNodeID(1).
			ExecN(ctx, 1)
		Expect(err).ToNot(HaveOccurred())
		writeMockData(builder, 10*telem.Second, 10, 1, channels...)
		iter = openIter(1, services, builder, channel.KeysFromChannels(channels))
	})
	AfterAll(func() {
		Expect(iter.Close()).To(Succeed())
		_, ok := <-iter.Responses()
		Expect(ok).To(BeFalse())
		Expect(builder.Close()).To(Succeed())

	})
	// Behavioral accuracy tests check whether the iterator returns the correct
	// boolean acknowledgements and segment counts. These tests DO NOT check
	// for data accuracy.
	Context("Behavioral Accuracy", func() {
		Describe("First", func() {
			It("Should return the first segment in the iterator", func() {
				Expect(iter.First()).To(BeTrue())
				res := <-iter.Responses()
				Expect(res.Error).To(BeNil())
				Expect(res.Segments).To(HaveLen(1))
			})
		})
		Describe("SeekFirst + TraverseTo", func() {
			It("Should return the next segment in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next()).To(BeTrue())
				res := <-iter.Responses()
				Expect(res.Error).To(BeNil())
				Expect(res.Segments).To(HaveLen(1))
			})
		})
		Describe("SeekLast + Prev", func() {
			It("Should return the previous segment in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Prev()).To(BeTrue())
				res := <-iter.Responses()
				Expect(res.Error).To(BeNil())
				Expect(res.Segments).To(HaveLen(1))
			})
		})
		Describe("NextSpan", func() {
			It("Should return the next span in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.NextSpan(20 * telem.Second)).To(BeTrue())
				res := <-iter.Responses()
				Expect(res.Error).To(BeNil())
				Expect(res.Segments).To(HaveLen(1))
				res2 := <-iter.Responses()
				Expect(res2.Error).To(BeNil())
				Expect(res2.Segments).To(HaveLen(1))
			})
		})
		Describe("PrevSpan", func() {
			It("Should return the previous span in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.PrevSpan(20 * telem.Second)).To(BeTrue())
				res := <-iter.Responses()
				Expect(res.Error).To(BeNil())
				Expect(res.Segments).To(HaveLen(1))
				res2 := <-iter.Responses()
				Expect(res2.Error).To(BeNil())
				Expect(res2.Segments).To(HaveLen(1))
			})
		})
	})
})
