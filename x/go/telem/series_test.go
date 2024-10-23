package telem_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
)

var _ = Describe("Series", func() {
	Describe("Len", func() {
		It("Should correctly return the number of samples in a series with a fixed length data type", func() {
			s := telem.NewSeriesV[int64](1, 2, 3)
			Expect(s.Len()).To(Equal(int64(3)))
		})
		It("Should correctly return the number of samples in a series with a variable length data type", func() {
			s := telem.NewStringsV("bob", "alice", "charlie")
			Expect(s.Len()).To(Equal(int64(3)))
		})
	})
	Describe("Factory", func() {
		It("Should marshal and unmarshal a float64 series correctly", func() {
			d := []float64{1.0, 2.0, 3.0}
			s := telem.NewSeries[float64](d)
			Expect(s.DataType).To(Equal(telem.Float64T))
			Expect(s.Len()).To(Equal(int64(3)))
			Expect(telem.Unmarshal[float64](s)).To(Equal(d))
		})
		It("Should marshal and unmarshal a float32 series correctly", func() {
			d := []float32{1.0, 2.0, 3.0}
			s := telem.NewSeries[float32](d)
			Expect(s.DataType).To(Equal(telem.Float32T))
			Expect(s.Len()).To(Equal(int64(3)))
			Expect(telem.Unmarshal[float32](s)).To(Equal(d))
		})
		It("Should marshal and unmarshal a int64 series correctly", func() {
			d := []int64{1, 2, 3}
			s := telem.NewSeries[int64](d)
			Expect(s.DataType).To(Equal(telem.Int64T))
			Expect(s.Len()).To(Equal(int64(3)))
			Expect(telem.Unmarshal[int64](s)).To(Equal(d))
		})
		It("Should marshal and unmarshal a int32 series correctly", func() {
			d := []int32{1, 2, 3}
			s := telem.NewSeries[int32](d)
			Expect(s.DataType).To(Equal(telem.Int32T))
			Expect(s.Len()).To(Equal(int64(3)))
			Expect(telem.Unmarshal[int32](s)).To(Equal(d))
		})
		It("Should marshal and unmarshal a int16 series correctly", func() {
			d := []int16{1, 2, 3}
			s := telem.NewSeries[int16](d)
			Expect(s.DataType).To(Equal(telem.Int16T))
			Expect(s.Len()).To(Equal(int64(3)))
			Expect(telem.Unmarshal[int16](s)).To(Equal(d))
		})
		It("Should marshal and unmarshal a int8 series correctly", func() {
			d := []int8{1, 2, 3}
			s := telem.NewSeries[int8](d)
			Expect(s.DataType).To(Equal(telem.Int8T))
			Expect(s.Len()).To(Equal(int64(3)))
			Expect(telem.Unmarshal[int8](s)).To(Equal(d))
		})
		It("Should marshal and unmarshal a uint64 series correctly", func() {
			d := []uint64{1, 2, 3}
			s := telem.NewSeries[uint64](d)
			Expect(s.DataType).To(Equal(telem.Uint64T))
			Expect(s.Len()).To(Equal(int64(3)))
			Expect(telem.Unmarshal[uint64](s)).To(Equal(d))
		})
		It("Should marshal and unmarshal a uint32 series correctly", func() {
			d := []uint32{1, 2, 3}
			s := telem.NewSeries[uint32](d)
			Expect(s.DataType).To(Equal(telem.Uint32T))
			Expect(s.Len()).To(Equal(int64(3)))
			Expect(telem.Unmarshal[uint32](s)).To(Equal(d))
		})
		It("Should marshal and unmarshal a uint16 series correctly", func() {
			d := []uint16{1, 2, 3}
			s := telem.NewSeries[uint16](d)
			Expect(s.DataType).To(Equal(telem.Uint16T))
			Expect(s.Len()).To(Equal(int64(3)))
			Expect(telem.Unmarshal[uint16](s)).To(Equal(d))
		})
		It("Should marshal and unmarshal a uint8 series correctly", func() {
			d := []uint8{1, 2, 3}
			s := telem.NewSeries[uint8](d)
			Expect(s.DataType).To(Equal(telem.Uint8T))
			Expect(s.Len()).To(Equal(int64(3)))
			Expect(telem.Unmarshal[uint8](s)).To(Equal(d))
		})
	})
})
