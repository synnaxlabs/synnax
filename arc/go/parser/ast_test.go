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
	"github.com/antlr4-go/antlr/v4"
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
			Entry("raw string", `r"hello"`),
			Entry("format string", `f"hi {x}"`),
			Entry("multi-line string", "`a\nb`"),
			Entry("multi-line format string", "f`hi {x}\nbye`"),
			Entry("raw format string", `rf"path: {p}"`),
			Entry("unit literal", "5ms"),
			Entry("negated integer", "-1"),
			Entry("negated float", "-3.14"),
			Entry("negated unit literal", "-5ms"),
		)

		DescribeTable("false cases",
			func(code string) { Expect(parser.IsLiteral(parseExpr(code))).To(BeFalse()) },
			Entry("addition", "1 + 2"),
			Entry("logical not", "not 1"),
			Entry("identifier", "x"),
			Entry("function call", "foo()"),
			Entry("index", "arr[0]"),
			Entry("parenthesized", "(42)"),
			Entry("comparison", "1 > 0"),
			Entry("logical", "1 and 0"),
		)
	})

	Describe("IsNegatedLiteral", func() {
		DescribeTable("true cases",
			func(code string) { Expect(parser.IsNegatedLiteral(parseExpr(code))).To(BeTrue()) },
			Entry("negated integer", "-1"),
			Entry("negated float", "-3.14"),
			Entry("negated unit literal", "-5ms"),
		)

		DescribeTable("false cases",
			func(code string) { Expect(parser.IsNegatedLiteral(parseExpr(code))).To(BeFalse()) },
			Entry("positive integer", "42"),
			Entry("positive float", "3.14"),
			Entry("positive unit literal", "5ms"),
			Entry("string", `"hello"`),
			Entry("identifier", "x"),
			Entry("addition", "1 + 2"),
			Entry("logical not", "not 1"),
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
			Entry("raw string", `r"hello"`, `r"hello"`),
			Entry("multi-line string preserves newline", "`a\nb`", "`a\nb`"),
			Entry("format string", `f"hi {x}"`, `f"hi {x}"`),
			Entry("unit literal", "5ms", "5ms"),
			Entry("negated integer extracts inner literal", "-1", "1"),
			Entry("negated float extracts inner literal", "-3.14", "3.14"),
			Entry("negated unit extracts inner literal", "-5ms", "5ms"),
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
			Entry("raw string", `r"hello"`),
			Entry("multi-line string", "`a\nb`"),
			Entry("format string", `f"hi {x}"`),
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
			Entry("format string literal", `f"hi {x}"`, func(p parser.IPrimaryExpressionContext) {
				Expect(p.Literal().GetText()).To(Equal(`f"hi {x}"`))
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

	Describe("Imports", func() {
		It("Should return nil for a nil program", func() {
			Expect(parser.Imports(nil)).To(BeNil())
		})

		It("Should return nil when no import statements are present", func() {
			prog := MustSucceed(parser.Parse(`func f() {}`))
			Expect(parser.Imports(prog)).To(BeNil())
		})

		It("Should collect a single bare import", func() {
			prog := MustSucceed(parser.Parse(`import time`))
			entries := parser.Imports(prog)
			Expect(entries).To(HaveLen(1))
			Expect(entries[0].Path).To(Equal("time"))
			Expect(entries[0].Alias).To(Equal("time"))
			Expect(entries[0].AST).ToNot(BeNil())
		})

		It("Should collect multiple modules in one block", func() {
			prog := MustSucceed(parser.Parse(`import ( time math status )`))
			entries := parser.Imports(prog)
			Expect(entries).To(HaveLen(3))
			Expect(entries[0].Path).To(Equal("time"))
			Expect(entries[1].Path).To(Equal("math"))
			Expect(entries[2].Path).To(Equal("status"))
		})

		It("Should record the alias when present", func() {
			prog := MustSucceed(parser.Parse(`import time as t`))
			entries := parser.Imports(prog)
			Expect(entries).To(HaveLen(1))
			Expect(entries[0].Path).To(Equal("time"))
			Expect(entries[0].Alias).To(Equal("t"))
		})

		It("Should join hierarchical path segments with dots", func() {
			prog := MustSucceed(parser.Parse(`import math.trig`))
			entries := parser.Imports(prog)
			Expect(entries).To(HaveLen(1))
			Expect(entries[0].Path).To(Equal("math.trig"))
		})

		It("Should default alias to the last path segment on hierarchical paths", func() {
			prog := MustSucceed(parser.Parse(`import math.trig`))
			entries := parser.Imports(prog)
			Expect(entries[0].Alias).To(Equal("trig"))
		})

		It("Should preserve a hierarchical alias when AS is present", func() {
			prog := MustSucceed(parser.Parse(`import math.trig as t`))
			entries := parser.Imports(prog)
			Expect(entries[0].Path).To(Equal("math.trig"))
			Expect(entries[0].Alias).To(Equal("t"))
		})

		It("Should collect items across multiple import statements", func() {
			prog := MustSucceed(parser.Parse(`
				import time
				import math
			`))
			entries := parser.Imports(prog)
			Expect(entries).To(HaveLen(2))
			Expect(entries[0].Path).To(Equal("time"))
			Expect(entries[1].Path).To(Equal("math"))
		})
	})

	Describe("CollectIdentifiers", func() {
		DescribeTable("collects primary-expression identifiers",
			func(code string, expected []string) {
				Expect(parser.CollectIdentifiers(parseExpr(code))).To(Equal(expected))
			},
			Entry("single identifier", "x", []string{"x"}),
			Entry("deduplicates repeats", "x + x", []string{"x"}),
			Entry("source order", "b + a + c", []string{"b", "a", "c"}),
			Entry("nested and indexed", "a * (b + c[i]) - d",
				[]string{"a", "b", "c", "i", "d"}),
			Entry("function call name and args", "foo(a, b)",
				[]string{"foo", "a", "b"}),
			Entry("literal only", "42", nil),
			Entry("format string braces are token text, not identifiers",
				`f"hi {x}"`, nil),
			Entry("qualified names are excluded", "math.avg + x", []string{"x"}),
		)
	})

	Describe("StringTerminal", func() {
		It("Should return the terminal for a string literal", func() {
			lit := parser.GetLiteral(parseExpr(`"hello"`))
			Expect(parser.StringTerminal(lit)).ToNot(BeNil())
		})

		It("Should return the terminal for a multi-line string literal", func() {
			lit := parser.GetLiteral(parseExpr("`a\nb`"))
			Expect(parser.StringTerminal(lit)).ToNot(BeNil())
		})

		It("Should return nil for a numeric literal", func() {
			lit := parser.GetLiteral(parseExpr("42"))
			Expect(parser.StringTerminal(lit)).To(BeNil())
		})
	})

	Describe("QualifiedNameParts and QualifiedName", func() {
		qualified := func(code string) parser.IQualifiedIdentifierContext {
			primary := parser.GetPrimaryExpression(parseExpr(code))
			Expect(primary).ToNot(BeNil())
			qid := primary.QualifiedIdentifier()
			Expect(qid).ToNot(BeNil())
			return qid
		}

		It("Should split a qualified identifier into head and tail", func() {
			head, tail := parser.QualifiedNameParts(qualified("math.avg"))
			Expect(head).To(Equal("math"))
			Expect(tail).To(Equal("avg"))
		})

		It("Should read the authority keyword as a head", func() {
			head, tail := parser.QualifiedNameParts(qualified("authority.absolute"))
			Expect(head).To(Equal("authority"))
			Expect(tail).To(Equal("absolute"))
		})

		It("Should join head and tail with a dot", func() {
			Expect(parser.QualifiedName(qualified("math.avg"))).To(Equal("math.avg"))
		})
	})

	Describe("FunctionNameParts and FunctionName", func() {
		var findFunction func(t antlr.Tree) parser.IFunctionContext
		findFunction = func(t antlr.Tree) parser.IFunctionContext {
			if fn, ok := t.(parser.IFunctionContext); ok {
				return fn
			}
			for i := 0; i < t.GetChildCount(); i++ {
				if found := findFunction(t.GetChild(i)); found != nil {
					return found
				}
			}
			return nil
		}
		// Flow statements are allowed at top level and in stage/sequence
		// bodies, but not in func bodies, which is the rule ParseStatement
		// parses. Parse a whole program instead.
		functionOf := func(code string) parser.IFunctionContext {
			prog := MustSucceed(parser.Parse(code))
			fn := findFunction(prog)
			Expect(fn).ToNot(BeNil())
			return fn
		}

		It("Should return the bare name with an empty tail", func() {
			head, tail := parser.FunctionNameParts(functionOf("x -> calc{}"))
			Expect(head).To(Equal("calc"))
			Expect(tail).To(Equal(""))
		})

		It("Should split a qualified function name", func() {
			head, tail := parser.FunctionNameParts(functionOf("x -> math.avg{}"))
			Expect(head).To(Equal("math"))
			Expect(tail).To(Equal("avg"))
		})

		It("Should join names only when a tail is present", func() {
			Expect(parser.FunctionName(functionOf("x -> calc{}"))).To(Equal("calc"))
			Expect(parser.FunctionName(functionOf("x -> math.avg{}"))).
				To(Equal("math.avg"))
		})
	})

	Describe("PrimaryNameParts and PrimaryName", func() {
		primaryOf := func(code string) parser.IPrimaryExpressionContext {
			primary := parser.GetPrimaryExpression(parseExpr(code))
			Expect(primary).ToNot(BeNil())
			return primary
		}

		It("Should return the bare name with an empty tail", func() {
			head, tail := parser.PrimaryNameParts(primaryOf("x"))
			Expect(head).To(Equal("x"))
			Expect(tail).To(Equal(""))
		})

		It("Should split a qualified identifier", func() {
			head, tail := parser.PrimaryNameParts(primaryOf("math.avg"))
			Expect(head).To(Equal("math"))
			Expect(tail).To(Equal("avg"))
		})

		It("Should return empty parts for a non-identifier primary", func() {
			head, tail := parser.PrimaryNameParts(primaryOf("42"))
			Expect(head).To(Equal(""))
			Expect(tail).To(Equal(""))
		})

		It("Should join or blank the name to match the parts", func() {
			Expect(parser.PrimaryName(primaryOf("x"))).To(Equal("x"))
			Expect(parser.PrimaryName(primaryOf("math.avg"))).To(Equal("math.avg"))
			Expect(parser.PrimaryName(primaryOf("42"))).To(Equal(""))
		})
	})
})
