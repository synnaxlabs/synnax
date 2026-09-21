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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

// marshalSample converts the first sample of s and returns its JSON text.
func marshalSample(
	s telem.Series,
	target xjson.Type,
	enums xjson.ReverseEnumMap,
) string {
	GinkgoHelper()
	v := MustSucceed(xjson.FromSample(s.DataType, s.At(0), target, enums))
	return string(MustSucceed(json.Marshal(v)))
}

var _ = Describe("FromSample", func() {
	Describe("ParseType", func() {
		DescribeTable("Should parse each type and round trip its text",
			func(text string, expected xjson.Type) {
				Expect(MustSucceed(xjson.ParseType(text))).To(Equal(expected))
				Expect(expected.String()).To(Equal(text))
			},
			Entry("number", "number", xjson.Number),
			Entry("string", "string", xjson.String),
			Entry("boolean", "boolean", xjson.Boolean),
		)
		It("Should reject an unknown type", func() {
			Expect(xjson.ParseType("array")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`unknown JSON type "array"`)),
			))
		})
	})

	Describe("ZeroValue", func() {
		DescribeTable("Should return the zero value of each type",
			func(t xjson.Type, expected string) {
				Expect(MustSucceed(json.Marshal(xjson.ZeroValue(t)))).
					To(BeEquivalentTo(expected))
			},
			Entry("number", xjson.Number, "0"),
			Entry("string", xjson.String, `""`),
			Entry("boolean", xjson.Boolean, "false"),
		)
	})

	Describe("CheckFromSample", func() {
		DescribeTable("Should accept a supported conversion",
			func(dt telem.DataType, target xjson.Type) {
				Expect(xjson.CheckFromSample(dt, target)).To(Succeed())
			},
			Entry("float64 to number", telem.Float64T, xjson.Number),
			Entry("int8 to string", telem.Int8T, xjson.String),
			Entry("uint64 to boolean", telem.Uint64T, xjson.Boolean),
			Entry("string to string", telem.StringT, xjson.String),
		)
		DescribeTable("Should reject an unsupported conversion",
			func(dt telem.DataType, target xjson.Type, message string) {
				Expect(xjson.CheckFromSample(dt, target)).To(SatisfyAll(
					MatchError(xjson.ErrConversion),
					MatchError(ContainSubstring(message)),
				))
			},
			Entry("string to number",
				telem.StringT, xjson.Number, "cannot convert string to a JSON number"),
			Entry(
				"string to boolean",
				telem.StringT,
				xjson.Boolean,
				"cannot convert string to a JSON boolean",
			),
			Entry("timestamp to number",
				telem.TimestampT, xjson.Number,
				"cannot convert timestamp to a JSON number"),
			Entry("uuid to string",
				telem.UUIDT, xjson.String, "cannot convert uuid to a JSON string"),
		)
	})

	Describe("FromSample", func() {
		DescribeTable("Should convert to a number",
			func(s telem.Series, expected string) {
				Expect(marshalSample(s, xjson.Number, nil)).To(Equal(expected))
			},
			Entry("float64", telem.NewSeriesV(1.5), "1.5"),
			Entry("float32", telem.NewSeriesV[float32](0.1), "0.10000000149011612"),
			Entry("int64 minimum",
				telem.NewSeriesV[int64](math.MinInt64), "-9223372036854775808"),
			Entry("int32", telem.NewSeriesV[int32](-7), "-7"),
			Entry("int16", telem.NewSeriesV[int16](-300), "-300"),
			Entry("int8", telem.NewSeriesV[int8](-128), "-128"),
			Entry("uint64 maximum",
				telem.NewSeriesV[uint64](math.MaxUint64), "18446744073709551615"),
			Entry("uint32", telem.NewSeriesV[uint32](4000000000), "4000000000"),
			Entry("uint16", telem.NewSeriesV[uint16](65535), "65535"),
			Entry("uint8", telem.NewSeriesV[uint8](255), "255"),
		)
		DescribeTable("Should convert to a boolean",
			func(s telem.Series, expected string) {
				Expect(marshalSample(s, xjson.Boolean, nil)).To(Equal(expected))
			},
			Entry("zero", telem.NewSeriesV[uint8](0), "false"),
			Entry("one", telem.NewSeriesV[uint8](1), "true"),
			Entry("negative", telem.NewSeriesV[int32](-2), "true"),
			Entry("fraction", telem.NewSeriesV(0.25), "true"),
		)
		DescribeTable(
			"Should convert to a string",
			func(s telem.Series, enums xjson.ReverseEnumMap, expected string) {
				Expect(marshalSample(s, xjson.String, enums)).To(Equal(expected))
			},
			Entry("string", telem.NewSeriesV("open"), nil, `"open"`),
			Entry("float64", telem.NewSeriesV(2.5), nil, `"2.5"`),
			Entry(
				"float32 shortest text",
				telem.NewSeriesV[float32](0.1),
				nil,
				`"0.1"`,
			),
			Entry("negative integer", telem.NewSeriesV[int16](-12), nil, `"-12"`),
			Entry("enum label",
				telem.NewSeriesV[uint8](1), xjson.ReverseEnumMap{1: "open"}, `"open"`),
			Entry("no matching enum label",
				telem.NewSeriesV[uint8](2), xjson.ReverseEnumMap{1: "open"}, `"2"`),
		)
		It("Should not apply enum labels to a number target", func() {
			s := telem.NewSeriesV[uint8](1)
			enums := xjson.ReverseEnumMap{1: "open"}
			Expect(marshalSample(s, xjson.Number, enums)).To(Equal("1"))
		})
		It("Should convert a sample that is not the first of its series", func() {
			s := telem.NewSeriesV[int32](1, 2, 3)
			v := MustSucceed(xjson.FromSample(s.DataType, s.At(2), xjson.Number, nil))
			Expect(v).To(BeEquivalentTo(3))
		})
		DescribeTable("Should reject a sample it cannot convert",
			func(s telem.Series, target xjson.Type, message string) {
				Expect(xjson.FromSample(s.DataType, s.At(0), target, nil)).Error().
					To(SatisfyAll(
						MatchError(xjson.ErrConversion),
						MatchError(ContainSubstring(message)),
					))
			},
			Entry("string to number", telem.NewSeriesV("a"), xjson.Number,
				"cannot convert string to a JSON number"),
			Entry("timestamp", telem.NewSeriesV(telem.TimeStamp(1)), xjson.Number,
				"cannot convert timestamp to a JSON number"),
			Entry("NaN", telem.NewSeriesV(math.NaN()), xjson.Number,
				"value is not finite"),
			Entry("infinity", telem.NewSeriesV(math.Inf(1)), xjson.String,
				"value is not finite"),
		)
	})

	Describe("FromTimeStamp", func() {
		const ts = telem.TimeStamp(1767323045678000000)
		DescribeTable("Should format a timestamp",
			func(ts telem.TimeStamp, format xjson.TimeFormat, expected string) {
				v := xjson.FromTimeStamp(ts, format)
				Expect(MustSucceed(json.Marshal(v))).To(BeEquivalentTo(expected))
			},
			Entry("ISO 8601", ts, xjson.ISO8601, `"2026-01-02T03:04:05.678Z"`),
			Entry("ISO 8601 whole second",
				telem.TimeStamp(1767323045000000000), xjson.ISO8601,
				`"2026-01-02T03:04:05Z"`),
			Entry("nanoseconds", ts+1, xjson.UnixNanosecond, "1767323045678000001"),
			Entry("microseconds", ts, xjson.UnixMicrosecond, "1767323045678000"),
			Entry("milliseconds", ts, xjson.UnixMillisecond, "1767323045678"),
			Entry("seconds", ts, xjson.UnixSecond, "1767323045.678"),
			Entry("fractional microseconds",
				ts+1, xjson.UnixMicrosecond, "1767323045678000.001"),
			Entry("before the epoch", telem.TimeStamp(-1500), xjson.UnixMicrosecond,
				"-1.5"),
			Entry("less than one unit before the epoch",
				telem.TimeStamp(-500), xjson.UnixMicrosecond, "-0.5"),
			Entry("the epoch", telem.TimeStamp(0), xjson.UnixSecond, "0"),
		)
		It("Should round trip through AppendSample", func() {
			v := xjson.FromTimeStamp(ts, xjson.ISO8601)
			data := MustSucceed(
				xjson.AppendSample(nil, telem.TimestampT, v, xjson.ISO8601, nil),
			)
			s := telem.Series{DataType: telem.TimestampT, Data: data}
			Expect(s).To(telem.MatchSeriesData(telem.NewSeriesV(ts)))
		})
	})
})
