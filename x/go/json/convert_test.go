// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package json_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

func decode(text string) any {
	GinkgoHelper()
	return MustSucceed(xjson.Decode([]byte(text)))
}

func convert(
	text string,
	dt telem.DataType,
	format xjson.TimeFormat,
	enums xjson.EnumMap,
) telem.Series {
	GinkgoHelper()
	data := MustSucceed(xjson.AppendSample(nil, dt, decode(text), format, enums))
	return telem.Series{DataType: dt, Data: data}
}

var _ = Describe("Convert", func() {
	Describe("Decode", func() {
		It("Should keep the exact text of a number", func() {
			Expect(decode(`{"a":[18446744073709551615]}`)).To(Equal(map[string]any{
				"a": []any{xjson.NumberText("18446744073709551615")},
			}))
		})
		It("Should keep the last value of a duplicate name", func() {
			Expect(decode(`{"a":1,"a":2}`)).
				To(Equal(map[string]any{"a": xjson.NumberText("2")}))
		})
		It("Should replace invalid UTF-8", func() {
			Expect(decode("\"a\xffb\"")).To(Equal("a\uFFFDb"))
		})
		It("Should reject text after the document", func() {
			Expect(xjson.Decode([]byte("1 2"))).Error().To(MatchError(
				ContainSubstring("invalid character '2' after top-level value"),
			))
		})
	})

	Describe("Marshal", func() {
		It("Should sort object names and keep number text", func() {
			doc := map[string]any{"b": xjson.NumberText("1.50"), "a": "x"}
			Expect(xjson.Marshal(doc)).To(BeEquivalentTo(`{"a":"x","b":1.50}`))
		})
	})

	Describe("ParseTimeFormat", func() {
		DescribeTable("Should parse each format",
			func(text string, expected xjson.TimeFormat) {
				Expect(MustSucceed(xjson.ParseTimeFormat(text))).To(Equal(expected))
			},
			Entry("iso8601", "iso8601", xjson.ISO8601),
			Entry("unix_sec", "unix_sec", xjson.UnixSecond),
			Entry("unix_ms", "unix_ms", xjson.UnixMillisecond),
			Entry("unix_us", "unix_us", xjson.UnixMicrosecond),
			Entry("unix_ns", "unix_ns", xjson.UnixNanosecond),
		)
		It("Should reject an unknown format", func() {
			Expect(xjson.ParseTimeFormat("unix_min")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`unknown time format "unix_min"`)),
			))
		})
	})

	Describe("CheckToSample", func() {
		DescribeTable("Should accept a data type that AppendSample produces",
			func(dt telem.DataType) { Expect(xjson.CheckToSample(dt)).To(Succeed()) },
			Entry("float64", telem.Float64T),
			Entry("float32", telem.Float32T),
			Entry("int64", telem.Int64T),
			Entry("int32", telem.Int32T),
			Entry("int16", telem.Int16T),
			Entry("int8", telem.Int8T),
			Entry("uint64", telem.Uint64T),
			Entry("uint32", telem.Uint32T),
			Entry("uint16", telem.Uint16T),
			Entry("uint8", telem.Uint8T),
			Entry("timestamp", telem.TimestampT),
			Entry("string", telem.StringT),
		)
		DescribeTable("Should reject a data type that AppendSample cannot produce",
			func(dt telem.DataType) {
				Expect(xjson.CheckToSample(dt)).To(SatisfyAll(
					MatchError(xjson.ErrConversion),
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"cannot convert a JSON value to "+string(dt),
					)),
				))
			},
			Entry("uuid", telem.UUIDT),
			Entry("json", telem.JSONT),
			Entry("bytes", telem.BytesT),
		)
	})

	Describe("AppendSample", func() {
		It("Should append to the existing buffer", func() {
			data := MustSucceed(
				xjson.AppendSample(nil, telem.Int16T, decode("1"), xjson.ISO8601, nil),
			)
			data = MustSucceed(
				xjson.AppendSample(
					data,
					telem.Int16T,
					decode("-2"),
					xjson.ISO8601,
					nil,
				),
			)
			Expect(telem.Series{DataType: telem.Int16T, Data: data}).
				To(telem.MatchSeriesData(telem.NewSeriesV[int16](1, -2)))
		})

		DescribeTable("Should convert numeric values",
			func(text string, expected telem.Series) {
				Expect(convert(text, expected.DataType, xjson.ISO8601, nil)).
					To(telem.MatchSeriesData(expected))
			},
			Entry("a float to float64", "23.5", telem.NewSeriesV(23.5)),
			Entry("an integer to float32", "7", telem.NewSeriesV[float32](7)),
			Entry("a whole float to int32", "4.0", telem.NewSeriesV[int32](4)),
			Entry("an exponent to int32", "1e3", telem.NewSeriesV[int32](1000)),
			Entry("a negative integer to int8", "-128", telem.NewSeriesV[int8](-128)),
			Entry("the uint8 maximum", "255", telem.NewSeriesV[uint8](255)),
			Entry(
				"an int64 beyond float64 precision",
				"9007199254740993",
				telem.NewSeriesV[int64](9007199254740993),
			),
			Entry(
				"the uint64 maximum",
				"18446744073709551615",
				telem.NewSeriesV[uint64](18446744073709551615),
			),
			Entry(
				"a whole float beyond float64 precision",
				"9007199254740993.0",
				telem.NewSeriesV[int64](9007199254740993),
			),
			Entry(
				"an exponent beyond float64 precision",
				"9007199254740993e0",
				telem.NewSeriesV[int64](9007199254740993),
			),
			Entry(
				"the uint64 maximum with an exponent",
				"1.8446744073709551615E19",
				telem.NewSeriesV[uint64](18446744073709551615),
			),
			Entry(
				"a negative exponent that leaves an integer",
				"-12500e-2",
				telem.NewSeriesV[int16](-125),
			),
			Entry(
				"the int64 minimum with an exponent",
				"-9.223372036854775808e18",
				telem.NewSeriesV[int64](math.MinInt64),
			),
			Entry("a negative zero fraction", "-0.0", telem.NewSeriesV[uint8](0)),
			Entry("true to uint8", "true", telem.NewSeriesV[uint8](1)),
			Entry("false to float64", "false", telem.NewSeriesV[float64](0)),
			Entry("a numeric string to float64", `"12.25"`, telem.NewSeriesV(12.25)),
			Entry("a signed integer string", `"+42"`, telem.NewSeriesV[uint16](42)),
		)

		It("Should convert a float64 from a default decode", func() {
			data := MustSucceed(
				xjson.AppendSample(nil, telem.Int32T, float64(12), xjson.ISO8601, nil),
			)
			Expect(telem.Series{DataType: telem.Int32T, Data: data}).
				To(telem.MatchSeriesData(telem.NewSeriesV[int32](12)))
		})

		DescribeTable(
			"Should convert a Go integer",
			func(value any, dt telem.DataType, expected telem.Series) {
				data := MustSucceed(
					xjson.AppendSample(nil, dt, value, xjson.UnixNanosecond, nil),
				)
				Expect(telem.Series{DataType: dt, Data: data}).
					To(telem.MatchSeriesData(expected))
			},
			Entry(
				"int64 to int32",
				int64(-12),
				telem.Int32T,
				telem.NewSeriesV[int32](-12),
			),
			Entry(
				"int64 to uint16",
				int64(12),
				telem.Uint16T,
				telem.NewSeriesV[uint16](12),
			),
			Entry(
				"uint64 to float64",
				uint64(12),
				telem.Float64T,
				telem.NewSeriesV(12.0),
			),
			Entry(
				"uint64 to string",
				uint64(12),
				telem.StringT,
				telem.NewSeriesV("12"),
			),
			Entry(
				"int64 nanoseconds to timestamp",
				int64(1_000),
				telem.TimestampT,
				telem.NewSeriesV(telem.TimeStamp(1_000)),
			),
		)

		It("Should reject a negative int64 for an unsigned type", func() {
			Expect(
				xjson.AppendSample(nil, telem.Uint8T, int64(-1), xjson.ISO8601, nil),
			).
				Error().
				To(MatchError(ContainSubstring("out of bounds")))
		})

		It("Should map a string through the enum map", func() {
			enums := xjson.EnumMap{"ON": 1, "OFF": 0}
			Expect(convert(`"ON"`, telem.Uint8T, xjson.ISO8601, enums)).
				To(telem.MatchSeriesData(telem.NewSeriesV[uint8](1)))
		})

		It("Should append strings in the series encoding", func() {
			data := MustSucceed(
				xjson.AppendSample(nil, telem.StringT, "open", xjson.ISO8601, nil),
			)
			data = MustSucceed(
				xjson.AppendSample(
					data,
					telem.StringT,
					decode("2"),
					xjson.ISO8601,
					nil,
				),
			)
			Expect(data).To(Equal([]byte{
				4, 0, 0, 0, 'o', 'p', 'e', 'n',
				1, 0, 0, 0, '2',
			}))
		})

		DescribeTable("Should convert to a string",
			func(text, expected string) {
				Expect(convert(text, telem.StringT, xjson.ISO8601, nil)).
					To(telem.MatchSeriesData(telem.NewSeriesV(expected)))
			},
			Entry("a string as is", `"open"`, "open"),
			Entry("a number as its JSON text", "23.5", "23.5"),
			Entry("an object as its JSON text", `{"a":1}`, `{"a":1}`),
			Entry("null as its JSON text", "null", "null"),
		)

		DescribeTable("Should convert timestamps",
			func(text string, format xjson.TimeFormat, expected telem.TimeStamp) {
				Expect(convert(text, telem.TimestampT, format, nil)).
					To(telem.MatchSeriesData(telem.NewSeriesV(expected)))
			},
			Entry("seconds", "1700000000", xjson.UnixSecond,
				telem.TimeStamp(1700000000)*telem.SecondTS),
			Entry("fractional seconds", "1.5", xjson.UnixSecond,
				telem.TimeStamp(1500)*telem.MillisecondTS),
			Entry("seconds with nanosecond digits", "1700000000.000000001",
				xjson.UnixSecond, telem.TimeStamp(1700000000000000001)),
			Entry("milliseconds with nanosecond digits", "1700000000123.456789",
				xjson.UnixMillisecond, telem.TimeStamp(1700000000123456789)),
			Entry("a fraction of a nanosecond toward zero", "-1.0000000009",
				xjson.UnixSecond, -telem.SecondTS),
			Entry("milliseconds", "1700000000123", xjson.UnixMillisecond,
				telem.TimeStamp(1700000000123)*telem.MillisecondTS),
			Entry("microseconds", "1700000000123456", xjson.UnixMicrosecond,
				telem.TimeStamp(1700000000123456)*telem.MicrosecondTS),
			Entry("exact nanoseconds", "1700000000123456789", xjson.UnixNanosecond,
				telem.TimeStamp(1700000000123456789)),
			Entry("exact nanoseconds with an exponent", "1.700000000123456789e18",
				xjson.UnixNanosecond, telem.TimeStamp(1700000000123456789)),
			Entry("a numeric string", `"1700000000"`, xjson.UnixSecond,
				telem.TimeStamp(1700000000)*telem.SecondTS),
			Entry("ISO 8601 in UTC", `"2023-11-14T22:13:20.5Z"`, xjson.ISO8601,
				telem.TimeStamp(1700000000)*telem.SecondTS+500*telem.MillisecondTS),
			Entry("ISO 8601 with an offset", `"2023-11-14T23:13:20+01:00"`,
				xjson.ISO8601, telem.TimeStamp(1700000000)*telem.SecondTS),
			Entry("ISO 8601 with a space and lowercase z", `"2023-11-14 22:13:20z"`,
				xjson.ISO8601, telem.TimeStamp(1700000000)*telem.SecondTS),
			Entry("the latest ISO 8601 timestamp", `"2262-04-11T23:47:16.854775807Z"`,
				xjson.ISO8601, telem.TimeStampMax),
			Entry("the earliest ISO 8601 timestamp", `"1677-09-21T00:12:43.145224192Z"`,
				xjson.ISO8601, telem.TimeStamp(math.MinInt64)),
		)

		DescribeTable("Should reject a value it cannot convert",
			func(
				text string,
				dt telem.DataType,
				format xjson.TimeFormat,
				message string,
			) {
				Expect(xjson.AppendSample(nil, dt, decode(text), format, nil)).
					Error().To(SatisfyAll(
					MatchError(xjson.ErrConversion),
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
			},
			Entry("a fraction for an integer", "1.5", telem.Int32T, xjson.ISO8601,
				"cannot convert 1.5 to int32: value has a fractional component"),
			Entry("above the float32 maximum", "1e40", telem.Float32T, xjson.ISO8601,
				"cannot convert 1e40 to float32: value is out of bounds"),
			Entry("above the uint8 maximum", "256", telem.Uint8T, xjson.ISO8601,
				"cannot convert 256 to uint8: value is out of bounds"),
			Entry("below the int8 minimum", "-129", telem.Int8T, xjson.ISO8601,
				"value is out of bounds"),
			Entry("a negative for an unsigned type", "-1", telem.Uint64T,
				xjson.ISO8601, "value is out of bounds"),
			Entry("above the int64 maximum", "9223372036854775808", telem.Int64T,
				xjson.ISO8601, "value is out of bounds"),
			Entry("a float above the int64 maximum", "1e19", telem.Int64T,
				xjson.ISO8601, "value is out of bounds"),
			Entry("a fraction beyond float64 precision", "9007199254740993.5",
				telem.Int64T, xjson.ISO8601, "value has a fractional component"),
			Entry("an exponent above the uint64 maximum", "1.8446744073709551616e19",
				telem.Uint64T, xjson.ISO8601, "value is out of bounds"),
			Entry("an exponent below the int64 minimum", "-9.223372036854775809e18",
				telem.Int64T, xjson.ISO8601, "value is out of bounds"),
			Entry("an exponent at the int maximum", "1e9223372036854775807",
				telem.Int64T, xjson.ISO8601, "value is out of bounds"),
			Entry("trailing zeros with an exponent at the int maximum",
				"100e9223372036854775807", telem.Int64T, xjson.ISO8601,
				"value is out of bounds"),
			Entry("an exponent beyond the int maximum", "1e9223372036854775808",
				telem.Int64T, xjson.ISO8601, "value is out of bounds"),
			Entry("a fraction with an exponent at the int minimum",
				"1.5e-9223372036854775808", telem.Int64T, xjson.ISO8601,
				"value has a fractional component"),
			Entry("an exponent beyond the int minimum", "1e-9223372036854775809",
				telem.Int64T, xjson.ISO8601, "value has a fractional component"),
			Entry("a NaN string for a signed integer", `"NaN"`, telem.Int32T,
				xjson.ISO8601, `cannot convert "NaN" to int32: value is out of bounds`),
			Entry("a NaN string for an unsigned integer", `"NaN"`, telem.Uint8T,
				xjson.ISO8601, `cannot convert "NaN" to uint8: value is out of bounds`),
			Entry("a non-numeric string", `"abc"`, telem.Float64T, xjson.ISO8601,
				`cannot convert "abc" to float64: not a valid number`),
			Entry("an empty string", `""`, telem.Float64T, xjson.ISO8601,
				"not a valid number"),
			Entry("an object for a number", `{"a":1}`, telem.Float64T, xjson.ISO8601,
				`cannot convert {"a":1} to float64`),
			Entry("null for a number", "null", telem.Float64T, xjson.ISO8601,
				"cannot convert null to float64"),
			Entry("an unsupported target", "1", telem.UUIDT, xjson.ISO8601,
				"cannot convert 1 to uuid"),
			Entry("a number with the ISO 8601 format", "1700000000",
				telem.TimestampT, xjson.ISO8601,
				"numeric values cannot be converted with ISO 8601 format"),
			Entry("a malformed ISO 8601 string", `"yesterday"`, telem.TimestampT,
				xjson.ISO8601, "not a valid ISO 8601 timestamp"),
			Entry("ISO 8601 with no timezone", `"2023-11-14T22:13:20"`,
				telem.TimestampT, xjson.ISO8601, "not a valid ISO 8601 timestamp"),
			Entry("ISO 8601 after 2262", `"2500-01-01T00:00:00Z"`, telem.TimestampT,
				xjson.ISO8601, "value is out of bounds"),
			Entry("ISO 8601 before 1677", `"1600-01-01T00:00:00Z"`, telem.TimestampT,
				xjson.ISO8601, "value is out of bounds"),
			Entry("a boolean for a timestamp", "true", telem.TimestampT,
				xjson.UnixSecond, "cannot convert true to timestamp"),
			Entry("seconds that overflow", "1e30", telem.TimestampT,
				xjson.UnixSecond, "value is out of bounds"),
		)
	})
})
