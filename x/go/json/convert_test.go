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
	"encoding/json"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// decode parses JSON text the way a payload decoder does, keeping number precision.
func decode(text string) any {
	GinkgoHelper()
	dec := json.NewDecoder(strings.NewReader(text))
	dec.UseNumber()
	var v any
	Expect(dec.Decode(&v)).To(Succeed())
	return v
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

	Describe("SupportsSampleTarget", func() {
		DescribeTable("Should report support",
			func(dt telem.DataType, expected bool) {
				Expect(xjson.SupportsSampleTarget(dt)).To(Equal(expected))
			},
			Entry("float64", telem.Float64T, true),
			Entry("uint8", telem.Uint8T, true),
			Entry("timestamp", telem.TimestampT, true),
			Entry("string", telem.StringT, true),
			Entry("uuid", telem.UUIDT, false),
			Entry("json", telem.JSONT, false),
			Entry("bytes", telem.BytesT, false),
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

		It("Should map a string through the enum map", func() {
			enums := xjson.EnumMap{"ON": 1, "OFF": 0}
			Expect(convert(`"ON"`, telem.Uint8T, xjson.ISO8601, enums)).
				To(telem.MatchSeriesData(telem.NewSeriesV[uint8](1)))
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
			Entry("milliseconds", "1700000000123", xjson.UnixMillisecond,
				telem.TimeStamp(1700000000123)*telem.MillisecondTS),
			Entry("microseconds", "1700000000123456", xjson.UnixMicrosecond,
				telem.TimeStamp(1700000000123456)*telem.MicrosecondTS),
			Entry("exact nanoseconds", "1700000000123456789", xjson.UnixNanosecond,
				telem.TimeStamp(1700000000123456789)),
			Entry("a numeric string", `"1700000000"`, xjson.UnixSecond,
				telem.TimeStamp(1700000000)*telem.SecondTS),
			Entry("ISO 8601 in UTC", `"2023-11-14T22:13:20.5Z"`, xjson.ISO8601,
				telem.TimeStamp(1700000000)*telem.SecondTS+500*telem.MillisecondTS),
			Entry("ISO 8601 with an offset", `"2023-11-14T23:13:20+01:00"`,
				xjson.ISO8601, telem.TimeStamp(1700000000)*telem.SecondTS),
			Entry("ISO 8601 with a space and lowercase z", `"2023-11-14 22:13:20z"`,
				xjson.ISO8601, telem.TimeStamp(1700000000)*telem.SecondTS),
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
			Entry("a boolean for a timestamp", "true", telem.TimestampT,
				xjson.UnixSecond, "cannot convert true to timestamp"),
			Entry("seconds that overflow", "1e30", telem.TimestampT,
				xjson.UnixSecond, "value is out of bounds"),
		)
	})
})
