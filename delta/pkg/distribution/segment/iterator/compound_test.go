package iterator_test

import (
	"context"
	"github.com/arya-analytics/delta/pkg/distribution/channel"
	"github.com/arya-analytics/delta/pkg/distribution/core"
	"github.com/arya-analytics/delta/pkg/distribution/core/mock"
	"github.com/arya-analytics/delta/pkg/distribution/segment/iterator"
	"github.com/arya-analytics/x/telem"
	"github.com/cockroachdb/errors"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"go.uber.org/zap"
	"time"
)

func assertResponse(
	c,
	n int,
	iter iterator.Iterator,
	timeout time.Duration,
) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	for i := 0; i < c; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case v := <-iter.Responses():
			if len(v.Segments) != n {
				return errors.Newf("expected %v segments, received %v", n, len(v.Segments))
			}
		}
	}
	select {
	case <-iter.Responses():
		return errors.Newf("expected no more iter, received extra response")
	case <-ctx.Done():
		return nil
	}
}

var _ = Describe("Compound", Ordered, func() {
	var (
		log       *zap.Logger
		iter      iterator.Iterator
		builder   *mock.CoreBuilder
		services  map[core.NodeID]serviceContainer
		nChannels int
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
		nChannels = len(channels)
		writeMockData(builder, 10*telem.Second, 10, 10, channels...)

		Eventually(func(g Gomega) {
			g.Expect(services[2].channel.NewRetrieve().WhereKeys(channel.KeysFromChannels(channels)...).Exists(ctx)).To(BeTrue())
		}).Should(Succeed())

		iter = openIter(2, services, builder, channel.KeysFromChannels(channels))
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
				Expect(assertResponse(nChannels, 1, iter, sensibleTimeoutThreshold)).To(Succeed())
			})
		})
		Describe("SeekFirst + TraverseTo", func() {
			It("Should return the first segment in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.Next()).To(BeTrue())
				Expect(assertResponse(nChannels, 1, iter, sensibleTimeoutThreshold)).To(Succeed())
			})
		})
		Describe("SeekLast + Prev", func() {
			It("Should return the last segment in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.Prev()).To(BeTrue())
				Expect(assertResponse(nChannels, 1, iter, sensibleTimeoutThreshold)).To(Succeed())
			})
		})
		Describe("NextSpan", func() {
			It("Should return the next span in the iterator", func() {
				Expect(iter.SeekFirst()).To(BeTrue())
				Expect(iter.NextSpan(20 * telem.Second)).To(BeTrue())
				Expect(assertResponse(nChannels*2, 1, iter, sensibleTimeoutThreshold)).To(Succeed())
			})
		})
		Describe("PrevSpan", func() {
			It("Should return the previous span in the iterator", func() {
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.PrevSpan(20 * telem.Second)).To(BeTrue())
				Expect(assertResponse(nChannels*2, 1, iter, sensibleTimeoutThreshold)).To(Succeed())
			})
		})
		Describe("NextRange", func() {
			It("Should return the next range of data in the iterator", func() {
				Expect(iter.NextRange(telem.TimeRange{Start: 0, End: telem.TimeStamp(30 * telem.Second)})).To(BeTrue())
				Expect(assertResponse(nChannels*3, 1, iter, 30*time.Millisecond))
			})
		})
	})
})
