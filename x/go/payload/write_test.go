// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package payload_test

import (
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	xjson "github.com/synnaxlabs/x/json"
	"github.com/synnaxlabs/x/payload"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("WriteField", func() {
	base := payload.BaseWriteField{Key: "k", Pointer: "/a/b"}

	Describe("TemplateField", func() {
		DescribeTable("should place the JSON text of a fixed value",
			func(v payload.WriteFieldVariant, expected string) {
				field, _ := MustSucceed2(payload.TemplateField(payload.WriteField{
					Variant: v,
				}))
				Expect(field.Label).To(Equal(`field "k"`))
				Expect(field.Pointer).To(Equal(xjson.Pointer{"a", "b"}))
				Expect(string(field.Literal)).To(Equal(expected))
			},
			Entry("float",
				payload.FloatWriteField{BaseWriteField: base, Value: 1.5}, "1.5"),
			Entry("whole float",
				payload.FloatWriteField{BaseWriteField: base, Value: 1e21}, "1e+21"),
			Entry("integer",
				payload.IntegerWriteField{BaseWriteField: base, Value: math.MinInt64},
				"-9223372036854775808"),
			Entry("unsigned",
				payload.UnsignedWriteField{
					BaseWriteField: base,
					Value:          math.MaxUint64,
				},
				"18446744073709551615"),
			Entry("string",
				payload.StringWriteField{BaseWriteField: base, Value: `say "hi"`},
				`"say \"hi\""`),
			Entry("boolean",
				payload.BooleanWriteField{BaseWriteField: base, Value: false}, "false"),
			Entry(
				"json",
				payload.JsonWriteField{
					BaseWriteField: base,
					Value:          ` [1, {"a": null}] `,
				},
				`[1,{"a":null}]`,
			),
			Entry("bytes",
				payload.BytesWriteField{BaseWriteField: base, Value: "AQID"}, `"AQID"`),
		)

		It("should generate a new UUID for a uuid field", func() {
			field, gen := MustSucceed2(payload.TemplateField(payload.WriteField{
				Variant: payload.UuidWriteField{BaseWriteField: base},
			}))
			Expect(field.Literal).To(BeNil())
			a := MustSucceed(gen.Append(nil, 0))
			b := MustSucceed(gen.Append(nil, 0))
			Expect(a).NotTo(Equal(b))
			Expect(uuid.Parse(string(a[1 : len(a)-1]))).Error().To(Succeed())
		})

		It("should generate the time for a timestamp field", func() {
			field, gen := MustSucceed2(payload.TemplateField(payload.WriteField{
				Variant: payload.TimestampWriteField{
					BaseWriteField: base,
					TimeFormat:     payload.TimeFormatUnixMs,
				},
			}))
			Expect(field.Literal).To(BeNil())
			Expect(string(MustSucceed(gen.Append(nil, 3*telem.SecondTS)))).
				To(Equal("3000"))
		})

		DescribeTable("should reject an invalid field",
			func(f payload.WriteField, msg string) {
				Expect(payload.TemplateField(f)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry("null", payload.WriteField{}, "a field is null"),
			Entry("empty pointer",
				payload.WriteField{Variant: payload.UuidWriteField{
					BaseWriteField: payload.BaseWriteField{Key: "k"},
				}},
				`field "k": pointer must not be empty`),
			Entry("non-finite float",
				payload.WriteField{Variant: payload.FloatWriteField{
					BaseWriteField: base,
					Value:          payload.Float(math.Inf(1)),
				}},
				`field "k": value +Inf must be finite`),
		)

		It("should reject an unknown time format", func() {
			Expect(payload.TemplateField(payload.WriteField{
				Variant: payload.TimestampWriteField{
					BaseWriteField: base,
					TimeFormat:     "unix_hours",
				},
			})).Error().To(MatchError(ContainSubstring(`field "k"`)))
		})
	})

	Describe("ValidateWriteField", func() {
		DescribeTable("should accept a valid value",
			func(v payload.WriteFieldVariant) {
				Expect(payload.ValidateWriteField(payload.WriteField{Variant: v})).
					To(Succeed())
			},
			Entry("float", payload.FloatWriteField{Value: -0.5}),
			Entry("json null", payload.JsonWriteField{Value: "null"}),
			Entry("json with duplicate names",
				payload.JsonWriteField{Value: `{"a": 1, "a": 2}`}),
			Entry("json with a number that underflows",
				payload.JsonWriteField{Value: `[1e-400]`}),
			Entry("empty bytes", payload.BytesWriteField{}),
			Entry("uuid with no pointer", payload.UuidWriteField{}),
		)

		DescribeTable("should reject a value that cannot go into a payload",
			func(f payload.WriteField, msg string) {
				Expect(payload.ValidateWriteField(f)).To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry("null", payload.WriteField{}, "a field is null"),
			Entry("NaN",
				payload.WriteField{Variant: payload.FloatWriteField{
					Value: payload.Float(math.NaN()),
				}},
				"value NaN must be finite"),
			Entry("negative infinity",
				payload.WriteField{Variant: payload.FloatWriteField{
					Value: payload.Float(math.Inf(-1)),
				}},
				"value -Inf must be finite"),
			Entry("empty json",
				payload.WriteField{Variant: payload.JsonWriteField{}},
				"value is not JSON"),
			Entry("invalid json",
				payload.WriteField{Variant: payload.JsonWriteField{Value: "{"}},
				"value is not JSON"),
			Entry("json with a lone surrogate",
				payload.WriteField{Variant: payload.JsonWriteField{Value: `"\ud800"`}},
				"value is not JSON"),
			Entry(
				"json with a number past float64 range",
				payload.WriteField{
					Variant: payload.JsonWriteField{Value: `{"a": 1e400}`},
				},
				"value is not JSON",
			),
			Entry("two json values",
				payload.WriteField{Variant: payload.JsonWriteField{Value: "1 2"}},
				"value is not JSON"),
			Entry("invalid base64",
				payload.WriteField{Variant: payload.BytesWriteField{Value: "AQ*"}},
				"value is not base64"),
			Entry("base64 with nonzero trailing bits",
				payload.WriteField{Variant: payload.BytesWriteField{Value: "AR=="}},
				"value is not base64"),
			Entry("unpadded base64",
				payload.WriteField{Variant: payload.BytesWriteField{Value: "AQI"}},
				"value is not base64"),
			Entry("base64 with a line break",
				payload.WriteField{Variant: payload.BytesWriteField{Value: "AQ\r\nI="}},
				"value is not base64"),
		)
	})
})
