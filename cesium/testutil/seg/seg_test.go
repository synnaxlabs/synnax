package seg_test

import (
	"github.com/arya-analytics/cesium"
	"github.com/arya-analytics/cesium/testutil/seg"
	seg2 "github.com/arya-analytics/cesium/testutil/seg"
	"github.com/arya-analytics/x/binary"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Seg", func() {
	Describe("data Factories", func() {
		Describe("Float64", func() {
			Describe("Sequential", func() {
				It("Should generate the correct floating point values", func() {
					f := seg.SeqFloat64(10)
					Expect(f).To(HaveLen(80))
					f64 := binary.ToFloat64(f)
					Expect(f64).To(HaveLen(10))
					Expect(f64[0]).To(Equal(float64(0)))
					Expect(f64[1]).To(Equal(float64(1.0)))
				})
			})
			Describe("Random", func() {
				It("Should generate the correct floating point values", func() {
					f := seg.RandFloat64(10)
					Expect(f).To(HaveLen(80))
					f64 := binary.ToFloat64(f)
					Expect(f64).To(HaveLen(10))
				})
			})
		})
	})
	Describe("segment Factories", func() {
		Describe("new", func() {
			It("Should create a new segment correctly", func() {
				c := cesium.Channel{Key: 1, DataRate: 1, DataType: cesium.Float64}
				seg := seg2.New(c, seg.SeqFloat64, 0, 10*cesium.Second)
				Expect(seg.Data).To(HaveLen(80))
				f64 := binary.ToFloat64(seg.Data)
				Expect(f64).To(HaveLen(10))
				Expect(f64[0]).To(Equal(float64(0)))
				Expect(c.DataRate.Span(len(f64))).To(Equal(10 * cesium.Second))
			})
		})
		Describe("NewSet", func() {
			It("Should create a new set of segments correctly", func() {
				c := cesium.Channel{Key: 1, DataRate: 1, DataType: cesium.Float64}
				segs := seg2.NewSet(c, seg.SeqFloat64, 0, 10*cesium.Second, 10)
				Expect(segs).To(HaveLen(10))
				Expect(segs[0].Data).To(HaveLen(80))
				f64 := binary.ToFloat64(segs[0].Data)
				Expect(f64).To(HaveLen(10))
				Expect(f64[0]).To(Equal(float64(0)))
				lastSeg := segs[len(segs)-1]
				Expect(lastSeg.Data).To(HaveLen(80))
				Expect(lastSeg.Range(c.DataRate, c.DataType).End).To(Equal(cesium.TimeStamp(0).Add(100 * cesium.Second)))
			})
		})
		Describe("SequentialFactory", func() {
			It("Should create a seg of segments sequentially", func() {
				c := cesium.Channel{Key: 1, DataRate: 1, DataType: cesium.Float64}
				sf := seg2.NewSequentialFactory(seg.SeqFloat64, 10*cesium.Second, c)
				Expect(sf.Next()[0].Start).To(Equal(cesium.TimeStamp(0)))
				segs := sf.NextN(2)
				Expect(segs).To(HaveLen(2))
				Expect(segs[1].Start).To(Equal(cesium.TimeStamp(0).Add(20 * cesium.Second)))
				segs = sf.NextN(2)
				Expect(segs).To(HaveLen(2))
				Expect(segs[1].Start).To(Equal(cesium.TimeStamp(0).Add(40 * cesium.Second)))
			})
		})
	})
})
