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
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/diagnostics"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Backtick Format String Analyzer Diagnostics", func() {
	fmtResolver := func() symbol.MapResolver {
		return symbol.MapResolver{
			"chI32": {Name: "chI32", Kind: symbol.KindChannel, Type: types.Chan(types.I32())},
			"chF64": {Name: "chF64", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
			"chStr": {Name: "chStr", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			"chU8":  {Name: "chU8", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			"trig":  {Name: "trig", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			"log":   {Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
		}
	}

	analyze := func(specCtx SpecContext, code string) diagnostics.Diagnostics {
		ast := MustSucceed(parser.Parse(code))
		ctx := acontext.CreateRoot(specCtx, ast, fmtResolver())
		analyzer.AnalyzeProgram(ctx)
		return *ctx.Diagnostics
	}

	findError := func(diags diagnostics.Diagnostics, substr string) *diagnostics.Diagnostic {
		for i := range diags {
			d := diags[i]
			if d.Severity == diagnostics.SeverityError && strings.Contains(d.Message, substr) {
				return &d
			}
		}
		return nil
	}

	countErrors := func(diags diagnostics.Diagnostics, substr string) int {
		n := 0
		for _, d := range diags {
			if d.Severity == diagnostics.SeverityError && strings.Contains(d.Message, substr) {
				n++
			}
		}
		return n
	}

	expectError := func(specCtx SpecContext, code, substr string) diagnostics.Diagnostic {
		diags := analyze(specCtx, code)
		Expect(diags.Ok()).To(BeFalse(),
			fmt.Sprintf("expected an error matching %q but analysis succeeded", substr))
		got := findError(diags, substr)
		Expect(got).ToNot(BeNil(),
			fmt.Sprintf("no error matched %q; got:\n%s", substr, diags.String()))
		return *got
	}

	expectSuccess := func(specCtx SpecContext, code string) {
		diags := analyze(specCtx, code)
		Expect(diags.Ok()).To(BeTrue(), diags.String())
	}

	wrap := func(body string) string {
		return `func f() {
` + body + `
}
trig -> f{}`
	}

	Describe("Body parse errors (fmtstring.Parse)", func() {
		DescribeTable("rejects malformed format string bodies",
			func(specCtx SpecContext, body, errSubstr string) {
				expectError(specCtx, wrap(`    log = `+body), errSubstr)
			},
			Entry("unmatched closing brace", "`}`", "unmatched '}'"),
			Entry("unmatched opening brace at end", "`{x`", "unmatched '{'"),
			Entry("unmatched opening brace mid-text", "`pre {x more`", "unmatched '{'"),
			Entry("nested unmatched open inside placeholder", "`{x{y}`", "unmatched"),
			Entry("empty placeholder body", "`{}`", "must contain an expression"),
			Entry("empty spec after percent", "`{chI32%}`", "format spec after '%' is empty"),
			Entry("empty expression before percent", "`{%d}`", "must contain an expression before '%'"),
		)
	})

	Describe("Placeholder expression parse errors (parser.ParseExpression)", func() {
		DescribeTable("rejects unparseable placeholder bodies",
			func(specCtx SpecContext, body, errSubstr string) {
				expectError(specCtx, wrap(`    log = `+body), errSubstr)
			},
			Entry("trailing operator", "`{chI32 +}`", "invalid placeholder expression"),
			Entry("leading operator", "`{* chI32}`", "invalid placeholder expression"),
			Entry("unclosed paren in expression", "`{(chI32}`", "invalid placeholder expression"),
		)
	})

	Describe("Placeholder type checks", func() {
		It("accepts a numeric (i32) placeholder", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+"`{chI32}`"))
		})

		It("accepts a string-typed placeholder", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+"`{chStr}`"))
		})

		It("accepts a numeric literal placeholder", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+"`{42}`"))
		})

		It("rejects a placeholder referencing an undeclared identifier", func(specCtx SpecContext) {
			diags := analyze(specCtx, wrap(`    log = `+"`{undeclared}`"))
			Expect(diags.Ok()).To(BeFalse(),
				"expected an error for undeclared identifier")
		})

		It("anchors the placeholder type diagnostic on the {...} span for an undeclared identifier", func(specCtx SpecContext) {
			// Source line 2: "    log = `pre {undeclared} post`"
			// '{' sits at col 15 and '}' at col 26 in the analyzer's column scheme,
			// so the placeholder span runs [15, 27).
			code := "func f() {\n    log = `pre {undeclared} post`\n}\ntrig -> f{}"
			diags := analyze(specCtx, code)
			Expect(diags.Ok()).To(BeFalse())
			d := findError(diags, `placeholder "undeclared"`)
			Expect(d).ToNot(BeNil(),
				"expected a placeholder type diagnostic naming \"undeclared\"; got:\n%s", diags.String())
			Expect(d.Start.Line).To(Equal(2))
			Expect(d.End.Line).To(Equal(2))
			Expect(d.Start.Col).To(Equal(15), "span should start at the '{' column")
			Expect(d.End.Col).To(Equal(27), "span should end one past the '}' column")
		})
	})

	Describe("Format spec validation", func() {
		DescribeTable("rejects float-only specs on integer placeholders",
			func(specCtx SpecContext, body string) {
				expectError(specCtx, wrap(`    log = `+body), "invalid format spec")
			},
			Entry("i32 channel %f", "`{chI32%f}`"),
			Entry("i32 channel %.2f", "`{chI32%.2f}`"),
			Entry("i32 channel %e", "`{chI32%e}`"),
			Entry("i32 channel %g", "`{chI32%g}`"),
		)

		DescribeTable("rejects integer-only specs on float placeholders",
			func(specCtx SpecContext, body string) {
				expectError(specCtx, wrap(`    log = `+body), "invalid format spec")
			},
			Entry("f64 channel %d", "`{chF64%d}`"),
			Entry("f64 channel %o", "`{chF64%o}`"),
		)

		DescribeTable("rejects invalid verbs on string placeholders",
			func(specCtx SpecContext, body string) {
				expectError(specCtx, wrap(`    log = `+body), "invalid format spec")
			},
			Entry("string channel %d", "`{chStr%d}`"),
			Entry("string channel %.2f", "`{chStr%.2f}`"),
		)

		DescribeTable("rejects blacklisted verbs across placeholder types",
			func(specCtx SpecContext, body string) {
				expectError(specCtx, wrap(`    log = `+body), "invalid format spec")
			},
			Entry("i32 channel %T", "`{chI32%T}`"),
			Entry("f64 channel %T", "`{chF64%T}`"),
			Entry("string channel %T", "`{chStr%T}`"),
			Entry("i32 channel %v", "`{chI32%v}`"),
			Entry("f64 channel %v", "`{chF64%v}`"),
			Entry("string channel %v", "`{chStr%v}`"),
			Entry("integer literal %T", "`{42%T}`"),
		)

		DescribeTable("accepts valid specs",
			func(specCtx SpecContext, body string) {
				expectSuccess(specCtx, wrap(`    log = `+body))
			},
			Entry("i32 channel %d", "`{chI32%d}`"),
			Entry("i32 channel %05d", "`{chI32%05d}`"),
			Entry("i32 channel %x", "`{chI32%x}`"),
			Entry("f64 channel %.2f", "`{chF64%.2f}`"),
			Entry("f64 channel %e", "`{chF64%e}`"),
			Entry("u8 channel %d", "`{chU8%d}`"),
			Entry("string channel %s", "`{chStr%s}`"),
			Entry("string channel %q", "`{chStr%q}`"),
			Entry("integer literal %d", "`{123%d}`"),
			Entry("float literal %.2f", "`{3.14%.2f}`"),
		)

		// Go's fmt accepts %x and %b on float64; pin so a future validator change cannot regress.
		DescribeTable("accepts Go-fmt-valid integer-shaped specs on floats",
			func(specCtx SpecContext, body string) {
				expectSuccess(specCtx, wrap(`    log = `+body))
			},
			Entry("f64 channel %x (hex float form)", "`{chF64%x}`"),
			Entry("f64 channel %b (binary scientific form)", "`{chF64%b}`"),
		)
	})

	Describe("Multiple '%' in placeholder (last '%' is the spec separator)", func() {
		It("treats the rightmost '%' in `{x%y%d}` as the spec separator (i32)", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    x i32 := 10
    y i32 := 3
    log = `+"`{x%y%d}`"))
		})

		It("treats the rightmost '%' in `{x%y%.2f}` as the spec separator (f64)", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    x f64 := 1.0
    y f64 := 0.5
    log = `+"`{x%y%.2f}`"))
		})

		It("treats the rightmost '%' in three-'%' bodies as the spec separator", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    a i32 := 7
    b i32 := 3
    c i32 := 2
    log = `+"`{a%b%c%d}`"))
		})

		It("rejects when the rightmost '%' produces an invalid spec, regardless of earlier '%'", func(specCtx SpecContext) {
			expectError(specCtx, wrap(`    x i32 := 10
    y i32 := 3
    log = `+"`{x%y%z}`"), "invalid format spec")
		})

		It("rejects when the rightmost '%' splits a spec invalid for the resulting expression type", func(specCtx SpecContext) {
			expectError(specCtx, wrap(`    x i32 := 10
    y i32 := 3
    log = `+"`{x%y%f}`"), "invalid format spec")
		})

		It("with `{x%y}` (single '%' between two i32 vars) splits y as the spec, not as a modulo operand", func(specCtx SpecContext) {
			expectError(specCtx, wrap(`    x i32 := 10
    y i32 := 3
    log = `+"`{x%y}`"), `invalid format spec "y"`)
		})
	})

	Describe("Whitespace flanking '%' (modulo expression vs spec)", func() {
		It("treats `{x % y}` (both spaces) as a modulo expression on i32", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    x i32 := 10
    y i32 := 3
    log = `+"`{x % y}`"))
		})

		It("accepts `{chF64 % .2f}` because the body parses as an f64 modulo expression", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+"`{chF64 % .2f}`"))
		})

		It("rejects `{chI32% .2f}` (right-space only) with a type mismatch", func(specCtx SpecContext) {
			expectError(specCtx, wrap(`    log = `+"`{chI32% .2f}`"), "type mismatch")
		})

		It("rejects `{chI32 %.2f}` (left-space only) with a type mismatch", func(specCtx SpecContext) {
			expectError(specCtx, wrap(`    log = `+"`{chI32 %.2f}`"), "type mismatch")
		})

		It("accepts `{chF64%.2f}` (no spaces) as a valid float spec", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+"`{chF64%.2f}`"))
		})
	})

	Describe("Empty body and trivial cases", func() {
		It("accepts an empty raw string (no placeholders)", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+"``"))
		})

		It("accepts a literal-only raw string", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+"`hello world`"))
		})

		It("accepts escaped braces with no placeholder", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+"`\\{ \\}`"))
		})
	})

	Describe("Diagnostic position anchoring", func() {
		It("anchors a placeholder spec error on the same line as the literal", func(specCtx SpecContext) {
			code := "func f() {\n    log = `{chStr%d}`\n}\ntrig -> f{}"
			d := expectError(specCtx, code, "invalid format spec")
			Expect(d.Start.Line).To(Equal(2),
				"expected diagnostic on line 2, got line %d (col %d)", d.Start.Line, d.Start.Col)
			Expect(d.End.Line).To(BeNumerically(">=", d.Start.Line))
			if d.End.Line == d.Start.Line {
				Expect(d.End.Col).To(BeNumerically(">", d.Start.Col),
					"expected nonzero placeholder span")
			}
		})

		It("anchors a placeholder spec error on a later line for a multi-line raw string", func(specCtx SpecContext) {
			code := "func f() {\n    log = `line1\nline2\n{chStr%d}`\n}\ntrig -> f{}"
			d := expectError(specCtx, code, "invalid format spec")
			Expect(d.Start.Line).To(Equal(4),
				"expected diagnostic on line 4 (third line of literal), got line %d col %d",
				d.Start.Line, d.Start.Col)
		})

		It("anchors a placeholder error past the opening backtick on a single-line literal", func(specCtx SpecContext) {
			code := "func f() {\n    log = `pre {chStr%d} post`\n}\ntrig -> f{}"
			d := expectError(specCtx, code, "invalid format spec")
			Expect(d.Start.Line).To(Equal(2))
			Expect(d.Start.Col).To(BeNumerically(">", 11),
				"placeholder column %d should be past the opening backtick", d.Start.Col)
		})
	})

	Describe("Defensive guards", func() {
		// findRawStrTerminal locates the first STR_LITERAL_RAW terminal in tree.
		var findRawStrTerminal func(t antlr.Tree) antlr.TerminalNode
		findRawStrTerminal = func(t antlr.Tree) antlr.TerminalNode {
			if tn, ok := t.(antlr.TerminalNode); ok {
				if tok := tn.GetSymbol(); tok != nil &&
					tok.GetTokenType() == parser.ArcParserSTR_LITERAL_RAW {
					return tn
				}
			}
			for i := 0; i < t.GetChildCount(); i++ {
				if found := findRawStrTerminal(t.GetChild(i)); found != nil {
					return found
				}
			}
			return nil
		}

		It("emits a diagnostic when raw token text lacks backticks", func(specCtx SpecContext) {
			ast := MustSucceed(parser.Parse("func f() {\n    log = `x`\n}\ntrig -> f{}"))
			rawStr := findRawStrTerminal(ast)
			Expect(rawStr).ToNot(BeNil())
			// Mutate token text to drop the backticks. This is unreachable via
			// the grammar but exercises the StripDelimiters guard.
			rawStr.GetSymbol().SetText("no_backticks")
			ctx := acontext.CreateRoot(specCtx, ast, fmtResolver())
			expression.AnalyzeStringFmtLiteral(ctx, rawStr)
			Expect(ctx.Diagnostics.Ok()).To(BeFalse())
			Expect(findError(*ctx.Diagnostics, "invalid raw string literal")).
				ToNot(BeNil())
		})
	})

	Describe("Multiple errors in one literal", func() {
		It("emits one diagnostic per offending placeholder", func(specCtx SpecContext) {
			code := wrap(`    log = ` + "`{chStr%d} {chF64%d}`")
			diags := analyze(specCtx, code)
			Expect(diags.Ok()).To(BeFalse())
			Expect(countErrors(diags, "invalid format spec")).To(Equal(2),
				"expected one diagnostic per placeholder; got:\n%s", diags.String())
		})

		It("continues analyzing later placeholders after an earlier spec error", func(specCtx SpecContext) {
			code := wrap(`    log = ` + "`{chStr%d} and {chF64%d}`")
			diags := analyze(specCtx, code)
			Expect(diags.Ok()).To(BeFalse())
			Expect(countErrors(diags, "invalid format spec")).To(Equal(2),
				"expected later placeholder error; got:\n%s", diags.String())
		})
	})
})
