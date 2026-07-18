// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package telem_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
)

func valueAtTest[T telem.FixedSample](value T, dt telem.DataType) func() {
	return func() {
		s := telem.NewSeriesV(value)
		ExpectWithOffset(1, s.DataType).To(Equal(dt))
		ExpectWithOffset(1, telem.ValueAt[T](s, 0)).To(Equal(value))
	}
}

var _ = Describe("Series", func() {
	Describe("ValueAt", func() {
		Describe("Happy Path", func() {
			Specify("uint8", valueAtTest(uint8(1), telem.Uint8T))
			Specify("uint16", valueAtTest(uint16(1), telem.Uint16T))
			Specify("uint32", valueAtTest(uint32(1), telem.Uint32T))
			Specify("uint64", valueAtTest(uint64(1), telem.Uint64T))
			Specify("int8", valueAtTest(int8(1), telem.Int8T))
			Specify("int16", valueAtTest(int16(1), telem.Int16T))
			Specify("int32", valueAtTest(int32(1), telem.Int32T))
			Specify("int64", valueAtTest(int64(1), telem.Int64T))
			Specify("float32", valueAtTest(float32(1.0), telem.Float32T))
			Specify("float64", valueAtTest(float64(1.0), telem.Float64T))
			Specify("timestamp", valueAtTest(telem.TimeStamp(1), telem.TimeStampT))
			Specify("uuid", valueAtTest(uuid.New(), telem.UUIDT))
		})
		Describe("Negative Index", func() {
			It("Should return a value at the given negative index", func() {
				data := []int64{1, 2, 3}
				s := telem.NewSeries(data)
				Expect(telem.ValueAt[int64](s, -1)).To(Equal(data[2]))
				Expect(telem.ValueAt[int64](s, -2)).To(Equal(data[1]))
				Expect(telem.ValueAt[int64](s, -3)).To(Equal(data[0]))
			})
		})
		Describe("Out of Bounds", func() {
			It("Should panic when the index is out of bounds", func() {
				s := telem.NewSeriesV[int64](1, 2, 3)
				Expect(func() { telem.ValueAt[int64](s, 3) }).To(Panic())
				Expect(func() { telem.ValueAt[int64](s, -4) }).To(Panic())
			})
		})
	})
	Describe("SetValueAt", func() {
		Describe("Per Type", func() {
			Specify("Uint8", func() {
				s := telem.NewSeriesV[uint8](1)
				Expect(telem.ValueAt[uint8](s, 0)).To(Equal(uint8(1)))
				telem.SetValueAt(s, 0, uint8(10))
				Expect(telem.ValueAt[uint8](s, 0)).To(Equal(uint8(10)))
			})
			Specify("Uint16", func() {
				s := telem.NewSeriesV[uint16](2)
				Expect(telem.ValueAt[uint16](s, 0)).To(Equal(uint16(2)))
				telem.SetValueAt(s, 0, uint16(20))
				Expect(telem.ValueAt[uint16](s, 0)).To(Equal(uint16(20)))
			})
			Specify("Uint32", func() {
				s := telem.NewSeriesV[uint32](4)
				Expect(telem.ValueAt[uint32](s, 0)).To(Equal(uint32(4)))
				telem.SetValueAt(s, 0, uint32(40))
				Expect(telem.ValueAt[uint32](s, 0)).To(Equal(uint32(40)))
			})
			Specify("Uint64", func() {
				s := telem.NewSeriesV[uint64](8)
				Expect(telem.ValueAt[uint64](s, 0)).To(Equal(uint64(8)))
				telem.SetValueAt(s, 0, uint64(80))
				Expect(telem.ValueAt[uint64](s, 0)).To(Equal(uint64(80)))
			})
			Specify("Float32", func() {
				s := telem.NewSeriesV[float32](4)
				Expect(telem.ValueAt[float32](s, 0)).To(Equal(float32(4)))
				telem.SetValueAt(s, 0, float32(40))
				Expect(telem.ValueAt[float32](s, 0)).To(Equal(float32(40)))
			})
			Specify("Float64", func() {
				s := telem.NewSeriesV[float64](8)
				Expect(telem.ValueAt[float64](s, 0)).To(Equal(float64(8)))
				telem.SetValueAt(s, 0, float64(80))
				Expect(telem.ValueAt[float64](s, 0)).To(Equal(float64(80)))
			})
			Specify("Int64", func() {
				s := telem.NewSeriesV[int64](8)
				Expect(telem.ValueAt[int64](s, 0)).To(Equal(int64(8)))
				telem.SetValueAt(s, 0, int64(80))
				Expect(telem.ValueAt[int64](s, 0)).To(Equal(int64(80)))
			})
			Specify("Int32", func() {
				s := telem.NewSeriesV[int32](4)
				Expect(telem.ValueAt[int32](s, 0)).To(Equal(int32(4)))
				telem.SetValueAt(s, 0, int32(40))
				Expect(telem.ValueAt[int32](s, 0)).To(Equal(int32(40)))
			})
			Specify("Int16", func() {
				s := telem.NewSeriesV[int16](4)
				Expect(telem.ValueAt[int16](s, 0)).To(Equal(int16(4)))
				telem.SetValueAt(s, 0, int16(40))
				Expect(telem.ValueAt[int16](s, 0)).To(Equal(int16(40)))
			})
			Specify("Int8", func() {
				s := telem.NewSeriesV[int8](4)
				Expect(telem.ValueAt[int8](s, 0)).To(Equal(int8(4)))
				telem.SetValueAt(s, 0, int8(40))
				Expect(telem.ValueAt[int8](s, 0)).To(Equal(int8(40)))
			})
			Specify("TimeStamp", func() {
				s := telem.NewSeriesV[telem.TimeStamp](8)
				Expect(telem.ValueAt[telem.TimeStamp](s, 0)).To(Equal(telem.TimeStamp(8)))
				telem.SetValueAt(s, 0, telem.TimeStamp(80))
				Expect(telem.ValueAt[telem.TimeStamp](s, 0)).To(Equal(telem.TimeStamp(80)))
			})
			Specify("UUID", func() {
				v1 := uuid.New()
				s := telem.NewSeriesV(v1)
				Expect(telem.ValueAt[uuid.UUID](s, 0)).To(Equal(v1))
				v2 := uuid.New()
				telem.SetValueAt(s, 0, v2)
				Expect(telem.ValueAt[uuid.UUID](s, 0)).To(Equal(v2))
			})
		})

		It("Should set the value at the given index", func() {
			s := telem.NewSeriesV[int64](1, 2, 3)
			telem.SetValueAt[int64](s, 0, 4)
			Expect(telem.ValueAt[int64](s, 0)).To(Equal(int64(4)))
		})

		It("Should support negative indices", func() {
			s := telem.NewSeriesV[int64](1, 2, 3)
			telem.SetValueAt[int64](s, -1, 4)
			Expect(telem.ValueAt[int64](s, -1)).To(Equal(int64(4)))
		})

		It("Should panic when the index is out of bounds", func() {
			s := telem.NewSeriesV[int64](1, 2, 3)
			Expect(func() { telem.SetValueAt[int64](s, 3, 4) }).To(Panic())
			Expect(func() { telem.SetValueAt[int64](s, -4, 4) }).To(Panic())
		})
	})
	Describe("MultiSeries", func() {
		Describe("NewMultiSeries", func() {
			It("Should construct a multi-series from a slice of series", func() {
				s1 := telem.NewSeriesSecondsTSV(1, 2, 3)
				s2 := telem.NewSeriesSecondsTSV(4, 5, 6)
				ms := telem.NewMultiSeriesV(s1, s2)
				Expect(ms.Len()).To(Equal(int64(6)))
			})
			It("Should sort the series by alignment on construction", func() {
				s1 := telem.NewSeriesSecondsTSV(1, 2, 3)
				s1.Alignment = telem.NewAlignment(0, 0)
				s2 := telem.NewSeriesSecondsTSV(4, 5, 6)
				s2.Alignment = telem.NewAlignment(0, 3)
				ms := telem.NewMultiSeriesV(s2, s1)
				Expect(ms.Series[0].Alignment).To(Equal(s1.Alignment))
				Expect(ms.Series[1].Alignment).To(Equal(s2.Alignment))
			})
			It("Should panic when trying to construct the series out of different data types", func() {
				s1 := telem.NewSeriesSecondsTSV(1, 2, 3)
				s2 := telem.NewSeriesV("a", "b", "c")
				Expect(func() { telem.NewMultiSeriesV(s1, s2) }).To(Panic())
			})
		})
		Describe("MultiSeriesAtAlignment", func() {
			It("Should return the data at the specified alignment", func() {
				s1 := telem.NewSeriesV[uint8](1, 2, 3)
				s1.Alignment = telem.NewAlignment(1, 0)
				s2 := telem.NewSeriesV[uint8](4, 5, 6)
				s2.Alignment = telem.NewAlignment(1, 3)
				ms := telem.NewMultiSeriesV(s1, s2)
				Expect(telem.MultiSeriesAtAlignment[uint8](ms, telem.NewAlignment(1, 3))).To(Equal(uint8(4)))
			})

			It("Should panic when querying a value outside of the expected alignment", func() {
				s1 := telem.NewSeriesV[uint8](1, 2, 3)
				s1.Alignment = telem.NewAlignment(1, 0)
				ms := telem.NewMultiSeriesV(s1)
				Expect(func() {
					telem.MultiSeriesAtAlignment[uint8](ms, 5000)
				}).To(Panic())
			})
		})
	})
	Describe("CopyValue", func() {
		It("Should copy a value from one series to another", func() {
			src := telem.NewSeriesV[int64](1, 2, 3, 4, 5)
			dst := telem.NewSeriesV[int64](10, 20, 30, 40, 50)
			telem.CopyValue(dst, src, 0, 2)
			Expect(telem.ValueAt[int64](dst, 0)).To(Equal(int64(3)))
			Expect(telem.ValueAt[int64](dst, 1)).To(Equal(int64(20)))
		})

		It("Should copy values at different indices", func() {
			src := telem.NewSeriesV(1.1, 2.2, 3.3)
			dst := telem.NewSeriesV(0.0, 0.0, 0.0)
			telem.CopyValue(dst, src, 1, 2)
			Expect(telem.ValueAt[float64](dst, 1)).To(Equal(3.3))
			Expect(telem.ValueAt[float64](dst, 0)).To(Equal(0.0))
			Expect(telem.ValueAt[float64](dst, 2)).To(Equal(0.0))
		})

		It("Should work with different numeric types", func() {
			src := telem.NewSeriesV[uint8](10, 20, 30)
			dst := telem.NewSeriesV[uint8](0, 0, 0)
			telem.CopyValue(dst, src, 2, 1)
			Expect(telem.ValueAt[uint8](dst, 2)).To(Equal(uint8(20)))
		})

		It("Should panic when data types do not match", func() {
			src := telem.NewSeriesV[int64](1, 2, 3)
			dst := telem.NewSeriesV[int32](10, 20, 30)
			Expect(func() {
				telem.CopyValue(dst, src, 0, 0)
			}).To(Panic())
		})

		It("Should panic when source is variable density", func() {
			src := telem.NewSeriesV("a", "b", "c")
			dst := telem.NewSeriesV("x", "y", "z")
			Expect(func() {
				telem.CopyValue(dst, src, 0, 0)
			}).To(Panic())
		})

		It("Should panic when destination is variable density", func() {
			src := telem.NewSeriesV[int64](1, 2, 3)
			dst := telem.NewSeriesV("x", "y", "z")
			Expect(func() {
				telem.CopyValue(dst, src, 0, 0)
			}).To(Panic())
		})
	})
})
