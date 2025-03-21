// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

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
	Describe("ValueAt", func() {
		Describe("Happy Path", func() {

			It("Should return a uint8 value at the given index", func() {
				s := telem.NewSeriesV[uint8](1, 2, 3)
				Expect(telem.ValueAt[uint8](s, 0)).To(Equal(uint8(1)))
				Expect(telem.ValueAt[uint8](s, 1)).To(Equal(uint8(2)))
				Expect(telem.ValueAt[uint8](s, 2)).To(Equal(uint8(3)))
			})
			It("Should return a uint16 value at the given index", func() {
				s := telem.NewSeriesV[uint16](1, 2, 3)
				Expect(telem.ValueAt[uint16](s, 0)).To(Equal(uint16(1)))
				Expect(telem.ValueAt[uint16](s, 1)).To(Equal(uint16(2)))
				Expect(telem.ValueAt[uint16](s, 2)).To(Equal(uint16(3)))
			})
			It("Should return a uint32 value at the given index", func() {
				s := telem.NewSeriesV[uint32](1, 2, 3)
				Expect(telem.ValueAt[uint32](s, 0)).To(Equal(uint32(1)))
				Expect(telem.ValueAt[uint32](s, 1)).To(Equal(uint32(2)))
				Expect(telem.ValueAt[uint32](s, 2)).To(Equal(uint32(3)))
			})
			It("Should return a uint64 value at the given index", func() {
				s := telem.NewSeriesV[uint64](1, 2, 3)
				Expect(telem.ValueAt[uint64](s, 0)).To(Equal(uint64(1)))
				Expect(telem.ValueAt[uint64](s, 1)).To(Equal(uint64(2)))
				Expect(telem.ValueAt[uint64](s, 2)).To(Equal(uint64(3)))
			})
			It("Should return a int8 value at the given index", func() {
				s := telem.NewSeriesV[int8](1, 2, 3)
				Expect(telem.ValueAt[int8](s, 0)).To(Equal(int8(1)))
				Expect(telem.ValueAt[int8](s, 1)).To(Equal(int8(2)))
				Expect(telem.ValueAt[int8](s, 2)).To(Equal(int8(3)))
			})
			It("Should return a int16 value at the given index", func() {
				s := telem.NewSeriesV[int16](1, 2, 3)
				Expect(telem.ValueAt[int16](s, 0)).To(Equal(int16(1)))
				Expect(telem.ValueAt[int16](s, 1)).To(Equal(int16(2)))
				Expect(telem.ValueAt[int16](s, 2)).To(Equal(int16(3)))
			})
			It("Should return a int32 value at the given index", func() {
				s := telem.NewSeriesV[int32](1, 2, 3)
				Expect(telem.ValueAt[int32](s, 0)).To(Equal(int32(1)))
				Expect(telem.ValueAt[int32](s, 1)).To(Equal(int32(2)))
				Expect(telem.ValueAt[int32](s, 2)).To(Equal(int32(3)))
			})
			It("Should return a int64 value at the given index", func() {
				s := telem.NewSeriesV[int64](1, 2, 3)
				Expect(telem.ValueAt[int64](s, 0)).To(Equal(int64(1)))
				Expect(telem.ValueAt[int64](s, 1)).To(Equal(int64(2)))
				Expect(telem.ValueAt[int64](s, 2)).To(Equal(int64(3)))
			})
			It("Should return a float32 value at the given index", func() {
				s := telem.NewSeriesV[float32](1.0, 2.0, 3.0)
				Expect(telem.ValueAt[float32](s, 0)).To(Equal(float32(1.0)))
				Expect(telem.ValueAt[float32](s, 1)).To(Equal(float32(2.0)))
				Expect(telem.ValueAt[float32](s, 2)).To(Equal(float32(3.0)))
			})
			It("Should return a float64 value at the given index", func() {
				s := telem.NewSeriesV[float64](1.0, 2.0, 3.0)
				Expect(telem.ValueAt[float64](s, 0)).To(Equal(float64(1.0)))
				Expect(telem.ValueAt[float64](s, 1)).To(Equal(float64(2.0)))
				Expect(telem.ValueAt[float64](s, 2)).To(Equal(float64(3.0)))
			})
		})
		Describe("Negative Index", func() {
			It("Should return a value at the given negative index", func() {
				s := telem.NewSeriesV[int64](1, 2, 3)
				Expect(telem.ValueAt[int64](s, -1)).To(Equal(int64(3)))
				Expect(telem.ValueAt[int64](s, -2)).To(Equal(int64(2)))
				Expect(telem.ValueAt[int64](s, -3)).To(Equal(int64(1)))
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
	Describe("String", func() {
		Context("Empty Series", func() {
			It("Should properly format an empty series", func() {
				s := telem.Series{DataType: telem.Uint64T}
				Expect(s.String()).To(ContainSubstring("Len: 0"))
				Expect(s.String()).To(ContainSubstring("Contents: []"))
			})
		})

		Context("Short Series", func() {
			It("Should show all values for series with <= 12 elements", func() {
				s := telem.NewSeriesV[int64](1, 2, 3, 4, 5)
				Expect(s.String()).To(Equal("Series{TimeRange: 0ns - 0ns, DataType: int64, Len: 5, Size: 40 bytes, Contents: [1 2 3 4 5]}"))
			})

			It("Should properly format float values", func() {
				s := telem.NewSeriesV[float64](1.1, 2.2, 3.3)
				str := s.String()
				Expect(str).To(ContainSubstring("DataType: float64"))
				Expect(str).To(ContainSubstring("[1.1 2.2 3.3]"))
			})

			It("Should properly format string values", func() {
				s := telem.NewStringsV("a", "b", "c")
				str := s.String()
				Expect(str).To(ContainSubstring("DataType: string"))
				Expect(str).To(ContainSubstring("[a b c]"))
			})
		})

		Context("Long Series", func() {
			It("Should truncate series with > 12 elements", func() {
				values := make([]int64, 20)
				for i := range values {
					values[i] = int64(i + 1)
				}
				s := telem.NewSeriesV(values...)
				str := s.String()
				Expect(str).To(ContainSubstring("Len: 20"))
				Expect(str).To(ContainSubstring("[1 2 3 4 5 ... 16 17 18 19 20]"))
			})

			It("Should truncate long float series", func() {
				values := make([]float64, 15)
				for i := range values {
					values[i] = float64(i) + 0.5
				}
				s := telem.NewSeriesV(values...)
				str := s.String()
				Expect(str).To(ContainSubstring("[0.5 1.5 2.5 3.5 4.5 ... 10.5 11.5 12.5 13.5 14.5]"))
			})

			It("Should truncate long string series", func() {
				values := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n"}
				s := telem.NewStringsV(values...)
				str := s.String()
				Expect(str).To(ContainSubstring("[a b c d e ... j k l m n]"))
			})
		})

		Context("Different Data Types", func() {
			It("Should handle uint8 values", func() {
				s := telem.NewSeriesV[uint8](1, 2, 3)
				Expect(s.String()).To(ContainSubstring("DataType: uint8"))
				Expect(s.String()).To(ContainSubstring("[1 2 3]"))
			})

			It("Should handle int16 values", func() {
				s := telem.NewSeriesV[int16](1000, 2000, 3000)
				Expect(s.String()).To(ContainSubstring("DataType: int16"))
				Expect(s.String()).To(ContainSubstring("[1000 2000 3000]"))
			})
		})
	})
})
