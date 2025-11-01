// Copyright 2025 Synnax Labs, Inc.
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
	. "github.com/synnaxlabs/x/testutil"
)

// Helper to parse a numeric literal from text
func parseNumeric(text string, targetType types.Type) (literal.ParsedValue, error) {
	expr := MustSucceed(parser.ParseExpression(text))
	logicalOr := expr.LogicalOrExpression()
	logicalAnd := logicalOr.AllLogicalAndExpression()[0]
	equality := logicalAnd.AllEqualityExpression()[0]
	relational := equality.AllRelationalExpression()[0]
	additive := relational.AllAdditiveExpression()[0]
	multiplicative := additive.AllMultiplicativeExpression()[0]
	power := multiplicative.AllPowerExpression()[0]
	unary := power.UnaryExpression()

	// Check if this is a unary minus expression (e.g., -128)
	if unary.UnaryExpression() != nil && unary.MINUS() != nil {
		// Get the inner literal
		innerUnary := unary.UnaryExpression()
		postfix := innerUnary.PostfixExpression()
		primary := postfix.PrimaryExpression()
		lit := primary.Literal()
		numLit := lit.NumericLiteral()

		// Parse as positive first
		parsed, err := literal.ParseNumeric(numLit, targetType)
		if err != nil {
			return literal.ParsedValue{}, err
		}

		// Negate the value
		switch v := parsed.Value.(type) {
		case int8:
			parsed.Value = -v
		case int16:
			parsed.Value = -v
		case int32:
			parsed.Value = -v
		case int64:
			parsed.Value = -v
		case float32:
			parsed.Value = -v
		case float64:
			parsed.Value = -v
		}
		return parsed, nil
	}

	// Regular literal without unary operator
	postfix := unary.PostfixExpression()
	primary := postfix.PrimaryExpression()
	lit := primary.Literal()
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
		Entry("i8 out of range (positive)", "128", types.I8(), false, nil, types.Type{}, "out of range for i8"),
		Entry("i8 zero", "0", types.I8(), true, int8(0), types.I8(), ""),

		// i16 tests
		Entry("i16 max value", "32767", types.I16(), true, int16(32767), types.I16(), ""),
		Entry("i16 out of range (positive)", "32768", types.I16(), false, nil, types.Type{}, "out of range for i16"),

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

		// Default type inference
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
})
