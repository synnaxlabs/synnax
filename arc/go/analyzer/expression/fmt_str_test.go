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
		return NewRoot(nil,
			symbol.Symbol{Name: "chI32", Kind: symbol.KindChannel, Type: types.Chan(types.I32())},
			symbol.Symbol{Name: "chF64", Kind: symbol.KindChannel, Type: types.Chan(types.F64())},
			symbol.Symbol{Name: "chStr", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
			symbol.Symbol{Name: "chU8", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			symbol.Symbol{Name: "trig", Kind: symbol.KindChannel, Type: types.Chan(types.U8())},
			symbol.Symbol{Name: "log", Kind: symbol.KindChannel, Type: types.Chan(types.String())},
		)
	}

	analyze := func(specCtx SpecContext, code string) diagnostics.Diagnostics {
		ast := MustSucceed(parser.Parse(code))
		ctx := acontext.NewRoot(specCtx, ast, fmtResolver())
		analyzer.AnalyzeProgram(ctx)
		return *ctx.Diagnostics
	}

	findError := func(diags diagnostics.Diagnostics, substr string) *diagnostics.Diagnostic {
		for i := range diags {
			d := diags[i]
			if d.Severity == protocol.DiagnosticSeverityError && strings.Contains(d.Message, substr) {
				return &d
			}
		}
		return nil
	}

	countErrors := func(diags diagnostics.Diagnostics, substr string) int {
		n := 0
		for _, d := range diags {
			if d.Severity == protocol.DiagnosticSeverityError && strings.Contains(d.Message, substr) {
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

	Describe("Body parse errors (literal.FmtStrParse)", func() {
		DescribeTable("rejects malformed format string bodies",
			func(specCtx SpecContext, body, errSubstr string) {
				expectError(specCtx, wrap(`    log = `+body), errSubstr)
			},
			Entry("unmatched opening brace at end", `f"{x"`, "unmatched '{'"),
			Entry("unmatched opening brace mid-text", `f"pre {x more"`, "unmatched '{'"),
			Entry("nested unmatched open inside placeholder", `f"{x{y}"`, "unmatched"),
			Entry("empty placeholder body", `f"{}"`, "must contain an expression"),
			Entry("empty spec after colon", `f"{chI32:}"`, "format spec after ':' is empty"),
			Entry("empty expression before colon", `f"{:d}"`, "must contain an expression before ':'"),
		)
	})

	Describe("Placeholder expression parse errors (parser.ParseExpression)", func() {
		DescribeTable("rejects unparseable placeholder bodies",
			func(specCtx SpecContext, body, errSubstr string) {
				expectError(specCtx, wrap(`    log = `+body), errSubstr)
			},
			Entry("trailing operator", `f"{chI32 +}"`, "invalid placeholder expression"),
			Entry("leading operator", `f"{* chI32}"`, "invalid placeholder expression"),
			Entry("unclosed paren in expression", `f"{(chI32}"`, "invalid placeholder expression"),
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

		It("rejects a placeholder referencing an undeclared identifier", func(specCtx SpecContext) {
			diags := analyze(specCtx, wrap(`    log = `+`f"{undeclared}"`))
			Expect(diags.Ok()).To(BeFalse(),
				"expected an error for undeclared identifier")
		})

		It("anchors the placeholder type diagnostic on the {...} span for an undeclared identifier", func(specCtx SpecContext) {
			// Source line 2: `    log = f"pre {undeclared} post"`
			// '{' sits at col 16 and '}' at col 27 in the analyzer's column scheme,
			// so the placeholder span runs [16, 28).
			code := "func f() {\n    log = f\"pre {undeclared} post\"\n}\ntrig -> f{}"
			diags := analyze(specCtx, code)
			Expect(diags.Ok()).To(BeFalse())
			d := findError(diags, `placeholder "undeclared"`)
			Expect(d).ToNot(BeNil(),
				"expected a placeholder type diagnostic naming \"undeclared\"; got:\n%s", diags.String())
			Expect(d.Range.Start.Line).To(Equal(uint32(1)))
			Expect(d.Range.End.Line).To(Equal(uint32(1)))
			Expect(d.Range.Start.Character).To(Equal(uint32(16)), "span should start at the '{' column")
			Expect(d.Range.End.Character).To(Equal(uint32(28)), "span should end one past the '}' column")
		})
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
		It("treats the rightmost ':' in `{x:y:d}` as the spec separator (i32)", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    x i32 := 10
    y i32 := 3
    log = `+`f"{x:y:d}"`))
		})

		It("treats the rightmost ':' in `{x:y:.2f}` as the spec separator (f64)", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    x f64 := 1.0
    y f64 := 0.5
    log = `+`f"{x:y:.2f}"`))
		})

		It("treats the rightmost ':' in three-':' bodies as the spec separator", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    a i32 := 7
    b i32 := 3
    c i32 := 2
    log = `+`f"{a:b:c:d}"`))
		})

		It("rejects when the rightmost ':' produces an invalid spec, regardless of earlier ':'", func(specCtx SpecContext) {
			expectError(specCtx, wrap(`    x i32 := 10
    y i32 := 3
    log = `+`f"{x:y:z}"`), "invalid format spec")
		})

		It("rejects when the rightmost ':' splits a spec invalid for the resulting expression type", func(specCtx SpecContext) {
			expectError(specCtx, wrap(`    x i32 := 10
    y i32 := 3
    log = `+`f"{x:y:f}"`), "invalid format spec")
		})

		It("with `{x:y}` (single ':' between two i32 vars) splits y as the spec", func(specCtx SpecContext) {
			expectError(specCtx, wrap(`    x i32 := 10
    y i32 := 3
    log = `+`f"{x:y}"`), `invalid format spec "y"`)
		})
	})

	Describe("Empty body and trivial cases", func() {
		It("accepts an empty format string (no placeholders)", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+`f""`))
		})

		It("accepts a literal-only format string", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+`f"hello world"`))
		})

		It("accepts a doubled open brace and bare close with no placeholder", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+`f"{{ }"`))
		})

		It("accepts an rf-prefixed format string with a placeholder", func(specCtx SpecContext) {
			expectSuccess(specCtx, wrap(`    log = `+`rf"v={chI32}"`))
		})

		It("rejects an invalid spec inside an rf-prefixed format string", func(specCtx SpecContext) {
			expectError(specCtx, wrap(`    log = `+`rf"v={chStr:d}"`), "invalid format spec")
		})

		It("accepts an rf-prefixed multi-line format string with placeholders across newlines", func(specCtx SpecContext) {
			code := "func f() {\n    log = rf`v={chI32}\nt={chF64}`\n}\ntrig -> f{}"
			expectSuccess(specCtx, code)
		})
	})

	Describe("Diagnostic position anchoring", func() {
		It("anchors a placeholder spec error on the same line as the literal", func(specCtx SpecContext) {
			code := "func f() {\n    log = f\"{chStr:d}\"\n}\ntrig -> f{}"
			d := expectError(specCtx, code, "invalid format spec")
			Expect(d.Range.Start.Line).To(Equal(uint32(1)),
				"expected diagnostic on line 2, got line %d (col %d)", d.Range.Start.Line, d.Range.Start.Character)
			Expect(d.Range.End.Line).To(BeNumerically(">=", d.Range.Start.Line))
			if d.Range.End.Line == d.Range.Start.Line {
				Expect(d.Range.End.Character).To(BeNumerically(">", d.Range.Start.Character),
					"expected nonzero placeholder span")
			}
		})

		It("anchors a placeholder spec error on a later line for a multi-line format string", func(specCtx SpecContext) {
			code := "func f() {\n    log = f`line1\nline2\n{chStr:d}`\n}\ntrig -> f{}"
			d := expectError(specCtx, code, "invalid format spec")
			Expect(d.Range.Start.Line).To(Equal(uint32(3)),
				"expected diagnostic on line 4 (third line of literal), got line %d col %d",
				d.Range.Start.Line, d.Range.Start.Character)
		})

		It("anchors a placeholder error past the opening quote on a single-line literal", func(specCtx SpecContext) {
			code := "func f() {\n    log = f\"pre {chStr:d} post\"\n}\ntrig -> f{}"
			d := expectError(specCtx, code, "invalid format spec")
			Expect(d.Range.Start.Line).To(Equal(uint32(1)))
			Expect(d.Range.Start.Character).To(BeNumerically(">", 11),
				"placeholder column %d should be past the opening quote", d.Range.Start.Character)
		})
	})

	Describe("Defensive guards", func() {
		// findStringTerminal locates the first STR_LITERAL or STR_LITERAL_MULTI
		// terminal in tree.
		var findStringTerminal func(t antlr.Tree) antlr.TerminalNode
		findStringTerminal = func(t antlr.Tree) antlr.TerminalNode {
			if tn, ok := t.(antlr.TerminalNode); ok {
				if tok := tn.GetSymbol(); tok != nil {
					tt := tok.GetTokenType()
					if tt == parser.ArcParserSTR_LITERAL || tt == parser.ArcParserSTR_LITERAL_MULTI {
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

		It("emits a diagnostic when string token text lacks delimiters", func(specCtx SpecContext) {
			ast := MustSucceed(parser.Parse("func f() {\n    log = f\"x\"\n}\ntrig -> f{}"))
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
		})
	})

	Describe("Multiple errors in one literal", func() {
		It("emits one diagnostic per offending placeholder", func(specCtx SpecContext) {
			code := wrap(`    log = ` + `f"{chStr:d} {chF64:d}"`)
			diags := analyze(specCtx, code)
			Expect(diags.Ok()).To(BeFalse())
			Expect(countErrors(diags, "invalid format spec")).To(Equal(2),
				"expected one diagnostic per placeholder; got:\n%s", diags.String())
		})

		It("continues analyzing later placeholders after an earlier spec error", func(specCtx SpecContext) {
			code := wrap(`    log = ` + `f"{chStr:d} and {chF64:d}"`)
			diags := analyze(specCtx, code)
			Expect(diags.Ok()).To(BeFalse())
			Expect(countErrors(diags, "invalid format spec")).To(Equal(2),
				"expected later placeholder error; got:\n%s", diags.String())
		})
	})
})
