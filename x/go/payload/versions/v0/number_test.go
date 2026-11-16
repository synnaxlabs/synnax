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
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io"
	"math"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/x/payload/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Number", func() {
	Describe("Float", func() {
		DescribeTable("Should encode a float as a number or its text",
			func(f v0.Float, expected string) {
				Expect(string(MustSucceed(json.Marshal(f)))).To(Equal(expected))
			},
			Entry("finite", v0.Float(1.5), "1.5"),
			Entry("NaN", v0.Float(math.NaN()), `"NaN"`),
			Entry("positive infinity", v0.Float(math.Inf(1)), `"Infinity"`),
			Entry("negative infinity", v0.Float(math.Inf(-1)), `"-Infinity"`),
		)
		DescribeTable("Should decode a float from a number or its text",
			func(data string, expected float64) {
				var f v0.Float
				Expect(json.Unmarshal([]byte(data), &f)).To(Succeed())
				Expect(float64(f)).To(Equal(expected))
			},
			Entry("number", "-2.25", -2.25),
			Entry("positive infinity", `"Infinity"`, math.Inf(1)),
			Entry("negative infinity", `"-Infinity"`, math.Inf(-1)),
		)
		It("Should decode NaN from its text", func() {
			var f v0.Float
			Expect(json.Unmarshal([]byte(`"NaN"`), &f)).To(Succeed())
			Expect(math.IsNaN(float64(f))).To(BeTrue())
		})
		DescribeTable("Should reject a value that is not a float",
			func(data string) {
				var f v0.Float
				Expect(json.Unmarshal([]byte(data), &f)).
					To(MatchError(validate.ErrValidation))
			},
			Entry("other text", `"inf"`),
			Entry("boolean", "true"),
			Entry("null", "null"),
			Entry("overflow", "1e400"),
		)
	})

	Describe("Integer", func() {
		It("Should encode every digit of an integer as a string", func() {
			Expect(string(MustSucceed(json.Marshal(v0.Integer(math.MinInt64))))).
				To(Equal(`"-9223372036854775808"`))
		})
		DescribeTable("Should decode an integer from a string or a number",
			func(data string, expected v0.Integer) {
				var i v0.Integer
				Expect(json.Unmarshal([]byte(data), &i)).To(Succeed())
				Expect(i).To(Equal(expected))
			},
			Entry("string", `"9223372036854775807"`, v0.Integer(math.MaxInt64)),
			Entry("number", "-12", v0.Integer(-12)),
		)
		DescribeTable("Should reject a value that is not a signed 64-bit integer",
			func(data string) {
				var i v0.Integer
				Expect(json.Unmarshal([]byte(data), &i)).
					To(MatchError(validate.ErrValidation))
			},
			Entry("out of range", `"9223372036854775808"`),
			Entry("plus sign", `"+1"`),
			Entry("boolean", "true"),
		)
		It("Should return a decoder error as is", func() {
			var i v0.Integer
			err := i.UnmarshalJSONFrom(jsontext.NewDecoder(strings.NewReader("")))
			Expect(err).To(MatchError(io.EOF))
			Expect(err).ToNot(MatchError(validate.ErrValidation))
		})
	})

	Describe("Unsigned", func() {
		It("Should encode every digit of an unsigned integer as a string", func() {
			Expect(string(MustSucceed(json.Marshal(v0.Unsigned(math.MaxUint64))))).
				To(Equal(`"18446744073709551615"`))
		})
		DescribeTable("Should decode an unsigned integer from a string or a number",
			func(data string, expected v0.Unsigned) {
				var u v0.Unsigned
				Expect(json.Unmarshal([]byte(data), &u)).To(Succeed())
				Expect(u).To(Equal(expected))
			},
			Entry("string", `"18446744073709551615"`, v0.Unsigned(math.MaxUint64)),
			Entry("number", "12", v0.Unsigned(12)),
		)
		DescribeTable("Should reject a value that is not an unsigned 64-bit integer",
			func(data string) {
				var u v0.Unsigned
				Expect(json.Unmarshal([]byte(data), &u)).
					To(MatchError(validate.ErrValidation))
			},
			Entry("negative", `"-1"`),
			Entry("negative zero", `"-0"`),
			Entry("plus sign", `"+1"`),
			Entry("out of range", `"18446744073709551616"`),
			Entry("boolean", "true"),
		)
		It("Should return a decoder error as is", func() {
			var u v0.Unsigned
			err := u.UnmarshalJSONFrom(jsontext.NewDecoder(strings.NewReader("")))
			Expect(err).To(MatchError(io.EOF))
			Expect(err).ToNot(MatchError(validate.ErrValidation))
		})
	})

	Describe("FloatRead", func() {
		It("Should round-trip a non-finite fill through JSON", func() {
			fill := v0.Float(math.Inf(-1))
			b := MustSucceed(json.Marshal(v0.FloatRead{Fill: &fill}))
			Expect(string(b)).To(ContainSubstring(`"fill":"-Infinity"`))
			var decoded v0.FloatRead
			Expect(json.Unmarshal(b, &decoded)).To(Succeed())
			Expect(*decoded.Fill).To(Equal(fill))
		})
	})
})
