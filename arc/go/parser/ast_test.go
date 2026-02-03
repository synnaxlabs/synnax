// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package parser_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/parser"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("AST Utilities", func() {
	parseExpr := func(code string) parser.IExpressionContext {
		return MustSucceed(parser.ParseExpression(code))
	}

	Describe("IsLiteral", func() {
		DescribeTable("true cases",
			func(code string) { Expect(parser.IsLiteral(parseExpr(code))).To(BeTrue()) },
			Entry("integer", "42"),
			Entry("float", "3.14"),
			Entry("string", `"hello"`),
			Entry("unit literal", "5ms"),
		)

		DescribeTable("false cases",
			func(code string) { Expect(parser.IsLiteral(parseExpr(code))).To(BeFalse()) },
			Entry("addition", "1 + 2"),
			Entry("unary minus", "-1"),
			Entry("identifier", "x"),
			Entry("function call", "foo()"),
			Entry("index", "arr[0]"),
			Entry("parenthesized", "(42)"),
			Entry("comparison", "1 > 0"),
			Entry("logical", "1 and 0"),
		)
	})

	Describe("GetLiteral", func() {
		DescribeTable("extracts literal text",
			func(code string, expected string) {
				lit := parser.GetLiteral(parseExpr(code))
				Expect(lit).NotTo(BeNil())
				Expect(lit.GetText()).To(Equal(expected))
			},
			Entry("integer", "42", "42"),
			Entry("float", "3.14", "3.14"),
			Entry("string", `"hello"`, `"hello"`),
			Entry("unit literal", "5ms", "5ms"),
		)

		It("returns nil for non-literal", func() {
			Expect(parser.GetLiteral(parseExpr("1 + 2"))).To(BeNil())
		})
	})

	Describe("IsNumericLiteral", func() {
		DescribeTable("true cases",
			func(code string) { Expect(parser.IsNumericLiteral(parseExpr(code))).To(BeTrue()) },
			Entry("integer", "42"),
			Entry("float", "3.14"),
			Entry("negated integer", "-1"),
			Entry("negated float", "-3.14"),
			Entry("double negation", "--5"),
			Entry("unit literal", "5ms"),
		)

		DescribeTable("false cases",
			func(code string) { Expect(parser.IsNumericLiteral(parseExpr(code))).To(BeFalse()) },
			Entry("string", `"hello"`),
			Entry("identifier", "x"),
			Entry("addition", "1 + 2"),
			Entry("negated identifier", "-x"),
		)
	})

	Describe("GetPrimaryExpression", func() {
		DescribeTable("extracts primary",
			func(code string, check func(parser.IPrimaryExpressionContext)) {
				primary := parser.GetPrimaryExpression(parseExpr(code))
				Expect(primary).NotTo(BeNil())
				check(primary)
			},
			Entry("identifier", "foo", func(p parser.IPrimaryExpressionContext) {
				Expect(p.IDENTIFIER().GetText()).To(Equal("foo"))
			}),
			Entry("integer literal", "42", func(p parser.IPrimaryExpressionContext) {
				Expect(p.Literal().NumericLiteral().INTEGER_LITERAL().GetText()).To(Equal("42"))
			}),
			Entry("string literal", `"hi"`, func(p parser.IPrimaryExpressionContext) {
				Expect(p.Literal().GetText()).To(Equal(`"hi"`))
			}),
		)

		DescribeTable("returns nil for expressions with operators",
			func(code string) { Expect(parser.GetPrimaryExpression(parseExpr(code))).To(BeNil()) },
			Entry("addition", "1 + 2"),
			Entry("multiplication", "3 * 4"),
			Entry("comparison", "a > b"),
			Entry("logical or", "a or b"),
			Entry("unary minus", "-1"),
			Entry("power", "2 ^ 3"),
		)
	})

	Describe("GetExpressionText", func() {
		DescribeTable("extracts source text",
			func(code string, expected string) {
				Expect(parser.GetExpressionText(parseExpr(code))).To(Equal(expected))
			},
			Entry("identifier", "foo", "foo"),
			Entry("binary with spaces", "1 + 2", "1 + 2"),
			Entry("complex expression", "a * b + c", "a * b + c"),
			Entry("function call", "foo(1, 2)", "foo(1, 2)"),
		)

		It("returns empty string for nil", func() {
			Expect(parser.GetExpressionText(nil)).To(Equal(""))
		})
	})

	Describe("GetLiteralNode", func() {
		It("extracts literal from nested postfix context", func() {
			expr := parseExpr("42")
			postfix := expr.LogicalOrExpression().LogicalAndExpression(0).
				EqualityExpression(0).RelationalExpression(0).
				AdditiveExpression(0).MultiplicativeExpression(0).
				PowerExpression(0).UnaryExpression().PostfixExpression()
			lit := parser.GetLiteralNode(postfix)
			Expect(lit).NotTo(BeNil())
			Expect(lit.GetText()).To(Equal("42"))
		})
	})
})
