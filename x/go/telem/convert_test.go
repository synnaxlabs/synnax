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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Convert", func() {
	DescribeTable("Should convert the samples of a series to a data type",
		func(s telem.Series, dt telem.DataType, expected telem.Series) {
			Expect(MustSucceed(telem.Convert(s, dt))).To(telem.MatchSeries(expected))
		},
		Entry("int16 to float64",
			telem.NewSeriesV[int16](-1, 300),
			telem.Float64T,
			telem.NewSeriesV[float64](-1, 300),
		),
		Entry("float64 to int32, truncating",
			telem.NewSeriesV(1.9, -1.9),
			telem.Int32T,
			telem.NewSeriesV[int32](1, -1),
		),
		Entry("int64 to float64, losing integers above 2^53",
			telem.NewSeriesV[int64](1<<53+1),
			telem.Float64T,
			telem.NewSeriesV[float64](1<<53),
		),
		Entry("uint8 to boolean",
			telem.NewSeriesV[uint8](0, 2),
			telem.BooleanT,
			telem.NewSeriesV(false, true),
		),
		Entry("boolean to float32",
			telem.NewSeriesV(true, false),
			telem.Float32T,
			telem.NewSeriesV[float32](1, 0),
		),
		Entry("timestamp to int64",
			telem.NewSeriesV[telem.TimeStamp](-1, telem.SecondTS),
			telem.Int64T,
			telem.NewSeriesV[int64](-1, int64(telem.SecondTS)),
		),
		Entry("float32 to timestamp",
			telem.NewSeriesV[float32](2),
			telem.TimestampT,
			telem.NewSeriesV[telem.TimeStamp](2),
		),
	)

	It("Should keep every other field of the series", func() {
		s := telem.NewSeriesV[int16](1, 2)
		s.Alignment, s.AlignmentMultiple, s.Clock = 10, 2, 7
		s.TimeRange = telem.TimeRange{Start: 1, End: 3}
		expected := telem.NewSeriesV[float32](1, 2)
		expected.Alignment, expected.AlignmentMultiple, expected.Clock = 10, 2, 7
		expected.TimeRange = s.TimeRange
		Expect(MustSucceed(telem.Convert(s, telem.Float32T))).
			To(telem.MatchSeries(expected))
	})

	It("Should return a series of the data type unchanged", func() {
		s := telem.NewSeriesV[int16](1, 2)
		converted := MustSucceed(telem.Convert(s, telem.Int16T))
		Expect(&converted.Data[0]).To(BeIdenticalTo(&s.Data[0]))
	})

	DescribeTable("Should fail between text and numbers",
		func(s telem.Series, dt telem.DataType, msg string) {
			Expect(telem.Convert(s, dt)).Error().To(MatchError(msg))
		},
		Entry("string to float32",
			telem.NewSeriesV("a"),
			telem.Float32T,
			"cannot convert a string series to float32",
		),
		Entry("float32 to string",
			telem.NewSeriesV[float32](1),
			telem.StringT,
			"cannot convert a float32 series to string",
		),
		Entry("uuid to int64",
			telem.Series{DataType: telem.UUIDT, Data: make([]byte, 16)},
			telem.Int64T,
			"cannot convert a uuid series to int64",
		),
	)

	DescribeTable("Should hold both ends of each integer range",
		func(s telem.Series, dt telem.DataType, expected telem.Series) {
			Expect(MustSucceed(telem.Convert(s, dt))).To(telem.MatchSeries(expected))
		},
		Entry("uint8", telem.NewSeriesV[int16](0, 255), telem.Uint8T,
			telem.NewSeriesV[uint8](0, 255)),
		Entry("uint16", telem.NewSeriesV[int32](0, 65535), telem.Uint16T,
			telem.NewSeriesV[uint16](0, 65535)),
		Entry("uint32", telem.NewSeriesV[int64](0, 4294967295), telem.Uint32T,
			telem.NewSeriesV[uint32](0, 4294967295)),
		Entry("uint64", telem.NewSeriesV[int64](0, math.MaxInt64), telem.Uint64T,
			telem.NewSeriesV[uint64](0, math.MaxInt64)),
		Entry("int8", telem.NewSeriesV[int16](-128, 127), telem.Int8T,
			telem.NewSeriesV[int8](-128, 127)),
		Entry("int16", telem.NewSeriesV[int32](-32768, 32767), telem.Int16T,
			telem.NewSeriesV[int16](-32768, 32767)),
		Entry("int32", telem.NewSeriesV[int64](math.MinInt32, math.MaxInt32),
			telem.Int32T, telem.NewSeriesV[int32](math.MinInt32, math.MaxInt32)),
		Entry("int64", telem.NewSeriesV[uint64](0, math.MaxInt64), telem.Int64T,
			telem.NewSeriesV[int64](0, math.MaxInt64)),
		Entry("int64 from a float", telem.NewSeriesV(-0x1p63), telem.Int64T,
			telem.NewSeriesV[int64](math.MinInt64)),
	)

	DescribeTable("Should fail on a sample out of the range of the data type",
		func(s telem.Series, dt telem.DataType, msg string) {
			Expect(telem.Convert(s, dt)).Error().To(SatisfyAll(
				MatchError(telem.ErrOutOfRange),
				MatchError(ContainSubstring(msg)),
				BeAssignableToTypeOf(telem.OutOfRangeError{}),
			))
		},
		Entry("above uint8", telem.NewSeriesV[int16](256), telem.Uint8T,
			"value 256 of an int16 series is out of range of uint8"),
		Entry("below uint8", telem.NewSeriesV[int16](-1), telem.Uint8T,
			"value -1 of an int16 series is out of range of uint8"),
		Entry("above uint16", telem.NewSeriesV[int32](65536), telem.Uint16T,
			"value 65536 of an int32 series is out of range of uint16"),
		Entry("above uint32", telem.NewSeriesV[int64](4294967296), telem.Uint32T,
			"value 4294967296 of an int64 series is out of range of uint32"),
		Entry("below uint64", telem.NewSeriesV[int64](-1), telem.Uint64T,
			"value -1 of an int64 series is out of range of uint64"),
		Entry("above uint64", telem.NewSeriesV(0x1p64), telem.Uint64T,
			"of a float64 series is out of range of uint64"),
		Entry("above int8", telem.NewSeriesV[int16](128), telem.Int8T,
			"value 128 of an int16 series is out of range of int8"),
		Entry("below int8", telem.NewSeriesV[int16](-129), telem.Int8T,
			"value -129 of an int16 series is out of range of int8"),
		Entry("above int16", telem.NewSeriesV[int32](32768), telem.Int16T,
			"value 32768 of an int32 series is out of range of int16"),
		Entry("below int16", telem.NewSeriesV[int32](-32769), telem.Int16T,
			"value -32769 of an int32 series is out of range of int16"),
		Entry("above int32", telem.NewSeriesV[int64](math.MaxInt32+1), telem.Int32T,
			"value 2147483648 of an int64 series is out of range of int32"),
		Entry("below int32", telem.NewSeriesV[int64](math.MinInt32-1), telem.Int32T,
			"value -2147483649 of an int64 series is out of range of int32"),
		Entry("above int64", telem.NewSeriesV[uint64](math.MaxInt64+1), telem.Int64T,
			"value 9223372036854775808 of a uint64 series is out of range of int64"),
		Entry("below int64", telem.NewSeriesV(-0x1p64), telem.Int64T,
			"of a float64 series is out of range of int64"),
		Entry("NaN to an integer", telem.NewSeriesV(math.NaN()), telem.Int32T,
			"value NaN of a float64 series is out of range of int32"),
		Entry("above float32", telem.NewSeriesV(1e39), telem.Float32T,
			"of a float64 series is out of range of float32"),
	)

	It("Should keep a NaN and an infinity from float64 to float32", func() {
		converted := MustSucceed(telem.Convert(
			telem.NewSeriesV(math.NaN(), math.Inf(1)),
			telem.Float32T,
		))
		values := converted.Unmarshal[float32]()
		Expect(math.IsNaN(float64(values[0]))).To(BeTrue())
		Expect(math.IsInf(float64(values[1]), 1)).To(BeTrue())
	})

	Describe("ValueAs", func() {
		It("Should return a sample converted to the type", func() {
			Expect(
				MustSucceed(
					telem.ValueAs[float64](telem.NewSeriesV[int32](1, 300), -1),
				),
			).
				To(Equal(300.0))
			Expect(MustSucceed(telem.ValueAs[int32](telem.NewSeriesV(2.9, -2.9), 1))).
				To(Equal(int32(-2)))
		})

		It("Should return a sample of the same type as it is", func() {
			Expect(MustSucceed(telem.ValueAs[int16](telem.NewSeriesV[int16](-7), 0))).
				To(Equal(int16(-7)))
		})

		It("Should fail on a sample out of range", func() {
			Expect(telem.ValueAs[uint8](telem.NewSeriesV[int16](300), 0)).Error().
				To(SatisfyAll(
					MatchError(telem.ErrOutOfRange),
					MatchError(ContainSubstring(
						"value 300 of an int16 series is out of range of uint8",
					)),
				))
		})

		It("Should fail on text", func() {
			Expect(telem.ValueAs[float32](telem.NewSeriesV("a"), 0)).Error().
				To(MatchError("cannot convert a string series to float32"))
		})

		It("Should convert a sample with no allocation", func() {
			s := telem.NewSeriesV[int16](1, 300)
			var err error
			Expect(testing.AllocsPerRun(100, func() {
				_, err = telem.ValueAs[float64](s, -1)
			})).To(BeZero())
			Expect(err).To(Succeed())
		})
	})
})
