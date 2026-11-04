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
	"encoding/json/jsontext"
	"math"
	"time"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var onOff = []xjson.Enum{{Label: "OFF", Value: 0}, {Label: "ON", Value: 1}}

func sample[T telem.Sample](v T) []byte { return telem.NewSeriesV(v).Data }

func text(s string) []byte { return telem.MarshalVariableSample([]byte(s)) }

var id = uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

var _ = Describe("Field", func() {
	Describe("ParseType", func() {
		DescribeTable("Should round trip the config form",
			func(text string) {
				Expect(MustSucceed(xjson.ParseType(text)).String()).To(Equal(text))
			},
			Entry("number", "number"),
			Entry("string", "string"),
			Entry("boolean", "boolean"),
		)
		It("Should reject an unknown type", func() {
			Expect(xjson.ParseType("object")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`unknown JSON type "object"`)),
			))
		})
	})

	Describe("Decoder", func() {
		DescribeTable("Should convert a JSON value into a sample",
			func(f xjson.Field, raw string, expected []byte) {
				d := MustSucceed(xjson.NewDecoder(f))
				Expect(d.Append(nil, jsontext.Value(raw))).To(Equal(expected))
			},
			Entry("a float", xjson.Field{DataType: telem.Float64T}, `1.5`, sample(1.5)),
			Entry("a float32", xjson.Field{DataType: telem.Float32T}, `0.25`,
				sample[float32](0.25)),
			Entry("an int64 past the precision of a float",
				xjson.Field{DataType: telem.Int64T}, `9007199254740993`,
				sample[int64](9007199254740993)),
			Entry("the largest uint64", xjson.Field{DataType: telem.Uint64T},
				`18446744073709551615`, sample[uint64](math.MaxUint64)),
			Entry("a negative int8", xjson.Field{DataType: telem.Int8T}, `-2`,
				sample[int8](-2)),
			Entry("a whole float into an integer", xjson.Field{DataType: telem.Int16T},
				`-3.0`, sample[int16](-3)),
			Entry("true into a number", xjson.Field{DataType: telem.Uint8T}, `true`,
				sample[uint8](1)),
			Entry("a number into a boolean", xjson.Field{DataType: telem.BooleanT},
				`2`, []byte{1}),
			Entry("an enum label", xjson.Field{DataType: telem.Uint8T, Enums: onOff},
				`"ON"`, sample[uint8](1)),
			Entry("an escaped enum label",
				xjson.Field{DataType: telem.Uint8T, Enums: onOff}, `"\u004fN"`,
				sample[uint8](1)),
			Entry("a string", xjson.Field{DataType: telem.StringT}, `"a\nb"`,
				text("a\nb")),
			Entry("JSON as the document spells it", xjson.Field{DataType: telem.JSONT},
				`{ "z" : 1.0, "a" : [ 1 , 2 ] }`, text(`{"z":1.0,"a":[1,2]}`)),
			Entry("a scalar as JSON", xjson.Field{DataType: telem.JSONT}, `1e999`,
				text(`1e999`)),
			Entry("a UUID", xjson.Field{DataType: telem.UUIDT}, `"`+id.String()+`"`,
				id[:]),
			Entry(
				"an ISO 8601 timestamp",
				xjson.Field{DataType: telem.TimestampT, TimeFormat: xjson.ISO8601},
				`"2026-01-02T03:04:05.000000006Z"`,
				sample(
					telem.NewTimeStamp(time.Date(2026, 1, 2, 3, 4, 5, 6, time.UTC)),
				),
			),
			Entry(
				"an ISO 8601 timestamp with a space and a lowercase zone",
				xjson.Field{DataType: telem.TimestampT, TimeFormat: xjson.ISO8601},
				`"2026-01-02 03:04:05z"`,
				sample(
					telem.NewTimeStamp(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)),
				),
			),
			Entry("Unix seconds with every digit to the nanosecond",
				xjson.Field{DataType: telem.TimestampT, TimeFormat: xjson.UnixSecond},
				`1767323045.123456789`, sample(telem.TimeStamp(1767323045123456789))),
			Entry("Unix milliseconds in a string",
				xjson.Field{
					DataType:   telem.TimestampT,
					TimeFormat: xjson.UnixMillisecond,
				},
				`"1767323045123"`, sample(telem.TimeStamp(1767323045123000000))),
			Entry("Unix microseconds with an exponent",
				xjson.Field{
					DataType:   telem.TimestampT,
					TimeFormat: xjson.UnixMicrosecond,
				},
				`1.5e3`, sample(telem.TimeStamp(1500000))),
			Entry("negative Unix seconds",
				xjson.Field{DataType: telem.TimestampT, TimeFormat: xjson.UnixSecond},
				`-1.5`, sample(telem.TimeStamp(-1500000000))),
			Entry("a fraction of a nanosecond, truncated",
				xjson.Field{
					DataType:   telem.TimestampT,
					TimeFormat: xjson.UnixNanosecond,
				},
				`10.9`, sample(telem.TimeStamp(10))),
		)

		DescribeTable("Should reject a JSON value that does not convert",
			func(f xjson.Field, raw, message string) {
				d := MustSucceed(xjson.NewDecoder(f))
				dst := []byte{9}
				out, err := d.Append(dst, jsontext.Value(raw))
				Expect(err).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
				Expect(out).To(Equal(dst))
			},
			Entry("a number into a string", xjson.Field{DataType: telem.StringT}, `1`,
				"expected a string, got a number"),
			Entry("a string into a number", xjson.Field{DataType: telem.Float64T},
				`"1"`, `unknown enum label "1"`),
			Entry("null into a number", xjson.Field{DataType: telem.Float64T}, `null`,
				"expected a number or boolean, got null"),
			Entry("a number past an int8", xjson.Field{DataType: telem.Int8T}, `128`,
				"128 is out of range for int8"),
			Entry("a negative number into a uint", xjson.Field{DataType: telem.Uint8T},
				`-1`, "-1 is out of range for uint8"),
			Entry("a fraction into an integer", xjson.Field{DataType: telem.Int32T},
				`1.5`, "1.5 is not an integer"),
			Entry("a float past a float32", xjson.Field{DataType: telem.Float32T},
				`1e39`, "1e+39 overflows float32"),
			Entry("an object into a UUID", xjson.Field{DataType: telem.UUIDT}, `{}`,
				"expected a UUID string, got an object"),
			Entry("text that is not a UUID", xjson.Field{DataType: telem.UUIDT},
				`"x"`, `invalid UUID "x"`),
			Entry("a number in ISO 8601",
				xjson.Field{DataType: telem.TimestampT, TimeFormat: xjson.ISO8601}, `1`,
				"expected an ISO 8601 string, got a number"),
			Entry("text that is not ISO 8601",
				xjson.Field{DataType: telem.TimestampT, TimeFormat: xjson.ISO8601},
				`"now"`, `invalid ISO 8601 timestamp "now"`),
			Entry("a boolean in Unix seconds",
				xjson.Field{DataType: telem.TimestampT, TimeFormat: xjson.UnixSecond},
				`true`, "expected a number, got a boolean"),
			Entry("text that is not a number in Unix seconds",
				xjson.Field{DataType: telem.TimestampT, TimeFormat: xjson.UnixSecond},
				`"soon"`, `invalid number "soon"`),
			Entry("Unix seconds past the range of a timestamp",
				xjson.Field{DataType: telem.TimestampT, TimeFormat: xjson.UnixSecond},
				`1e12`, "1e12 is out of range for a unix_sec timestamp"),
			Entry("JSON with a repeated member", xjson.Field{DataType: telem.JSONT},
				`{"a":1,"a":2}`, "duplicate object member name"),
		)

		It("Should reject a data type that no JSON value converts to", func() {
			Expect(xjson.NewDecoder(xjson.Field{DataType: telem.BytesT})).Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(
						"cannot convert a JSON value to bytes",
					)),
				))
		})

		It("Should append samples one after another", func() {
			d := MustSucceed(xjson.NewDecoder(xjson.Field{DataType: telem.StringT}))
			dst := MustSucceed(d.Append(nil, jsontext.Value(`"a"`)))
			dst = MustSucceed(d.Append(dst, jsontext.Value(`"bc"`)))
			Expect(dst).To(Equal(append(text("a"), text("bc")...)))
		})
	})

	Describe("Encoder", func() {
		DescribeTable("Should convert a sample into a JSON value",
			func(f xjson.Field, s []byte, expected string) {
				e := MustSucceed(xjson.NewEncoder(f))
				Expect(string(MustSucceed(e.Append(nil, s)))).To(Equal(expected))
			},
			Entry("a whole float", xjson.Field{DataType: telem.Float64T},
				sample(100000.0), "100000"),
			Entry("a fraction", xjson.Field{DataType: telem.Float64T}, sample(0.1),
				"0.1"),
			Entry("negative zero", xjson.Field{DataType: telem.Float64T},
				sample(math.Copysign(0, -1)), "-0"),
			Entry("a large float", xjson.Field{DataType: telem.Float64T}, sample(1e21),
				"1e+21"),
			Entry("a small float", xjson.Field{DataType: telem.Float64T},
				sample(1e-7), "1e-7"),
			Entry("the largest uint64", xjson.Field{DataType: telem.Uint64T},
				sample[uint64](math.MaxUint64), "18446744073709551615"),
			Entry("a negative int16", xjson.Field{DataType: telem.Int16T},
				sample[int16](-300), "-300"),
			Entry("a number as a boolean",
				xjson.Field{DataType: telem.Uint8T, Type: xjson.Boolean},
				sample[uint8](2), "true"),
			Entry("a number as a string",
				xjson.Field{DataType: telem.Float32T, Type: xjson.String},
				sample[float32](0.5), `"0.5"`),
			Entry("a value as its enum label",
				xjson.Field{DataType: telem.Uint8T, Type: xjson.String, Enums: onOff},
				sample[uint8](1), `"ON"`),
			Entry("a string", xjson.Field{DataType: telem.StringT, Type: xjson.String},
				[]byte("a\"b"), `"a\"b"`),
			Entry("a UUID", xjson.Field{DataType: telem.UUIDT, Type: xjson.String},
				id[:], `"`+id.String()+`"`),
			Entry("an ISO 8601 timestamp",
				xjson.Field{DataType: telem.TimestampT, TimeFormat: xjson.ISO8601},
				sample(telem.TimeStamp(1500000006)),
				`"1970-01-01T00:00:01.500000006Z"`),
			Entry("Unix seconds with every digit",
				xjson.Field{DataType: telem.TimestampT, TimeFormat: xjson.UnixSecond},
				sample(telem.TimeStamp(1767323045123456789)), "1767323045.123456789"),
			Entry("whole Unix milliseconds",
				xjson.Field{
					DataType:   telem.TimestampT,
					TimeFormat: xjson.UnixMillisecond,
				},
				sample(telem.TimeStamp(5000000)), "5"),
			Entry("negative Unix seconds",
				xjson.Field{DataType: telem.TimestampT, TimeFormat: xjson.UnixSecond},
				sample(telem.TimeStamp(-500000000)), "-0.5"),
			Entry("Unix nanoseconds",
				xjson.Field{
					DataType:   telem.TimestampT,
					TimeFormat: xjson.UnixNanosecond,
				},
				sample(telem.TimeStamp(math.MinInt64)), "-9223372036854775808"),
		)

		DescribeTable("Should reject a sample with no JSON value",
			func(f xjson.Field, s []byte, message string) {
				e := MustSucceed(xjson.NewEncoder(f))
				Expect(e.Append(nil, s)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
			},
			Entry("NaN", xjson.Field{DataType: telem.Float64T}, sample(math.NaN()),
				"NaN is not a JSON number"),
			Entry("infinity", xjson.Field{DataType: telem.Float32T},
				sample(float32(math.Inf(1))), "+Inf is not a JSON number"),
			Entry("a value with no enum label",
				xjson.Field{DataType: telem.Uint8T, Type: xjson.String, Enums: onOff},
				sample[uint8](5), "no enum label for value 5"),
		)

		DescribeTable("Should reject a JSON type that cannot hold the data type",
			func(dt telem.DataType, t xjson.Type) {
				Expect(xjson.NewEncoder(xjson.Field{DataType: dt, Type: t})).Error().
					To(SatisfyAll(
						MatchError(validate.ErrValidation),
						MatchError(ContainSubstring("only serializes as a string")),
					))
			},
			Entry("a string as a number", telem.StringT, xjson.Number),
			Entry("a UUID as a boolean", telem.UUIDT, xjson.Boolean),
			Entry("JSON as a number", telem.JSONT, xjson.Number),
		)
	})
})
