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
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

var _ = Describe("Hover", func() {
	var (
		server *lsp.Server
		docURI uri.URI
	)

	BeforeEach(func() {
		server, docURI = SetupTestServer()
	})

	DescribeTable("keyword hover",
		func(ctx SpecContext, content string, char uint32, expectedTitle string, expectedSubstring string) {
			OpenArcDocument(server, ctx, docURI, content)
			hover := Hover(server, ctx, docURI, 0, char)
			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### " + expectedTitle))
			if expectedSubstring != "" {
				Expect(HoverContents(hover)).To(ContainSubstring(expectedSubstring))
			}
		},
		Entry("func", "func add(x i32, y i32) i32 {\n    return x + y\n}", uint32(2), "func", "Declares a function"),
		Entry("stage", "sequence main { stage first {} }", uint32(18), "stage", "within a sequence"),
		Entry("if", "if x > 10 { return 1 }", uint32(1), "if", "Conditional"),
		Entry("return", "return 42", uint32(3), "return", ""),
		Entry("sequence", "sequence main { stage first {} }", uint32(4), "sequence", "state machine"),
		Entry("authority", "authority 200", uint32(4), "authority", "control authority"),
	)

	DescribeTable("type hover with range",
		func(ctx SpecContext, content string, char uint32, expectedType string) {
			OpenArcDocument(server, ctx, docURI, content)
			hover := Hover(server, ctx, docURI, 0, char)
			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### " + expectedType))
			Expect(HoverContents(hover)).To(ContainSubstring("Range:"))
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
		func(ctx SpecContext, content string, line, char uint32, expectedType, expectedSubstring string) {
			OpenArcDocument(server, ctx, docURI, content)
			hover := Hover(server, ctx, docURI, line, char)
			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### " + expectedType))
			Expect(HoverContents(hover)).To(ContainSubstring(expectedSubstring))
		},
		Entry("f32", "x f32 := 3.14", uint32(0), uint32(2), "f32", "32-bit floating point"),
		Entry("f64", "x f32 := 3.14\ny f64 := 2.71828", uint32(1), uint32(2), "f64", "64-bit floating point"),
		Entry("series", "data series f64 := [1.0, 2.0, 3.0]", uint32(0), uint32(7), "series", "Homogeneous array"),
		Entry("chan", "ch chan f64", uint32(0), uint32(4), "chan", "Bidirectional channel"),
	)

	Describe("Built-in Functions", func() {
		It("should provide hover for 'len' function", func(ctx SpecContext) {
			content := "length := len(data)"
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 0, Character: 11}, // l|en
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### len"))
			Expect(HoverContents(hover)).To(ContainSubstring("length of a series"))
		})

		It("should provide hover for 'set_authority' function", func(ctx SpecContext) {
			content := "set_authority{value=255}"
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 0, Character: 5},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### set_authority"))
			Expect(HoverContents(hover)).To(ContainSubstring("control.set_authority"))
		})

		It("should provide hover for 'control.set_authority' function", func(ctx SpecContext) {
			content := "import control\n\ntrig -> control.set_authority{value=255}"
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 2, Character: 20}, // control.set_a|uthority
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### control.set_authority"))
			Expect(HoverContents(hover)).To(ContainSubstring("control authority"))
		})

		It("should provide hover for 'math.avg' function", func(ctx SpecContext) {
			content := "import math\n\nsensor -> math.avg{} -> output"
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 2, Character: 16},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### math.avg"))
			Expect(HoverContents(hover)).To(ContainSubstring("running average"))
		})

		It("should provide hover for 'select' function", func(ctx SpecContext) {
			content := "flag -> select{} -> output"
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 0, Character: 11},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### select"))
			Expect(HoverContents(hover)).To(ContainSubstring("Routes input values"))
		})

		It("should provide hover for 'stable.for' function", func(ctx SpecContext) {
			content := "import stable\n\nsensor -> stable.for{duration=5s} -> output"
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 2, Character: 18},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### stable.for"))
			Expect(HoverContents(hover)).To(ContainSubstring("remained stable"))
		})

		It("should provide hover for 'time.now' function", func(ctx SpecContext) {
			content := "import time\n\nfunc test() i64 {\n    return time.now()\n}"
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 3, Character: 17}, // n|ow
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### time.now"))
			Expect(HoverContents(hover)).To(ContainSubstring("current timestamp"))
		})

		It("should provide deprecation hover for bare 'now' function", func(ctx SpecContext) {
			content := "t := now()"
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 0, Character: 6}, // n|ow
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("deprecated"))
			Expect(HoverContents(hover)).To(ContainSubstring("time.now"))
		})

		It("should provide hover for 'time.interval' function", func(ctx SpecContext) {
			content := "import time\n\ntime.interval{period=100ms} -> output"
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 2, Character: 7}, // time.i|nterval
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### time.interval"))
			Expect(HoverContents(hover)).To(ContainSubstring("Fires repeatedly"))
		})

		It("should provide deprecation hover for bare 'interval' function", func(ctx SpecContext) {
			content := "interval{period=100ms}"
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 0, Character: 2}, // i|nterval
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("deprecated"))
			Expect(HoverContents(hover)).To(ContainSubstring("time.interval"))
		})

		It("should provide hover for 'time.wait' function", func(ctx SpecContext) {
			content := "import time\n\ntime.wait{duration=500ms} -> output"
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 2, Character: 7}, // time.w|ait
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### time.wait"))
			Expect(HoverContents(hover)).To(ContainSubstring("Fires once"))
		})

		It("should provide deprecation hover for bare 'wait' function", func(ctx SpecContext) {
			content := "wait{duration=500ms}"
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 0, Character: 2}, // w|ait
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("deprecated"))
			Expect(HoverContents(hover)).To(ContainSubstring("time.wait"))
		})
	})

	Describe("User-Defined Symbols", func() {
		It("should provide hover for user-defined functions", func(ctx SpecContext) {
			content := `func add(x i32, y i32) i32 {
    return x + y
}

func main() {
    result := add(1, 2)
}`
			OpenArcDocument(server, ctx, docURI, content)

			// Hover over 'add' in the function call
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 5, Character: 15}, // add|(1, 2)
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### add"))
			Expect(HoverContents(hover)).To(ContainSubstring("func add"))
			Expect(HoverContents(hover)).To(ContainSubstring("x i32"))
			Expect(HoverContents(hover)).To(ContainSubstring("y i32"))
			Expect(HoverContents(hover)).To(ContainSubstring("i32"))
		})

		It("should provide hover for user-defined functions", func(ctx SpecContext) {
			content := `func max{} (value f32) f32 {
    max_val $= value
    if (value > max_val) {
        max_val = value
    }
    return max_val
}`
			OpenArcDocument(server, ctx, docURI, content)

			// Hover over 'max' in the function declaration
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 0, Character: 6}, // func m|ax
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### max"))
			Expect(HoverContents(hover)).To(ContainSubstring("func max"))
			Expect(HoverContents(hover)).To(ContainSubstring("value f32"))
		})

		It("should provide hover for stages with config", func(ctx SpecContext) {
			content := `func threshold{
    limit f64
} (value f64) u8 {
    if (value > limit) {
        return u8(1)
    }
    return u8(0)
}`
			OpenArcDocument(server, ctx, docURI, content)

			// Hover over 'threshold' in the function declaration
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 0, Character: 8}, // func t|hreshold
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### threshold"))
			Expect(HoverContents(hover)).To(ContainSubstring("func threshold"))
			Expect(HoverContents(hover)).To(ContainSubstring("limit f64"))
			Expect(HoverContents(hover)).To(ContainSubstring("value f64"))
		})

		It("should provide hover for variables", func(ctx SpecContext) {
			content := `func test() {
    x i32 := 42
    y := x + 10
}
`
			OpenArcDocument(server, ctx, docURI, content)

			// Hover over 'x' in the expression
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 2, Character: 9}, // x| + 10
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### x"))
			Expect(HoverContents(hover)).To(ContainSubstring("Variable"))
			Expect(HoverContents(hover)).To(ContainSubstring("i32"))
		})

		It("should provide hover for stateful variables", func(ctx SpecContext) {
			content := `func counter{} () u32 {
    count u32 $= 0
    count = count + 1
    return count
}
`
			OpenArcDocument(server, ctx, docURI, content)

			// Hover over 'count' on line 2
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 2, Character: 5}, // count| = count + 1
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### count"))
			Expect(HoverContents(hover)).To(ContainSubstring("Stateful Variable"))
			Expect(HoverContents(hover)).To(ContainSubstring("Persists across executions"))
		})

		It("should provide hover for function parameters", func(ctx SpecContext) {
			content := `func multiply(x f64, y f64) f64 {
    return x * y
}
`
			OpenArcDocument(server, ctx, docURI, content)

			// Hover over 'x' parameter in function body
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 1, Character: 11}, // x| * y
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### x"))
			Expect(HoverContents(hover)).To(ContainSubstring("Input Parameter"))
			Expect(HoverContents(hover)).To(ContainSubstring("f64"))
		})

		It("should provide hover for sequence declarations", func(ctx SpecContext) {
			content := `sequence main {
    stage first {}
    stage second {}
}`
			OpenArcDocument(server, ctx, docURI, content)

			// Hover over 'main' sequence name
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 0, Character: 10}, // sequence m|ain
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### main"))
			Expect(HoverContents(hover)).To(ContainSubstring("Sequence"))
			Expect(HoverContents(hover)).To(ContainSubstring("first"))
			Expect(HoverContents(hover)).To(ContainSubstring("second"))
		})

		It("should provide hover for stage declarations within sequence", func(ctx SpecContext) {
			content := `sequence main {
    stage first {}
}`
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 1, Character: 11},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### first"))
			Expect(HoverContents(hover)).To(ContainSubstring("Stage"))
		})

		It("should include single-line doc comment in hover", func(ctx SpecContext) {
			content := `// Adds two numbers together
func add(x i32, y i32) i32 {
    return x + y
}`
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 1, Character: 6},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### add"))
			Expect(HoverContents(hover)).To(ContainSubstring("Adds two numbers together"))
		})

		It("should include multi-line doc comment in hover", func(ctx SpecContext) {
			content := `/* Computes the maximum of two values */
func max(a i32, b i32) i32 {
    if a > b { return a }
    return b
}`
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 1, Character: 6},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### max"))
			Expect(HoverContents(hover)).To(ContainSubstring("Computes the maximum of two values"))
		})

		It("should include multiple consecutive single-line comments in hover", func(ctx SpecContext) {
			content := `// Threshold function
// Returns 1 if value exceeds limit, 0 otherwise
func threshold(value f64) u8 {
    return u8(0)
}`
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 2, Character: 6},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("Threshold function"))
			Expect(HoverContents(hover)).To(ContainSubstring("Returns 1 if value exceeds limit"))
		})

		It("should not include comment separated by code from symbol", func(ctx SpecContext) {
			content := `// Comment for helper
func helper() i32 {
    return 0
}

func add(a i32, b i32) i32 {
    return a + b
}`
			OpenArcDocument(server, ctx, docURI, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
					Position:     protocol.Position{Line: 5, Character: 6},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### add"))
			Expect(HoverContents(hover)).ToNot(ContainSubstring("Comment for helper"))
		})
	})

	DescribeTable("kind label hover",
		func(
			ctx SpecContext,
			content string,
			line, char uint32,
			expectedTitle, expectedKind string,
		) {
			OpenArcDocument(server, ctx, docURI, content)
			hover := Hover(server, ctx, docURI, line, char)
			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### " + expectedTitle))
			Expect(HoverContents(hover)).To(ContainSubstring(expectedKind))
		},
		Entry("function",
			"func foo() i32 { return 0 }\n",
			uint32(0), uint32(5),
			"foo", "Node"),
		Entry("variable",
			"func test() {\n    x i32 := 42\n    y := x + 10\n}\n",
			uint32(2), uint32(9),
			"x", "Variable"),
		Entry("stateful variable",
			"func counter{} () u32 {\n    count u32 $= 0\n    count = count + 1\n    return count\n}\n",
			uint32(2), uint32(5),
			"count", "Stateful Variable"),
		Entry("input parameter",
			"func multiply(x f64, y f64) f64 {\n    return x * y\n}\n",
			uint32(1), uint32(11),
			"x", "Input Parameter"),
		Entry("output parameter",
			"func compute() result f64 {\n    result = 1.0\n}\n",
			uint32(1), uint32(4),
			"result", "Output Parameter"),
		Entry("config parameter",
			"func compute{k f64}() f64 {\n    return k\n}\n",
			uint32(1), uint32(11),
			"k", "Configuration Parameter"),
		Entry("module alias",
			"import time\n\nfunc test() {\n    time.now()\n}\n",
			uint32(3), uint32(5),
			"time", "Module"),
		Entry("sequence",
			"sequence main {\n    stage first {}\n}\n",
			uint32(0), uint32(10),
			"main", "Sequence"),
		Entry("stage",
			"sequence main {\n    stage first {}\n}\n",
			uint32(1), uint32(11),
			"first", "Stage"),
	)

	DescribeTable("operator hover",
		func(ctx SpecContext, content string, char uint32, expectedOp, expectedSubstring string) {
			OpenArcDocument(server, ctx, docURI, content)
			hover := Hover(server, ctx, docURI, 0, char)
			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring(expectedOp))
			if expectedSubstring != "" {
				Expect(HoverContents(hover)).To(ContainSubstring(expectedSubstring))
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
		It("should return nil for unknown words", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "unknown_identifier")
			Expect(Hover(server, ctx, docURI, 0, 5)).To(BeNil())
		})

		It("should return nil for position out of bounds", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func test() {}")
			Expect(Hover(server, ctx, docURI, 10, 0)).To(BeNil())
		})

		It("should return nil for closed document", func(ctx SpecContext) {
			hover := Hover(server, ctx, "file:///nonexistent.arc", 0, 0)
			Expect(hover).To(BeNil())
		})

		It("should handle hovering at end of word", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func")
			hover := Hover(server, ctx, docURI, 0, 3)
			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### func"))
		})

		It("should handle hovering at start of word", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func")
			hover := Hover(server, ctx, docURI, 0, 0)
			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### func"))
		})

		It("should handle empty lines", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "\n\nfunc test() {}")
			Expect(Hover(server, ctx, docURI, 0, 0)).To(BeNil())
		})
	})

	Describe("GlobalResolver", func() {
		It("should provide hover for global variables from GlobalResolver", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol {
				return NewRoot(nil, symbol.Symbol{Name: "myGlobal",
					Type: types.I32(),
					Kind: symbol.KindVariable})
			}}))
			server.SetClient(&MockClient{})

			OpenArcDocument(server, ctx, docURI, "func test() i32 {\n    return myGlobal\n}")
			hover := Hover(server, ctx, docURI, 1, 12)
			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("myGlobal"))
			Expect(HoverContents(hover)).To(ContainSubstring("i32"))
		})
	})

	Describe("Qualified Module Identifiers", func() {
		It("Should provide hover for qualified module function", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{
				NewRoot: func() *symbol.Symbol {
					return NewRoot(nil)
				},
			}))
			server.SetClient(&MockClient{})

			content := "import time\n\nfunc test() i64 {\n    return time.now()\n}"
			OpenArcDocument(server, ctx, docURI, content)
			hover := Hover(server, ctx, docURI, 3, 14)
			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("now"))
		})

		It("Should not provide hover for invalid module prefix", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{
				NewRoot: func() *symbol.Symbol {
					return NewRoot(nil)
				},
			}))
			server.SetClient(&MockClient{})

			content := "func test() {\n    fake.nonexistent()\n}"
			OpenArcDocument(server, ctx, docURI, content)
			hover := Hover(server, ctx, docURI, 1, 10)
			Expect(hover).To(BeNil())
		})

		It("Should not provide hover for a member of an unimported module", func(ctx SpecContext) {
			// `time` is not imported, so `time.now` is an undefined reference
			// to the analyzer. Hover must not render docs as if it were valid.
			content := "func test() i64 {\n    return time.now()\n}"
			OpenArcDocument(server, ctx, docURI, content)
			hover := Hover(server, ctx, docURI, 1, 17) // n|ow
			Expect(hover).To(BeNil())
		})

		It("Should provide hover for a member of an imported module", func(ctx SpecContext) {
			content := "import time\n\nfunc test() i64 {\n    return time.now()\n}"
			OpenArcDocument(server, ctx, docURI, content)
			hover := Hover(server, ctx, docURI, 3, 17) // n|ow
			Expect(hover).ToNot(BeNil())
			Expect(HoverContents(hover)).To(ContainSubstring("#### time.now"))
		})
	})

	Describe("SemanticTokens", func() {
		DescribeTable("Keywords",
			func(ctx SpecContext, content string, expectedType uint32) {
				OpenArcDocument(server, ctx, docURI, content)
				tokens := SemanticTokens(server, ctx, docURI)
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
			Entry("for", "for i {}", uint32(lsp.SemanticTokenTypeKeyword)),
			Entry("break", "break", uint32(lsp.SemanticTokenTypeKeyword)),
			Entry("continue", "continue", uint32(lsp.SemanticTokenTypeKeyword)),
		)

		DescribeTable("Types",
			func(ctx SpecContext, content string, expectedType uint32) {
				OpenArcDocument(server, ctx, docURI, content)
				tokens := SemanticTokens(server, ctx, docURI)
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
			func(ctx SpecContext, content string, expectedType uint32) {
				OpenArcDocument(server, ctx, docURI, content)
				tokens := SemanticTokens(server, ctx, docURI)
				Expect(tokens).ToNot(BeNil())
				Expect(len(tokens.Data)).To(BeNumerically(">=", 5))
				found := false
				for i := 3; i < len(tokens.Data); i += 5 {
					if tokens.Data[i] == expectedType {
						found = true
						break
					}
				}
				Expect(found).To(BeTrue(), "expected token type %d not found", expectedType)
			},
			Entry("declare :=", "x := 1", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("state declare $=", "x $= 1", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("assign =", "x = 1", uint32(lsp.SemanticTokenTypeOperator)),
			Entry("arrow ->", "x -> y", uint32(lsp.SemanticTokenTypeEdgeContinuous)),
			Entry("transition =>", "x => y", uint32(lsp.SemanticTokenTypeEdgeConditional)),
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
			Entry("and", "x and y", uint32(lsp.SemanticTokenTypeKeyword)),
			Entry("or", "x or y", uint32(lsp.SemanticTokenTypeKeyword)),
		)

		DescribeTable("Single token types",
			func(ctx SpecContext, content string, expectedType uint32) {
				OpenArcDocument(server, ctx, docURI, content)
				tokens := SemanticTokens(server, ctx, docURI)
				Expect(tokens).ToNot(BeNil())
				Expect(len(tokens.Data)).To(BeNumerically(">=", 5))
				Expect(tokens.Data[3]).To(Equal(expectedType))
			},
			Entry("not keyword", "not x", uint32(lsp.SemanticTokenTypeKeyword)),
			Entry("variable", "myVariable", uint32(lsp.SemanticTokenTypeVariable)),
			Entry("string literal", `"hello world"`, uint32(lsp.SemanticTokenTypeString)),
			Entry("integer", "42", uint32(lsp.SemanticTokenTypeNumber)),
			Entry("float", "3.14", uint32(lsp.SemanticTokenTypeNumber)),
			Entry("float starting with dot", ".5", uint32(lsp.SemanticTokenTypeNumber)),
			Entry("single-line comment", "// comment", uint32(lsp.SemanticTokenTypeComment)),
			Entry("multi-line comment", "/* comment */", uint32(lsp.SemanticTokenTypeComment)),
		)

		It("should tokenize function names as function type", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func myFunc() {}")
			tokens := SemanticTokens(server, ctx, docURI)
			Expect(tokens).ToNot(BeNil())
			Expect(len(tokens.Data)).To(BeNumerically(">=", 10))
			Expect(tokens.Data[3]).To(Equal(uint32(lsp.SemanticTokenTypeKeyword)))
			Expect(tokens.Data[8]).To(Equal(uint32(lsp.SemanticTokenTypeFunction)))
		})

		It("should tokenize input parameters as input type", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func myFunc(x f32) {}")
			tokens := SemanticTokens(server, ctx, docURI)
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

		It("should tokenize sequence names as function type", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "sequence main { stage init {} }")
			tokens := SemanticTokens(server, ctx, docURI)
			Expect(tokens).ToNot(BeNil())
			Expect(len(tokens.Data)).To(BeNumerically(">=", 10))
			Expect(tokens.Data[3]).To(Equal(uint32(lsp.SemanticTokenTypeKeyword)))
			Expect(tokens.Data[8]).To(Equal(uint32(lsp.SemanticTokenTypeFunction)))
		})

		It("should tokenize stage names as function type", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "sequence main { stage init {} }")
			tokens := SemanticTokens(server, ctx, docURI)
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
			Expect(tokens.Data[stageKeywordIdx+8]).To(Equal(uint32(lsp.SemanticTokenTypeFunction)))
		})

		It("should tokenize stateful variables as statefulVariable type", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func counter{} () u32 {\n    count u32 $= 0\n    return count\n}")
			tokens := SemanticTokens(server, ctx, docURI)
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

		It("should tokenize channel variables as channel type", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol {
				return NewRoot(nil, symbol.Symbol{Name: "sensorData",
					Type: types.Chan(types.F64()),
					Kind: symbol.KindChannel})
			}}))
			server.SetClient(&MockClient{})

			OpenArcDocument(server, ctx, docURI, "func test() { x := sensorData }")
			tokens := SemanticTokens(server, ctx, docURI)
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

		It("should tokenize module prefix as variable in qualified calls", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "time.interval{period=100ms}")
			tokens := SemanticTokens(server, ctx, docURI)
			Expect(tokens).ToNot(BeNil())
			Expect(len(tokens.Data)).To(BeNumerically(">=", 5))
			Expect(tokens.Data[3]).To(Equal(uint32(lsp.SemanticTokenTypeVariable)))
		})

		It("should tokenize member name as function in qualified calls", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "import time\n\ntime.interval{period=100ms} -> output")
			tokens := SemanticTokens(server, ctx, docURI)
			Expect(tokens).ToNot(BeNil())
			foundFunction := false
			for i := 3; i < len(tokens.Data); i += 5 {
				if tokens.Data[i] == uint32(lsp.SemanticTokenTypeFunction) {
					foundFunction = true
					break
				}
			}
			Expect(foundFunction).To(BeTrue())
		})

		It("should tokenize keyword normally when not a module prefix", func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "authority 200")
			tokens := SemanticTokens(server, ctx, docURI)
			Expect(tokens).ToNot(BeNil())
			Expect(len(tokens.Data)).To(BeNumerically(">=", 5))
			Expect(tokens.Data[3]).To(Equal(uint32(lsp.SemanticTokenTypeKeyword)))
		})
	})
})
