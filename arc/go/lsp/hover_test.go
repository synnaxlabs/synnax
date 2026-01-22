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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var _ = Describe("Hover", func() {
	var (
		ctx    context.Context
		server *lsp.Server
		uri    protocol.DocumentURI
	)

	BeforeEach(func() {
		ctx = context.Background()
		server, uri = SetupTestServer()
	})

	DescribeTable("keyword hover",
		func(content string, char uint32, expectedTitle string, expectedSubstring string) {
			OpenDocument(server, ctx, uri, content)
			hover := Hover(server, ctx, uri, 0, char)
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### " + expectedTitle))
			if expectedSubstring != "" {
				Expect(hover.Contents.Value).To(ContainSubstring(expectedSubstring))
			}
		},
		Entry("func", "func add(x i32, y i32) i32 {\n    return x + y\n}", uint32(2), "func", "Declares a function"),
		Entry("stage", "sequence main { stage first {} }", uint32(18), "stage", "within a sequence"),
		Entry("if", "if x > 10 { return 1 }", uint32(1), "if", "Conditional"),
		Entry("return", "return 42", uint32(3), "return", ""),
		Entry("sequence", "sequence main { stage first {} }", uint32(4), "sequence", "state machine"),
	)

	DescribeTable("type hover with range",
		func(content string, char uint32, expectedType string) {
			OpenDocument(server, ctx, uri, content)
			hover := Hover(server, ctx, uri, 0, char)
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### " + expectedType))
			Expect(hover.Contents.Value).To(ContainSubstring("Range:"))
		},
		Entry("i8", "x i8 := 127", uint32(2), "i8"),
		Entry("i16", "y i16 := 32767", uint32(2), "i16"),
		Entry("i32", "z i32 := 2147483647", uint32(2), "i32"),
		Entry("i64", "a i64 := 9223372036854775807", uint32(2), "i64"),
		Entry("u8", "b u8 := 255", uint32(2), "u8"),
		Entry("u16", "c u16 := 65535", uint32(2), "u16"),
		Entry("u32", "d u32 := 4294967295", uint32(2), "u32"),
		Entry("u64", "e u64 := 18446744073709551615", uint32(2), "u64"),
	)

	DescribeTable("type hover",
		func(content string, line, char uint32, expectedType, expectedSubstring string) {
			OpenDocument(server, ctx, uri, content)
			hover := Hover(server, ctx, uri, line, char)
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### " + expectedType))
			Expect(hover.Contents.Value).To(ContainSubstring(expectedSubstring))
		},
		Entry("f32", "x f32 := 3.14", uint32(0), uint32(2), "f32", "32-bit floating point"),
		Entry("f64", "x f32 := 3.14\ny f64 := 2.71828", uint32(1), uint32(2), "f64", "64-bit floating point"),
		Entry("series", "data series f64 := [1.0, 2.0, 3.0]", uint32(0), uint32(7), "series", "Homogeneous array"),
		Entry("chan", "ch chan f64", uint32(0), uint32(4), "chan", "Bidirectional channel"),
	)

	Describe("Built-in Functions", func() {
		It("should provide hover for 'len' function", func() {
			content := "length := len(data)"
			OpenDocument(server, ctx, uri, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 11}, // l|en
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### len"))
			Expect(hover.Contents.Value).To(ContainSubstring("length of a series"))
		})

		It("should provide hover for 'now' function", func() {
			content := "time := now()"
			OpenDocument(server, ctx, uri, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 9}, // n|ow
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### now"))
			Expect(hover.Contents.Value).To(ContainSubstring("current timestamp"))
		})
	})

	Describe("User-Defined Symbols", func() {
		It("should provide hover for user-defined functions", func() {
			content := `func add(x i32, y i32) i32 {
    return x + y
}

func main() {
    result := add(1, 2)
}`
			OpenDocument(server, ctx, uri, content)

			// Hover over 'add' in the function call
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 5, Character: 15}, // add|(1, 2)
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### add"))
			Expect(hover.Contents.Value).To(ContainSubstring("func add"))
			Expect(hover.Contents.Value).To(ContainSubstring("x i32"))
			Expect(hover.Contents.Value).To(ContainSubstring("y i32"))
			Expect(hover.Contents.Value).To(ContainSubstring("i32"))
		})

		It("should provide hover for user-defined functions", func() {
			content := `func max{} (value f32) f32 {
    max_val $= value
    if (value > max_val) {
        max_val = value
    }
    return max_val
}`
			OpenDocument(server, ctx, uri, content)

			// Hover over 'max' in the function declaration
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 6}, // func m|ax
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### max"))
			Expect(hover.Contents.Value).To(ContainSubstring("func max"))
			Expect(hover.Contents.Value).To(ContainSubstring("value f32"))
		})

		It("should provide hover for stages with config", func() {
			content := `func threshold{
    limit f64
} (value f64) u8 {
    if (value > limit) {
        return u8(1)
    }
    return u8(0)
}`
			OpenDocument(server, ctx, uri, content)

			// Hover over 'threshold' in the function declaration
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 8}, // func t|hreshold
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### threshold"))
			Expect(hover.Contents.Value).To(ContainSubstring("func threshold"))
			Expect(hover.Contents.Value).To(ContainSubstring("limit f64"))
			Expect(hover.Contents.Value).To(ContainSubstring("value f64"))
		})

		It("should provide hover for variables", func() {
			content := `func test() {
    x i32 := 42
    y := x + 10
}
`
			OpenDocument(server, ctx, uri, content)

			// Hover over 'x' in the expression
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 2, Character: 9}, // x| + 10
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### x"))
			Expect(hover.Contents.Value).To(ContainSubstring("Variable"))
			Expect(hover.Contents.Value).To(ContainSubstring("i32"))
		})

		It("should provide hover for stateful variables", func() {
			content := `func counter{} () u32 {
    count u32 $= 0
    count = count + 1
    return count
}
`
			OpenDocument(server, ctx, uri, content)

			// Hover over 'count' on line 2
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 2, Character: 5}, // count| = count + 1
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### count"))
			Expect(hover.Contents.Value).To(ContainSubstring("Stateful Variable"))
			Expect(hover.Contents.Value).To(ContainSubstring("Persists across executions"))
		})

		It("should provide hover for function parameters", func() {
			content := `func multiply(x f64, y f64) f64 {
    return x * y
}
`
			OpenDocument(server, ctx, uri, content)

			// Hover over 'x' parameter in function body
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 11}, // x| * y
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### x"))
			Expect(hover.Contents.Value).To(ContainSubstring("Input Parameter"))
			Expect(hover.Contents.Value).To(ContainSubstring("f64"))
		})

		It("should provide hover for sequence declarations", func() {
			content := `sequence main {
    stage first {}
    stage second {}
}`
			OpenDocument(server, ctx, uri, content)

			// Hover over 'main' sequence name
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 10}, // sequence m|ain
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### main"))
			Expect(hover.Contents.Value).To(ContainSubstring("Sequence"))
			Expect(hover.Contents.Value).To(ContainSubstring("first"))
			Expect(hover.Contents.Value).To(ContainSubstring("second"))
		})

		It("should provide hover for stage declarations within sequence", func() {
			content := `sequence main {
    stage first {}
}`
			OpenDocument(server, ctx, uri, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 11},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### first"))
			Expect(hover.Contents.Value).To(ContainSubstring("Stage"))
		})

		It("should include single-line doc comment in hover", func() {
			content := `// Adds two numbers together
func add(x i32, y i32) i32 {
    return x + y
}`
			OpenDocument(server, ctx, uri, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 6},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### add"))
			Expect(hover.Contents.Value).To(ContainSubstring("Adds two numbers together"))
		})

		It("should include multi-line doc comment in hover", func() {
			content := `/* Computes the maximum of two values */
func max(a i32, b i32) i32 {
    if a > b { return a }
    return b
}`
			OpenDocument(server, ctx, uri, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 6},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### max"))
			Expect(hover.Contents.Value).To(ContainSubstring("Computes the maximum of two values"))
		})

		It("should include multiple consecutive single-line comments in hover", func() {
			content := `// Threshold function
// Returns 1 if value exceeds limit, 0 otherwise
func threshold(value f64) u8 {
    return u8(0)
}`
			OpenDocument(server, ctx, uri, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 2, Character: 6},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("Threshold function"))
			Expect(hover.Contents.Value).To(ContainSubstring("Returns 1 if value exceeds limit"))
		})

		It("should not include comment separated by code from symbol", func() {
			content := `// Comment for helper
func helper() i32 {
    return 0
}

func add(a i32, b i32) i32 {
    return a + b
}`
			OpenDocument(server, ctx, uri, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 5, Character: 6},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### add"))
			Expect(hover.Contents.Value).ToNot(ContainSubstring("Comment for helper"))
		})
	})

	DescribeTable("operator hover",
		func(content string, char uint32, expectedOp, expectedSubstring string) {
			OpenDocument(server, ctx, uri, content)
			hover := Hover(server, ctx, uri, 0, char)
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring(expectedOp))
			if expectedSubstring != "" {
				Expect(hover.Contents.Value).To(ContainSubstring(expectedSubstring))
			}
		},
		Entry(":=", "x := 42", uint32(2), ":=", "Declares and initializes"),
		Entry("$=", "count $= 0", uint32(6), "$=", "stateful"),
		Entry("=>", "if ready => next_stage", uint32(9), "=>", "Transitions"),
		Entry("->", "value -> channel", uint32(6), "->", "channel"),
		Entry("==", "x == y", uint32(2), "==", ""),
		Entry("!=", "x != y", uint32(2), "!=", ""),
		Entry("<=", "x <= y", uint32(2), "<=", ""),
		Entry(">=", "x >= y", uint32(2), ">=", ""),
		Entry("+=", "x += 5", uint32(2), "+=", ""),
		Entry("-=", "x -= 5", uint32(2), "-=", ""),
		Entry("*=", "x *= 5", uint32(2), "*=", ""),
		Entry("/=", "x /= 5", uint32(2), "/=", ""),
		Entry("%=", "x %= 5", uint32(2), "%=", ""),
	)

	Describe("Edge Cases", func() {
		It("should return nil for unknown words", func() {
			OpenDocument(server, ctx, uri, "unknown_identifier")
			Expect(Hover(server, ctx, uri, 0, 5)).To(BeNil())
		})

		It("should return nil for position out of bounds", func() {
			OpenDocument(server, ctx, uri, "func test() {}")
			Expect(Hover(server, ctx, uri, 10, 0)).To(BeNil())
		})

		It("should return nil for closed document", func() {
			hover := Hover(server, ctx, "file:///nonexistent.arc", 0, 0)
			Expect(hover).To(BeNil())
		})

		It("should handle hovering at end of word", func() {
			OpenDocument(server, ctx, uri, "func")
			hover := Hover(server, ctx, uri, 0, 3)
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### func"))
		})

		It("should handle hovering at start of word", func() {
			OpenDocument(server, ctx, uri, "func")
			hover := Hover(server, ctx, uri, 0, 0)
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### func"))
		})

		It("should handle empty lines", func() {
			OpenDocument(server, ctx, uri, "\n\nfunc test() {}")
			Expect(Hover(server, ctx, uri, 0, 0)).To(BeNil())
		})
	})

	Describe("GlobalResolver", func() {
		It("should provide hover for global variables from GlobalResolver", func() {
			globalResolver := symbol.MapResolver{
				"myGlobal": symbol.Symbol{
					Name: "myGlobal",
					Type: types.I32(),
					Kind: symbol.KindVariable,
				},
			}

			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			OpenDocument(server, ctx, uri, "func test() i32 {\n    return myGlobal\n}")
			hover := Hover(server, ctx, uri, 1, 12)
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("myGlobal"))
			Expect(hover.Contents.Value).To(ContainSubstring("i32"))
		})
	})

	Describe("SemanticTokens", func() {
		DescribeTable("Keywords",
			func(content string, expectedType uint32) {
				OpenDocument(server, ctx, uri, content)
				tokens := SemanticTokens(server, ctx, uri)
				Expect(tokens).ToNot(BeNil())
				Expect(len(tokens.Data)).To(BeNumerically(">=", 5))
				Expect(tokens.Data[3]).To(Equal(expectedType))
			},
			Entry("func", "func test() {}", uint32(lsp.SemanticTokenTypeKeyword)),
			Entry("if", "if x {}", uint32(lsp.SemanticTokenTypeKeyword)),
			Entry("else", "else {}", uint32(lsp.SemanticTokenTypeKeyword)),
			Entry("return", "return 1", uint32(lsp.SemanticTokenTypeKeyword)),
			Entry("sequence", "sequence main {}", uint32(lsp.SemanticTokenTypeKeyword)),
			Entry("stage", "stage first {}", uint32(lsp.SemanticTokenTypeKeyword)),
			Entry("next", "next foo", uint32(lsp.SemanticTokenTypeKeyword)),
		)

		DescribeTable("Types",
			func(content string, expectedType uint32) {
				OpenDocument(server, ctx, uri, content)
				tokens := SemanticTokens(server, ctx, uri)
				Expect(tokens).ToNot(BeNil())
				Expect(len(tokens.Data)).To(BeNumerically(">=", 10))
				Expect(tokens.Data[8]).To(Equal(expectedType))
			},
			Entry("i8", "x i8", uint32(lsp.SemanticTokenTypeType)),
			Entry("i16", "x i16", uint32(lsp.SemanticTokenTypeType)),
			Entry("i32", "x i32", uint32(lsp.SemanticTokenTypeType)),
			Entry("i64", "x i64", uint32(lsp.SemanticTokenTypeType)),
			Entry("u8", "x u8", uint32(lsp.SemanticTokenTypeType)),
			Entry("u16", "x u16", uint32(lsp.SemanticTokenTypeType)),
			Entry("u32", "x u32", uint32(lsp.SemanticTokenTypeType)),
			Entry("u64", "x u64", uint32(lsp.SemanticTokenTypeType)),
			Entry("f32", "x f32", uint32(lsp.SemanticTokenTypeType)),
			Entry("f64", "x f64", uint32(lsp.SemanticTokenTypeType)),
			Entry("str", "x str", uint32(lsp.SemanticTokenTypeType)),
			Entry("series", "x series", uint32(lsp.SemanticTokenTypeType)),
			Entry("chan", "x chan", uint32(lsp.SemanticTokenTypeType)),
		)

		DescribeTable("Operators",
			func(content string, expectedType uint32) {
				OpenDocument(server, ctx, uri, content)
				tokens := SemanticTokens(server, ctx, uri)
				Expect(tokens).ToNot(BeNil())
				Expect(len(tokens.Data)).To(BeNumerically(">=", 10))
				Expect(tokens.Data[8]).To(Equal(expectedType))
			},
			Entry("declare :=", "x := 1", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("state declare $=", "x $= 1", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("assign =", "x = 1", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("arrow ->", "x -> y", uint32(lsp.SemanticTokenTypeEdgeContinuous)),
			Entry("transition =>", "x => y", uint32(lsp.SemanticTokenTypeEdgeOneShot)),
			Entry("plus +", "x + y", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("minus -", "x - y", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("star *", "x * y", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("slash /", "x / y", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("percent %", "x % y", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("caret ^", "x ^ y", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("eq ==", "x == y", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("neq !=", "x != y", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("lt <", "x < y", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("gt >", "x > y", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("leq <=", "x <= y", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("geq >=", "x >= y", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("and", "x and y", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("or", "x or y", uint32(lsp.SemanticTokenTypeOperator)),
		)

		DescribeTable("Single token types",
			func(content string, expectedType uint32) {
				OpenDocument(server, ctx, uri, content)
				tokens := SemanticTokens(server, ctx, uri)
				Expect(tokens).ToNot(BeNil())
				Expect(len(tokens.Data)).To(BeNumerically(">=", 5))
				Expect(tokens.Data[3]).To(Equal(expectedType))
			},
			Entry("not operator", "not x", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("variable", "myVariable", uint32(lsp.SemanticTokenTypeVariable)),
			Entry("string literal", `"hello world"`, uint32(lsp.SemanticTokenTypeString)),
			Entry("integer", "42", uint32(lsp.SemanticTokenTypeNumber)),
			Entry("float", "3.14", uint32(lsp.SemanticTokenTypeNumber)),
			Entry("float starting with dot", ".5", uint32(lsp.SemanticTokenTypeNumber)),
			Entry("single-line comment", "// comment", uint32(lsp.SemanticTokenTypeComment)),
			Entry("multi-line comment", "/* comment */", uint32(lsp.SemanticTokenTypeComment)),
		)

		It("should tokenize function names as function type", func() {
			OpenDocument(server, ctx, uri, "func myFunc() {}")
			tokens := SemanticTokens(server, ctx, uri)
			Expect(tokens).ToNot(BeNil())
			Expect(len(tokens.Data)).To(BeNumerically(">=", 10))
			Expect(tokens.Data[3]).To(Equal(uint32(lsp.SemanticTokenTypeKeyword)))
			Expect(tokens.Data[8]).To(Equal(uint32(lsp.SemanticTokenTypeFunction)))
		})

		It("should tokenize input parameters as input type", func() {
			OpenDocument(server, ctx, uri, "func myFunc(x f32) {}")
			tokens := SemanticTokens(server, ctx, uri)
			Expect(tokens).ToNot(BeNil())
			foundInput := false
			for i := 3; i < len(tokens.Data); i += 5 {
				if tokens.Data[i] == uint32(lsp.SemanticTokenTypeInput) {
					foundInput = true
					break
				}
			}
			Expect(foundInput).To(BeTrue())
		})

		It("should tokenize sequence names as sequence type", func() {
			OpenDocument(server, ctx, uri, "sequence main { stage init {} }")
			tokens := SemanticTokens(server, ctx, uri)
			Expect(tokens).ToNot(BeNil())
			Expect(len(tokens.Data)).To(BeNumerically(">=", 10))
			Expect(tokens.Data[3]).To(Equal(uint32(lsp.SemanticTokenTypeKeyword)))
			Expect(tokens.Data[8]).To(Equal(uint32(lsp.SemanticTokenTypeSequence)))
		})

		It("should tokenize stage names as stage type", func() {
			OpenDocument(server, ctx, uri, "sequence main { stage init {} }")
			tokens := SemanticTokens(server, ctx, uri)
			Expect(tokens).ToNot(BeNil())
			stageKeywordIdx := -1
			for i := 0; i < len(tokens.Data)-5; i += 5 {
				if tokens.Data[i+3] == uint32(lsp.SemanticTokenTypeKeyword) {
					if i >= 10 {
						stageKeywordIdx = i
						break
					}
				}
			}
			Expect(stageKeywordIdx).ToNot(Equal(-1))
			Expect(tokens.Data[stageKeywordIdx+8]).To(Equal(uint32(lsp.SemanticTokenTypeStage)))
		})

		It("should tokenize stateful variables as statefulVariable type", func() {
			OpenDocument(server, ctx, uri, "func counter{} () u32 {\n    count u32 $= 0\n    return count\n}")
			tokens := SemanticTokens(server, ctx, uri)
			Expect(tokens).ToNot(BeNil())
			foundStateful := false
			for i := 3; i < len(tokens.Data); i += 5 {
				if tokens.Data[i] == uint32(lsp.SemanticTokenTypeStatefulVariable) {
					foundStateful = true
					break
				}
			}
			Expect(foundStateful).To(BeTrue())
		})

		It("should tokenize channel variables as channel type", func() {
			globalResolver := symbol.MapResolver{
				"sensorData": symbol.Symbol{
					Name: "sensorData",
					Type: types.Chan(types.F64()),
					Kind: symbol.KindChannel,
				},
			}

			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			OpenDocument(server, ctx, uri, "func test() { x := sensorData }")
			tokens := SemanticTokens(server, ctx, uri)
			Expect(tokens).ToNot(BeNil())
			foundChannel := false
			for i := 3; i < len(tokens.Data); i += 5 {
				if tokens.Data[i] == uint32(lsp.SemanticTokenTypeChannel) {
					foundChannel = true
					break
				}
			}
			Expect(foundChannel).To(BeTrue())
		})
	})
})
