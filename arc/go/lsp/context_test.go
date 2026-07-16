// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package lsp_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	"github.com/synnaxlabs/x/lsp/protocol"
)

var _ = Describe("Context Detection", func() {
	DescribeTable("DetectCompletionContext",
		func(content string, line, char uint32, expected lsp.CompletionContext) {
			pos := protocol.Position{Line: line, Character: char}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(expected))
		},
		// Comment Context
		Entry("single-line comment at start",
			"// this is a comment", uint32(0), uint32(5), lsp.ContextComment),
		Entry("single-line comment at end",
			"// this is a comment", uint32(0), uint32(20), lsp.ContextComment),
		Entry("single-line comment after code",
			"x := 1 // comment here", uint32(0), uint32(15), lsp.ContextComment),
		Entry("multi-line comment on first line",
			"/* multi\nline\ncomment */", uint32(0), uint32(5), lsp.ContextComment),
		Entry("multi-line comment on middle line",
			"/* multi\nline\ncomment */", uint32(1), uint32(2), lsp.ContextComment),
		Entry("multi-line comment on last line",
			"/* multi\nline\ncomment */", uint32(2), uint32(5), lsp.ContextComment),
		Entry("position before comment start is not comment",
			"x := 1 // comment", uint32(0), uint32(3), lsp.ContextUnknown),

		// Type Annotation Context
		Entry("type annotation after LPAREN and identifier",
			"func foo(x ", uint32(0), uint32(11), lsp.ContextTypeAnnotation),
		Entry("type annotation after COMMA and identifier",
			"func foo(x i32, y ", uint32(0), uint32(18), lsp.ContextTypeAnnotation),
		Entry("partial type annotation",
			"func foo(x i", uint32(0), uint32(12), lsp.ContextTypeAnnotation),
		Entry("type annotation in func with empty input block",
			"func foo{} (x ", uint32(0), uint32(14), lsp.ContextTypeAnnotation),
		Entry("type annotation in func with input block params",
			"func controller{sensor chan f64} (enable ", uint32(0), uint32(41), lsp.ContextTypeAnnotation),

		// Expression Context
		Entry("expression after DECLARE",
			"x := ", uint32(0), uint32(5), lsp.ContextExpression),
		Entry("expression after STATE_DECLARE",
			"x $= ", uint32(0), uint32(5), lsp.ContextExpression),
		Entry("expression after ASSIGN",
			"x = ", uint32(0), uint32(4), lsp.ContextExpression),
		Entry("expression after PLUS",
			"x := y + ", uint32(0), uint32(9), lsp.ContextExpression),
		Entry("expression after MINUS",
			"x := y - ", uint32(0), uint32(9), lsp.ContextExpression),
		Entry("expression after comparison operator",
			"if x > ", uint32(0), uint32(7), lsp.ContextExpression),
		Entry("expression after RETURN",
			"return ", uint32(0), uint32(7), lsp.ContextExpression),
		Entry("expression after LPAREN in function call",
			"foo(", uint32(0), uint32(4), lsp.ContextExpression),
		Entry("expression after COMMA in function call",
			"foo(a, ", uint32(0), uint32(7), lsp.ContextExpression),
		Entry("expression inside parenthesized expression after return",
			"return (o", uint32(0), uint32(9), lsp.ContextExpression),
		Entry("expression inside nested parentheses in assignment",
			"x := (o", uint32(0), uint32(7), lsp.ContextExpression),
		Entry("expression inside parenthesized expression after operator",
			"x := y + (o", uint32(0), uint32(11), lsp.ContextExpression),

		// Statement Start Context
		Entry("no completions on the same line as opening LBRACE",
			"func foo() { ", uint32(0), uint32(13), lsp.ContextNone),
		Entry("statement start on new line in function body",
			"func foo() {\n    ", uint32(1), uint32(4), lsp.ContextStatementStart),
		Entry("statement start after closing brace on new line",
			"if true {\n    x := 1\n}\n", uint32(3), uint32(0), lsp.ContextStatementStart),
		Entry("statement start at empty file",
			"", uint32(0), uint32(0), lsp.ContextStatementStart),

		// Statement-level comma (regression: a stray comma at the end of a
		// statement inside a block body must not be treated as expression
		// context, otherwise the editor offers channel completions in places
		// where the next token cannot be an expression). The same statement-
		// start context applies whether the cursor is still on the line with
		// the comma or has moved to the next line — comma is a statement
		// terminator, not an opener, so it does not need the same-line
		// suppression that ContextNone applies after `{`.
		Entry("statement-level comma inside sequence body (same line)",
			"sequence {\n    0 -> \"hello\",", uint32(1), uint32(17), lsp.ContextStatementStart),
		Entry("statement-level comma inside sequence body (next line)",
			"sequence {\n    0 -> \"hello\",\n", uint32(2), uint32(0), lsp.ContextStatementStart),
		Entry("statement-level comma inside stage body (same line)",
			"stage foo {\n    x = \"bar\",", uint32(1), uint32(14), lsp.ContextStatementStart),
		Entry("statement-level comma inside stage body (next line)",
			"stage foo {\n    x = \"bar\",\n    ", uint32(2), uint32(4), lsp.ContextStatementStart),
		Entry("statement-level comma inside function body (same line)",
			"func foo() {\n    x := 1,", uint32(1), uint32(11), lsp.ContextStatementStart),
		Entry("statement-level comma inside function body (next line)",
			"func foo() {\n    x := 1,\n    ", uint32(2), uint32(4), lsp.ContextStatementStart),
		Entry("statement-level comma inside nested stage body (same line)",
			"sequence main {\n    stage press {\n        0 -> \"hello\",",
			uint32(2), uint32(21), lsp.ContextStatementStart),
		Entry("statement-level comma inside nested stage body (next line)",
			"sequence main {\n    stage press {\n        0 -> \"hello\",\n        ",
			uint32(3), uint32(8), lsp.ContextStatementStart),

		// Expression-level commas must stay expression context regardless of
		// whether the cursor wraps to the next line. Guards the fix above
		// against widening too far and breaking multi-line function calls or
		// array literals.
		Entry("expression-level comma in function call wraps to next line",
			"foo(a,\n    ", uint32(1), uint32(4), lsp.ContextExpression),
		Entry("expression-level comma in array literal wraps to next line",
			"[1,\n    ", uint32(1), uint32(4), lsp.ContextExpression),

		// Unknown Context
		Entry("unknown for ambiguous positions",
			"func foo() i32", uint32(0), uint32(14), lsp.ContextUnknown),

		// Input Param Name Context
		Entry("input param name in empty braces",
			"myFunc{", uint32(0), uint32(7), lsp.ContextInputParamName),
		Entry("input param name after comma",
			"myFunc{a=1, ", uint32(0), uint32(12), lsp.ContextInputParamName),
		Entry("input param name when typing",
			"myFunc{thr", uint32(0), uint32(10), lsp.ContextInputParamName),
		Entry("input param name after comma with partial name",
			"myFunc{a=1, b", uint32(0), uint32(13), lsp.ContextInputParamName),
		Entry("input context for function inside stage",
			"sequence main {\n    stage press {\n        wait{", uint32(2), uint32(13), lsp.ContextInputParamName),

		// Input Param Value Context
		Entry("input param value after equals",
			"myFunc{threshold=", uint32(0), uint32(17), lsp.ContextInputParamValue),
		Entry("input param value with multiple params",
			"myFunc{a=1, b=", uint32(0), uint32(14), lsp.ContextInputParamValue),
	)

	DescribeTable("should not detect input context in body contexts",
		func(content string, line, char uint32) {
			pos := protocol.Position{Line: line, Character: char}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).ToNot(Equal(lsp.ContextInputParamName))
			Expect(ctx).ToNot(Equal(lsp.ContextInputParamValue))
		},
		Entry("function body braces",
			"func foo() { ", uint32(0), uint32(13)),
		Entry("after function call",
			"foo() { ", uint32(0), uint32(8)),
		Entry("inside stage body",
			"stage press { ", uint32(0), uint32(14)),
		Entry("inside stage body on new line",
			"stage press {\n    ", uint32(1), uint32(4)),
		Entry("inside sequence body",
			"sequence main { ", uint32(0), uint32(16)),
		Entry("inside nested stage",
			"sequence main {\n    stage press {\n        ", uint32(2), uint32(8)),
	)
})
