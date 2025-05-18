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
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
)

func MarshalTest[T telem.Sample](data []T, dt telem.DataType) func() {
	return func() {
		s := telem.NewSeries(data)
		ExpectWithOffset(1, s.DataType).To(Equal(dt))
		ExpectWithOffset(1, s.Len()).To(Equal(int64(len(data))))
		ExpectWithOffset(1, telem.Unmarshal[T](s)).To(Equal(data))
	}
}

func ValueAtTest[T telem.Sample](value T, dt telem.DataType) func() {
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
			s := telem.NewStringsV("bob", "alice", "charlie")
			Expect(s.Len()).To(Equal(int64(3)))
		})
	})

	Describe("Factory", func() {
		Describe("Marshal", func() {
			Specify("float64", MarshalTest([]float64{1.0, 2.0, 3.0}, telem.Float64T))
			Specify("float32", MarshalTest([]float32{1.0, 2.0, 3.0}, telem.Float32T))
			Specify("int64", MarshalTest([]int64{1, 2, 3}, telem.Int64T))
			Specify("int32", MarshalTest([]int32{1, 2, 3}, telem.Int32T))
			Specify("int16", MarshalTest([]int16{1, 2, 3}, telem.Int16T))
			Specify("int8", MarshalTest([]int8{1, 2, 3}, telem.Int8T))
			Specify("uint64", MarshalTest([]uint64{1, 2, 3}, telem.Uint64T))
			Specify("uint32", MarshalTest([]uint32{1, 2, 3}, telem.Uint32T))
			Specify("uint16", MarshalTest([]uint16{1, 2, 3}, telem.Uint16T))
			Specify("uint8", MarshalTest([]uint8{1, 2, 3}, telem.Uint8T))
			Specify("timestamp", MarshalTest([]telem.TimeStamp{1, 2, 3}, telem.TimeStampT))
		})

		Describe("StaticJSONV", func() {
			It("Should correctly marshal a static JSON data structure", func() {
				data := map[string]any{
					"cat": map[string]any{
						"one": "two",
					},
				}
				s := telem.NewStaticJSONV(data)
				Expect(s.Len()).To(Equal(int64(1)))
			})
		})
	})

	Describe("MakeSeries", func() {
		It("Should make a series with the specified length", func() {
			s := telem.MakeSeries(telem.Int64T, 20)
			Expect(s.Len()).To(Equal(int64(20)))
			Expect(s.Size()).To(Equal(telem.ByteSize * 20 * 8))
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
				s := telem.NewStringsV("a", "b", "c")
				Expect(s.At(0)).To(Equal([]byte("a")))
				Expect(s.At(1)).To(Equal([]byte("b")))
				Expect(s.At(2)).To(Equal([]byte("c")))
			})

			It("Should panic when the index is out of bounds", func() {
				s := telem.NewStringsV("a", "b", "c")
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
			Specify("uint8", ValueAtTest(uint8(1), telem.Uint8T))
			Specify("uint16", ValueAtTest(uint16(1), telem.Uint16T))
			Specify("uint32", ValueAtTest(uint32(1), telem.Uint32T))
			Specify("uint64", ValueAtTest(uint64(1), telem.Uint64T))
			Specify("int8", ValueAtTest(int8(1), telem.Int8T))
			Specify("int16", ValueAtTest(int16(1), telem.Int16T))
			Specify("int32", ValueAtTest(int32(1), telem.Int32T))
			Specify("int64", ValueAtTest(int64(1), telem.Int64T))
			Specify("float32", ValueAtTest(float32(1.0), telem.Float32T))
			Specify("float64", ValueAtTest(float64(1.0), telem.Float64T))
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
				s := telem.NewSeriesV(1.1, 2.2, 3.3)
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
			Entry("string", telem.NewStringsV("a", "b", "c"), "[a b c]"),
			Entry("json", telem.NewStaticJSONV(map[string]any{"a": 1, "b": 2, "c": 3}), `[{"a":1,"b":2,"c":3}]`),
			Entry("timestamp", telem.NewSecondsTSV(1, 2, 3), "[1970-01-01T00:00:01Z +1s +2s]"),
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
				s := telem.NewStringsV(values...)
				str := s.String()
				Expect(str).To(ContainSubstring("[a b c d e f ... i j k l m n]"))
			})

			It("Should truncate a long timestamp series", func() {
				values := telem.NewSecondsTSV(1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20)
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

	Describe("DownSample", func() {
		Context("Fixed Length Data Types", func() {
			It("Should correctly downsample a series with a factor of 2", func() {
				original := telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7, 8)
				downsampled := original.DownSample(2)

				Expect(downsampled.Len()).To(Equal(int64(4)))
				Expect(telem.Unmarshal[int64](downsampled)).To(Equal([]int64{1, 3, 5, 7}))
				Expect(downsampled.DataType).To(Equal(original.DataType))
				Expect(downsampled.TimeRange).To(Equal(original.TimeRange))
				Expect(downsampled.Alignment).To(Equal(original.Alignment))
			})

			It("Should correctly downsample a series with a factor of 3", func() {
				original := telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7, 8, 9)
				downsampled := original.DownSample(3)

				Expect(downsampled.Len()).To(Equal(int64(3)))
				Expect(telem.Unmarshal[int64](downsampled)).To(Equal([]int64{1, 4, 7}))
			})

			It("Should work when the factor is not an even multiple of the length", func() {
				original := telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6, 7, 8, 9, 10)
				downsampled := original.DownSample(3)

				Expect(downsampled.Len()).To(Equal(int64(4)))
				Expect(telem.Unmarshal[int64](downsampled)).To(Equal([]int64{1, 4, 7, 10}))
			})

			It("Should work with different numeric types", func() {
				original := telem.NewSeriesV(1.1, 2.2, 3.3, 4.4, 5.5, 6.6)
				downsampled := original.DownSample(2)

				Expect(downsampled.Len()).To(Equal(int64(3)))
				Expect(telem.Unmarshal[float64](downsampled)).To(Equal([]float64{1.1, 3.3, 5.5}))
			})

			It("Should preserve alignment information", func() {
				original := telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6)
				original.Alignment = telem.NewAlignment(1, 5)
				downsampled := original.DownSample(2)

				Expect(downsampled.Alignment).To(Equal(original.Alignment))
			})

			It("Should preserve time range information", func() {
				original := telem.NewSeriesV[int64](1, 2, 3, 4, 5, 6)
				original.TimeRange = telem.TimeRange{Start: 100, End: 600}
				downsampled := original.DownSample(2)

				Expect(downsampled.TimeRange).To(Equal(original.TimeRange))
			})
		})

		Context("Variable Length Data Types", func() {
			It("Should correctly down sample a string series", func() {
				original := telem.NewStringsV("a", "b", "c", "d", "e", "f")
				downSampled := original.DownSample(2)

				Expect(downSampled.Len()).To(Equal(int64(3)))
				Expect(telem.UnmarshalStrings(downSampled.Data)).To(Equal([]string{"a", "c", "e"}))
			})

			It("Should correctly down sample a JSON series", func() {
				data := []map[string]any{
					{"id": 1},
					{"id": 2},
					{"id": 3},
					{"id": 4},
				}

				s := telem.NewStaticJSONV(data...)
				downSampled := s.DownSample(2)
				Expect(downSampled.Len()).To(Equal(int64(2)))
				split := bytes.Split(downSampled.Data, []byte("\n"))
				Expect(len(split)).To(Equal(3)) // 2 items + empty string after last newline
			})
		})

		Context("Edge Cases", func() {
			It("Should return the original series if factor is <= 1", func() {
				original := telem.NewSeriesV[int64](1, 2, 3)
				downSampled := original.DownSample(0)
				Expect(downSampled).To(Equal(original))
				downSampled = original.DownSample(1)
				Expect(downSampled).To(Equal(original))
				downSampled = original.DownSample(-1)
				Expect(downSampled).To(Equal(original))
			})

			It("Should return the maximum possible downSampling if series length is <= factor", func() {
				original := telem.NewSeriesV[int64](1, 2, 3)
				downSampled := original.DownSample(3)
				Expect(downSampled.Len()).To(Equal(int64(1)))
				Expect(telem.Unmarshal[int64](downSampled)).To(Equal([]int64{1}))
				downSampled = original.DownSample(10)
				Expect(downSampled.Len()).To(Equal(int64(1)))
				Expect(telem.Unmarshal[int64](downSampled)).To(Equal([]int64{1}))
			})

			It("Should handle empty series correctly", func() {
				original := telem.Series{DataType: telem.Int64T}
				downSampled := original.DownSample(2)
				Expect(downSampled).To(Equal(original))
				Expect(downSampled.Len()).To(Equal(int64(0)))
			})
		})
	})

	Describe("MultiSeries", func() {
		Describe("NewMultiSeries", func() {
			It("Should construct a multi-series from a slice of series", func() {
				s1 := telem.NewSecondsTSV(1, 2, 3)
				s2 := telem.NewSecondsTSV(4, 5, 6)
				ms := telem.NewMultiSeriesV(s1, s2)
				Expect(ms.Len()).To(Equal(int64(6)))
			})
			It("Should sort the series by alignment on construction", func() {
				s1 := telem.NewSecondsTSV(1, 2, 3)
				s1.Alignment = telem.NewAlignment(0, 0)
				s2 := telem.NewSecondsTSV(4, 5, 6)
				s2.Alignment = telem.NewAlignment(0, 3)
				ms := telem.NewMultiSeriesV(s2, s1)
				Expect(ms.Series[0].Alignment).To(Equal(s1.Alignment))
				Expect(ms.Series[1].Alignment).To(Equal(s2.Alignment))
			})
			It("Should panic when trying to construct the series out of different data types", func() {
				s1 := telem.NewSecondsTSV(1, 2, 3)
				s2 := telem.NewStringsV("a", "b", "c")
				Expect(func() { telem.NewMultiSeriesV(s1, s2) }).To(Panic())
			})
		})

		Describe("AlignmentBounds", func() {
			It("Should return the alignment bounds of the multi-series", func() {
				s1 := telem.NewSecondsTSV(1, 2, 3)
				s1.Alignment = telem.NewAlignment(0, 0)
				s2 := telem.NewSecondsTSV(4, 5, 6)
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
				s1 := telem.NewSecondsTSV(1, 2, 3)
				s1.TimeRange = telem.TimeRange{Start: 0, End: 3}
				s2 := telem.NewSecondsTSV(4, 5, 6)
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
				s1 := telem.NewSecondsTSV(1, 2, 3)
				s2 := telem.NewSecondsTSV(4, 5, 6)
				ms := telem.NewMultiSeriesV(s1)
				ms = ms.Append(s2)
				Expect(ms.Len()).To(Equal(int64(6)))
				Expect(ms.Series[0].Alignment).To(Equal(s1.Alignment))
				Expect(ms.Series[1].Alignment).To(Equal(s2.Alignment))
			})

			It("Should panic if the series data types do not match", func() {
				s1 := telem.NewSecondsTSV(1, 2, 3)
				s2 := telem.NewSeriesV[int32](1, 2, 3)
				ms := telem.NewMultiSeriesV(s1)
				Expect(func() {
					ms = ms.Append(s2)
				}).To(Panic())
			})

			It("Should not panic when appending to an empty series", func() {
				s1 := telem.NewSecondsTSV(1, 2, 3)
				ms := telem.MultiSeries{}
				Expect(func() {
					ms.Append(s1)
				}).NotTo(Panic())
			})
		})

		Describe("FilterGreaterThanOrEqualTo", func() {
			It("Should remove series with alignment bounds that are less than the target threshold", func() {
				s1 := telem.NewSecondsTSV(1, 2, 3)
				s1.Alignment = telem.NewAlignment(0, 0)
				s2 := telem.NewSecondsTSV(4, 5, 6)
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
				s1 := telem.NewSecondsTSV(1, 2, 3)
				s1.Alignment = 500
				s2 := telem.NewSecondsTSV(4, 5, 6)
				s2.Alignment = 5000
				ms := telem.NewMultiSeriesV(s1, s2)
				ms = ms.FilterGreaterThanOrEqualTo(5)
				Expect(ms.Len()).To(Equal(int64(6)))
			})

			It("Should filter all series when alignment bounds is very high", func() {
				s1 := telem.NewSecondsTSV(1, 2, 3)
				s1.Alignment = 0
				s2 := telem.NewSecondsTSV(4, 5, 6)
				s2.Alignment = 3
				ms := telem.NewMultiSeriesV(s1, s2)
				ms = ms.FilterGreaterThanOrEqualTo(5000)
				Expect(ms).To(Equal(telem.MultiSeries{}))
			})
		})

		Describe("Len", func() {
			It("Should return the accumulated length of all series", func() {
				s1 := telem.NewSecondsTSV(1, 2, 3)
				s2 := telem.NewSecondsTSV(4, 5, 6)
				ms := telem.NewMultiSeriesV(s1, s2)
				Expect(ms.Len()).To(Equal(int64(6)))
			})
		})

		Describe("DataType", func() {
			It("Should return the data type of the multi-series", func() {
				s1 := telem.NewSecondsTSV(1, 2, 3)
				s2 := telem.NewSecondsTSV(4, 5, 6)
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
				values = append(values, telem.UnmarshalF[int64](s.DataType)(sample))
			}
			Expect(values).To(Equal([]int64{1, 2, 3, 4, 5}))
		})

		It("iterates variable length correctly", func() {
			s := telem.NewStringsV("foo", "bar", "baz")
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
				values = append(values, telem.UnmarshalF[int64](s.DataType)(sample))
				count++
				if count > 2 {
					break
				}
			}
			Expect(values).To(Equal([]int64{1, 2, 3}))
		})

		It("Should allow for early termination in variable length series", func() {
			s := telem.NewStringsV("foo", "bar", "baz")
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
})
