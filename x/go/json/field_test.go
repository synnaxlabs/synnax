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

var onOff = []xjson.Enum{xjson.NewEnum("OFF", 0.0), xjson.NewEnum("ON", 1.0)}

// past2To53 is the first integer that a float64 does not hold.
const past2To53 = 1<<53 + 1

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
			Entry("json", "json"),
		)
		It("Should reject an unknown type", func() {
			Expect(xjson.ParseType("object")).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring(`unknown JSON type "object"`)),
			))
		})
	})

	Describe("NewEnum", func() {
		DescribeTable("Should read its label as its sample and write it back",
			func(en xjson.Enum, dt telem.DataType, expected []byte) {
				f := xjson.Field{
					DataType: dt,
					Type:     xjson.String,
					Enums:    []xjson.Enum{en},
				}
				d := MustSucceed(xjson.NewDecoder(f))
				Expect(d.Append(nil, jsontext.Value(`"L"`))).To(Equal(expected))
				e := MustSucceed(xjson.NewEncoder(f))
				written := expected
				if dt == telem.StringT {
					written, _ = telem.UnmarshalVariableSample(expected)
				}
				Expect(string(MustSucceed(e.Append(nil, written)))).To(Equal(`"L"`))
			},
			Entry("an int", xjson.NewEnum("L", int32(-7)), telem.Int32T,
				sample[int32](-7)),
			Entry("a uint", xjson.NewEnum("L", uint16(7)), telem.Uint16T,
				sample[uint16](7)),
			Entry("a float", xjson.NewEnum("L", 2.5), telem.Float64T, sample(2.5)),
			Entry("a string", xjson.NewEnum("L", "on"), telem.StringT, text("on")),
			Entry("a bool", xjson.NewEnum("L", true), telem.BooleanT, []byte{1}),
		)
	})

	Describe("Convert", func() {
		It("Should convert a value to a sample of another data type", func() {
			Expect(xjson.Convert(int64(-7), telem.Int8T)).To(Equal(sample[int8](-7)))
			Expect(xjson.Convert(math.NaN(), telem.Float32T)).To(
				HaveLen(int(telem.Float32T.Density())),
			)
			Expect(xjson.Convert(true, telem.Uint8T)).To(Equal([]byte{1}))
			Expect(xjson.Convert(telem.SecondTS, telem.TimestampT)).To(
				Equal(sample(telem.SecondTS)),
			)
			Expect(xjson.Convert("on", telem.StringT)).To(Equal([]byte("on")))
		})

		It("Should reject a value that does not fit the data type", func() {
			Expect(xjson.Convert(int64(200), telem.Int8T)).Error().To(
				MatchError(ContainSubstring("200 is out of range for int8")),
			)
			Expect(xjson.Convert(math.NaN(), telem.Int32T)).Error().To(
				MatchError(validate.ErrValidation),
			)
			Expect(xjson.Convert("on", telem.Uint8T)).Error().To(
				MatchError(
					ContainSubstring("a string value does not convert to uint8"),
				),
			)
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
			Entry("a whole decimal past the precision of a float",
				xjson.Field{DataType: telem.Int64T}, `9007199254740993.0`,
				sample[int64](9007199254740993)),
			Entry("an exponent past the precision of a float",
				xjson.Field{DataType: telem.Int64T}, `9007199254740993e0`,
				sample[int64](9007199254740993)),
			Entry("a whole number with a negative exponent",
				xjson.Field{DataType: telem.Int32T}, `4000e-3`, sample[int32](4)),
			Entry("the largest uint64 with an exponent",
				xjson.Field{DataType: telem.Uint64T}, `1.8446744073709551615e19`,
				sample[uint64](math.MaxUint64)),
			Entry("the smallest int64 with a decimal point",
				xjson.Field{DataType: telem.Int64T}, `-9223372036854775808.0`,
				sample[int64](math.MinInt64)),
			Entry("a negative zero into an unsigned integer",
				xjson.Field{DataType: telem.Uint8T}, `-0.0`, sample[uint8](0)),
			Entry("a negative zero into a float", xjson.Field{DataType: telem.Float64T},
				`-0.0`, sample(math.Copysign(0, -1))),
			Entry("true into a number", xjson.Field{DataType: telem.Uint8T}, `true`,
				sample[uint8](1)),
			Entry("a number into a boolean", xjson.Field{DataType: telem.BooleanT},
				`2`, []byte{1}),
			Entry("an enum label", xjson.Field{DataType: telem.Uint8T, Enums: onOff},
				`"ON"`, sample[uint8](1)),
			Entry("an escaped enum label",
				xjson.Field{DataType: telem.Uint8T, Enums: onOff}, `"\u004fN"`,
				sample[uint8](1)),
			Entry("an int64 enum past 2^53",
				xjson.Field{
					DataType: telem.Int64T,
					Enums:    []xjson.Enum{xjson.NewEnum("BIG", int64(past2To53))},
				},
				`"BIG"`, sample[int64](past2To53)),
			Entry("the largest uint64 enum",
				xjson.Field{
					DataType: telem.Uint64T,
					Enums: []xjson.Enum{
						xjson.NewEnum("MAX", uint64(math.MaxUint64)),
					},
				},
				`"MAX"`, sample[uint64](math.MaxUint64)),
			Entry("a NaN enum",
				xjson.Field{
					DataType: telem.Float32T,
					Enums:    []xjson.Enum{xjson.NewEnum("FAULT", math.NaN())},
				},
				`"FAULT"`, sample(float32(math.NaN()))),
			Entry("a boolean enum into a number",
				xjson.Field{
					DataType: telem.Int8T,
					Enums:    []xjson.Enum{xjson.NewEnum("ON", true)},
				},
				`"ON"`, sample[int8](1)),
			Entry("a string enum",
				xjson.Field{
					DataType: telem.StringT,
					Enums:    []xjson.Enum{xjson.NewEnum("ON", "1")},
				},
				`"ON"`, text("1")),
			Entry("a string that is not an enum label",
				xjson.Field{
					DataType: telem.StringT,
					Enums:    []xjson.Enum{xjson.NewEnum("ON", "1")},
				},
				`"OFF"`, text("OFF")),
			Entry("a string", xjson.Field{DataType: telem.StringT}, `"a\nb"`,
				text("a\nb")),
			Entry("JSON as the document spells it", xjson.Field{DataType: telem.JSONT},
				`{ "z" : 1.0, "a" : [ 1 , 2 ] }`, text(`{"z":1.0,"a":[1,2]}`)),
			Entry("a scalar as JSON", xjson.Field{DataType: telem.JSONT}, `1e999`,
				text(`1e999`)),
			Entry("a UUID", xjson.Field{DataType: telem.UUIDT}, `"`+id.String()+`"`,
				id[:]),
			Entry("bytes as base64", xjson.Field{DataType: telem.BytesT}, `"AP8/"`,
				telem.MarshalVariableSample([]byte{0x00, 0xff, 0x3f})),
			Entry("base64 with an escaped slash", xjson.Field{DataType: telem.BytesT},
				`"AP8\/"`, telem.MarshalVariableSample([]byte{0x00, 0xff, 0x3f})),
			Entry("empty bytes", xjson.Field{DataType: telem.BytesT}, `""`, text("")),
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
				out, err := d.Append([]byte{9}, jsontext.Value(raw))
				Expect(err).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
				Expect(out).To(BeNil())
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
			Entry("a fraction that a float rounds away",
				xjson.Field{DataType: telem.Int64T}, `9007199254740993.5`,
				"9.007199254740994e+15 is not an integer"),
			Entry("a whole decimal past an int64", xjson.Field{DataType: telem.Int64T},
				`1e19`, "10000000000000000000 is out of range for int64"),
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
			Entry("a number into bytes", xjson.Field{DataType: telem.BytesT}, `1`,
				"expected a base64 string, got a number"),
			Entry("text that is not base64", xjson.Field{DataType: telem.BytesT},
				`"a*b="`, "illegal base64 data at input byte 1"),
			Entry("base64 without padding", xjson.Field{DataType: telem.BytesT},
				`"YQ"`, "illegal base64 data at input byte 0"),
			Entry("base64 with padding bits set", xjson.Field{DataType: telem.BytesT},
				`"YR=="`, "illegal base64 data at input byte 2"),
			Entry("base64 with a line feed", xjson.Field{DataType: telem.BytesT},
				`"Y\nQ=="`, "base64 must not hold a line break"),
			Entry("base64 with a carriage return", xjson.Field{DataType: telem.BytesT},
				`"YQ==\r"`, "base64 must not hold a line break"),
		)

		It("Should reject a data type that no JSON value converts to", func() {
			Expect(xjson.NewDecoder(xjson.Field{DataType: telem.UnknownT})).Error().
				To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring("cannot convert a JSON value to")),
				))
		})

		DescribeTable(
			"Should reject an enum value that does not convert",
			func(dt telem.DataType, en xjson.Enum, message string) {
				f := xjson.Field{
					DataType: dt,
					Type:     xjson.String,
					Enums:    []xjson.Enum{en},
				}
				matcher := SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				)
				Expect(xjson.NewDecoder(f)).Error().To(matcher)
				Expect(xjson.NewEncoder(f)).Error().To(matcher)
			},
			Entry(
				"a fraction into an integer",
				telem.Uint8T,
				xjson.NewEnum("HALF", 0.5),
				`enum label "HALF": 0.5 is not an integer`,
			),
			Entry(
				"a value past the data type",
				telem.Int8T,
				xjson.NewEnum("BIG", int64(128)),
				"128 is out of range for int8",
			),
			Entry(
				"a NaN into an integer",
				telem.Int32T,
				xjson.NewEnum("FAULT", math.NaN()),
				"NaN is not an integer",
			),
			Entry(
				"a fraction into a boolean",
				telem.BooleanT,
				xjson.NewEnum("HALF", 0.5),
				"0.5 is not a boolean",
			),
			Entry("a negative number into a boolean", telem.BooleanT,
				xjson.NewEnum("ERR", int64(-1)), "-1 is not a boolean"),
			Entry(
				"a NaN into a boolean",
				telem.BooleanT,
				xjson.NewEnum("FAULT", math.NaN()),
				"NaN is not a boolean",
			),
			Entry("a number into a string", telem.StringT, xjson.NewEnum("ON", 1.0),
				"a float64 value does not convert to string"),
			Entry("a string into a number", telem.Float64T, xjson.NewEnum("ON", "1"),
				"a string value does not convert to float64"),
			Entry("a timestamp", telem.TimestampT, xjson.NewEnum("EPOCH", int64(0)),
				"a int64 value does not convert to timestamp"),
		)

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
			Entry("a value with no enum label as text",
				xjson.Field{DataType: telem.Uint8T, Type: xjson.String, Enums: onOff},
				sample[uint8](5), `"5"`),
			Entry("an enum label for a number",
				xjson.Field{DataType: telem.Float64T, Enums: onOff}, sample(1.0),
				`"ON"`),
			Entry("an int64 past 2^53 by its exact enum",
				xjson.Field{
					DataType: telem.Int64T,
					Type:     xjson.String,
					Enums: []xjson.Enum{
						xjson.NewEnum("LOW", int64(past2To53-1)),
						xjson.NewEnum("HIGH", int64(past2To53)),
					},
				},
				sample[int64](past2To53), `"HIGH"`),
			Entry("a uint64 by its exact enum",
				xjson.Field{
					DataType: telem.Uint64T,
					Type:     xjson.String,
					Enums: []xjson.Enum{
						xjson.NewEnum("LOW", uint64(math.MaxUint64-1)),
						xjson.NewEnum("HIGH", uint64(math.MaxUint64)),
					},
				},
				sample[uint64](math.MaxUint64), `"HIGH"`),
			Entry("negative zero as the enum of zero",
				xjson.Field{DataType: telem.Float64T, Type: xjson.String, Enums: onOff},
				sample(math.Copysign(0, -1)), `"OFF"`),
			Entry("any NaN as the NaN enum",
				xjson.Field{
					DataType: telem.Float32T,
					Enums:    []xjson.Enum{xjson.NewEnum("FAULT", math.NaN())},
				},
				sample(math.Float32frombits(0x7fc00001)), `"FAULT"`),
			Entry("infinity as its enum",
				xjson.Field{
					DataType: telem.Float64T,
					Enums:    []xjson.Enum{xjson.NewEnum("OVER", math.Inf(1))},
				},
				sample(math.Inf(1)), `"OVER"`),
			Entry("a float32 infinity as its enum",
				xjson.Field{
					DataType: telem.Float32T,
					Enums:    []xjson.Enum{xjson.NewEnum("UNDER", math.Inf(-1))},
				},
				sample(float32(math.Inf(-1))), `"UNDER"`),
			Entry("a float enum narrowed to a float32",
				xjson.Field{
					DataType: telem.Float32T,
					Type:     xjson.String,
					Enums:    []xjson.Enum{xjson.NewEnum("TENTH", 0.1)},
				},
				sample(float32(0.1)), `"TENTH"`),
			Entry("a string by its enum",
				xjson.Field{
					DataType: telem.StringT,
					Type:     xjson.String,
					Enums:    []xjson.Enum{xjson.NewEnum("ON", "1")},
				},
				[]byte("1"), `"ON"`),
			Entry("a boolean by its enum",
				xjson.Field{
					DataType: telem.BooleanT,
					Type:     xjson.Boolean,
					Enums:    []xjson.Enum{xjson.NewEnum("ON", true)},
				},
				[]byte{1}, `"ON"`),
			Entry("a nonzero number as the label of its truth value",
				xjson.Field{
					DataType: telem.Float64T,
					Type:     xjson.Boolean,
					Enums: []xjson.Enum{
						xjson.NewEnum("OFF", false),
						xjson.NewEnum("ON", true),
					},
				},
				sample(0.5), `"ON"`),
			Entry("a negative integer as the label of true",
				xjson.Field{
					DataType: telem.Int64T,
					Type:     xjson.Boolean,
					Enums:    []xjson.Enum{xjson.NewEnum("ON", true)},
				},
				sample(int64(-2)), `"ON"`),
			Entry("NaN as text",
				xjson.Field{DataType: telem.Float64T, Type: xjson.String},
				sample(math.NaN()), `"NaN"`),
			Entry("infinity as text",
				xjson.Field{DataType: telem.Float32T, Type: xjson.String},
				sample(float32(math.Inf(1))), `"Infinity"`),
			Entry("negative infinity as text",
				xjson.Field{DataType: telem.Float64T, Type: xjson.String},
				sample(math.Inf(-1)), `"-Infinity"`),
			Entry("infinity as a boolean",
				xjson.Field{DataType: telem.Float64T, Type: xjson.Boolean},
				sample(math.Inf(-1)), "true"),
			Entry("a string", xjson.Field{DataType: telem.StringT, Type: xjson.String},
				[]byte("a\"b"), `"a\"b"`),
			Entry("a JSON object as it is",
				xjson.Field{DataType: telem.JSONT, Type: xjson.JSON},
				[]byte(`{"a":[1,"b"]}`), `{"a":[1,"b"]}`),
			Entry("a JSON string as it is",
				xjson.Field{DataType: telem.JSONT, Type: xjson.JSON},
				[]byte(`"on"`), `"on"`),
			Entry("a JSON object as a string",
				xjson.Field{DataType: telem.JSONT, Type: xjson.String},
				[]byte(`{"a":1}`), `"{\"a\":1}"`),
			Entry("a UUID", xjson.Field{DataType: telem.UUIDT, Type: xjson.String},
				id[:], `"`+id.String()+`"`),
			Entry("bytes as base64",
				xjson.Field{DataType: telem.BytesT, Type: xjson.String},
				[]byte{0x00, 0xff, 0x3f}, `"AP8/"`),
			Entry("empty bytes",
				xjson.Field{DataType: telem.BytesT, Type: xjson.String},
				[]byte{}, `""`),
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
				sample(float32(math.Inf(1))), "Infinity is not a JSON number"),
			Entry("NaN with no enum",
				xjson.Field{DataType: telem.Float64T, Enums: onOff},
				sample(math.NaN()), "NaN is not a JSON number"),
			Entry("NaN as a boolean",
				xjson.Field{DataType: telem.Float32T, Type: xjson.Boolean},
				sample(float32(math.NaN())), "NaN is not a JSON boolean"),
			Entry("NaN as a boolean with a true label",
				xjson.Field{
					DataType: telem.Float64T,
					Type:     xjson.Boolean,
					Enums:    []xjson.Enum{xjson.NewEnum("ON", true)},
				},
				sample(math.NaN()), "NaN is not a JSON boolean"),
		)

		DescribeTable("Should report enum values that convert to the same sample",
			func(dt telem.DataType, a, b xjson.Enum, message string) {
				Expect(xjson.CheckEnums(xjson.Field{
					DataType: dt,
					Type:     xjson.String,
					Enums:    []xjson.Enum{a, b},
				})).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(message)),
				))
			},
			Entry("floats that narrow to one float32", telem.Float32T,
				xjson.NewEnum("A", 0.1), xjson.NewEnum("B", 0.1000000001),
				`enum labels "A" and "B" have the same float32 value`),
			Entry("zero and negative zero", telem.Float64T,
				xjson.NewEnum("A", 0.0), xjson.NewEnum("B", math.Copysign(0, -1)),
				`enum labels "A" and "B" have the same float64 value`),
			Entry("two NaNs", telem.Float64T,
				xjson.NewEnum("A", math.NaN()), xjson.NewEnum("B", math.NaN()),
				`enum labels "A" and "B" have the same float64 value`),
			Entry("a boolean and one", telem.BooleanT,
				xjson.NewEnum("A", true), xjson.NewEnum("B", uint8(1)),
				`enum labels "A" and "B" have the same boolean value`),
			Entry("two strings", telem.StringT,
				xjson.NewEnum("A", "on"), xjson.NewEnum("B", "on"),
				`enum labels "A" and "B" have the same string value`),
		)

		It("Should write the first label of two values that match", func() {
			e := MustSucceed(xjson.NewEncoder(xjson.Field{
				DataType: telem.Float32T,
				Type:     xjson.String,
				Enums: []xjson.Enum{
					xjson.NewEnum("A", 0.1),
					xjson.NewEnum("B", 0.1000000001),
				},
			}))
			Expect(
				string(MustSucceed(e.Append(nil, sample(float32(0.1))))),
			).To(Equal(`"A"`))
		})

		It("Should let two values share a label", func() {
			e := MustSucceed(xjson.NewEncoder(xjson.Field{
				DataType: telem.Uint8T,
				Enums: []xjson.Enum{
					xjson.NewEnum("A", uint8(0)),
					xjson.NewEnum("A", uint8(1)),
				},
			}))
			Expect(
				string(MustSucceed(e.Append(nil, sample[uint8](1)))),
			).To(Equal(`"A"`))
		})

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
			Entry("bytes as a number", telem.BytesT, xjson.Number),
			Entry("JSON as a number", telem.JSONT, xjson.Number),
		)

		It("Should reject the JSON type for a data type that is not JSON", func() {
			Expect(xjson.NewEncoder(xjson.Field{
				DataType: telem.StringT,
				Type:     xjson.JSON,
			})).Error().To(SatisfyAll(
				MatchError(validate.ErrValidation),
				MatchError(ContainSubstring("has no JSON value of its own")),
			))
		})
	})

	It("Should round trip bytes through base64", func() {
		f := xjson.Field{DataType: telem.BytesT, Type: xjson.String}
		data := make([]byte, 256)
		for i := range data {
			data[i] = byte(i)
		}
		raw := MustSucceed(MustSucceed(xjson.NewEncoder(f)).Append(nil, data))
		Expect(MustSucceed(xjson.NewDecoder(f)).Append(nil, jsontext.Value(raw))).
			To(Equal(telem.MarshalVariableSample(data)))
	})
})
