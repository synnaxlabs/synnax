// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	"encoding/binary"
	"math"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/x/telem/types/v0"
	. "github.com/synnaxlabs/x/testutil"
)

func newFixedSeriesRoundtripTest[T v0.FixedSample](
	data []T,
	dt v0.DataType,
) func() {
	return func() {
		s := v0.NewSeries(data)
		ExpectWithOffset(1, s.DataType).To(Equal(dt))
		ExpectWithOffset(1, s.Len()).To(BeEquivalentTo(len(data)))
		for i, v := range data {
			ExpectWithOffset(1, v0.ValueAt[T](s, i)).To(Equal(v))
		}
		if len(data) > 0 {
			ExpectWithOffset(1, v0.UnmarshalSeries[T](s)).To(Equal(data))
		} else {
			ExpectWithOffset(1, v0.UnmarshalSeries[T](s)).To(BeNil())
		}
		ExpectWithOffset(1, v0.NewSeriesV(data...)).To(Equal(s))
	}
}

func newVariableSeriesRoundtripTest[T v0.VariableSample](
	data []T,
	dt v0.DataType,
) func() {
	return func() {
		s := v0.NewSeries(data)
		ExpectWithOffset(1, s.DataType).To(Equal(dt))
		ExpectWithOffset(1, s.Len()).To(BeEquivalentTo(len(data)))
		if len(data) > 0 {
			ExpectWithOffset(1, v0.UnmarshalSeries[T](s)).To(Equal(data))
		} else {
			ExpectWithOffset(1, v0.UnmarshalSeries[T](s)).To(BeEmpty())
		}
		ExpectWithOffset(1, v0.NewSeriesV(data...)).To(Equal(s))
	}
}

var _ = Describe("SeriesFactory", func() {
	Describe("NewSeries,NewSeriesV,UnmarshalSeries", func() {
		Describe("Fixed Types", func() {
			Specify("uint8", newFixedSeriesRoundtripTest([]uint8{1, 2, 3}, v0.Uint8T))
			Specify("uint16", newFixedSeriesRoundtripTest([]uint16{1, 2, 3}, v0.Uint16T))
			Specify("uint32", newFixedSeriesRoundtripTest([]uint32{1, 2, 3}, v0.Uint32T))
			Specify("uint64", newFixedSeriesRoundtripTest([]uint64{1, 2, 3}, v0.Uint64T))
			Specify("int8", newFixedSeriesRoundtripTest([]int8{1, 2, 3}, v0.Int8T))
			Specify("int16", newFixedSeriesRoundtripTest([]int16{1, 2, 3}, v0.Int16T))
			Specify("int32", newFixedSeriesRoundtripTest([]int32{1, 2, 3}, v0.Int32T))
			Specify("int64", newFixedSeriesRoundtripTest([]int64{1, 2, 3}, v0.Int64T))
			Specify("float32", newFixedSeriesRoundtripTest([]float32{1.0, 2.0, 3.0}, v0.Float32T))
			Specify("float64", newFixedSeriesRoundtripTest([]float64{1.0, 2.0, 3.0}, v0.Float64T))
			Specify("timestamp", newFixedSeriesRoundtripTest([]v0.TimeStamp{1, 2, 3}, v0.TimeStampT))
			Specify("uuid", newFixedSeriesRoundtripTest([]uuid.UUID{uuid.New(), uuid.New(), uuid.New()}, v0.UUIDT))
			Specify("empty", newFixedSeriesRoundtripTest([]int64{}, v0.Int64T))
			Specify("nil", func() {
				s := v0.NewSeries[int64](nil)
				Expect(s.DataType).To(Equal(v0.Int64T))
				Expect(s.Len()).To(BeZero())
				Expect(v0.UnmarshalSeries[int64](s)).To(BeNil())
			})
			Specify("single value", newFixedSeriesRoundtripTest([]int64{1}, v0.Int64T))
		})
		Describe("Variable Types", func() {
			Specify("string", newVariableSeriesRoundtripTest([]string{"hello", "world"}, v0.StringT))
			Specify("single string", newVariableSeriesRoundtripTest([]string{"hello"}, v0.StringT))
			Specify("empty strings", newVariableSeriesRoundtripTest([]string{"", "", ""}, v0.StringT))
			Specify("different length strings", newVariableSeriesRoundtripTest([]string{"hello", "", "foo"}, v0.StringT))
			Specify("empty", newVariableSeriesRoundtripTest([]string{}, v0.StringT))
			Specify("[]byte", newVariableSeriesRoundtripTest([][]byte{{1, 2, 3}, {4, 5, 6}}, v0.BytesT))
			Specify("single []byte", newVariableSeriesRoundtripTest([][]byte{{1}}, v0.BytesT))
			Specify("empty []byte values", newVariableSeriesRoundtripTest([][]byte{{}, {}, {}}, v0.BytesT))
			Specify("empty []byte", newVariableSeriesRoundtripTest([][]byte{}, v0.BytesT))
			Specify("different length []byte", newVariableSeriesRoundtripTest([][]byte{{1, 2, 3}, {}, {4}}, v0.BytesT))
			Specify("single JSON", func() {
				s := MustSucceed(v0.NewJSONSeriesV(map[string]any{"a": 1.0}))
				Expect(s.DataType).To(Equal(v0.JSONT))
				Expect(s.Len()).To(BeEquivalentTo(1))
				Expect(string(s.At(0))).To(Equal(`{"a":1}`))
			})
			Specify("empty JSON", func() {
				s := MustSucceed(v0.NewJSONSeries([]map[string]any{}))
				Expect(s.DataType).To(Equal(v0.JSONT))
				Expect(s.Len()).To(BeEquivalentTo(0))
			})
		})
	})

	Describe("MakeSeries", func() {
		It("Should allocate a series with the specified length", func() {
			s := v0.MakeSeries(v0.Int64T, 20)
			Expect(s.Len()).To(BeEquivalentTo(20))
			Expect(s.Size()).To(Equal(v0.Byte * 20 * 8))
		})

		It("Should allocate with zero length", func() {
			s := v0.MakeSeries(v0.Float32T, 0)
			Expect(s.Len()).To(BeZero())
			Expect(s.Size()).To(BeZero())
		})

		It("Should work with different data types", func() {
			s := v0.MakeSeries(v0.Uint16T, 10)
			Expect(s.Len()).To(BeEquivalentTo(10))
			Expect(s.Size()).To(Equal(v0.Byte * 10 * 2))
		})
	})

	Describe("NewSeriesSecondsTSV", func() {
		It("Should multiply timestamps by SecondTS", func() {
			s := v0.NewSeriesSecondsTSV(1, 2, 3)
			Expect(s.DataType).To(Equal(v0.TimeStampT))
			Expect(s.Len()).To(BeEquivalentTo(3))
			data := v0.UnmarshalSeries[v0.TimeStamp](s)
			Expect(data[0]).To(Equal(v0.SecondTS))
			Expect(data[1]).To(Equal(2 * v0.SecondTS))
			Expect(data[2]).To(Equal(3 * v0.SecondTS))
		})

		It("Should handle a single timestamp", func() {
			s := v0.NewSeriesSecondsTSV(5)
			Expect(s.Len()).To(BeEquivalentTo(1))
			data := v0.UnmarshalSeries[v0.TimeStamp](s)
			Expect(data).To(Equal([]v0.TimeStamp{5 * v0.SecondTS}))
		})

		It("Should handle zero", func() {
			s := v0.NewSeriesSecondsTSV(0)
			data := v0.UnmarshalSeries[v0.TimeStamp](s)
			Expect(data).To(Equal([]v0.TimeStamp{0}))
		})
	})

	Describe("NewJSONSeries / UnmarshalJSONSeries", func() {
		It("Should marshal a map", func() {
			data := []map[string]any{{"cat": map[string]any{"one": "two"}}}
			s := MustSucceed(v0.NewJSONSeries(data))
			Expect(s.DataType).To(Equal(v0.JSONT))
			Expect(s.Len()).To(BeEquivalentTo(1))
			Expect(string(s.At(0))).To(Equal(`{"cat":{"one":"two"}}`))
			Expect(v0.UnmarshalJSONSeries[map[string]any](s)).To(Equal(data))
		})

		It("Should marshal multiple values", func() {
			data := []map[string]any{{"a": 1}, {"b": 2}}
			s := MustSucceed(v0.NewJSONSeries(data))
			Expect(s.DataType).To(Equal(v0.JSONT))
			Expect(string(s.At(0))).To(Equal(`{"a":1}`))
			Expect(string(s.At(1))).To(Equal(`{"b":2}`))
			Expect(s.Len()).To(BeEquivalentTo(2))
		})

		It("Should return an error for invalid JSON", func() {
			data := []chan int{make(chan int)}
			Expect(v0.NewJSONSeries(data)).Error().
				To(MatchError(ContainSubstring("json: unsupported type: chan int")))
		})
	})

	Describe("NewJSONSeriesV", func() {
		It("Should marshal variadic JSON values", func() {
			s := MustSucceed(v0.NewJSONSeriesV(
				map[string]any{"a": 1},
				map[string]any{"b": 2},
			))
			Expect(s.Len()).To(Equal(int64(2)))
			Expect(string(s.At(0))).To(Equal(`{"a":1}`))
			Expect(string(s.At(1))).To(Equal(`{"b":2}`))
		})

		It("Should marshal a slice value", func() {
			s := MustSucceed(v0.NewJSONSeriesV([]int{1, 2, 3}))
			Expect(string(s.At(0))).To(Equal(`[1,2,3]`))
		})

		It("Should roundtrip through UnmarshalJSONSeries", func() {
			s := MustSucceed(v0.NewJSONSeriesV([]int{1, 2, 3}))
			unmarshalled := MustSucceed(v0.UnmarshalJSONSeries[[]int](s))
			Expect(unmarshalled).To(Equal([][]int{{1, 2, 3}}))
		})

		It("Should return an error for invalid JSON", func() {
			Expect(v0.NewJSONSeriesV(make(chan int))).Error().
				To(MatchError(ContainSubstring("json: unsupported type: chan int")))
		})
	})

	Describe("UnmarshalJSONSeries", func() {
		It("Should unmarshal JSON into typed values", func() {
			s := MustSucceed(v0.NewJSONSeriesV([]int{1, 2, 3}))
			data := MustSucceed(v0.UnmarshalJSONSeries[[]int](s))
			Expect(data).To(Equal([][]int{{1, 2, 3}}))
		})

		It("Should unmarshal multiple JSON values", func() {
			s := MustSucceed(v0.NewJSONSeriesV(
				map[string]any{"a": 1.0},
				map[string]any{"b": 2.0},
			))
			data := MustSucceed(v0.UnmarshalJSONSeries[map[string]any](s))
			Expect(data).To(HaveLen(2))
			Expect(data[0]).To(HaveKeyWithValue("a", 1.0))
			Expect(data[1]).To(HaveKeyWithValue("b", 2.0))
		})

		It("Should return an error when unmarshalling into wrong type", func() {
			s := MustSucceed(v0.NewJSONSeriesV([]int{1, 2, 3}))
			Expect(v0.UnmarshalJSONSeries[string](s)).Error().
				To(MatchError(ContainSubstring(
					"json: cannot unmarshal array into Go value of type string",
				)))
		})
	})

	Describe("MarshalVariableSample", func() {
		It("Should marshal a typical sample with a length prefix", func() {
			sample := []byte("hello")
			result := v0.MarshalVariableSample(sample)
			Expect(result).To(HaveLen(9))
			Expect(binary.LittleEndian.Uint32(result[:4])).To(Equal(uint32(5)))
			Expect(result[4:]).To(Equal(sample))
		})

		It("Should marshal an empty sample", func() {
			result := v0.MarshalVariableSample([]byte{})
			Expect(result).To(HaveLen(4))
			Expect(binary.LittleEndian.Uint32(result[:4])).To(Equal(uint32(0)))
		})

		It("Should marshal a nil sample", func() {
			result := v0.MarshalVariableSample(nil)
			Expect(result).To(HaveLen(4))
			Expect(binary.LittleEndian.Uint32(result[:4])).To(Equal(uint32(0)))
		})

		It("Should marshal a single byte sample", func() {
			result := v0.MarshalVariableSample([]byte{0xFF})
			Expect(result).To(HaveLen(5))
			Expect(binary.LittleEndian.Uint32(result[:4])).To(Equal(uint32(1)))
			Expect(result[4]).To(Equal(byte(0xFF)))
		})

		It("Should produce output readable by Series.At", func() {
			samples := [][]byte{[]byte("foo"), []byte("barbaz"), []byte("")}
			var data []byte
			for _, s := range samples {
				data = append(data, v0.MarshalVariableSample(s)...)
			}
			series := v0.Series{DataType: v0.StringT, Data: data}
			Expect(series.Len()).To(Equal(int64(3)))
			Expect(series.At(0)).To(Equal([]byte("foo")))
			Expect(series.At(1)).To(Equal([]byte("barbaz")))
			Expect(series.At(2)).To(BeEmpty())
		})

		It("Should produce the same encoding as NewSeriesV", func() {
			fromFactory := v0.NewSeriesV([]byte{1, 2}, []byte{3, 4, 5})
			var manual []byte
			manual = append(manual, v0.MarshalVariableSample([]byte{1, 2})...)
			manual = append(manual, v0.MarshalVariableSample([]byte{3, 4, 5})...)
			Expect(manual).To(Equal(fromFactory.Data))
		})
	})

	Describe("Arrange", func() {
		It("Should create a series with the correct values for int64", func() {
			s := v0.Arrange[int64](0, 5, 2)
			Expect(s.Len()).To(BeEquivalentTo(5))
			Expect(s.DataType).To(Equal(v0.Int64T))
			Expect(v0.UnmarshalSeries[int64](s)).To(Equal([]int64{0, 2, 4, 6, 8}))
		})

		It("Should create a series with the correct values for float64", func() {
			s := v0.Arrange(0.0, 5, 0.5)
			Expect(s.Len()).To(BeEquivalentTo(5))
			Expect(s.DataType).To(Equal(v0.Float64T))
			Expect(v0.UnmarshalSeries[float64](s)).To(Equal([]float64{0.0, 0.5, 1.0, 1.5, 2.0}))
		})

		It("Should create a series with a single value when count is 1", func() {
			s := v0.Arrange[int32](10, 1, 5)
			Expect(s.Len()).To(BeEquivalentTo(1))
			Expect(s.DataType).To(Equal(v0.Int32T))
			Expect(v0.UnmarshalSeries[int32](s)).To(Equal([]int32{10}))
		})

		It("Should create a series with negative spacing", func() {
			s := v0.Arrange[int64](10, 5, -2)
			Expect(s.Len()).To(BeEquivalentTo(5))
			Expect(v0.UnmarshalSeries[int64](s)).To(Equal([]int64{10, 8, 6, 4, 2}))
		})

		It("Should create a series with timestamps", func() {
			s := v0.Arrange[v0.TimeStamp](0, 3, 100)
			Expect(s.DataType).To(Equal(v0.TimeStampT))
			Expect(v0.UnmarshalSeries[v0.TimeStamp](s)).
				To(Equal([]v0.TimeStamp{0, 100, 200}))
		})

		It("Should panic when count is less than 0", func() {
			Expect(func() {
				v0.Arrange[int64](0, -1, 1)
			}).To(Panic())
		})
	})

	Describe("NewSeriesFromAny", func() {
		id := uuid.New()

		DescribeTable("happy path", func(input any, dt v0.DataType, expected v0.Series) {
			s := v0.NewSeriesFromAny(input, dt)
			Expect(s).To(Equal(expected))
		},
			// same-type numeric
			Entry("int → Int64T", 42, v0.Int64T, v0.NewSeriesV(int64(42))),
			Entry("int64 → Int64T", int64(100), v0.Int64T, v0.NewSeriesV(int64(100))),
			Entry("int32 → Int32T", int32(50), v0.Int32T, v0.NewSeriesV(int32(50))),
			Entry("int16 → Int16T", int16(25), v0.Int16T, v0.NewSeriesV(int16(25))),
			Entry("int8 → Int8T", int8(12), v0.Int8T, v0.NewSeriesV(int8(12))),
			Entry("uint64 → Uint64T", uint64(200), v0.Uint64T, v0.NewSeriesV(uint64(200))),
			Entry("uint32 → Uint32T", uint32(150), v0.Uint32T, v0.NewSeriesV(uint32(150))),
			Entry("uint16 → Uint16T", uint16(75), v0.Uint16T, v0.NewSeriesV(uint16(75))),
			Entry("uint8 → Uint8T", uint8(37), v0.Uint8T, v0.NewSeriesV(uint8(37))),
			Entry("float64 → Float64T", 3.14, v0.Float64T, v0.NewSeriesV(float64(3.14))),
			Entry("float32 → Float32T", float32(2.5), v0.Float32T, v0.NewSeriesV(float32(2.5))),

			// cross-type numeric conversions
			Entry("int → Float64T", 42, v0.Float64T, v0.NewSeriesV(float64(42))),
			Entry("float64 → Int64T (truncates)", 42.7, v0.Int64T, v0.NewSeriesV(int64(42))),
			Entry("uint32 → Int32T", uint32(100), v0.Int32T, v0.NewSeriesV(int32(100))),
			Entry("int32 → Uint32T", int32(50), v0.Uint32T, v0.NewSeriesV(uint32(50))),
			Entry("float32 → Float64T", float32(1.5), v0.Float64T, v0.NewSeriesV(float64(float32(1.5)))),
			Entry("float64 → Float32T", 2.5, v0.Float32T, v0.NewSeriesV(float32(2.5))),
			Entry("int64 → Int32T", int64(100), v0.Int32T, v0.NewSeriesV(int32(100))),
			Entry("int16 → Int8T", int16(25), v0.Int8T, v0.NewSeriesV(int8(25))),
			Entry("int8 → Int64T", int8(12), v0.Int64T, v0.NewSeriesV(int64(12))),
			Entry("uint8 → Uint64T", uint8(37), v0.Uint64T, v0.NewSeriesV(uint64(37))),
			Entry("uint → Int64T", uint(42), v0.Int64T, v0.NewSeriesV(int64(42))),

			// edge values
			Entry("zero → Int64T", 0, v0.Int64T, v0.NewSeriesV(int64(0))),
			Entry("negative → Int32T", -42, v0.Int32T, v0.NewSeriesV(int32(-42))),
			Entry("max uint64", uint64(18446744073709551615), v0.Uint64T, v0.NewSeriesV(uint64(18446744073709551615))),

			// timestamp
			Entry("TimeStamp → TimeStampT", v0.TimeStamp(1000), v0.TimeStampT, v0.NewSeriesV(v0.TimeStamp(1000))),
			Entry("int64 → TimeStampT", int64(5000), v0.TimeStampT, v0.NewSeriesV(v0.TimeStamp(5000))),
			Entry("TimeSpan → TimeStampT", v0.TimeSpan(1000), v0.TimeStampT, v0.NewSeriesV(v0.TimeStamp(1000))),
			Entry("TimeSpan → Int64T", v0.TimeSpan(2000), v0.Int64T, v0.NewSeriesV(int64(2000))),

			// string
			Entry("string → StringT", "hello", v0.StringT, v0.NewSeriesV("hello")),
			Entry("int → StringT", 42, v0.StringT, v0.NewSeriesV("42")),
			Entry("float → StringT", 3.14, v0.StringT, v0.NewSeriesV("3.14")),
			Entry("uuid → StringT", id, v0.StringT, v0.NewSeriesV(id.String())),

			// uuid
			Entry("uuid → UUIDT", id, v0.UUIDT, v0.NewSeriesV(id)),
			Entry("string → UUIDT", id.String(), v0.UUIDT, v0.NewSeriesV(id)),
			Entry("[]byte → UUIDT", id[:], v0.UUIDT, v0.NewSeriesV(id)),

			// bytes
			Entry("[]byte → BytesT", []byte{1, 2, 3}, v0.BytesT, v0.NewSeriesV([]byte{1, 2, 3})),
			Entry("string → BytesT", "hello", v0.BytesT, v0.NewSeriesV([]byte("hello"))),
			Entry("byte -> BytesT", byte(1), v0.BytesT, v0.NewSeriesV([]byte{1})),
			Entry("uint8 -> BytesT", uint8(2), v0.BytesT, v0.NewSeriesV([]byte{2})),
			Entry("uint16 -> BytesT", uint16(3), v0.BytesT, v0.NewSeriesV([]byte{3, 0})),
			Entry("uint32 -> BytesT", uint32(4), v0.BytesT, v0.NewSeriesV([]byte{4, 0, 0, 0})),
			Entry("uint64 -> BytesT", uint64(5), v0.BytesT, v0.NewSeriesV([]byte{5, 0, 0, 0, 0, 0, 0, 0})),
			Entry("int8 -> BytesT", int8(6), v0.BytesT, v0.NewSeriesV([]byte{6})),
			Entry("int16 -> BytesT", int16(7), v0.BytesT, v0.NewSeriesV([]byte{7, 0})),
			Entry("int32 -> BytesT", int32(8), v0.BytesT, v0.NewSeriesV([]byte{8, 0, 0, 0})),
			Entry("int64 -> BytesT", int64(9), v0.BytesT, v0.NewSeriesV([]byte{9, 0, 0, 0, 0, 0, 0, 0})),
			Entry("timestamp -> BytesT", v0.TimeStamp(1), v0.BytesT, v0.NewSeriesV([]byte{1, 0, 0, 0, 0, 0, 0, 0})),
			Entry("timespan -> BytesT", v0.TimeSpan(10), v0.BytesT, v0.NewSeriesV([]byte{10, 0, 0, 0, 0, 0, 0, 0})),
			Entry("float32 -> BytesT", float32(1.5), v0.BytesT, v0.NewSeriesV(v0.ByteOrder.AppendUint32(nil, math.Float32bits(1.5)))),
			Entry("float64 -> BytesT", 2.5, v0.BytesT, v0.NewSeriesV(v0.ByteOrder.AppendUint64(nil, math.Float64bits(2.5)))),
			Entry("uuid -> BytesT", id, v0.BytesT, v0.NewSeriesV(id[:])),
			// json
			Entry("int → JSONT", 42, v0.JSONT, MustSucceed(v0.NewJSONSeriesV(42))),
		)

		DescribeTable("should panic", func(input any, dt v0.DataType, msg string) {
			Expect(func() {
				v0.NewSeriesFromAny(input, dt)
			}).To(PanicWith(ContainSubstring(msg)))
		},
			Entry("string → Int64T", "not a number", v0.Int64T, "cannot cast string to int64"),
			Entry("string → TimeStampT", "2024-01-01", v0.TimeStampT, "cannot cast string to v0.TimeStamp"),
			Entry("int → UUIDT", 42, v0.UUIDT, "cannot cast int to uuid.UUID"),
			Entry("invalid string → UUIDT", "not-a-uuid", v0.UUIDT, "invalid UUID"),
			Entry("short []byte → UUIDT", []byte{1, 2, 3}, v0.UUIDT, "invalid UUID (got 3 bytes)"),
			Entry("nil → Int64T", nil, v0.Int64T, "cannot cast <nil> to int64"),
			Entry("int → unsupported", 42, v0.UnknownT, "unsupported data type"),
			Entry("chan int → BytesT", make(chan int), v0.BytesT, "cannot cast chan int to []byte"),
		)
	})
})
