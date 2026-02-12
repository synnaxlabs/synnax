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
	"math"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

func newFixedSeriesRoundtripTest[T telem.FixedSample](
	data []T,
	dt telem.DataType,
) func() {
	return func() {
		s := telem.NewSeries(data)
		ExpectWithOffset(1, s.DataType).To(Equal(dt))
		ExpectWithOffset(1, s.Len()).To(BeEquivalentTo(len(data)))
		for i, v := range data {
			ExpectWithOffset(1, telem.ValueAt[T](s, i)).To(Equal(v))
		}
		if len(data) > 0 {
			ExpectWithOffset(1, telem.UnmarshalSeries[T](s)).To(Equal(data))
		} else {
			ExpectWithOffset(1, telem.UnmarshalSeries[T](s)).To(BeNil())
		}
		ExpectWithOffset(1, telem.NewSeriesV(data...)).To(Equal(s))
	}
}

func newVariableSeriesRoundtripTest[T telem.VariableSample](
	data []T,
	dt telem.DataType,
) func() {
	return func() {
		s := telem.NewSeries(data)
		ExpectWithOffset(1, s.DataType).To(Equal(dt))
		ExpectWithOffset(1, s.Len()).To(BeEquivalentTo(len(data)))
		if len(data) > 0 {
			ExpectWithOffset(1, telem.UnmarshalSeries[T](s)).To(Equal(data))
		} else {
			ExpectWithOffset(1, telem.UnmarshalSeries[T](s)).To(BeEmpty())
		}
		ExpectWithOffset(1, telem.NewSeriesV(data...)).To(Equal(s))
	}
}

var _ = Describe("SeriesFactory", func() {
	Describe("NewSeries,NewSeriesV,UnmarshalSeries", func() {
		Describe("Fixed Types", func() {
			Specify("uint8", newFixedSeriesRoundtripTest([]uint8{1, 2, 3}, telem.Uint8T))
			Specify("uint16", newFixedSeriesRoundtripTest([]uint16{1, 2, 3}, telem.Uint16T))
			Specify("uint32", newFixedSeriesRoundtripTest([]uint32{1, 2, 3}, telem.Uint32T))
			Specify("uint64", newFixedSeriesRoundtripTest([]uint64{1, 2, 3}, telem.Uint64T))
			Specify("int8", newFixedSeriesRoundtripTest([]int8{1, 2, 3}, telem.Int8T))
			Specify("int16", newFixedSeriesRoundtripTest([]int16{1, 2, 3}, telem.Int16T))
			Specify("int32", newFixedSeriesRoundtripTest([]int32{1, 2, 3}, telem.Int32T))
			Specify("int64", newFixedSeriesRoundtripTest([]int64{1, 2, 3}, telem.Int64T))
			Specify("float32", newFixedSeriesRoundtripTest([]float32{1.0, 2.0, 3.0}, telem.Float32T))
			Specify("float64", newFixedSeriesRoundtripTest([]float64{1.0, 2.0, 3.0}, telem.Float64T))
			Specify("timestamp", newFixedSeriesRoundtripTest([]telem.TimeStamp{1, 2, 3}, telem.TimeStampT))
			Specify("uuid", newFixedSeriesRoundtripTest([]uuid.UUID{uuid.New(), uuid.New(), uuid.New()}, telem.UUIDT))
			Specify("empty", newFixedSeriesRoundtripTest([]int64{}, telem.Int64T))
			Specify("nil", func() {
				s := telem.NewSeries[int64](nil)
				Expect(s.DataType).To(Equal(telem.Int64T))
				Expect(s.Len()).To(BeZero())
				Expect(telem.UnmarshalSeries[int64](s)).To(BeNil())
			})
			Specify("single value", newFixedSeriesRoundtripTest([]int64{1}, telem.Int64T))
		})
		Describe("Variable Types", func() {
			Specify("string", newVariableSeriesRoundtripTest([]string{"hello", "world"}, telem.StringT))
			Specify("single string", newVariableSeriesRoundtripTest([]string{"hello"}, telem.StringT))
			Specify("empty strings", newVariableSeriesRoundtripTest([]string{"", "", ""}, telem.StringT))
			Specify("different length strings", newVariableSeriesRoundtripTest([]string{"hello", "", "foo"}, telem.StringT))
			Specify("empty", newVariableSeriesRoundtripTest([]string{}, telem.StringT))
			Specify("[]byte", newVariableSeriesRoundtripTest([][]byte{{1, 2, 3}, {4, 5, 6}}, telem.BytesT))
		})
	})

	Describe("MakeSeries", func() {
		It("Should allocate a series with the specified length", func() {
			s := telem.MakeSeries(telem.Int64T, 20)
			Expect(s.Len()).To(BeEquivalentTo(20))
			Expect(s.Size()).To(Equal(telem.Byte * 20 * 8))
		})

		It("Should allocate with zero length", func() {
			s := telem.MakeSeries(telem.Float32T, 0)
			Expect(s.Len()).To(BeZero())
			Expect(s.Size()).To(BeZero())
		})

		It("Should work with different data types", func() {
			s := telem.MakeSeries(telem.Uint16T, 10)
			Expect(s.Len()).To(BeEquivalentTo(10))
			Expect(s.Size()).To(Equal(telem.Byte * 10 * 2))
		})
	})

	Describe("NewSeriesSecondsTSV", func() {
		It("Should multiply timestamps by SecondTS", func() {
			s := telem.NewSeriesSecondsTSV(1, 2, 3)
			Expect(s.DataType).To(Equal(telem.TimeStampT))
			Expect(s.Len()).To(BeEquivalentTo(3))
			data := telem.UnmarshalSeries[telem.TimeStamp](s)
			Expect(data[0]).To(Equal(telem.SecondTS))
			Expect(data[1]).To(Equal(2 * telem.SecondTS))
			Expect(data[2]).To(Equal(3 * telem.SecondTS))
		})

		It("Should handle a single timestamp", func() {
			s := telem.NewSeriesSecondsTSV(5)
			Expect(s.Len()).To(BeEquivalentTo(1))
			data := telem.UnmarshalSeries[telem.TimeStamp](s)
			Expect(data).To(Equal([]telem.TimeStamp{5 * telem.SecondTS}))
		})

		It("Should handle zero", func() {
			s := telem.NewSeriesSecondsTSV(0)
			data := telem.UnmarshalSeries[telem.TimeStamp](s)
			Expect(data).To(Equal([]telem.TimeStamp{0}))
		})
	})

	Describe("NewJSONSeries / UnmarshalJSONSeries", func() {
		It("Should marshal a map", func() {
			data := []map[string]any{{"cat": map[string]any{"one": "two"}}}
			s := MustSucceed(telem.NewJSONSeries(data))
			Expect(s.DataType).To(Equal(telem.JSONT))
			Expect(s.Len()).To(BeEquivalentTo(1))
			Expect(string(s.At(0))).To(Equal(`{"cat":{"one":"two"}}`))
			Expect(telem.UnmarshalJSONSeries[map[string]any](s)).To(Equal(data))
		})

		It("Should marshal multiple values", func() {
			data := []map[string]any{{"a": 1}, {"b": 2}}
			s := MustSucceed(telem.NewJSONSeries(data))
			Expect(s.DataType).To(Equal(telem.JSONT))
			Expect(string(s.At(0))).To(Equal(`{"a":1}`))
			Expect(string(s.At(1))).To(Equal(`{"b":2}`))
			Expect(s.Len()).To(BeEquivalentTo(2))
		})

		It("Should return an error for invalid JSON", func() {
			data := []chan int{make(chan int)}
			Expect(telem.NewJSONSeries(data)).Error().
				To(MatchError(ContainSubstring("json: unsupported type: chan int")))
		})
	})

	Describe("NewJSONSeriesV", func() {
		It("Should marshal variadic JSON values", func() {
			s := MustSucceed(telem.NewJSONSeriesV(
				map[string]any{"a": 1},
				map[string]any{"b": 2},
			))
			Expect(s.Len()).To(Equal(int64(2)))
			Expect(string(s.At(0))).To(Equal(`{"a":1}`))
			Expect(string(s.At(1))).To(Equal(`{"b":2}`))
		})

		It("Should marshal a slice value", func() {
			s := MustSucceed(telem.NewJSONSeriesV([]int{1, 2, 3}))
			Expect(string(s.At(0))).To(Equal(`[1,2,3]`))
		})

		It("Should roundtrip through UnmarshalJSONSeries", func() {
			s := MustSucceed(telem.NewJSONSeriesV([]int{1, 2, 3}))
			unmarshalled := MustSucceed(telem.UnmarshalJSONSeries[[]int](s))
			Expect(unmarshalled).To(Equal([][]int{{1, 2, 3}}))
		})

		It("Should return an error for invalid JSON", func() {
			Expect(telem.NewJSONSeriesV(make(chan int))).Error().
				To(MatchError(ContainSubstring("json: unsupported type: chan int")))
		})
	})

	Describe("UnmarshalJSONSeries", func() {
		It("Should unmarshal JSON into typed values", func() {
			s := MustSucceed(telem.NewJSONSeriesV([]int{1, 2, 3}))
			data := MustSucceed(telem.UnmarshalJSONSeries[[]int](s))
			Expect(data).To(Equal([][]int{{1, 2, 3}}))
		})

		It("Should unmarshal multiple JSON values", func() {
			s := MustSucceed(telem.NewJSONSeriesV(
				map[string]any{"a": 1.0},
				map[string]any{"b": 2.0},
			))
			data := MustSucceed(telem.UnmarshalJSONSeries[map[string]any](s))
			Expect(data).To(HaveLen(2))
			Expect(data[0]).To(HaveKeyWithValue("a", 1.0))
			Expect(data[1]).To(HaveKeyWithValue("b", 2.0))
		})

		It("Should return an error when unmarshalling into wrong type", func() {
			s := MustSucceed(telem.NewJSONSeriesV([]int{1, 2, 3}))
			Expect(telem.UnmarshalJSONSeries[string](s)).Error().
				To(MatchError(ContainSubstring(
					"json: cannot unmarshal array into Go value of type string",
				)))
		})
	})

	Describe("Arrange", func() {
		It("Should create a series with the correct values for int64", func() {
			s := telem.Arrange[int64](0, 5, 2)
			Expect(s.Len()).To(BeEquivalentTo(5))
			Expect(s.DataType).To(Equal(telem.Int64T))
			Expect(telem.UnmarshalSeries[int64](s)).To(Equal([]int64{0, 2, 4, 6, 8}))
		})

		It("Should create a series with the correct values for float64", func() {
			s := telem.Arrange(0.0, 5, 0.5)
			Expect(s.Len()).To(BeEquivalentTo(5))
			Expect(s.DataType).To(Equal(telem.Float64T))
			Expect(telem.UnmarshalSeries[float64](s)).To(Equal([]float64{0.0, 0.5, 1.0, 1.5, 2.0}))
		})

		It("Should create a series with a single value when count is 1", func() {
			s := telem.Arrange[int32](10, 1, 5)
			Expect(s.Len()).To(BeEquivalentTo(1))
			Expect(s.DataType).To(Equal(telem.Int32T))
			Expect(telem.UnmarshalSeries[int32](s)).To(Equal([]int32{10}))
		})

		It("Should create a series with negative spacing", func() {
			s := telem.Arrange[int64](10, 5, -2)
			Expect(s.Len()).To(BeEquivalentTo(5))
			Expect(telem.UnmarshalSeries[int64](s)).To(Equal([]int64{10, 8, 6, 4, 2}))
		})

		It("Should create a series with timestamps", func() {
			s := telem.Arrange[telem.TimeStamp](0, 3, 100)
			Expect(s.DataType).To(Equal(telem.TimeStampT))
			Expect(telem.UnmarshalSeries[telem.TimeStamp](s)).
				To(Equal([]telem.TimeStamp{0, 100, 200}))
		})

		It("Should panic when count is less than 0", func() {
			Expect(func() {
				telem.Arrange[int64](0, -1, 1)
			}).To(Panic())
		})
	})

	Describe("NewSeriesFromAny", func() {
		id := uuid.New()

		DescribeTable("happy path", func(input any, dt telem.DataType, expected telem.Series) {
			s := telem.NewSeriesFromAny(input, dt)
			Expect(s).To(Equal(expected))
		},
			// same-type numeric
			Entry("int → Int64T", 42, telem.Int64T, telem.NewSeriesV(int64(42))),
			Entry("int64 → Int64T", int64(100), telem.Int64T, telem.NewSeriesV(int64(100))),
			Entry("int32 → Int32T", int32(50), telem.Int32T, telem.NewSeriesV(int32(50))),
			Entry("int16 → Int16T", int16(25), telem.Int16T, telem.NewSeriesV(int16(25))),
			Entry("int8 → Int8T", int8(12), telem.Int8T, telem.NewSeriesV(int8(12))),
			Entry("uint64 → Uint64T", uint64(200), telem.Uint64T, telem.NewSeriesV(uint64(200))),
			Entry("uint32 → Uint32T", uint32(150), telem.Uint32T, telem.NewSeriesV(uint32(150))),
			Entry("uint16 → Uint16T", uint16(75), telem.Uint16T, telem.NewSeriesV(uint16(75))),
			Entry("uint8 → Uint8T", uint8(37), telem.Uint8T, telem.NewSeriesV(uint8(37))),
			Entry("float64 → Float64T", 3.14, telem.Float64T, telem.NewSeriesV(float64(3.14))),
			Entry("float32 → Float32T", float32(2.5), telem.Float32T, telem.NewSeriesV(float32(2.5))),

			// cross-type numeric conversions
			Entry("int → Float64T", 42, telem.Float64T, telem.NewSeriesV(float64(42))),
			Entry("float64 → Int64T (truncates)", 42.7, telem.Int64T, telem.NewSeriesV(int64(42))),
			Entry("uint32 → Int32T", uint32(100), telem.Int32T, telem.NewSeriesV(int32(100))),
			Entry("int32 → Uint32T", int32(50), telem.Uint32T, telem.NewSeriesV(uint32(50))),
			Entry("float32 → Float64T", float32(1.5), telem.Float64T, telem.NewSeriesV(float64(float32(1.5)))),
			Entry("float64 → Float32T", 2.5, telem.Float32T, telem.NewSeriesV(float32(2.5))),
			Entry("int64 → Int32T", int64(100), telem.Int32T, telem.NewSeriesV(int32(100))),
			Entry("int16 → Int8T", int16(25), telem.Int8T, telem.NewSeriesV(int8(25))),
			Entry("int8 → Int64T", int8(12), telem.Int64T, telem.NewSeriesV(int64(12))),
			Entry("uint8 → Uint64T", uint8(37), telem.Uint64T, telem.NewSeriesV(uint64(37))),
			Entry("uint → Int64T", uint(42), telem.Int64T, telem.NewSeriesV(int64(42))),

			// edge values
			Entry("zero → Int64T", 0, telem.Int64T, telem.NewSeriesV(int64(0))),
			Entry("negative → Int32T", -42, telem.Int32T, telem.NewSeriesV(int32(-42))),
			Entry("max uint64", uint64(18446744073709551615), telem.Uint64T, telem.NewSeriesV(uint64(18446744073709551615))),

			// timestamp
			Entry("TimeStamp → TimeStampT", telem.TimeStamp(1000), telem.TimeStampT, telem.NewSeriesV(telem.TimeStamp(1000))),
			Entry("int64 → TimeStampT", int64(5000), telem.TimeStampT, telem.NewSeriesV(telem.TimeStamp(5000))),

			// string
			Entry("string → StringT", "hello", telem.StringT, telem.NewSeriesV("hello")),
			Entry("int → StringT", 42, telem.StringT, telem.NewSeriesV("42")),
			Entry("float → StringT", 3.14, telem.StringT, telem.NewSeriesV("3.14")),
			Entry("uuid → StringT", id, telem.StringT, telem.NewSeriesV(id.String())),

			// uuid
			Entry("uuid → UUIDT", id, telem.UUIDT, telem.NewSeriesV(id)),
			Entry("string → UUIDT", id.String(), telem.UUIDT, telem.NewSeriesV(id)),
			Entry("[]byte → UUIDT", id[:], telem.UUIDT, telem.NewSeriesV(id)),

			// bytes
			Entry("[]byte → BytesT", []byte{1, 2, 3}, telem.BytesT, telem.NewSeriesV([]byte{1, 2, 3})),
			Entry("string → BytesT", "hello", telem.BytesT, telem.NewSeriesV([]byte("hello"))),
			Entry("byte -> BytesT", byte(1), telem.BytesT, telem.NewSeriesV([]byte{1})),
			Entry("uint8 -> BytesT", uint8(2), telem.BytesT, telem.NewSeriesV([]byte{2})),
			Entry("uint16 -> BytesT", uint16(3), telem.BytesT, telem.NewSeriesV([]byte{3, 0})),
			Entry("uint32 -> BytesT", uint32(4), telem.BytesT, telem.NewSeriesV([]byte{4, 0, 0, 0})),
			Entry("uint64 -> BytesT", uint64(5), telem.BytesT, telem.NewSeriesV([]byte{5, 0, 0, 0, 0, 0, 0, 0})),
			Entry("int8 -> BytesT", int8(6), telem.BytesT, telem.NewSeriesV([]byte{6})),
			Entry("int16 -> BytesT", int16(7), telem.BytesT, telem.NewSeriesV([]byte{7, 0})),
			Entry("int32 -> BytesT", int32(8), telem.BytesT, telem.NewSeriesV([]byte{8, 0, 0, 0})),
			Entry("int64 -> BytesT", int64(9), telem.BytesT, telem.NewSeriesV([]byte{9, 0, 0, 0, 0, 0, 0, 0})),
			Entry("timestamp -> BytesT", telem.TimeStamp(1), telem.BytesT, telem.NewSeriesV([]byte{1, 0, 0, 0, 0, 0, 0, 0})),
			Entry("float32 -> BytesT", float32(1.5), telem.BytesT, telem.NewSeriesV(telem.ByteOrder.AppendUint32(nil, math.Float32bits(1.5)))),
			Entry("float64 -> BytesT", 2.5, telem.BytesT, telem.NewSeriesV(telem.ByteOrder.AppendUint64(nil, math.Float64bits(2.5)))),
			Entry("uuid -> BytesT", id, telem.BytesT, telem.NewSeriesV(id[:])),
			// json
			Entry("int → JSONT", 42, telem.JSONT, MustSucceed(telem.NewJSONSeriesV(42))),
		)

		DescribeTable("should panic", func(input any, dt telem.DataType, msg string) {
			Expect(func() {
				telem.NewSeriesFromAny(input, dt)
			}).To(PanicWith(ContainSubstring(msg)))
		},
			Entry("string → Int64T", "not a number", telem.Int64T, "cannot cast string to int64"),
			Entry("string → TimeStampT", "2024-01-01", telem.TimeStampT, "cannot cast string to telem.TimeStamp"),
			Entry("int → UUIDT", 42, telem.UUIDT, "cannot cast int to uuid.UUID"),
			Entry("invalid string → UUIDT", "not-a-uuid", telem.UUIDT, "invalid UUID"),
			Entry("short []byte → UUIDT", []byte{1, 2, 3}, telem.UUIDT, "invalid UUID (got 3 bytes)"),
			Entry("nil → Int64T", nil, telem.Int64T, "cannot cast <nil> to int64"),
			Entry("int → unsupported", 42, telem.UnknownT, "unsupported data type"),
			Entry("chan int → BytesT", make(chan int), telem.BytesT, "cannot cast chan int to []byte"),
		)
	})
})
