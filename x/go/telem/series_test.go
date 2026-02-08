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
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	xunsafe "github.com/synnaxlabs/x/unsafe"
)

func marshalSeriesTest[T telem.Sample](data []T, dt telem.DataType) func() {
	return func() {
		s := telem.NewSeries(data)
		ExpectWithOffset(1, s.DataType).To(Equal(dt))
		ExpectWithOffset(1, s.Len()).To(Equal(int64(len(data))))
		ExpectWithOffset(1, telem.UnmarshalSeries[T](s)).To(Equal(data))
	}
}

func marshalUnmarshalSliceTest[T telem.Sample](data []T, dt telem.DataType) func() {
	return func() {
		s := telem.MarshalSlice(data)
		Expect(telem.UnmarshalSlice[T](s, dt)).To(Equal(data))
	}
}

func valueAtTest[T telem.Sample](value T, dt telem.DataType) func() {
	return func() {
		s := telem.NewSeriesV(value)
		ExpectWithOffset(1, s.DataType).To(Equal(dt))
		ExpectWithOffset(1, telem.ValueAt[T](s, 0)).To(Equal(value))
	}
}

var _ = Describe("Series", func() {
	Describe("Len", func() {
		It("Should correctly return the number of samples in a series with a fixed length data type", func() {
			s := telem.NewSeriesV[int64](1, 2, 3)
			Expect(s.Len()).To(Equal(int64(3)))
		})
		It("Should correctly return the number of samples in a series with a variable length data type", func() {
			s := telem.NewSeriesVariableV("bob", "alice", "charlie")
			Expect(s.Len()).To(Equal(int64(3)))
		})
	})

	Describe("Factory", func() {
		Describe("MarshalSeries", func() {
			Specify("float64", marshalSeriesTest([]float64{1.0, 2.0, 3.0}, telem.Float64T))
			Specify("float32", marshalSeriesTest([]float32{1.0, 2.0, 3.0}, telem.Float32T))
			Specify("int64", marshalSeriesTest([]int64{1, 2, 3}, telem.Int64T))
			Specify("int32", marshalSeriesTest([]int32{1, 2, 3}, telem.Int32T))
			Specify("int16", marshalSeriesTest([]int16{1, 2, 3}, telem.Int16T))
			Specify("int8", marshalSeriesTest([]int8{1, 2, 3}, telem.Int8T))
			Specify("uint64", marshalSeriesTest([]uint64{1, 2, 3}, telem.Uint64T))
			Specify("uint32", marshalSeriesTest([]uint32{1, 2, 3}, telem.Uint32T))
			Specify("uint16", marshalSeriesTest([]uint16{1, 2, 3}, telem.Uint16T))
			Specify("uint8", marshalSeriesTest([]uint8{1, 2, 3}, telem.Uint8T))
			Specify("timestamp", marshalSeriesTest([]telem.TimeStamp{1, 2, 3}, telem.TimeStampT))
		})

		Describe("MarshalSlice", func() {
			Specify("float64", marshalUnmarshalSliceTest([]float64{1.0, 2.0, 3.0}, telem.Float64T))
			Specify("float32", marshalUnmarshalSliceTest([]float32{1.0, 2.0, 3.0}, telem.Float32T))
			Specify("int64", marshalUnmarshalSliceTest([]int64{1, 2, 3}, telem.Int64T))
			Specify("int32", marshalUnmarshalSliceTest([]int32{1, 2, 3}, telem.Int32T))
			Specify("int16", marshalUnmarshalSliceTest([]int16{1, 2, 3}, telem.Int16T))
			Specify("int8", marshalUnmarshalSliceTest([]int8{1, 2, 3}, telem.Int8T))
			Specify("uint64", marshalUnmarshalSliceTest([]uint64{1, 2, 3}, telem.Uint64T))
			Specify("uint32", marshalUnmarshalSliceTest([]uint32{1, 2, 3}, telem.Uint32T))
			Specify("uint16", marshalUnmarshalSliceTest([]uint16{1, 2, 3}, telem.Uint16T))
			Specify("uint8", marshalUnmarshalSliceTest([]uint8{1, 2, 3}, telem.Uint8T))
			Specify("timestamp", marshalUnmarshalSliceTest([]telem.TimeStamp{1, 2, 3}, telem.TimeStampT))
		})

		Describe("Series ValueAt/SetValueAt", func() {
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
		})

		Describe("String", func() {
			It("Should correctly marshal a static string data structure", func() {
				data := "hello"
				s := telem.NewSeriesVariableV(data)
				Expect(string(s.At(0))).To(Equal("hello"))
				Expect(s.Len()).To(BeEquivalentTo(1))
				unmarshalled := telem.UnmarshalVariable[string](s.Data)
				Expect(unmarshalled).To(Equal([]string{"hello"}))
			})
			It("Should correctly marshal empty strings", func() {
				s := telem.NewSeriesVariableV("", "", "")
				v := s.At(0)
				Expect(string(v)).To(Equal(""))
				Expect(s.Len()).To(BeEquivalentTo(3))
				unmarshalled := telem.UnmarshalVariable[string](s.Data)
				Expect(unmarshalled).To(Equal([]string{"", "", ""}))
			})
			It("Should correctly marshal and unmarshal byte slices", func() {
				data := [][]byte{{1, 2, 3}, {4, 5, 6}}
				s := telem.NewSeriesVariable(data)
				Expect(s.At(0)).To(Equal([]byte{1, 2, 3}))
				Expect(s.Len()).To(BeEquivalentTo(2))
				unmarshalled := telem.UnmarshalVariable[[]byte](s.Data)
				Expect(unmarshalled).To(Equal([][]byte{{1, 2, 3}, {4, 5, 6}}))
			})
		})

		Describe("JSON", func() {
			It("Should correctly marshal a static JSON data structure", func() {
				data := map[string]any{
					"cat": map[string]any{
						"one": "two",
					},
				}
				s := MustSucceed(telem.NewSeriesJSONV(data))
				v := s.At(0)
				Expect(string(v)).To(Equal(`{"cat":{"one":"two"}}`))
				Expect(s.Len()).To(Equal(int64(1)))
				unmarshalled := telem.UnmarshalVariable[string](s.Data)
				Expect(unmarshalled).To(Equal([]string{`{"cat":{"one":"two"}}`}))
			})
			It("Should correctly marshal a slice of JSON values", func() {
				data := []int{1, 2, 3}
				s := MustSucceed(telem.NewSeriesJSONV(data))
				v := s.At(0)
				Expect(string(v)).To(Equal(`[1,2,3]`))
				Expect(s.Len()).To(Equal(int64(1)))
				unmarshalled := MustSucceed(telem.UnmarshalJSON[[]int](s.Data))
				Expect(unmarshalled).To(Equal([][]int{{1, 2, 3}}))
			})
			It("should return an error when marshalling invalid JSON", func() {
				data := []chan int{make(chan int), make(chan int), make(chan int)}
				Expect(telem.NewSeriesJSONV(data)).Error().
					To(MatchError(ContainSubstring("json: unsupported type: chan int")))
			})
			It("should return an error when marshalling invalid JSON", func() {
				data := []chan int{make(chan int)}
				Expect(telem.MarshalJSON(data)).Error().
					To(MatchError(ContainSubstring("json: unsupported type: chan int")))
			})
			It("Should return an error when unmarshalling invalid JSON", func() {
				data := []int{1, 2, 3}
				s := MustSucceed(telem.NewSeriesJSONV(data))
				Expect(telem.UnmarshalJSON[string](s.Data)).Error().
					To(MatchError(ContainSubstring(
						"json: cannot unmarshal array into Go value of type string",
					)))
			})
		})

		Describe("Arrange", func() {
			It("Should create a series with the correct values for int64", func() {
				s := telem.Arrange[int64](0, 5, 2)
				Expect(s.Len()).To(Equal(int64(5)))
				Expect(s.DataType).To(Equal(telem.Int64T))
				data := telem.UnmarshalSeries[int64](s)
				Expect(data).To(Equal([]int64{0, 2, 4, 6, 8}))
			})
			It("Should create a series with the correct values for float64", func() {
				s := telem.Arrange(0.0, 5, 0.5)
				Expect(s.Len()).To(Equal(int64(5)))
				Expect(s.DataType).To(Equal(telem.Float64T))
				data := telem.UnmarshalSeries[float64](s)
				Expect(data).To(Equal([]float64{0.0, 0.5, 1.0, 1.5, 2.0}))
			})
			It("Should create a series with a single value when count is 1", func() {
				s := telem.Arrange[int32](10, 1, 5)
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(s.DataType).To(Equal(telem.Int32T))
				data := telem.UnmarshalSeries[int32](s)
				Expect(data).To(Equal([]int32{10}))
			})
			It("Should create a series with negative spacing", func() {
				s := telem.Arrange[int64](10, 5, -2)
				Expect(s.Len()).To(Equal(int64(5)))
				data := telem.UnmarshalSeries[int64](s)
				Expect(data).To(Equal([]int64{10, 8, 6, 4, 2}))
			})
			It("Should panic when count is less than 0", func() {
				Expect(func() {
					telem.Arrange[int64](0, -1, 1)
				}).To(Panic())
			})
		})
	})

	Describe("MakeSeries", func() {
		It("Should make a series with the specified length", func() {
			s := telem.MakeSeries(telem.Int64T, 20)
			Expect(s.Len()).To(Equal(int64(20)))
			Expect(s.Size()).To(Equal(telem.Byte * 20 * 8))
		})
	})

	Describe("At", func() {
		Context("Fixed Density", func() {
			It("Should return the the value at the given index", func() {
				s := telem.NewSeriesV[uint8](1, 2, 3)
				Expect(s.At(0)).To(Equal([]byte{1}))
				Expect(s.At(1)).To(Equal([]byte{2}))
				Expect(s.At(2)).To(Equal([]byte{3}))
			})

			It("Should panic when the index is out of bounds", func() {
				s := telem.NewSeriesV[uint8](1, 2, 3)
				Expect(func() {
					s.At(5)
				}).To(Panic())
				Expect(func() {
					s.At(-10)
				}).To(Panic())
			})
		})

		Context("Variable Density", func() {
			It("Should return the value at the given index", func() {
				s := telem.NewSeriesVariableV("a", "b", "c")
				Expect(s.At(0)).To(Equal([]byte("a")))
				Expect(s.At(1)).To(Equal([]byte("b")))
				Expect(s.At(2)).To(Equal([]byte("c")))
			})

			It("Should panic when the index is out of bounds", func() {
				s := telem.NewSeriesVariableV("a", "b", "c")
				Expect(func() {
					s.At(5)
				}).To(Panic())
				Expect(func() {
					s.At(-10)
				}).To(Panic())
			})
		})
	})

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
				Expect(s.String()).To(Equal("Series{Alignment: 0-0, TimeRange: 1970-01-01T00:00:00Z - 00:00:00 (0s), DataType: int64, Len: 5, Size: 40 bytes, Contents: [1 2 3 4 5]}"))
			})

			It("Should properly format float values", func() {
				s := telem.NewSeriesV(1.1, 2.2, 3.3)
				str := s.String()
				Expect(str).To(ContainSubstring("DataType: float64"))
				Expect(str).To(ContainSubstring("[1.1 2.2 3.3]"))
			})

			It("Should properly format string values", func() {
				s := telem.NewSeriesVariableV("a", "b", "c")
				str := s.String()
				Expect(str).To(ContainSubstring("DataType: string"))
				Expect(str).To(ContainSubstring("[a b c]"))
			})
		})

		DescribeTable("DataString", func(s telem.Series, expected string) {
			Expect(s.DataString()).To(Equal(expected))
		},
			Entry("uint8", telem.NewSeriesV[uint8](1, 2, 3), "[1 2 3]"),
			Entry("uint16", telem.NewSeriesV[uint16](1, 2, 3), "[1 2 3]"),
			Entry("uint32", telem.NewSeriesV[uint32](1, 2, 3), "[1 2 3]"),
			Entry("uint64", telem.NewSeriesV[uint64](1, 2, 3), "[1 2 3]"),
			Entry("int8", telem.NewSeriesV[int8](1, 2, 3), "[1 2 3]"),
			Entry("int16", telem.NewSeriesV[int16](1, 2, 3), "[1 2 3]"),
			Entry("int32", telem.NewSeriesV[int32](1, 2, 3), "[1 2 3]"),
			Entry("int64", telem.NewSeriesV[int64](1, 2, 3), "[1 2 3]"),
			Entry("float32", telem.NewSeriesV[float32](1.0, 2.0, 3.0), "[1 2 3]"),
			Entry("float64", telem.NewSeriesV(1.0, 2.0, 3.0), "[1 2 3]"),
			Entry("string", telem.NewSeriesVariableV("a", "b", "c"), "[a b c]"),
			Entry("json", MustSucceed(telem.NewSeriesJSONV(map[string]any{"a": 1, "b": 2, "c": 3})), `[{"a":1,"b":2,"c":3}]`),
			Entry("timestamp", telem.NewSeriesSecondsTSV(1, 2, 3), "[1970-01-01T00:00:01Z +1s +2s]"),
		)

		Context("Long Series", func() {
			It("Should truncate series with > 14 elements", func() {
				values := make([]int64, 20)
				for i := range values {
					values[i] = int64(i + 1)
				}
				s := telem.NewSeriesV(values...)
				str := s.String()
				Expect(str).To(ContainSubstring("Len: 20"))
				Expect(str).To(ContainSubstring("[1 2 3 4 5 6 ... 15 16 17 18 19 20]"))
			})

			It("Should truncate long float series", func() {
				values := make([]float64, 15)
				for i := range values {
					values[i] = float64(i) + 0.5
				}
				s := telem.NewSeriesV(values...)
				str := s.String()
				Expect(str).To(ContainSubstring("[0.5 1.5 2.5 3.5 4.5 5.5 ... 9.5 10.5 11.5 12.5 13.5 14.5]"))
			})

			It("Should truncate long string series", func() {
				values := []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n"}
				s := telem.NewSeriesVariableV(values...)
				str := s.String()
				Expect(str).To(ContainSubstring("[a b c d e f ... i j k l m n]"))
			})

			It("Should truncate a long timestamp series", func() {
				values := telem.NewSeriesSecondsTSV(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20)
				Expect(values.DataString()).To(Equal("[1970-01-01T00:00:01Z +1s +2s +3s +4s +5s ... +14s +15s +16s +17s +18s +19s]"))
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

	Describe("Downsample", func() {
		Context("Fixed Length Data Types", func() {
			It("Should correctly downsample a series with a factor of 2", func() {
				original := telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7, 8)
				downsampled := original.Downsample(2)

				Expect(downsampled.Len()).To(Equal(int64(4)))
				Expect(telem.UnmarshalSeries[int64](downsampled)).To(Equal([]int64{1, 3, 5, 7}))
				Expect(downsampled.DataType).To(Equal(original.DataType))
				Expect(downsampled.TimeRange).To(Equal(original.TimeRange))
				Expect(downsampled.Alignment).To(Equal(original.Alignment))
			})

			It("Should correctly downsample a series with a factor of 3", func() {
				original := telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7, 8, 9)
				downsampled := original.Downsample(3)

				Expect(downsampled.Len()).To(Equal(int64(3)))
				Expect(telem.UnmarshalSeries[int64](downsampled)).To(Equal([]int64{1, 4, 7}))
			})

			It("Should work when the factor is not an even multiple of the length", func() {
				original := telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
				downsampled := original.Downsample(3)

				Expect(downsampled.Len()).To(Equal(int64(4)))
				Expect(telem.UnmarshalSeries[int64](downsampled)).To(Equal([]int64{1, 4, 7, 10}))
			})

			It("Should work with different numeric types", func() {
				original := telem.NewSeriesV(1.1, 2.2, 3.3, 4.4, 5.5, 6.6)
				downsampled := original.Downsample(2)

				Expect(downsampled.Len()).To(Equal(int64(3)))
				Expect(telem.UnmarshalSeries[float64](downsampled)).To(Equal([]float64{1.1, 3.3, 5.5}))
			})

			It("Should preserve alignment information", func() {
				original := telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6)
				original.Alignment = telem.NewAlignment(1, 5)
				downsampled := original.Downsample(2)

				Expect(downsampled.Alignment).To(Equal(original.Alignment))
			})

			It("Should preserve time range information", func() {
				original := telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6)
				original.TimeRange = telem.TimeRange{Start: 100, End: 600}
				downsampled := original.Downsample(2)

				Expect(downsampled.TimeRange).To(Equal(original.TimeRange))
			})
		})

		Context("Variable Length Data Types", func() {
			It("Should correctly down sample a string series", func() {
				original := telem.NewSeriesVariableV("a", "b", "c", "d", "e", "f")
				downsampled := original.Downsample(2)

				Expect(downsampled.Len()).To(Equal(int64(3)))
				Expect(telem.UnmarshalVariable[string](downsampled.Data)).To(Equal([]string{"a", "c", "e"}))
			})

			It("Should correctly down sample a JSON series", func() {
				data := []map[string]any{
					{"id": 1},
					{"id": 2},
					{"id": 3},
					{"id": 4},
				}

				s := MustSucceed(telem.NewSeriesJSONV(data...))
				downsampled := s.Downsample(2)
				Expect(downsampled.Len()).To(Equal(int64(2)))
				split := bytes.Split(downsampled.Data, []byte("\n"))
				Expect(len(split)).To(Equal(3)) // 2 items + empty string after last newline
			})
		})

		Context("Edge Cases", func() {
			It("Should return the original series if factor is <= 1", func() {
				original := telem.NewSeriesV[int64](1, 2, 3)
				downsampled := original.Downsample(0)
				Expect(downsampled).To(Equal(original))
				downsampled = original.Downsample(1)
				Expect(downsampled).To(Equal(original))
				downsampled = original.Downsample(-1)
				Expect(downsampled).To(Equal(original))
			})

			It("Should return the maximum possible downSampling if series length is <= factor", func() {
				original := telem.NewSeriesV[int64](1, 2, 3)
				downsampled := original.Downsample(3)
				Expect(downsampled.Len()).To(Equal(int64(1)))
				Expect(telem.UnmarshalSeries[int64](downsampled)).To(Equal([]int64{1}))
				downsampled = original.Downsample(10)
				Expect(downsampled.Len()).To(Equal(int64(1)))
				Expect(telem.UnmarshalSeries[int64](downsampled)).To(Equal([]int64{1}))
			})

			It("Should handle empty series correctly", func() {
				original := telem.Series{DataType: telem.Int64T}
				downsampled := original.Downsample(2)
				Expect(downsampled).To(Equal(original))
				Expect(downsampled.Len()).To(Equal(int64(0)))
			})
		})
	})

	Describe("Resize", func() {
		Context("Fixed Length Data Types", func() {
			It("Should shrink a series by truncating data", func() {
				s := telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6)
				s.Resize(3)
				Expect(s.Len()).To(Equal(int64(3)))
				Expect(telem.UnmarshalSeries[int64](s)).To(Equal([]int64{1, 2, 3}))
			})

			It("Should grow a series by extending with zeros", func() {
				s := telem.NewSeriesV[int64](1, 2, 3)
				s.Resize(6)
				Expect(s.Len()).To(Equal(int64(6)))
				Expect(telem.UnmarshalSeries[int64](s)).To(Equal([]int64{1, 2, 3, 0, 0, 0}))
			})

			It("Should be a no-op when resizing to the same length", func() {
				original := telem.NewSeriesV[int64](1, 2, 3, 4)
				originalData := make([]byte, len(original.Data))
				copy(originalData, original.Data)
				original.Resize(4)
				Expect(original.Len()).To(Equal(int64(4)))
				Expect(original.Data).To(Equal(originalData))
			})

			It("Should work with different numeric types", func() {
				s := telem.NewSeriesV(1.1, 2.2, 3.3, 4.4, 5.5)
				s.Resize(3)
				Expect(s.Len()).To(Equal(int64(3)))
				Expect(telem.UnmarshalSeries[float64](s)).To(Equal([]float64{1.1, 2.2, 3.3}))
			})

			It("Should work with uint8", func() {
				s := telem.NewSeriesV[uint8](1, 2, 3)
				s.Resize(5)
				Expect(s.Len()).To(Equal(int64(5)))
				Expect(telem.UnmarshalSeries[uint8](s)).To(Equal([]uint8{1, 2, 3, 0, 0}))
			})

			It("Should work with float32", func() {
				s := telem.NewSeriesV[float32](1.0, 2.0, 3.0, 4.0)
				s.Resize(2)
				Expect(s.Len()).To(Equal(int64(2)))
				Expect(telem.UnmarshalSeries[float32](s)).To(Equal([]float32{1.0, 2.0}))
			})

			It("Should work with timestamps", func() {
				s := telem.NewSeriesSecondsTSV(1, 2, 3)
				s.Resize(5)
				Expect(s.Len()).To(Equal(int64(5)))
				result := telem.UnmarshalSeries[telem.TimeStamp](s)
				Expect(result[0]).To(Equal(telem.TimeStamp(1 * telem.Second)))
				Expect(result[1]).To(Equal(telem.TimeStamp(2 * telem.Second)))
				Expect(result[2]).To(Equal(telem.TimeStamp(3 * telem.Second)))
				Expect(result[3]).To(Equal(telem.TimeStamp(0)))
				Expect(result[4]).To(Equal(telem.TimeStamp(0)))
			})

			It("Should resize to zero length", func() {
				s := telem.NewSeriesV[int64](1, 2, 3, 4, 5)
				s.Resize(0)
				Expect(s.Len()).To(Equal(int64(0)))
				Expect(len(s.Data)).To(Equal(0))
			})

			It("Should handle resizing an empty series", func() {
				s := telem.Series{DataType: telem.Int64T}
				s.Resize(3)
				Expect(s.Len()).To(Equal(int64(3)))
				Expect(telem.UnmarshalSeries[int64](s)).To(Equal([]int64{0, 0, 0}))
			})

			It("Should handle large resize operations", func() {
				s := telem.NewSeriesV[int32](1, 2, 3)
				s.Resize(1000)
				Expect(s.Len()).To(Equal(int64(1000)))
				result := telem.UnmarshalSeries[int32](s)
				Expect(result[0]).To(Equal(int32(1)))
				Expect(result[1]).To(Equal(int32(2)))
				Expect(result[2]).To(Equal(int32(3)))
				for i := 3; i < 1000; i++ {
					Expect(result[i]).To(Equal(int32(0)))
				}
			})
		})

		Context("Variable Length Data Types", func() {
			It("Should panic when trying to resize a string series", func() {
				s := telem.NewSeriesVariableV("a", "b", "c")
				Expect(func() { s.Resize(5) }).To(Panic())
			})

			It("Should panic when trying to resize a JSON series", func() {
				s := MustSucceed(telem.NewSeriesJSONV(map[string]any{"a": 1}))
				Expect(func() { s.Resize(3) }).To(Panic())
			})
		})

		Context("Error Cases", func() {
			It("Should panic when resizing to a negative length", func() {
				s := telem.NewSeriesV[int64](1, 2, 3)
				Expect(func() { s.Resize(-1) }).To(Panic())
			})

			It("Should panic with a meaningful message for negative length", func() {
				s := telem.NewSeriesV[int64](1, 2, 3)
				defer func() {
					if r := recover(); r != nil {
						Expect(r).To(Equal("cannot resize series to negative length"))
					}
				}()
				s.Resize(-10)
			})

			It("Should panic with a meaningful message for variable-density types", func() {
				s := telem.NewSeriesVariableV("a", "b", "c")
				defer func() {
					if r := recover(); r != nil {
						Expect(r).To(Equal("cannot resize variable-density series"))
					}
				}()
				s.Resize(5)
			})
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
				s2 := telem.NewSeriesVariableV("a", "b", "c")
				Expect(func() { telem.NewMultiSeriesV(s1, s2) }).To(Panic())
			})
		})

		Describe("AlignmentBounds", func() {
			It("Should return the alignment bounds of the multi-series", func() {
				s1 := telem.NewSeriesSecondsTSV(1, 2, 3)
				s1.Alignment = telem.NewAlignment(0, 0)
				s2 := telem.NewSeriesSecondsTSV(4, 5, 6)
				s2.Alignment = telem.NewAlignment(0, 3)
				ms := telem.NewMultiSeriesV(s1, s2)
				Expect(ms.AlignmentBounds()).To(Equal(telem.AlignmentBounds{
					Lower: 0,
					Upper: 6,
				}))
			})

			It("Should return [0, 0) for an empty multi-series", func() {
				var ms telem.MultiSeries
				Expect(ms.AlignmentBounds()).To(Equal(telem.AlignmentBounds{
					Lower: 0,
					Upper: 0,
				}))
			})
		})

		Describe("TimeRange", func() {
			It("Should return the time range of the multi-series", func() {
				s1 := telem.NewSeriesSecondsTSV(1, 2, 3)
				s1.TimeRange = telem.TimeRange{Start: 0, End: 3}
				s2 := telem.NewSeriesSecondsTSV(4, 5, 6)
				s2.TimeRange = telem.TimeRange{Start: 3, End: 6}
				ms := telem.NewMultiSeriesV(s1, s2)
				Expect(ms.TimeRange()).To(Equal(telem.TimeRange{
					Start: s1.TimeRange.Start,
					End:   s2.TimeRange.End,
				}))
			})

			It("Should return a zero time range when the multi-series is empty", func() {
				ms := telem.MultiSeries{}
				Expect(ms.TimeRange()).To(Equal(telem.TimeRangeZero))
			})
		})

		Describe("Append", func() {
			It("Should append a series to the multi-series", func() {
				s1 := telem.NewSeriesSecondsTSV(1, 2, 3)
				s2 := telem.NewSeriesSecondsTSV(4, 5, 6)
				ms := telem.NewMultiSeriesV(s1)
				ms = ms.Append(s2)
				Expect(ms.Len()).To(Equal(int64(6)))
				Expect(ms.Series[0].Alignment).To(Equal(s1.Alignment))
				Expect(ms.Series[1].Alignment).To(Equal(s2.Alignment))
			})

			It("Should panic if the series data types do not match", func() {
				s1 := telem.NewSeriesSecondsTSV(1, 2, 3)
				s2 := telem.NewSeriesV[int32](1, 2, 3)
				ms := telem.NewMultiSeriesV(s1)
				Expect(func() {
					ms = ms.Append(s2)
				}).To(Panic())
			})

			It("Should not panic when appending to an empty series", func() {
				s1 := telem.NewSeriesSecondsTSV(1, 2, 3)
				ms := telem.MultiSeries{}
				Expect(func() {
					ms.Append(s1)
				}).NotTo(Panic())
			})
		})

		Describe("FilterGreaterThanOrEqualTo", func() {
			It("Should remove series with alignment bounds that are less than the target threshold", func() {
				s1 := telem.NewSeriesSecondsTSV(1, 2, 3)
				s1.Alignment = telem.NewAlignment(0, 0)
				s2 := telem.NewSeriesSecondsTSV(4, 5, 6)
				s2.Alignment = telem.NewAlignment(0, 3)
				ms := telem.NewMultiSeriesV(s1, s2)
				ms = ms.FilterGreaterThanOrEqualTo(telem.NewAlignment(0, 3))
				Expect(ms.Len()).To(Equal(int64(3)))
				Expect(ms.Series[0].Alignment).To(Equal(s2.Alignment))
			})

			It("Should correctly handle an empty multi-series", func() {
				var ms telem.MultiSeries
				Expect(ms.FilterGreaterThanOrEqualTo(0).Len()).To(Equal(int64(0)))
			})

			It("Should keep all series when alignment bounds is very low", func() {
				s1 := telem.NewSeriesSecondsTSV(1, 2, 3)
				s1.Alignment = 500
				s2 := telem.NewSeriesSecondsTSV(4, 5, 6)
				s2.Alignment = 5000
				ms := telem.NewMultiSeriesV(s1, s2)
				ms = ms.FilterGreaterThanOrEqualTo(5)
				Expect(ms.Len()).To(Equal(int64(6)))
			})

			It("Should filter all series when alignment bounds is very high", func() {
				s1 := telem.NewSeriesSecondsTSV(1, 2, 3)
				s1.Alignment = 0
				s2 := telem.NewSeriesSecondsTSV(4, 5, 6)
				s2.Alignment = 3
				ms := telem.NewMultiSeriesV(s1, s2)
				ms = ms.FilterGreaterThanOrEqualTo(5000)
				Expect(ms).To(Equal(telem.MultiSeries{}))
			})
		})

		Describe("Len", func() {
			It("Should return the accumulated length of all series", func() {
				s1 := telem.NewSeriesSecondsTSV(1, 2, 3)
				s2 := telem.NewSeriesSecondsTSV(4, 5, 6)
				ms := telem.NewMultiSeriesV(s1, s2)
				Expect(ms.Len()).To(Equal(int64(6)))
			})

			It("Should return 0 if there are no series", func() {
				ms := telem.MultiSeries{}
				Expect(ms.Len()).To(Equal(int64(0)))
			})
		})

		Describe("DataType", func() {
			It("Should return the data type of the multi-series", func() {
				s1 := telem.NewSeriesSecondsTSV(1, 2, 3)
				s2 := telem.NewSeriesSecondsTSV(4, 5, 6)
				ms := telem.NewMultiSeriesV(s1, s2)
				Expect(ms.DataType()).To(Equal(telem.TimeStampT))
			})
		})

		Describe("Data", func() {
			It("Should return the aggregate data of the multi-series as a single byte array", func() {
				s1 := telem.NewSeriesV[uint8](1, 2, 3)
				s2 := telem.NewSeriesV[uint8](4, 5, 6)
				ms := telem.NewMultiSeriesV(s1, s2)
				Expect(ms.Data()).To(Equal([]byte{1, 2, 3, 4, 5, 6}))
			})

			It("Should return an empty byte array if there are no series in the frame", func() {
				ts := telem.NewMultiSeriesV()
				Expect(ts.Data()).To(HaveLen(0))
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

	Describe("Samples", func() {
		It("iterates fixed length correctly", func() {
			s := telem.NewSeriesV[int64](1, 2, 3, 4, 5)
			values := make([]int64, 0, 5)
			for sample := range s.Samples() {
				values = append(values, MustSucceed(xunsafe.CastBytes[int64](sample)))
			}
			Expect(values).To(Equal([]int64{1, 2, 3, 4, 5}))
		})

		It("iterates variable length correctly", func() {
			s := telem.NewSeriesVariableV("foo", "bar", "baz")
			values := make([]string, 0, 3)
			for sample := range s.Samples() {
				values = append(values, string(sample))
			}
			Expect(values).To(Equal([]string{"foo", "bar", "baz"}))
		})

		It("allows early termination", func() {
			s := telem.NewSeriesV[int64](1, 2, 3, 4, 5)
			values := make([]int64, 0, 3)
			count := 0
			for sample := range s.Samples() {
				values = append(values, MustSucceed(xunsafe.CastBytes[int64](sample)))
				count++
				if count > 2 {
					break
				}
			}
			Expect(values).To(Equal([]int64{1, 2, 3}))
		})

		It("Should allow for early termination in variable length series", func() {
			s := telem.NewSeriesVariableV("foo", "bar", "baz")
			values := make([]string, 0, 3)
			count := 0
			for sample := range s.Samples() {
				count++
				values = append(values, string(sample))
				if count > 1 {
					break
				}
			}
			Expect(values).To(Equal([]string{"foo", "bar"}))
		})

		It("handles empty series", func() {
			s := telem.Series{DataType: telem.Int64T}
			count := 0
			s.Samples()(func(sample []byte) bool {
				count++
				return true
			})
			Expect(count).To(Equal(0))
		})
	})

	Describe("NewSeriesFromAny", func() {
		Describe("Int input types", func() {
			It("Should create a series from an int value", func() {
				s := telem.NewSeriesFromAny(42, telem.Int64T)
				Expect(s.DataType).To(Equal(telem.Int64T))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[int64](s, 0)).To(Equal(int64(42)))
			})

			It("Should create a series from an int64 value", func() {
				s := telem.NewSeriesFromAny(int64(100), telem.Int32T)
				Expect(s.DataType).To(Equal(telem.Int32T))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[int32](s, 0)).To(Equal(int32(100)))
			})

			It("Should create a series from an int32 value", func() {
				s := telem.NewSeriesFromAny(int32(50), telem.Int16T)
				Expect(s.DataType).To(Equal(telem.Int16T))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[int16](s, 0)).To(Equal(int16(50)))
			})

			It("Should create a series from an int16 value", func() {
				s := telem.NewSeriesFromAny(int16(25), telem.Int8T)
				Expect(s.DataType).To(Equal(telem.Int8T))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[int8](s, 0)).To(Equal(int8(25)))
			})

			It("Should create a series from an int8 value", func() {
				s := telem.NewSeriesFromAny(int8(12), telem.Int64T)
				Expect(s.DataType).To(Equal(telem.Int64T))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[int64](s, 0)).To(Equal(int64(12)))
			})
		})

		Describe("Uint input types", func() {
			It("Should create a series from a uint64 value", func() {
				s := telem.NewSeriesFromAny(uint64(200), telem.Uint32T)
				Expect(s.DataType).To(Equal(telem.Uint32T))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[uint32](s, 0)).To(Equal(uint32(200)))
			})

			It("Should create a series from a uint32 value", func() {
				s := telem.NewSeriesFromAny(uint32(150), telem.Uint16T)
				Expect(s.DataType).To(Equal(telem.Uint16T))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[uint16](s, 0)).To(Equal(uint16(150)))
			})

			It("Should create a series from a uint16 value", func() {
				s := telem.NewSeriesFromAny(uint16(75), telem.Uint8T)
				Expect(s.DataType).To(Equal(telem.Uint8T))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[uint8](s, 0)).To(Equal(uint8(75)))
			})

			It("Should create a series from a uint8 value", func() {
				s := telem.NewSeriesFromAny(uint8(37), telem.Uint64T)
				Expect(s.DataType).To(Equal(telem.Uint64T))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[uint64](s, 0)).To(Equal(uint64(37)))
			})
		})

		Describe("Float input types", func() {
			It("Should create a series from a float64 value", func() {
				s := telem.NewSeriesFromAny(3.14159, telem.Float64T)
				Expect(s.DataType).To(Equal(telem.Float64T))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[float64](s, 0)).To(BeNumerically("~", 3.14159))
			})

			It("Should create a series from a float32 value", func() {
				s := telem.NewSeriesFromAny(float32(2.718), telem.Float32T)
				Expect(s.DataType).To(Equal(telem.Float32T))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[float32](s, 0)).To(Equal(float32(2.718)))
			})
		})

		Describe("Type conversions", func() {
			It("Should convert int to float64", func() {
				s := telem.NewSeriesFromAny(42, telem.Float64T)
				Expect(s.DataType).To(Equal(telem.Float64T))
				Expect(telem.ValueAt[float64](s, 0)).To(Equal(float64(42)))
			})

			It("Should convert float64 to int", func() {
				s := telem.NewSeriesFromAny(42.7, telem.Int64T)
				Expect(s.DataType).To(Equal(telem.Int64T))
				Expect(telem.ValueAt[int64](s, 0)).To(Equal(int64(42)))
			})

			It("Should convert uint to int", func() {
				s := telem.NewSeriesFromAny(uint32(100), telem.Int32T)
				Expect(s.DataType).To(Equal(telem.Int32T))
				Expect(telem.ValueAt[int32](s, 0)).To(Equal(int32(100)))
			})

			It("Should convert int to uint", func() {
				s := telem.NewSeriesFromAny(int32(50), telem.Uint32T)
				Expect(s.DataType).To(Equal(telem.Uint32T))
				Expect(telem.ValueAt[uint32](s, 0)).To(Equal(uint32(50)))
			})

			It("Should convert float32 to float64", func() {
				s := telem.NewSeriesFromAny(float32(1.5), telem.Float64T)
				Expect(s.DataType).To(Equal(telem.Float64T))
				Expect(telem.ValueAt[float64](s, 0)).To(BeNumerically("~", 1.5))
			})

			It("Should convert float64 to float32", func() {
				s := telem.NewSeriesFromAny(2.5, telem.Float32T)
				Expect(s.DataType).To(Equal(telem.Float32T))
				Expect(telem.ValueAt[float32](s, 0)).To(BeNumerically("~", 2.5))
			})
		})

		Describe("Edge cases", func() {
			It("Should handle zero values", func() {
				s := telem.NewSeriesFromAny(0, telem.Int64T)
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[int64](s, 0)).To(Equal(int64(0)))
			})

			It("Should handle negative values", func() {
				s := telem.NewSeriesFromAny(-42, telem.Int32T)
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.ValueAt[int32](s, 0)).To(Equal(int32(-42)))
			})

			It("Should handle large uint64 values", func() {
				largeVal := uint64(18446744073709551615)
				s := telem.NewSeriesFromAny(largeVal, telem.Uint64T)
				Expect(s.Len()).To(Equal(int64(1)))
			})
		})

		Describe("String conversions", func() {
			It("Should convert string to StringT", func() {
				s := telem.NewSeriesFromAny("hello", telem.StringT)
				Expect(s.DataType).To(Equal(telem.StringT))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.UnmarshalVariable[string](s.Data)).To(Equal([]string{"hello"}))
			})

			It("Should convert int to string", func() {
				s := telem.NewSeriesFromAny(42, telem.StringT)
				Expect(s.DataType).To(Equal(telem.StringT))
				Expect(telem.UnmarshalVariable[string](s.Data)).To(Equal([]string{"42"}))
			})

			It("Should convert float to string", func() {
				s := telem.NewSeriesFromAny(3.14, telem.StringT)
				Expect(s.DataType).To(Equal(telem.StringT))
				Expect(telem.UnmarshalVariable[string](s.Data)[0]).To(ContainSubstring("3.14"))
			})

			It("Should panic when converting string to numeric type", func() {
				Expect(func() {
					telem.NewSeriesFromAny("not a number", telem.Int64T)
				}).To(Panic())
			})
		})

		Describe("JSON conversions", func() {
			It("Should convert JSON string to JSONT", func() {
				jsonStr := `{"key":"value"}`
				s := telem.NewSeriesFromAny(jsonStr, telem.JSONT)
				Expect(s.DataType).To(Equal(telem.JSONT))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(telem.UnmarshalVariable[string](s.Data)).To(Equal([]string{jsonStr}))
			})

			It("Should convert struct to JSON", func() {
				data := map[string]any{"name": "test", "value": 123}
				s := telem.NewSeriesFromAny(data, telem.JSONT)
				Expect(s.DataType).To(Equal(telem.JSONT))
				Expect(s.Len()).To(Equal(int64(1)))
				jsonStr := telem.UnmarshalVariable[string](s.Data)[0]
				Expect(jsonStr).To(ContainSubstring("name"))
				Expect(jsonStr).To(ContainSubstring("test"))
			})

			It("Should convert []byte to JSON", func() {
				jsonBytes := []byte(`{"test":true}`)
				s := telem.NewSeriesFromAny(jsonBytes, telem.JSONT)
				Expect(s.DataType).To(Equal(telem.JSONT))
				Expect(telem.UnmarshalVariable[string](s.Data)).To(Equal([]string{`{"test":true}`}))
			})

			It("Should convert numeric types to JSON", func() {
				s := telem.NewSeriesFromAny(42, telem.JSONT)
				Expect(s.DataType).To(Equal(telem.JSONT))
				Expect(telem.UnmarshalVariable[string](s.Data)).To(Equal([]string{"42"}))
			})
		})

		Describe("Bytes conversions", func() {
			It("Should convert []byte to BytesT", func() {
				data := []byte{0x01, 0x02, 0x03}
				s := telem.NewSeriesFromAny(data, telem.BytesT)
				Expect(s.DataType).To(Equal(telem.BytesT))
				Expect(s.Len()).To(Equal(int64(1)))
				Expect(s.Data[:len(data)]).To(Equal(data))
			})

			It("Should convert string to BytesT", func() {
				s := telem.NewSeriesFromAny("hello", telem.BytesT)
				Expect(s.DataType).To(Equal(telem.BytesT))
				Expect(s.Data[:5]).To(Equal([]byte("hello")))
			})

			It("Should convert numeric types to BytesT", func() {
				s := telem.NewSeriesFromAny(42, telem.BytesT)
				Expect(s.DataType).To(Equal(telem.BytesT))
				Expect(s.Data[:2]).To(Equal([]byte("42")))
			})
		})

		Describe("TimeStamp conversions", func() {
			It("Should convert TimeStamp to TimeStampT", func() {
				ts := telem.TimeStamp(1000)
				s := telem.NewSeriesFromAny(ts, telem.TimeStampT)
				Expect(s.DataType).To(Equal(telem.TimeStampT))
				Expect(telem.ValueAt[telem.TimeStamp](s, 0)).To(Equal(ts))
			})

			It("Should convert int64 to TimeStamp", func() {
				s := telem.NewSeriesFromAny(int64(5000), telem.TimeStampT)
				Expect(s.DataType).To(Equal(telem.TimeStampT))
				Expect(telem.ValueAt[telem.TimeStamp](s, 0)).To(Equal(telem.TimeStamp(5000)))
			})

			It("Should panic when converting string to TimeStamp", func() {
				Expect(func() {
					telem.NewSeriesFromAny("2024-01-01", telem.TimeStampT)
				}).To(Panic())
			})
		})

		Describe("Error cases", func() {
			It("Should panic with unsupported data type", func() {
				Expect(func() {
					telem.NewSeriesFromAny(42, telem.UUIDT)
				}).To(Panic())
			})

			It("Should panic with nil value for numeric type", func() {
				Expect(func() {
					telem.NewSeriesFromAny(nil, telem.Int64T)
				}).To(Panic())
			})
		})

		Describe("All data type targets", func() {
			It("Should create Int8T series", func() {
				s := telem.NewSeriesFromAny(5, telem.Int8T)
				Expect(s.DataType).To(Equal(telem.Int8T))
				Expect(telem.ValueAt[int8](s, 0)).To(Equal(int8(5)))
			})

			It("Should create Int16T series", func() {
				s := telem.NewSeriesFromAny(5, telem.Int16T)
				Expect(s.DataType).To(Equal(telem.Int16T))
				Expect(telem.ValueAt[int16](s, 0)).To(Equal(int16(5)))
			})

			It("Should create Int32T series", func() {
				s := telem.NewSeriesFromAny(5, telem.Int32T)
				Expect(s.DataType).To(Equal(telem.Int32T))
				Expect(telem.ValueAt[int32](s, 0)).To(Equal(int32(5)))
			})

			It("Should create Int64T series", func() {
				s := telem.NewSeriesFromAny(5, telem.Int64T)
				Expect(s.DataType).To(Equal(telem.Int64T))
				Expect(telem.ValueAt[int64](s, 0)).To(Equal(int64(5)))
			})

			It("Should create Uint8T series", func() {
				s := telem.NewSeriesFromAny(5, telem.Uint8T)
				Expect(s.DataType).To(Equal(telem.Uint8T))
				Expect(telem.ValueAt[uint8](s, 0)).To(Equal(uint8(5)))
			})

			It("Should create Uint16T series", func() {
				s := telem.NewSeriesFromAny(5, telem.Uint16T)
				Expect(s.DataType).To(Equal(telem.Uint16T))
				Expect(telem.ValueAt[uint16](s, 0)).To(Equal(uint16(5)))
			})

			It("Should create Uint32T series", func() {
				s := telem.NewSeriesFromAny(5, telem.Uint32T)
				Expect(s.DataType).To(Equal(telem.Uint32T))
				Expect(telem.ValueAt[uint32](s, 0)).To(Equal(uint32(5)))
			})

			It("Should create Uint64T series", func() {
				s := telem.NewSeriesFromAny(5, telem.Uint64T)
				Expect(s.DataType).To(Equal(telem.Uint64T))
				Expect(telem.ValueAt[uint64](s, 0)).To(Equal(uint64(5)))
			})

			It("Should create Float32T series", func() {
				s := telem.NewSeriesFromAny(5.5, telem.Float32T)
				Expect(s.DataType).To(Equal(telem.Float32T))
				Expect(telem.ValueAt[float32](s, 0)).To(BeNumerically("~", 5.5))
			})

			It("Should create Float64T series", func() {
				s := telem.NewSeriesFromAny(5.5, telem.Float64T)
				Expect(s.DataType).To(Equal(telem.Float64T))
				Expect(telem.ValueAt[float64](s, 0)).To(BeNumerically("~", 5.5))
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
			src := telem.NewSeriesVariableV("a", "b", "c")
			dst := telem.NewSeriesVariableV("x", "y", "z")
			Expect(func() {
				telem.CopyValue(dst, src, 0, 0)
			}).To(Panic())
		})

		It("Should panic when destination is variable density", func() {
			src := telem.NewSeriesV[int64](1, 2, 3)
			dst := telem.NewSeriesVariableV("x", "y", "z")
			Expect(func() {
				telem.CopyValue(dst, src, 0, 0)
			}).To(Panic())
		})
	})

	Describe("DeepCopy", func() {
		It("Should create a deep copy of a series", func() {
			original := telem.NewSeriesV[int64](1, 2, 3, 4, 5)
			original.TimeRange = telem.TimeRange{Start: 100, End: 200}
			original.Alignment = telem.NewAlignment(1, 5)

			copied := original.DeepCopy()

			Expect(copied.DataType).To(Equal(original.DataType))
			Expect(copied.Len()).To(Equal(original.Len()))
			Expect(copied.TimeRange).To(Equal(original.TimeRange))
			Expect(copied.Alignment).To(Equal(original.Alignment))
			Expect(telem.UnmarshalSeries[int64](copied)).To(Equal([]int64{1, 2, 3, 4, 5}))
		})

		It("Should create an independent copy that does not share data", func() {
			original := telem.NewSeriesV[int64](1, 2, 3)
			copied := original.DeepCopy()

			telem.SetValueAt[int64](original, 0, 99)

			Expect(telem.ValueAt[int64](original, 0)).To(Equal(int64(99)))
			Expect(telem.ValueAt[int64](copied, 0)).To(Equal(int64(1)))
		})

		It("Should work with different data types", func() {
			original := telem.NewSeriesV[float32](1.1, 2.2, 3.3)
			copied := original.DeepCopy()

			Expect(copied.DataType).To(Equal(telem.Float32T))
			Expect(telem.UnmarshalSeries[float32](copied)).To(Equal([]float32{1.1, 2.2, 3.3}))
		})

		It("Should work with variable density types", func() {
			original := telem.NewSeriesVariableV("foo", "bar", "baz")
			original.TimeRange = telem.TimeRange{Start: 10, End: 20}
			original.Alignment = telem.NewAlignment(2, 3)

			copied := original.DeepCopy()

			Expect(copied.DataType).To(Equal(telem.StringT))
			Expect(copied.Len()).To(Equal(int64(3)))
			Expect(copied.TimeRange).To(Equal(original.TimeRange))
			Expect(copied.Alignment).To(Equal(original.Alignment))
			Expect(telem.UnmarshalVariable[string](copied.Data)).To(Equal([]string{"foo", "bar", "baz"}))
		})

		It("Should work with empty series", func() {
			original := telem.Series{DataType: telem.Int64T}
			copied := original.DeepCopy()

			Expect(copied.Len()).To(Equal(int64(0)))
			Expect(copied.DataType).To(Equal(telem.Int64T))
			Expect(copied.Data).To(HaveLen(0))
		})

		It("Should preserve all fields correctly", func() {
			original := telem.NewSeriesV[uint32](100, 200, 300)
			original.TimeRange = telem.TimeRange{Start: telem.TimeStamp(1000), End: telem.TimeStamp(2000)}
			original.Alignment = telem.NewAlignment(5, 10)

			copied := original.DeepCopy()

			Expect(copied.TimeRange.Start).To(Equal(telem.TimeStamp(1000)))
			Expect(copied.TimeRange.End).To(Equal(telem.TimeStamp(2000)))
			Expect(copied.Alignment).To(Equal(telem.NewAlignment(5, 10)))
		})
	})
})
