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
	"go.lsp.dev/protocol"
)

var _ = Describe("Context Detection", func() {
	Describe("Comment Context", func() {
		It("should detect single-line comment at start", func() {
			content := "// this is a comment"
			pos := protocol.Position{Line: 0, Character: 5}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextComment))
		})

		It("should detect single-line comment at end", func() {
			content := "// this is a comment"
			pos := protocol.Position{Line: 0, Character: 20}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextComment))
		})

		It("should detect single-line comment after code", func() {
			content := "x := 1 // comment here"
			pos := protocol.Position{Line: 0, Character: 15}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextComment))
		})

		It("should detect multi-line comment on first line", func() {
			content := "/* multi\nline\ncomment */"
			pos := protocol.Position{Line: 0, Character: 5}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextComment))
		})

		It("should detect multi-line comment on middle line", func() {
			content := "/* multi\nline\ncomment */"
			pos := protocol.Position{Line: 1, Character: 2}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextComment))
		})

		It("should detect multi-line comment on last line", func() {
			content := "/* multi\nline\ncomment */"
			pos := protocol.Position{Line: 2, Character: 5}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextComment))
		})

		It("should not detect comment when position is before comment start", func() {
			content := "x := 1 // comment"
			pos := protocol.Position{Line: 0, Character: 3}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).ToNot(Equal(lsp.ContextComment))
		})
	})

	Describe("Type Annotation Context", func() {
		It("should detect type annotation after LPAREN and identifier", func() {
			content := "func foo(x "
			pos := protocol.Position{Line: 0, Character: 11}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextTypeAnnotation))
		})

		It("should detect type annotation after COMMA and identifier", func() {
			content := "func foo(x i32, y "
			pos := protocol.Position{Line: 0, Character: 18}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextTypeAnnotation))
		})

		It("should detect partial type annotation", func() {
			content := "func foo(x i"
			pos := protocol.Position{Line: 0, Character: 12}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextTypeAnnotation))
		})
	})

	Describe("Expression Context", func() {
		It("should detect expression after DECLARE", func() {
			content := "x := "
			pos := protocol.Position{Line: 0, Character: 5}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextExpression))
		})

		It("should detect expression after STATE_DECLARE", func() {
			content := "x $= "
			pos := protocol.Position{Line: 0, Character: 5}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextExpression))
		})

		It("should detect expression after ASSIGN", func() {
			content := "x = "
			pos := protocol.Position{Line: 0, Character: 4}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextExpression))
		})

		It("should detect expression after PLUS", func() {
			content := "x := y + "
			pos := protocol.Position{Line: 0, Character: 9}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextExpression))
		})

		It("should detect expression after MINUS", func() {
			content := "x := y - "
			pos := protocol.Position{Line: 0, Character: 9}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextExpression))
		})

		It("should detect expression after comparison operator", func() {
			content := "if x > "
			pos := protocol.Position{Line: 0, Character: 7}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextExpression))
		})

		It("should detect expression after RETURN", func() {
			content := "return "
			pos := protocol.Position{Line: 0, Character: 7}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextExpression))
		})

		It("should detect expression after LPAREN in function call", func() {
			content := "foo("
			pos := protocol.Position{Line: 0, Character: 4}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextExpression))
		})

		It("should detect expression after COMMA in function call", func() {
			content := "foo(a, "
			pos := protocol.Position{Line: 0, Character: 7}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextExpression))
		})
	})

	Describe("Statement Start Context", func() {
		It("should detect statement start after LBRACE", func() {
			content := "func foo() { "
			pos := protocol.Position{Line: 0, Character: 13}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextStatementStart))
		})

		It("should detect statement start on new line in function body", func() {
			content := "func foo() {\n    "
			pos := protocol.Position{Line: 1, Character: 4}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextStatementStart))
		})

		It("should detect statement start after closing brace on new line", func() {
			content := "if true {\n    x := 1\n}\n"
			pos := protocol.Position{Line: 3, Character: 0}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextStatementStart))
		})

		It("should detect statement start at empty file", func() {
			content := ""
			pos := protocol.Position{Line: 0, Character: 0}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextStatementStart))
		})
	})

	Describe("Unknown Context", func() {
		It("should return unknown for ambiguous positions", func() {
			content := "func foo() i32"
			pos := protocol.Position{Line: 0, Character: 14}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextUnknown))
		})
	})

	Describe("Config Param Name Context", func() {
		It("should detect config param name in empty braces", func() {
			content := "myFunc{"
			pos := protocol.Position{Line: 0, Character: 7}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextConfigParamName))
		})

		It("should detect config param name after comma", func() {
			content := "myFunc{a=1, "
			pos := protocol.Position{Line: 0, Character: 12}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextConfigParamName))
		})

		It("should detect config param name when typing", func() {
			content := "myFunc{thr"
			pos := protocol.Position{Line: 0, Character: 10}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextConfigParamName))
		})

		It("should detect config param name after comma with partial name", func() {
			content := "myFunc{a=1, b"
			pos := protocol.Position{Line: 0, Character: 13}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextConfigParamName))
		})

		It("should not detect config context in function body braces", func() {
			content := "func foo() { "
			pos := protocol.Position{Line: 0, Character: 13}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).ToNot(Equal(lsp.ContextConfigParamName))
			Expect(ctx).ToNot(Equal(lsp.ContextConfigParamValue))
		})

		It("should not detect config context after function call", func() {
			content := "foo() { "
			pos := protocol.Position{Line: 0, Character: 8}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).ToNot(Equal(lsp.ContextConfigParamName))
		})

		It("should not detect config context inside stage body", func() {
			content := "stage press { "
			pos := protocol.Position{Line: 0, Character: 14}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).ToNot(Equal(lsp.ContextConfigParamName))
			Expect(ctx).ToNot(Equal(lsp.ContextConfigParamValue))
		})

		It("should not detect config context inside stage body on new line", func() {
			content := "stage press {\n    "
			pos := protocol.Position{Line: 1, Character: 4}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).ToNot(Equal(lsp.ContextConfigParamName))
			Expect(ctx).ToNot(Equal(lsp.ContextConfigParamValue))
		})

		It("should not detect config context inside sequence body", func() {
			content := "sequence main { "
			pos := protocol.Position{Line: 0, Character: 16}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).ToNot(Equal(lsp.ContextConfigParamName))
			Expect(ctx).ToNot(Equal(lsp.ContextConfigParamValue))
		})

		It("should not detect config context inside nested stage", func() {
			content := "sequence main {\n    stage press {\n        "
			pos := protocol.Position{Line: 2, Character: 8}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).ToNot(Equal(lsp.ContextConfigParamName))
			Expect(ctx).ToNot(Equal(lsp.ContextConfigParamValue))
		})

		It("should still detect config context for function inside stage", func() {
			content := "sequence main {\n    stage press {\n        wait{"
			pos := protocol.Position{Line: 2, Character: 13}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextConfigParamName))
		})
	})

	Describe("Config Param Value Context", func() {
		It("should detect config param value after equals", func() {
			content := "myFunc{threshold="
			pos := protocol.Position{Line: 0, Character: 17}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextConfigParamValue))
		})

		It("should detect config param value with multiple params", func() {
			content := "myFunc{a=1, b="
			pos := protocol.Position{Line: 0, Character: 14}
			ctx := lsp.DetectCompletionContext(content, pos)
			Expect(ctx).To(Equal(lsp.ContextConfigParamValue))
		})
	})

	Describe("ExtractConfigContext", func() {
		It("should extract function name from config block", func() {
			content := "myFunc{"
			pos := protocol.Position{Line: 0, Character: 7}
			info := lsp.ExtractConfigContext(content, pos)
			Expect(info).ToNot(BeNil())
			Expect(info.FunctionName).To(Equal("myFunc"))
			Expect(info.ExistingParams).To(BeEmpty())
		})

		It("should extract existing params from config block", func() {
			content := "myFunc{a=1, b=2, "
			pos := protocol.Position{Line: 0, Character: 17}
			info := lsp.ExtractConfigContext(content, pos)
			Expect(info).ToNot(BeNil())
			Expect(info.FunctionName).To(Equal("myFunc"))
			Expect(info.ExistingParams).To(ConsistOf("a", "b"))
		})

		It("should extract current param name in value context", func() {
			content := "myFunc{threshold="
			pos := protocol.Position{Line: 0, Character: 17}
			info := lsp.ExtractConfigContext(content, pos)
			Expect(info).ToNot(BeNil())
			Expect(info.FunctionName).To(Equal("myFunc"))
			Expect(info.CurrentParamName).To(Equal("threshold"))
		})

		It("should return nil for non-config context", func() {
			content := "func foo() { "
			pos := protocol.Position{Line: 0, Character: 13}
			info := lsp.ExtractConfigContext(content, pos)
			Expect(info).To(BeNil())
		})

		It("should return nil for stage body context", func() {
			content := "stage press { "
			pos := protocol.Position{Line: 0, Character: 14}
			info := lsp.ExtractConfigContext(content, pos)
			Expect(info).To(BeNil())
		})

		It("should return nil for sequence body context", func() {
			content := "sequence main { "
			pos := protocol.Position{Line: 0, Character: 16}
			info := lsp.ExtractConfigContext(content, pos)
			Expect(info).To(BeNil())
		})
	})
})
