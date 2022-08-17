package kv_test

import (
	"github.com/arya-analytics/cesium/internal/channel"
	"github.com/arya-analytics/cesium/internal/kv"
	"github.com/arya-analytics/cesium/internal/segment"
	kvx "github.com/arya-analytics/x/kv"
	"github.com/arya-analytics/x/kv/memkv"
	"github.com/arya-analytics/x/telem"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Regression", func() {
	var (
		kve      kvx.DB
		headerKV *kv.Header
		chKV     *kv.Channel
	)
	BeforeEach(func() {
		kve = memkv.New()
		chKV = kv.NewChannel(kve)
		headerKV = kv.NewHeader(kve)

	})
	Context("25Hz Float64", func() {
		var (
			ch channel.Channel
		)
		BeforeEach(func() {
			ch = channel.Channel{
				Key:      1,
				DataRate: 25,
				DataType: telem.Float64,
			}
			Expect(chKV.Set(ch)).To(Succeed())
			span := 10 * telem.Second
			size := span.ByteSize(ch.DataRate, ch.DataType)
			var headers []segment.Header
			for i := 0; i < 10; i++ {
				headers = append(headers, segment.Header{
					ChannelKey: ch.Key,
					Start:      telem.TimeStamp(telem.TimeSpan(i) * span),
					Size:       size,
				})
			}
			Expect(headerKV.SetMultiple(headers)).To(Succeed())
		})
		Describe("PrevSpan", func() {
			It("Should move the iterator view correctly", func() {
				iter := kv.NewIterator(kve, telem.TimeRangeMax, ch.Key)
				Expect(iter.SeekLast()).To(BeTrue())
				Expect(iter.PrevSpan(20 * telem.Second)).To(BeTrue())
				Expect(iter.View()).To(Equal(telem.TimeRange{
					Start: telem.TimeStamp(80 * telem.Second),
					End:   telem.TimeStamp(100 * telem.Second),
				}))
				Expect(iter.Range().Headers).To(HaveLen(2))
				Expect(iter.Range().UnboundedRange()).To(Equal(iter.View()))
			})
		})

	})

})
