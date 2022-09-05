package cesium_test

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/cesium/testutil/seg"
	"github.com/arya-analytics/x/telem"
	. "github.com/arya-analytics/x/testutil"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Iterator", func() {
	var db cesium.DB
	BeforeEach(func() {
		db = MustSucceed(cesium.Open("", cesium.MemBacked(), cesium.WithLogger(logger)))
	})
	AfterEach(func() {
		Expect(db.Close()).To(Succeed())
	})
	Describe("Basic Iteration", func() {
		Context("Single Channel", func() {
			It("Should iterate over all segments in a time range", func() {
				ch := cesium.Channel{Rate: 1 * telem.Hz, Density: telem.Bit64}
				Expect(db.CreateChannel(&ch)).To(Succeed())
				factory := seg.NewSequentialFactory(&seg.RandomFloat64Factory{}, 10*telem.Second, ch)
				Expect(db.Write(factory.NextN(10))).To(Succeed())
				iter := MustSucceed(db.NewIterator(telem.TimeRangeMax, ch.Key))
				var segments []cesium.Segment
				for iter.First(); iter.Valid(); iter.Next() {
					segments = append(segments, iter.Value()...)
				}
				Expect(segments).To(HaveLen(10))
				Expect(iter.Close()).To(Succeed())
			})
		})
		Context("Multi Channel", func() {
			It("Should iterate over all segments in a time range", func() {
				ch := cesium.Channel{Rate: 1 * telem.Hz, Density: telem.Bit64}
				ch2 := cesium.Channel{Rate: 5 * telem.Hz, Density: telem.Bit64}
				Expect(db.CreateChannel(&ch)).To(Succeed())
				Expect(db.CreateChannel(&ch2)).To(Succeed())
				factory := seg.NewSequentialFactory(&seg.RandomFloat64Factory{}, 10*telem.Second, ch, ch2)
				Expect(db.Write(factory.NextN(10))).To(Succeed())
				iter := MustSucceed(db.NewIterator(telem.TimeRangeMax, ch.Key, ch2.Key))
				var segments []cesium.Segment
				for iter.First(); iter.Valid(); iter.Next() {
					Expect(iter.Error()).ToNot(HaveOccurred())
					segments = append(segments, iter.Value()...)
				}
				Expect(segments).To(HaveLen(20))
				Expect(iter.Close()).To(Succeed())
			})
		})
	})

	Describe("Iterating By Span", func() {
		It("Should iterate over segments a specified time span", func() {
			ch := cesium.Channel{Rate: 1 * telem.Hz, Density: telem.Bit64}
			Expect(db.CreateChannel(&ch)).To(Succeed())
			ch2 := cesium.Channel{Rate: 5 * telem.Hz, Density: telem.Bit64}
			Expect(db.CreateChannel(&ch2)).To(Succeed())
			factory := seg.NewSequentialFactory(&seg.RandomFloat64Factory{}, 10*telem.Second, ch, ch2)
			Expect(db.Write(factory.NextN(10))).To(Succeed())
			iter := MustSucceed(db.NewIterator(telem.TimeRangeMax, ch.Key, ch2.Key))
			Expect(iter.SeekFirst()).To(BeTrue())
			Expect(iter.NextSpan(20 * telem.Second)).To(BeTrue())
			Expect(iter.Value()).To(HaveLen(4))
			Expect(iter.NextSpan(20 * telem.Second)).To(BeTrue())
			Expect(iter.Value()).To(HaveLen(4))
			Expect(iter.PrevSpan(20 * telem.Second)).To(BeTrue())
			Expect(iter.Value()).To(HaveLen(4))
			Expect(iter.Close()).To(Succeed())
		})
	})

	Describe("Seeking", func() {
		It("Should seek to the correct segment in the range", func() {
			ch := cesium.Channel{Rate: 1 * telem.Hz, Density: telem.Bit64}
			Expect(db.CreateChannel(&ch)).To(Succeed())
			factory := seg.NewSequentialFactory(&seg.RandomFloat64Factory{}, 10*telem.Second, ch)
			Expect(db.Write(factory.NextN(10))).To(Succeed())
			iter := MustSucceed(db.NewIterator(telem.TimeRangeMax, ch.Key))

			Expect(iter.SeekGE(telem.TimeStamp(5 * telem.Second))).To(BeTrue())
			Expect(iter.Value()).To(HaveLen(0))
			Expect(iter.Valid()).To(BeFalse())

			Expect(iter.NextSpan(5 * telem.Second)).To(BeTrue())
			Expect(iter.Valid()).To(BeTrue())
			Expect(iter.View()).To(Equal(telem.TimeRange{
				Start: telem.TimeStamp(10 * telem.Second),
				End:   telem.TimeStamp(15 * telem.Second),
			}))
			Expect(iter.Value()).To(HaveLen(1))
			Expect(iter.Value()[0].Data).To(HaveLen(40))

			Expect(iter.SeekLT(telem.TimeStamp(5 * telem.Second))).To(BeTrue())
			Expect(iter.Valid()).To(BeFalse())

			Expect(iter.NextSpan(5 * telem.Second)).To(BeTrue())
			Expect(iter.View()).To(Equal(telem.TimeRange{
				Start: telem.TimeStamp(0),
				End:   telem.TimeStamp(5 * telem.Second),
			}))
			Expect(iter.Value()).To(HaveLen(1))
			Expect(iter.Value()[0].Data).To(HaveLen(40))

			Expect(iter.SeekLast()).To(BeTrue())
			Expect(iter.Prev()).To(BeTrue())
			Expect(iter.View()).To(Equal(telem.TimeRange{
				Start: telem.TimeStamp(90 * telem.Second),
				End:   telem.TimeStamp(100 * telem.Second),
			}))
			Expect(iter.Value()).To(HaveLen(1))
			Expect(iter.Value()[0].Data).To(HaveLen(80))

			Expect(iter.Close()).To(Succeed())
		})

	})

	Describe("ReadView", func() {
		It("Should set the iterator range correctly", func() {
			ch := cesium.Channel{Rate: 1 * telem.Hz, Density: telem.Bit64}
			Expect(db.CreateChannel(&ch)).To(Succeed())
			factory := seg.NewSequentialFactory(&seg.RandomFloat64Factory{}, 10*telem.Second, ch)
			Expect(db.Write(factory.NextN(10))).To(Succeed())
			iter := MustSucceed(db.NewIterator(telem.TimeRangeMax, ch.Key))
			Expect(iter.ReadView(telem.TimeRange{
				Start: telem.TimeStamp(5 * telem.Second),
				End:   telem.TimeStamp(15 * telem.Second),
			})).To(BeTrue())
			Expect(iter.Value()).To(HaveLen(2))
			Expect(iter.Value()[0].Data).To(HaveLen(40))
			Expect(iter.Value()[1].Data).To(HaveLen(40))
			Expect(iter.Close()).To(Succeed())
		})
	})

	Describe("Opening Error Cases", func() {
		It("Should return an error when the range has no data", func() {
			ch := cesium.Channel{Rate: 1 * telem.Hz, Density: telem.Bit64}
			Expect(db.CreateChannel(&ch)).To(Succeed())
			_, err := db.NewIterator(telem.TimeRangeMax, ch.Key)
			Expect(err).To(HaveOccurredAs(cesium.RangeHasNoData))
		})
		It("Should return an error when no channels are specified", func() {
			_, err := db.NewIterator(telem.TimeRangeMax)
			Expect(err).To(HaveOccurred())
		})
		It("Should return an error when a channel does not exist", func() {
			_, err := db.NewIterator(telem.TimeRangeMax, 1)
			Expect(err).To(HaveOccurredAs(cesium.NotFound))
		})
	})
})
