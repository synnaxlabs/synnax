// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression_test

import (
	"fmt"
	"strings"

	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	"github.com/synnaxlabs/arc/analyzer/expression"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var _ = Describe("Format String Analyzer Diagnostics", func() {
	fmtResolver := func() *symbol.Symbol {
		return NewRoot(
			nil,
			symbol.Symbol{
				Name: "chI32",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.I32()),
			},
			symbol.Symbol{
				Name: "chF64",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F64()),
			},
			symbol.Symbol{
				Name: "chStr",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.String()),
			},
			symbol.Symbol{
				Name: "chU8",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
			},
			symbol.Symbol{
				Name: "trig",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.U8()),
			},
			symbol.Symbol{
				Name: "log",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.String()),
			},
		)
	}

	analyze := func(specCtx SpecContext, code string) diagnostics.Diagnostics {
		GinkgoHelper()
		ast := MustSucceed(parser.Parse(code))
		ctx := acontext.NewRoot(specCtx, ast, fmtResolver())
		analyzer.AnalyzeProgram(ctx)
		return *ctx.Diagnostics
	}

	findError := func(diags diagnostics.Diagnostics, substr string) *diagnostics.Diagnostic {
		for i := range diags {
			d := diags[i]
			if d.Severity == protocol.DiagnosticSeverityError &&
				strings.Contains(d.Message, substr) {
				return &d
			}
		}
		return nil
	}

	countErrors := func(diags diagnostics.Diagnostics, substr string) int {
		n := 0
		for _, d := range diags {
			if d.Severity == protocol.DiagnosticSeverityError &&
				strings.Contains(d.Message, substr) {
				n++
			}
		}
		return n
	}

	expectError := func(specCtx SpecContext, code, substr string) diagnostics.Diagnostic {
		GinkgoHelper()
		diags := analyze(specCtx, code)
		Expect(diags.Ok()).To(BeFalse(),
			fmt.Sprintf("expected an error matching %q but analysis succeeded", substr))
		got := findError(diags, substr)
		Expect(got).ToNot(BeNil(),
			fmt.Sprintf("no error matched %q; got:\n%s", substr, diags.String()))
		return *got
	}

	expectSuccess := func(specCtx SpecContext, code string) {
		GinkgoHelper()
		diags := analyze(specCtx, code)
		Expect(diags.Ok()).To(BeTrue(), diags.String())
	}

	wrap := func(body string) string {
		return `func f() {
` + body + `
}
trig -> f()`
	}

	Describe("Body parse errors (literal.FmtStrParse)", func() {
		DescribeTable(
			"rejects malformed format string bodies",
			func(specCtx SpecContext, body, errSubstr string) {
				expectError(specCtx, wrap(`    log = `+body), errSubstr)
			},
			Entry("unmatched opening brace at end", `f"{x"`, "unmatched '{'"),
			Entry(
				"unmatched opening brace mid-text",
				`f"pre {x more"`,
				"unmatched '{'",
			),
			Entry("nested unmatched open inside placeholder", `f"{x{y}"`, "unmatched"),
			Entry("empty placeholder body", `f"{}"`, "must contain an expression"),
			Entry(
				"empty spec after colon",
				`f"{chI32:}"`,
				"format spec after ':' is empty",
			),
			Entry(
				"empty expression before colon",
				`f"{:d}"`,
				"must contain an expression before ':'",
			),
		)
	})

	Describe("Placeholder expression parse errors (parser.ParseExpression)", func() {
		DescribeTable(
			"rejects unparseable placeholder bodies",
			func(specCtx SpecContext, body, errSubstr string) {
				expectError(specCtx, wrap(`    log = `+body), errSubstr)
			},
			Entry(
				"trailing operator",
				`f"{chI32 +}"`,
				"invalid placeholder expression",
			),
			Entry("leading operator", `f"{* chI32}"`, "invalid placeholder expression"),
			Entry(
				"unclosed paren in expression",
				`f"{(chI32}"`,
				"invalid placeholder expression",
			),
		)
	})

	Describe("Placeholder type checks", func() {
		It("accepts a numeric (i32) placeholder", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+`f"{chI32}"`))
		})

		It("accepts a string-typed placeholder", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+`f"{chStr}"`))
		})

		It("accepts a numeric literal placeholder", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+`f"{42}"`))
		})

		It(
			"rejects a placeholder referencing an undeclared identifier",
			func(specCtx SpecContext) {
				diags := analyze(specCtx, wrap(`    log = `+`f"{undeclared}"`))
				Expect(diags.Ok()).To(BeFalse(),
					"expected an error for undeclared identifier")
			},
		)

		It(
			"anchors an unsupported type's diagnostic on the {...} span",
			func(specCtx SpecContext) {
				// Source line 3: `    log = f"pre {s} post"`
				// '{' sits at col 16 and '}' at col 18, so the span runs [16, 19).
				code := "func f() {\n    s series bool := [true, false]\n" +
					"    log = f\"pre {s} post\"\n}\ntrig -> f()"
				d := expectError(specCtx, code, `placeholder "s" has type`)
				Expect(d.Range.Start.Line).To(Equal(uint32(2)))
				Expect(d.Range.Start.Character).To(Equal(uint32(16)))
				Expect(d.Range.End.Character).To(Equal(uint32(19)))
			},
		)

		It(
			"reports an undeclared identifier in a placeholder where it is",
			func(specCtx SpecContext) {
				// Source line 2: `    log = f"pre {undeclared} post"`
				// The identifier runs [17, 27) in the analyzer's column scheme.
				code := "func f() {\n    log = f\"pre {undeclared} post\"\n}\ntrig -> f()"
				diags := analyze(specCtx, code)
				Expect(diags.Ok()).To(BeFalse())
				Expect(findError(diags, `placeholder "undeclared"`)).To(BeNil())
				d := findError(diags, "undefined symbol: undeclared")
				Expect(d).ToNot(BeNil(),
					"expected an undefined symbol diagnostic; got:\n%s", diags.String())
				Expect(d.Range.Start.Line).To(Equal(uint32(1)))
				Expect(d.Range.End.Line).To(Equal(uint32(1)))
				Expect(d.Range.Start.Character).To(Equal(uint32(17)))
				Expect(d.Range.End.Character).To(Equal(uint32(27)))
			},
		)
	})

	Describe("Placeholder calls", func() {
		It("Should accept a call that returns a value", func(specCtx SpecContext) {
			expectSuccess(specCtx, "func g() i64 { return 1 }\n"+
				"func f() {\n    log = f\"pre {g()} post\"\n}\ntrig -> f()")
		})

		It("Should reject a call that returns nothing on its placeholder", func(
			specCtx SpecContext,
		) {
			// Source line 2: `    log = f"pre {g()} post"`
			// '{' sits at col 16 and '}' at col 20, so the span runs [16, 21).
			code := "func g() {}\n" +
				"func f() {\n    log = f\"pre {g()} post\"\n}\ntrig -> f()"
			diags := analyze(specCtx, code)
			Expect(diags.Errors()).To(HaveLen(1), diags.String())
			d := findError(diags, `placeholder "g()" has type`)
			Expect(d).ToNot(BeNil(), diags.String())
			Expect(d.Range.Start.Line).To(Equal(uint32(2)))
			Expect(d.Range.Start.Character).To(Equal(uint32(16)))
			Expect(d.Range.End.Character).To(Equal(uint32(21)))
		})

		It("Should report an undeclared function once", func(specCtx SpecContext) {
			diags := analyze(specCtx, wrap(`    log = `+`f"{missing()}"`))
			Expect(diags.Errors()).To(HaveLen(1), diags.String())
			Expect(findError(diags, `placeholder "missing()"`)).To(BeNil())
		})

		It("Should report an undeclared name inside an expression once", func(
			specCtx SpecContext,
		) {
			diags := analyze(specCtx, wrap(`    log = `+`f"{undeclared + 1}"`))
			Expect(diags.Errors()).To(HaveLen(1), diags.String())
			Expect(findError(diags, "undefined symbol: undeclared")).ToNot(BeNil())
		})
	})

	DescribeTable("Should place a placeholder diagnostic after non-ASCII text",
		func(specCtx SpecContext, placeholder, message string) {
			diags := analyze(
				specCtx,
				"func f() {\n    log = f\"温度 "+placeholder+"\"\n}",
			)
			d := findError(diags, message)
			Expect(d).ToNot(BeNil(), diags.String())
			Expect(d.Range.Start.Line).To(Equal(uint32(1)))
			Expect(d.Range.Start.Character).To(Equal(uint32(15)))
			Expect(d.Range.End.Character).To(Equal(uint32(15 + len(placeholder))))
		},
		Entry("an empty placeholder", "{}", "must contain an expression"),
		Entry("a parse error", "{1 +}", "invalid placeholder expression"),
		Entry("a bad format spec", "{1:q}", "invalid format spec"),
	)

	It("Should count placeholder columns in runes", func(specCtx SpecContext) {
		code := "func f() {\n    log = f\"温度°: {1 + undeclared}\"\n}"
		diags := analyze(specCtx, code)
		d := findError(diags, "undeclared")
		Expect(d).ToNot(BeNil(), diags.String())
		Expect(d.Range.Start.Line).To(Equal(uint32(1)))
		Expect(d.Range.Start.Character).To(Equal(uint32(22)))
	})

	It("Should reuse the placeholders of a format string analyzed twice", func(
		specCtx SpecContext,
	) {
		expr := MustSucceed(parser.ParseExpression(`f"{1 + 2} {1 +} x"`))
		strTerm := parser.StringTerminal(parser.GetLiteral(expr))
		ctx := acontext.NewRoot(specCtx, expr, fmtResolver())
		expression.AnalyzeFmtStrLiteral(ctx, strTerm)
		first := ctx.FmtStrings[strTerm]
		Expect(first).To(HaveLen(4))
		Expect(first[0].Expr).ToNot(BeNil())
		Expect(first[2].Expr).To(BeNil())
		Expect(first[3]).To(Equal(parser.FmtSegment{Text: " x"}))
		expression.AnalyzeFmtStrLiteral(ctx, strTerm)
		second := ctx.FmtStrings[strTerm]
		Expect(second[0].Expr).To(BeIdenticalTo(first[0].Expr))
		Expect(second[2].Expr).To(BeNil())
	})

	It("Should report an invalid escape in literal text", func(specCtx SpecContext) {
		diags := analyze(specCtx, "func f() {\n    log = f\"\\u00zz{1}\"\n}")
		Expect(findError(diags, "invalid \\u escape")).ToNot(BeNil(), diags.String())
	})

	It("Should parse each placeholder at its position in the source", func(
		specCtx SpecContext,
	) {
		code := "func f() {\n    log = f`a {1 + 2}\n  {trig}`\n}\ntrig -> f()"
		ast := MustSucceed(parser.Parse(code))
		ctx := acontext.NewRoot(specCtx, ast, fmtResolver())
		analyzer.AnalyzeProgram(ctx)
		Expect(ctx.Diagnostics.Ok()).To(BeTrue(), ctx.Diagnostics.String())
		Expect(ctx.FmtStrings).To(HaveLen(1))
		var starts [][2]int
		for _, segments := range ctx.FmtStrings {
			for _, seg := range segments {
				if seg.Expr != nil {
					start := seg.Expr.GetStart()
					starts = append(starts, [2]int{start.GetLine(), start.GetColumn()})
				}
			}
		}
		Expect(starts).To(Equal([][2]int{{2, 15}, {3, 3}}))
	})

	It("Should anchor an error inside a placeholder on the expression", func(
		specCtx SpecContext,
	) {
		diags := analyze(specCtx, "func f() {\n    log = f\"pre {1 + undeclared}\"\n}")
		d := findError(diags, "undeclared")
		Expect(d).ToNot(BeNil(), diags.String())
		Expect(d.Range.Start.Line).To(Equal(uint32(1)))
		Expect(d.Range.Start.Character).To(Equal(uint32(21)))
	})

	Describe("Format spec validation", func() {
		DescribeTable("rejects float-only specs on integer placeholders",
			func(specCtx SpecContext, body string) {
				expectError(specCtx, wrap(`    log = `+body), "invalid format spec")
			},
			Entry("i32 channel :f", `f"{chI32:f}"`),
			Entry("i32 channel :.2f", `f"{chI32:.2f}"`),
			Entry("i32 channel :e", `f"{chI32:e}"`),
			Entry("i32 channel :g", `f"{chI32:g}"`),
		)

		DescribeTable("rejects integer-only specs on float placeholders",
			func(specCtx SpecContext, body string) {
				expectError(specCtx, wrap(`    log = `+body), "invalid format spec")
			},
			Entry("f64 channel :d", `f"{chF64:d}"`),
			Entry("f64 channel :o", `f"{chF64:o}"`),
		)

		DescribeTable("rejects invalid verbs on string placeholders",
			func(specCtx SpecContext, body string) {
				expectError(specCtx, wrap(`    log = `+body), "invalid format spec")
			},
			Entry("string channel :d", `f"{chStr:d}"`),
			Entry("string channel :.2f", `f"{chStr:.2f}"`),
		)

		DescribeTable("rejects blacklisted verbs across placeholder types",
			func(specCtx SpecContext, body string) {
				expectError(specCtx, wrap(`    log = `+body), "invalid format spec")
			},
			Entry("i32 channel :T", `f"{chI32:T}"`),
			Entry("f64 channel :T", `f"{chF64:T}"`),
			Entry("string channel :T", `f"{chStr:T}"`),
			Entry("i32 channel :v", `f"{chI32:v}"`),
			Entry("f64 channel :v", `f"{chF64:v}"`),
			Entry("string channel :v", `f"{chStr:v}"`),
			Entry("integer literal :T", `f"{42:T}"`),
			Entry("i32 channel :U", `f"{chI32:U}"`),
			Entry("u8 channel :U", `f"{chU8:U}"`),
			Entry("integer literal :U", `f"{42:U}"`),
			Entry("string channel :x", `f"{chStr:x}"`),
			Entry("string channel :X", `f"{chStr:X}"`),
		)

		DescribeTable("accepts valid specs",
			func(specCtx SpecContext, body string) {
				expectSuccess(specCtx, wrap(`    log = `+body))
			},
			Entry("i32 channel :d", `f"{chI32:d}"`),
			Entry("i32 channel :05d", `f"{chI32:05d}"`),
			Entry("i32 channel :x", `f"{chI32:x}"`),
			Entry("f64 channel :.2f", `f"{chF64:.2f}"`),
			Entry("f64 channel :e", `f"{chF64:e}"`),
			Entry("u8 channel :d", `f"{chU8:d}"`),
			Entry("string channel :s", `f"{chStr:s}"`),
			Entry("string channel :q", `f"{chStr:q}"`),
			Entry("integer literal :d", `f"{123:d}"`),
			Entry("float literal :.2f", `f"{3.14:.2f}"`),
		)
	})

	Describe("Multiple ':' in placeholder (last ':' is the spec separator)", func() {
		It(
			"treats the rightmost ':' in `{x:y:d}` as the spec separator (i32)",
			func(specCtx SpecContext) {
				expectSuccess(specCtx, wrap(`    x i32 := 10
    y i32 := 3
    log = `+`f"{x:y:d}"`))
			},
		)

		It(
			"treats the rightmost ':' in `{x:y:.2f}` as the spec separator (f64)",
			func(specCtx SpecContext) {
				expectSuccess(specCtx, wrap(`    x f64 := 1.0
    y f64 := 0.5
    log = `+`f"{x:y:.2f}"`))
			},
		)

		It(
			"treats the rightmost ':' in three-':' bodies as the spec separator",
			func(specCtx SpecContext) {
				expectSuccess(specCtx, wrap(`    a i32 := 7
    b i32 := 3
    c i32 := 2
    log = `+`f"{a:b:c:d}"`))
			},
		)

		It(
			"rejects when the rightmost ':' produces an invalid spec, regardless of earlier ':'",
			func(specCtx SpecContext) {
				expectError(specCtx, wrap(`    x i32 := 10
    y i32 := 3
    log = `+`f"{x:y:z}"`), "invalid format spec")
			},
		)

		It(
			"rejects when the rightmost ':' splits a spec invalid for the resulting expression type",
			func(specCtx SpecContext) {
				expectError(specCtx, wrap(`    x i32 := 10
    y i32 := 3
    log = `+`f"{x:y:f}"`), "invalid format spec")
			},
		)

		It(
			"with `{x:y}` (single ':' between two i32 vars) splits y as the spec",
			func(specCtx SpecContext) {
				expectError(specCtx, wrap(`    x i32 := 10
    y i32 := 3
    log = `+`f"{x:y}"`), `invalid format spec "y"`)
			},
		)
	})

	Describe("Empty body and trivial cases", func() {
		It(
			"accepts an empty format string (no placeholders)",
			func(specCtx SpecContext) {
				expectSuccess(specCtx, wrap(`    log = `+`f""`))
			},
		)

		It("accepts a literal-only format string", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+`f"hello world"`))
		})

		It(
			"accepts a doubled open brace and bare close with no placeholder",
			func(specCtx SpecContext) {
				expectSuccess(specCtx, wrap(`    log = `+`f"{{ }"`))
			},
		)

		It(
			"accepts an rf-prefixed format string with a placeholder",
			func(specCtx SpecContext) {
				expectSuccess(specCtx, wrap(`    log = `+`rf"v={chI32}"`))
			},
		)

		It(
			"rejects an invalid spec inside an rf-prefixed format string",
			func(specCtx SpecContext) {
				expectError(
					specCtx,
					wrap(`    log = `+`rf"v={chStr:d}"`),
					"invalid format spec",
				)
			},
		)

		It(
			"accepts an rf-prefixed multi-line format string with placeholders across newlines",
			func(specCtx SpecContext) {
				code := "func f() {\n    log = rf`v={chI32}\nt={chF64}`\n}\ntrig -> f()"
				expectSuccess(specCtx, code)
			},
		)
	})

	Describe("Diagnostic position anchoring", func() {
		It(
			"anchors a placeholder spec error on the same line as the literal",
			func(specCtx SpecContext) {
				code := "func f() {\n    log = f\"{chStr:d}\"\n}\ntrig -> f()"
				d := expectError(specCtx, code, "invalid format spec")
				Expect(d.Range.Start.Line).To(Equal(uint32(1)),
					"expected diagnostic on line 2, got line %d (col %d)", d.Range.Start.Line, d.Range.Start.Character)
				Expect(d.Range.End.Line).To(BeNumerically(">=", d.Range.Start.Line))
				if d.Range.End.Line == d.Range.Start.Line {
					Expect(
						d.Range.End.Character,
					).To(BeNumerically(">", d.Range.Start.Character),
						"expected nonzero placeholder span")
				}
			},
		)

		It(
			"anchors a placeholder spec error on a later line for a multi-line format string",
			func(specCtx SpecContext) {
				code := "func f() {\n    log = f`line1\nline2\n{chStr:d}`\n}\ntrig -> f()"
				d := expectError(specCtx, code, "invalid format spec")
				Expect(d.Range.Start.Line).To(Equal(uint32(3)),
					"expected diagnostic on line 4 (third line of literal), got line %d col %d",
					d.Range.Start.Line, d.Range.Start.Character)
			},
		)

		It(
			"anchors a placeholder error past the opening quote on a single-line literal",
			func(specCtx SpecContext) {
				code := "func f() {\n    log = f\"pre {chStr:d} post\"\n}\ntrig -> f()"
				d := expectError(specCtx, code, "invalid format spec")
				Expect(d.Range.Start.Line).To(Equal(uint32(1)))
				Expect(d.Range.Start.Character).To(BeNumerically(">", 11),
					"placeholder column %d should be past the opening quote", d.Range.Start.Character)
			},
		)
	})

	Describe("Defensive guards", func() {
		// findStringTerminal locates the first STR_LITERAL or STR_LITERAL_MULTI
		// terminal in tree.
		var findStringTerminal func(t antlr.Tree) antlr.TerminalNode
		findStringTerminal = func(t antlr.Tree) antlr.TerminalNode {
			if tn, ok := t.(antlr.TerminalNode); ok {
				if tok := tn.GetSymbol(); tok != nil {
					tt := tok.GetTokenType()
					if tt == parser.ArcParserSTR_LITERAL ||
						tt == parser.ArcParserSTR_LITERAL_MULTI {
						return tn
					}
				}
			}
			for i := 0; i < t.GetChildCount(); i++ {
				if found := findStringTerminal(t.GetChild(i)); found != nil {
					return found
				}
			}
			return nil
		}

		It(
			"emits a diagnostic when string token text lacks delimiters",
			func(specCtx SpecContext) {
				ast := MustSucceed(
					parser.Parse("func f() {\n    log = f\"x\"\n}\ntrig -> f()"),
				)
				strTerm := findStringTerminal(ast)
				Expect(strTerm).ToNot(BeNil())
				// Mutate token text to drop the delimiters. This is unreachable via
				// the grammar but exercises the StripQuotes guard.
				strTerm.GetSymbol().SetText("no_quotes")
				ctx := acontext.NewRoot(specCtx, ast, fmtResolver())
				expression.AnalyzeFmtStrLiteral(ctx, strTerm)
				Expect(ctx.Diagnostics.Ok()).To(BeFalse())
				Expect(findError(*ctx.Diagnostics, "invalid string literal")).
					ToNot(BeNil())
			},
		)
	})

	Describe("Multiple errors in one literal", func() {
		It("emits one diagnostic per offending placeholder", func(specCtx SpecContext) {
			code := wrap(`    log = ` + `f"{chStr:d} {chF64:d}"`)
			diags := analyze(specCtx, code)
			Expect(diags.Ok()).To(BeFalse())
			Expect(countErrors(diags, "invalid format spec")).To(Equal(2),
				"expected one diagnostic per placeholder; got:\n%s", diags.String())
		})

		It(
			"continues analyzing later placeholders after an earlier spec error",
			func(specCtx SpecContext) {
				code := wrap(`    log = ` + `f"{chStr:d} and {chF64:d}"`)
				diags := analyze(specCtx, code)
				Expect(diags.Ok()).To(BeFalse())
				Expect(countErrors(diags, "invalid format spec")).To(Equal(2),
					"expected later placeholder error; got:\n%s", diags.String())
			},
		)
	})
})
