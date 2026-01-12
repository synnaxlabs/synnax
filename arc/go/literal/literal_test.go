// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package literal_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/literal"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

// Helper to extract literal from expression text
func getLiteral(text string) parser.ILiteralContext {
	expr := MustSucceed(parser.ParseExpression(text))
	logicalOr := expr.LogicalOrExpression()
	logicalAnd := logicalOr.AllLogicalAndExpression()[0]
	equality := logicalAnd.AllEqualityExpression()[0]
	relational := equality.AllRelationalExpression()[0]
	additive := relational.AllAdditiveExpression()[0]
	multiplicative := additive.AllMultiplicativeExpression()[0]
	power := multiplicative.AllPowerExpression()[0]
	unary := power.UnaryExpression()
	postfix := unary.PostfixExpression()
	primary := postfix.PrimaryExpression()
	return primary.Literal()
}

// Helper to parse a numeric literal from text
func parseNumeric(text string, targetType types.Type) (literal.ParsedValue, error) {
	lit := getLiteral(text)
	numLit := lit.NumericLiteral()
	return literal.ParseNumeric(numLit, targetType)
}

var _ = Describe("Literal Parser", func() {
	// Note: Negative numbers like "-1" are unary expressions, not literals.
	// They should be tested through the full analyzer, not the literal parser.

	DescribeTable("Numeric literal parsing",
		func(
			input string,
			targetType types.Type,
			shouldSucceed bool,
			expectedValue any,
			expectedType types.Type,
			errorSubstring string,
		) {
			parsed, err := parseNumeric(input, targetType)
			if shouldSucceed {
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(Equal(expectedValue))
				Expect(parsed.Type).To(Equal(expectedType))
			} else {
				Expect(err).To(MatchError(ContainSubstring(errorSubstring)))
			}
		},
		// i8 tests
		Entry("i8 max value", "127", types.I8(), true, int8(127), types.I8(), ""),
		Entry("i8 out of range", "128", types.I8(), false, nil, types.Type{}, "out of range for i8"),
		Entry("i8 zero", "0", types.I8(), true, int8(0), types.I8(), ""),

		// i16 tests
		Entry("i16 max value", "32767", types.I16(), true, int16(32767), types.I16(), ""),
		Entry("i16 out of range", "32768", types.I16(), false, nil, types.Type{}, "out of range for i16"),

		// i32 tests
		Entry("i32 max value", "2147483647", types.I32(), true, int32(2147483647), types.I32(), ""),

		// i64 tests
		Entry("i64 typical value", "42", types.I64(), true, int64(42), types.I64(), ""),

		// u8 tests
		Entry("u8 max value", "255", types.U8(), true, uint8(255), types.U8(), ""),
		Entry("u8 zero", "0", types.U8(), true, uint8(0), types.U8(), ""),

		// u16 tests
		Entry("u16 max value", "65535", types.U16(), true, uint16(65535), types.U16(), ""),

		// u32 tests
		Entry("u32 typical value", "100", types.U32(), true, uint32(100), types.U32(), ""),

		// u64 tests
		Entry("u64 typical value", "100", types.U64(), true, uint64(100), types.U64(), ""),

		// Float tests
		Entry("f32 literal", "3.14", types.F32(), true, float32(3.14), types.F32(), ""),
		Entry("f64 literal", "3.14159", types.F64(), true, 3.14159, types.F64(), ""),

		// Value type inference
		Entry("integer defaults to i64", "42", types.Type{}, true, int64(42), types.I64(), ""),
		Entry("float defaults to f64", "3.14", types.Type{}, true, 3.14, types.F64(), ""),

		// Float to integer conversions (exact)
		Entry("exact float to i32 (3.0)", "3.0", types.I32(), true, int32(3), types.I32(), ""),
		Entry("exact float to i32 (0.0)", "0.0", types.I32(), true, int32(0), types.I32(), ""),

		// Float to integer conversions (non-exact - should fail)
		Entry("non-exact float to i32 (3.14)", "3.14", types.I32(), false, nil, types.Type{}, "cannot convert non-integer float"),
		Entry("non-exact float to u8 (3.5)", "3.5", types.U8(), false, nil, types.Type{}, "cannot convert non-integer float"),

		// Integer to float conversions
		Entry("int to f32", "42", types.F32(), true, float32(42), types.F32(), ""),
		Entry("int to f64", "42", types.F64(), true, float64(42), types.F64(), ""),
	)

	Describe("Unit literals", func() {
		Context("Type inference (no target type)", func() {
			It("Should infer int64 for exact integer result from integer literal", func() {
				lit := getLiteral("10s")
				parsed, err := literal.Parse(lit, types.Type{})
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(Equal(int64(10000000000))) // 10s = 10 billion ns
				Expect(parsed.Type.Kind).To(Equal(types.KindI64))
				Expect(parsed.Type.Unit).ToNot(BeNil())
				Expect(parsed.Type.Unit.Dimensions).To(Equal(types.DimTime))
			})

			It("Should infer int64 for 100kg (SI value 100 is exact int)", func() {
				lit := getLiteral("100kg")
				parsed, err := literal.Parse(lit, types.Type{})
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(Equal(int64(100)))
				Expect(parsed.Type.Kind).To(Equal(types.KindI64))
			})

			It("Should infer int64 for 100psi (SI value ~689476 is exact int)", func() {
				lit := getLiteral("100psi")
				parsed, err := literal.Parse(lit, types.Type{})
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(BeNumerically("~", int64(689476), 1))
				Expect(parsed.Type.Kind).To(Equal(types.KindI64))
			})

			It("Should infer int64 for 5km (SI value 5000 is exact int)", func() {
				lit := getLiteral("5km")
				parsed, err := literal.Parse(lit, types.Type{})
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(Equal(int64(5000)))
				Expect(parsed.Type.Kind).To(Equal(types.KindI64))
			})

			It("Should infer float64 for float literal even if SI value is exact int", func() {
				lit := getLiteral("5.0km")
				parsed, err := literal.Parse(lit, types.Type{})
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(Equal(5000.0))
				Expect(parsed.Type.Kind).To(Equal(types.KindF64))
			})

			It("Should infer int64 for 300ms (300 million ns is exact int)", func() {
				lit := getLiteral("300ms")
				parsed, err := literal.Parse(lit, types.Type{})
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(Equal(int64(300000000))) // 300ms = 300 million ns
				Expect(parsed.Type.Kind).To(Equal(types.KindI64))
			})

			It("Should infer float64 for float literal", func() {
				lit := getLiteral("100.5kg")
				parsed, err := literal.Parse(lit, types.Type{})
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(Equal(100.5))
				Expect(parsed.Type.Kind).To(Equal(types.KindF64))
			})
		})

		Context("With target type having a unit (scale conversion)", func() {
			It("Should convert 300ms to TimeSpan (nanoseconds)", func() {
				lit := getLiteral("300ms")
				parsed, err := literal.Parse(lit, types.TimeSpan())
				Expect(err).ToNot(HaveOccurred())
				// 300ms = 300 * (1e-3 / 1e-9) = 300,000,000 ns
				Expect(parsed.Value).To(Equal(telem.Millisecond * 300))
				Expect(parsed.Type.Kind).To(Equal(types.KindI64))
			})

			It("Should convert 5s to TimeSpan (nanoseconds)", func() {
				lit := getLiteral("5s")
				parsed, err := literal.Parse(lit, types.TimeSpan())
				Expect(err).ToNot(HaveOccurred())
				// 5s = 5 * (1 / 1e-9) = 5,000,000,000 ns
				Expect(parsed.Value).To(Equal(telem.Second * 5))
				Expect(parsed.Type.Kind).To(Equal(types.KindI64))
			})

			It("Should convert 1us to TimeSpan (nanoseconds)", func() {
				lit := getLiteral("1us")
				parsed, err := literal.Parse(lit, types.TimeSpan())
				Expect(err).ToNot(HaveOccurred())
				// 1us = 1 * (1e-6 / 1e-9) = 1000 ns
				Expect(parsed.Value).To(Equal(telem.Microsecond))
				Expect(parsed.Type.Kind).To(Equal(types.KindI64))
			})
		})

		Context("With target type kind (no unit)", func() {
			It("Should convert to i32 when SI value is exact integer", func() {
				lit := getLiteral("100psi")
				parsed, err := literal.Parse(lit, types.I32())
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(BeNumerically("~", int32(689476), 1))
				Expect(parsed.Type.Kind).To(Equal(types.KindI32))
			})

			It("Should convert to i64 when SI value is exact integer", func() {
				lit := getLiteral("5km")
				parsed, err := literal.Parse(lit, types.I64())
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(Equal(int64(5000)))
				Expect(parsed.Type.Kind).To(Equal(types.KindI64))
			})

			It("Should convert to f64", func() {
				lit := getLiteral("5km")
				parsed, err := literal.Parse(lit, types.F64())
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(Equal(5000.0))
				Expect(parsed.Type.Kind).To(Equal(types.KindF64))
			})

			It("Should convert to f32", func() {
				lit := getLiteral("5km")
				parsed, err := literal.Parse(lit, types.F32())
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(Equal(float32(5000)))
				Expect(parsed.Type.Kind).To(Equal(types.KindF32))
			})
		})

		Context("Go-like constant semantics (error on fractional)", func() {
			It("Should error when converting non-exact value to i32", func() {
				lit := getLiteral("1psi") // 1 psi = 6894.76 Pa (fractional)
				_, err := literal.Parse(lit, types.I32())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fractional part"))
			})

			It("Should error when converting non-exact value to i64", func() {
				lit := getLiteral("100.5kg")
				_, err := literal.Parse(lit, types.I64())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("fractional part"))
			})

			It("Should succeed when converting exact float literal to i32", func() {
				lit := getLiteral("1.0km")
				parsed, err := literal.Parse(lit, types.I32())
				Expect(err).ToNot(HaveOccurred())
				Expect(parsed.Value).To(Equal(int32(1000)))
			})
		})

		Context("Error cases", func() {
			It("Should return error for unknown units", func() {
				lit := getLiteral("5foobar")
				_, err := literal.Parse(lit, types.Type{})
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("unknown unit"))
			})

			It("Should return error for range overflow", func() {
				lit := getLiteral("300psi")
				_, err := literal.Parse(lit, types.I8())
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("out of range"))
			})
		})
	})

	Describe("String literals", func() {
		It("Should parse simple string", func() {
			lit := getLiteral(`"hello"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal("hello"))
			Expect(parsed.Type).To(Equal(types.String()))
		})

		It("Should parse empty string", func() {
			lit := getLiteral(`""`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal(""))
			Expect(parsed.Type).To(Equal(types.String()))
		})

		It("Should parse string with spaces", func() {
			lit := getLiteral(`"hello world"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal("hello world"))
		})

		It("Should handle newline escape", func() {
			lit := getLiteral(`"line1\nline2"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal("line1\nline2"))
		})

		It("Should handle tab escape", func() {
			lit := getLiteral(`"col1\tcol2"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal("col1\tcol2"))
		})

		It("Should handle carriage return escape", func() {
			lit := getLiteral(`"line1\rline2"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal("line1\rline2"))
		})

		It("Should handle escaped quote", func() {
			lit := getLiteral(`"say \"hello\""`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal(`say "hello"`))
		})

		It("Should handle escaped backslash", func() {
			lit := getLiteral(`"path\\to\\file"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal(`path\to\file`))
		})

		It("Should handle unicode escape", func() {
			lit := getLiteral(`"A\u0042C"`)
			parsed := MustSucceed(literal.Parse(lit, types.String()))
			Expect(parsed.Value).To(Equal("ABC"))
		})

		It("Should infer string type when no target type specified", func() {
			lit := getLiteral(`"hello"`)
			parsed := MustSucceed(literal.Parse(lit, types.Type{}))
			Expect(parsed.Value).To(Equal("hello"))
			Expect(parsed.Type).To(Equal(types.String()))
		})

		It("Should return error when assigning string to non-string type", func() {
			lit := getLiteral(`"hello"`)
			_, err := literal.Parse(lit, types.I32())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cannot assign string to"))
		})

		It("Should return error for malformed string literal (missing closing quote)", func() {
			Expect(
				literal.ParseString(`"hello`, types.String()),
			).Error().To(MatchError(ContainSubstring("invalid string literal")))
		})
	})

	Describe("Series literals", func() {
		It("Should return error for series literals (not supported for default values)", func() {
			lit := getLiteral("[1, 2, 3]")
			_, err := literal.Parse(lit, types.Series(types.I64()))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("series literals not supported for default values"))
		})
	})
})
