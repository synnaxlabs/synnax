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
	"github.com/synnaxlabs/arc/lsp/testutil"
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
		server, uri = testutil.SetupTestServer()
	})

	Describe("Keywords", func() {
		It("should provide hover for 'func' keyword", func() {
			testutil.OpenDocument(server, ctx, uri, "func add(x i32, y i32) i32 {\n    return x + y\n}")
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 2},
				},
			}))
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### func"))
			Expect(hover.Contents.Value).To(ContainSubstring("Declares a function"))
			Expect(hover.Contents.Kind).To(Equal(protocol.Markdown))
		})

		It("should provide hover for 'stage' keyword", func() {
			content := "sequence main { stage first {} }"
			testutil.OpenDocument(server, ctx, uri, content)
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 18}, // sta|ge
				},
			}))
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### stage"))
			Expect(hover.Contents.Value).To(ContainSubstring("within a sequence"))
		})

		It("should provide hover for 'if' keyword", func() {
			content := "if x > 10 { return 1 }"
			testutil.OpenDocument(server, ctx, uri, content)
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 1}, // i|f
				},
			}))
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### if"))
			Expect(hover.Contents.Value).To(ContainSubstring("Conditional"))
		})

		It("should provide hover for 'return' keyword", func() {
			content := "return 42"
			testutil.OpenDocument(server, ctx, uri, content)
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 3}, // ret|urn
				},
			}))
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### return"))
		})

		It("should provide hover for 'sequence' keyword", func() {
			content := "sequence main { stage first {} }"
			testutil.OpenDocument(server, ctx, uri, content)
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 4}, // sequ|ence
				},
			}))
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### sequence"))
			Expect(hover.Contents.Value).To(ContainSubstring("state machine"))
		})
	})

	Describe("Types", func() {
		It("should provide hover for integer types", func() {
			testCases := []struct {
				typeName string
				line     string
				pos      uint32
			}{
				{"i8", "x i8 := 127", 2},
				{"i16", "y i16 := 32767", 2},
				{"i32", "z i32 := 2147483647", 2},
				{"i64", "a i64 := 9223372036854775807", 2},
				{"u8", "b u8 := 255", 2},
				{"u16", "c u16 := 65535", 2},
				{"u32", "d u32 := 4294967295", 2},
				{"u64", "e u64 := 18446744073709551615", 2},
			}

			for _, tc := range testCases {
				testutil.OpenDocument(server, ctx, uri, tc.line)
				hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
					TextDocumentPositionParams: protocol.TextDocumentPositionParams{
						TextDocument: protocol.TextDocumentIdentifier{URI: uri},
						Position:     protocol.Position{Line: 0, Character: tc.pos},
					},
				}))
				Expect(hover).ToNot(BeNil(), "type: "+tc.typeName)
				Expect(hover.Contents.Value).To(ContainSubstring("#### "+tc.typeName), "type: "+tc.typeName)
				Expect(hover.Contents.Value).To(ContainSubstring("Range:"), "type: "+tc.typeName)
			}
		})

		It("should provide hover for float types", func() {
			content := "x f32 := 3.14\ny f64 := 2.71828"
			testutil.OpenDocument(server, ctx, uri, content)

			// Hover over f32
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 2},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### f32"))
			Expect(hover.Contents.Value).To(ContainSubstring("32-bit floating point"))

			// Hover over f64
			hover = MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 2},
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### f64"))
			Expect(hover.Contents.Value).To(ContainSubstring("64-bit floating point"))
		})

		It("should provide hover for series type", func() {
			content := "data series f64 := [1.0, 2.0, 3.0]"
			testutil.OpenDocument(server, ctx, uri, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 7}, // ser|ies
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### series"))
			Expect(hover.Contents.Value).To(ContainSubstring("Homogeneous array"))
		})

		It("should provide hover for chan type", func() {
			content := "ch chan f64"
			testutil.OpenDocument(server, ctx, uri, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 4}, // ch|an
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### chan"))
			Expect(hover.Contents.Value).To(ContainSubstring("Bidirectional channel"))
		})
	})

	Describe("Built-in Functions", func() {
		It("should provide hover for 'len' function", func() {
			content := "length := len(data)"
			testutil.OpenDocument(server, ctx, uri, content)

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
			testutil.OpenDocument(server, ctx, uri, content)

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
			testutil.OpenDocument(server, ctx, uri, content)

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
			testutil.OpenDocument(server, ctx, uri, content)

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
			testutil.OpenDocument(server, ctx, uri, content)

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
			testutil.OpenDocument(server, ctx, uri, content)

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
			testutil.OpenDocument(server, ctx, uri, content)

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
			testutil.OpenDocument(server, ctx, uri, content)

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
			testutil.OpenDocument(server, ctx, uri, content)

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
			testutil.OpenDocument(server, ctx, uri, content)

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
			testutil.OpenDocument(server, ctx, uri, content)

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
			testutil.OpenDocument(server, ctx, uri, content)

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
			testutil.OpenDocument(server, ctx, uri, content)

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
			testutil.OpenDocument(server, ctx, uri, content)

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

	Describe("Operators", func() {
		It("should provide hover for := operator", func() {
			content := "x := 42"
			testutil.OpenDocument(server, ctx, uri, content)
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 2},
				},
			}))
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring(":="))
			Expect(hover.Contents.Value).To(ContainSubstring("Declares and initializes"))
		})

		It("should provide hover for $= operator", func() {
			content := "count $= 0"
			testutil.OpenDocument(server, ctx, uri, content)
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 6},
				},
			}))
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("$="))
			Expect(hover.Contents.Value).To(ContainSubstring("stateful"))
		})

		It("should provide hover for => operator", func() {
			content := "if ready => next_stage"
			testutil.OpenDocument(server, ctx, uri, content)
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 9},
				},
			}))
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("=>"))
			Expect(hover.Contents.Value).To(ContainSubstring("Transitions"))
		})

		It("should provide hover for -> operator", func() {
			content := "value -> channel"
			testutil.OpenDocument(server, ctx, uri, content)
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 6},
				},
			}))
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("->"))
			Expect(hover.Contents.Value).To(ContainSubstring("channel"))
		})

		It("should provide hover for comparison operators", func() {
			testCases := []struct {
				op      string
				content string
				pos     uint32
			}{
				{"==", "x == y", 2},
				{"!=", "x != y", 2},
				{"<=", "x <= y", 2},
				{">=", "x >= y", 2},
			}
			for _, tc := range testCases {
				testutil.OpenDocument(server, ctx, uri, tc.content)
				hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
					TextDocumentPositionParams: protocol.TextDocumentPositionParams{
						TextDocument: protocol.TextDocumentIdentifier{URI: uri},
						Position:     protocol.Position{Line: 0, Character: tc.pos},
					},
				}))
				Expect(hover).ToNot(BeNil(), "operator: "+tc.op)
				Expect(hover.Contents.Value).To(ContainSubstring(tc.op), "operator: "+tc.op)
			}
		})

		It("should provide hover for compound assignment operators", func() {
			testCases := []struct {
				op      string
				content string
				pos     uint32
			}{
				{"+=", "x += 5", 2},
				{"-=", "x -= 5", 2},
				{"*=", "x *= 5", 2},
				{"/=", "x /= 5", 2},
				{"%=", "x %= 5", 2},
			}
			for _, tc := range testCases {
				testutil.OpenDocument(server, ctx, uri, tc.content)
				hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
					TextDocumentPositionParams: protocol.TextDocumentPositionParams{
						TextDocument: protocol.TextDocumentIdentifier{URI: uri},
						Position:     protocol.Position{Line: 0, Character: tc.pos},
					},
				}))
				Expect(hover).ToNot(BeNil(), "operator: "+tc.op)
				Expect(hover.Contents.Value).To(ContainSubstring(tc.op), "operator: "+tc.op)
			}
		})
	})

	Describe("Edge Cases", func() {
		It("should return nil for unknown words", func() {
			content := "unknown_identifier"
			testutil.OpenDocument(server, ctx, uri, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 5},
				},
			}))

			Expect(hover).To(BeNil())
		})

		It("should return nil for position out of bounds", func() {
			content := "func test() {}"
			testutil.OpenDocument(server, ctx, uri, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 10, Character: 0}, // Line doesn't exist
				},
			}))

			Expect(hover).To(BeNil())
		})

		It("should return nil for closed document", func() {
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.arc"},
					Position:     protocol.Position{Line: 0, Character: 0},
				},
			}))

			Expect(hover).To(BeNil())
		})

		It("should handle hovering at end of word", func() {
			content := "func"
			testutil.OpenDocument(server, ctx, uri, content)

			// Hover at last character
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 3}, // func|
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### func"))
		})

		It("should handle hovering at start of word", func() {
			content := "func"
			testutil.OpenDocument(server, ctx, uri, content)

			// Hover at first character
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 0}, // |func
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("#### func"))
		})

		It("should handle empty lines", func() {
			content := "\n\nfunc test() {}"
			testutil.OpenDocument(server, ctx, uri, content)

			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 0}, // Empty line
				},
			}))

			Expect(hover).To(BeNil())
		})
	})

	Describe("GlobalResolver", func() {
		It("should provide hover for global variables from GlobalResolver", func() {
			// Create a mock GlobalResolver with a global variable
			globalResolver := symbol.MapResolver{
				"myGlobal": symbol.Symbol{
					Name: "myGlobal",
					Type: types.I32(),
					Kind: symbol.KindVariable,
				},
			}

			// Create server with GlobalResolver
			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&testutil.MockClient{})

			content := "func test() i32 {\n    return myGlobal\n}"
			testutil.OpenDocument(server, ctx, uri, content)

			// Hover over myGlobal
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 12}, // myGl|obal
				},
			}))

			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("myGlobal"))
			Expect(hover.Contents.Value).To(ContainSubstring("i32"))
		})
	})

	Describe("SemanticTokens", func() {
		DescribeTable("Keywords",
			func(content string, expectedType uint32) {
				testutil.OpenDocument(server, ctx, uri, content)
				tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				}))
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
				testutil.OpenDocument(server, ctx, uri, content)
				tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				}))
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
				testutil.OpenDocument(server, ctx, uri, content)
				tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				}))
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

		It("should tokenize not operator", func() {
			testutil.OpenDocument(server, ctx, uri, "not x")
			tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			}))
			Expect(tokens).ToNot(BeNil())
			Expect(len(tokens.Data)).To(BeNumerically(">=", 5))
			Expect(tokens.Data[3]).To(Equal(uint32(lsp.SemanticTokenTypeOperator)))
		})

		It("should tokenize variables", func() {
			testutil.OpenDocument(server, ctx, uri, "myVariable")
			tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			}))
			Expect(tokens).ToNot(BeNil())
			Expect(len(tokens.Data)).To(BeNumerically(">=", 5))
			Expect(tokens.Data[3]).To(Equal(uint32(lsp.SemanticTokenTypeVariable)))
		})

		It("should tokenize string literals", func() {
			testutil.OpenDocument(server, ctx, uri, `"hello world"`)
			tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			}))
			Expect(tokens).ToNot(BeNil())
			Expect(len(tokens.Data)).To(BeNumerically(">=", 5))
			Expect(tokens.Data[3]).To(Equal(uint32(lsp.SemanticTokenTypeString)))
		})

		DescribeTable("Numbers",
			func(content string, expectedType uint32) {
				testutil.OpenDocument(server, ctx, uri, content)
				tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				}))
				Expect(tokens).ToNot(BeNil())
				Expect(len(tokens.Data)).To(BeNumerically(">=", 5))
				Expect(tokens.Data[3]).To(Equal(expectedType))
			},
			Entry("integer", "42", uint32(lsp.SemanticTokenTypeNumber)),
			Entry("float", "3.14", uint32(lsp.SemanticTokenTypeNumber)),
			Entry("float starting with dot", ".5", uint32(lsp.SemanticTokenTypeNumber)),
		)

		DescribeTable("Comments",
			func(content string, expectedType uint32) {
				testutil.OpenDocument(server, ctx, uri, content)
				tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				}))
				Expect(tokens).ToNot(BeNil())
				Expect(len(tokens.Data)).To(BeNumerically(">=", 5))
				Expect(tokens.Data[3]).To(Equal(expectedType))
			},
			Entry("single-line", "// comment", uint32(lsp.SemanticTokenTypeComment)),
			Entry("multi-line", "/* comment */", uint32(lsp.SemanticTokenTypeComment)),
		)

		It("should tokenize function names as function type", func() {
			testutil.OpenDocument(server, ctx, uri, "func myFunc() {}")
			tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			}))
			Expect(tokens).ToNot(BeNil())
			Expect(len(tokens.Data)).To(BeNumerically(">=", 10))
			Expect(tokens.Data[3]).To(Equal(uint32(lsp.SemanticTokenTypeKeyword)))
			Expect(tokens.Data[8]).To(Equal(uint32(lsp.SemanticTokenTypeFunction)))
		})

		It("should tokenize input parameters as input type", func() {
			testutil.OpenDocument(server, ctx, uri, "func myFunc(x f32) {}")
			tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			}))
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
			testutil.OpenDocument(server, ctx, uri, "sequence main { stage init {} }")
			tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			}))
			Expect(tokens).ToNot(BeNil())
			Expect(len(tokens.Data)).To(BeNumerically(">=", 10))
			Expect(tokens.Data[3]).To(Equal(uint32(lsp.SemanticTokenTypeKeyword)))
			Expect(tokens.Data[8]).To(Equal(uint32(lsp.SemanticTokenTypeSequence)))
		})

		It("should tokenize stage names as stage type", func() {
			testutil.OpenDocument(server, ctx, uri, "sequence main { stage init {} }")
			tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			}))
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
			testutil.OpenDocument(server, ctx, uri, `func counter{} () u32 {
    count u32 $= 0
    return count
}`)
			tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			}))
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
			server.SetClient(&testutil.MockClient{})

			testutil.OpenDocument(server, ctx, uri, "func test() { x := sensorData }")
			tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			}))
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
