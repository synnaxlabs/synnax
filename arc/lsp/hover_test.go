// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	"go.lsp.dev/protocol"
)

var _ = Describe("Hover", func() {
	var (
		server *lsp.Server
		ctx    context.Context
		uri    protocol.DocumentURI
	)

	BeforeEach(func() {
		ctx = context.Background()
		var err error
		server, err = lsp.New()
		Expect(err).ToNot(HaveOccurred())

		// Set a mock client (needed for diagnostics)
		server.SetClient(&mockClient{})

		uri = "file:///test.arc"
	})

	Describe("Keywords", func() {
		It("should provide hover for 'func' keyword", func() {
			content := "func add(x i32, y i32) i32 {\n    return x + y\n}"
			openDocument(server, ctx, uri, content)

			// Hover over 'func' at position 0:0
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 2}, // fu|nc
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## func"))
			Expect(hover.Contents.Value).To(ContainSubstring("Declares a function"))
			Expect(hover.Contents.Kind).To(Equal(protocol.Markdown))
		})

		It("should provide hover for 'stage' keyword", func() {
			content := "stage max{} (value f32) f32 { return value }"
			openDocument(server, ctx, uri, content)

			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 3}, // sta|ge
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## stage"))
			Expect(hover.Contents.Value).To(ContainSubstring("reactive stage"))
		})

		It("should provide hover for 'if' keyword", func() {
			content := "if x > 10 { return 1 }"
			openDocument(server, ctx, uri, content)

			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 1}, // i|f
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## if"))
			Expect(hover.Contents.Value).To(ContainSubstring("Conditional"))
		})

		It("should provide hover for 'return' keyword", func() {
			content := "return 42"
			openDocument(server, ctx, uri, content)

			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 3}, // ret|urn
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## return"))
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
				openDocument(server, ctx, uri, tc.line)

				hover, err := server.Hover(ctx, &protocol.HoverParams{
					TextDocumentPositionParams: protocol.TextDocumentPositionParams{
						TextDocument: protocol.TextDocumentIdentifier{URI: uri},
						Position:     protocol.Position{Line: 0, Character: tc.pos},
					},
				})

				Expect(err).ToNot(HaveOccurred(), "type: "+tc.typeName)
				Expect(hover).ToNot(BeNil(), "type: "+tc.typeName)
				Expect(hover.Contents.Value).To(ContainSubstring("## "+tc.typeName), "type: "+tc.typeName)
				Expect(hover.Contents.Value).To(ContainSubstring("Range:"), "type: "+tc.typeName)
			}
		})

		It("should provide hover for float types", func() {
			content := "x f32 := 3.14\ny f64 := 2.71828"
			openDocument(server, ctx, uri, content)

			// Hover over f32
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 2},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## f32"))
			Expect(hover.Contents.Value).To(ContainSubstring("32-bit floating point"))

			// Hover over f64
			hover, err = server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 2},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## f64"))
			Expect(hover.Contents.Value).To(ContainSubstring("64-bit floating point"))
		})

		It("should provide hover for temporal types", func() {
			content := "t timestamp := now()\nd timespan := 5s"
			openDocument(server, ctx, uri, content)

			// Hover over timestamp
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 5}, // times|tamp
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## timestamp"))
			Expect(hover.Contents.Value).To(ContainSubstring("nanoseconds since Unix epoch"))

			// Hover over timespan
			hover, err = server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 5}, // times|pan
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## timespan"))
			Expect(hover.Contents.Value).To(ContainSubstring("Duration"))
		})

		It("should provide hover for series type", func() {
			content := "data series f64 := [1.0, 2.0, 3.0]"
			openDocument(server, ctx, uri, content)

			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 7}, // ser|ies
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## series"))
			Expect(hover.Contents.Value).To(ContainSubstring("Homogeneous array"))
		})

		It("should provide hover for chan type", func() {
			content := "ch chan f64"
			openDocument(server, ctx, uri, content)

			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 4}, // ch|an
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## chan"))
			Expect(hover.Contents.Value).To(ContainSubstring("Bidirectional channel"))
		})
	})

	Describe("Built-in Functions", func() {
		It("should provide hover for 'len' function", func() {
			content := "length := len(data)"
			openDocument(server, ctx, uri, content)

			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 11}, // l|en
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## len"))
			Expect(hover.Contents.Value).To(ContainSubstring("length of a series"))
		})

		It("should provide hover for 'now' function", func() {
			content := "time := now()"
			openDocument(server, ctx, uri, content)

			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 9}, // n|ow
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## now"))
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
			openDocument(server, ctx, uri, content)

			// Hover over 'add' in the function call
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 5, Character: 15}, // add|(1, 2)
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## add"))
			Expect(hover.Contents.Value).To(ContainSubstring("func add"))
			Expect(hover.Contents.Value).To(ContainSubstring("x i32"))
			Expect(hover.Contents.Value).To(ContainSubstring("y i32"))
			Expect(hover.Contents.Value).To(ContainSubstring("i32"))
		})

		It("should provide hover for user-defined stages", func() {
			content := `func max{} (value f32) f32 {
    max_val $= value
    if (value > max_val) {
        max_val = value
    }
    return max_val
}`
			openDocument(server, ctx, uri, content)

			// Hover over 'max' in the function declaration
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 6}, // func m|ax
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## max"))
			Expect(hover.Contents.Value).To(ContainSubstring("func max"))
			Expect(hover.Contents.Value).To(ContainSubstring("value f32"))
			Expect(hover.Contents.Value).To(ContainSubstring("Reactive"))
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
			openDocument(server, ctx, uri, content)

			// Hover over 'threshold' in the function declaration
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 8}, // func t|hreshold
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## threshold"))
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
			openDocument(server, ctx, uri, content)

			// Hover over 'x' in the expression
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 2, Character: 9}, // x| + 10
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## x"))
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
			openDocument(server, ctx, uri, content)

			// Hover over 'count' on line 2
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 2, Character: 5}, // count| = count + 1
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## count"))
			Expect(hover.Contents.Value).To(ContainSubstring("Stateful Variable"))
			Expect(hover.Contents.Value).To(ContainSubstring("persists across executions"))
		})

		It("should provide hover for function parameters", func() {
			content := `func multiply(x f64, y f64) f64 {
    return x * y
}
`
			openDocument(server, ctx, uri, content)

			// Hover over 'x' parameter in function body
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 11}, // x| * y
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## x"))
			Expect(hover.Contents.Value).To(ContainSubstring("Input Parameter"))
			Expect(hover.Contents.Value).To(ContainSubstring("f64"))
		})
	})

	Describe("Edge Cases", func() {
		It("should return nil for unknown words", func() {
			content := "unknown_identifier"
			openDocument(server, ctx, uri, content)

			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 5},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).To(BeNil())
		})

		It("should return nil for position out of bounds", func() {
			content := "func test() {}"
			openDocument(server, ctx, uri, content)

			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 10, Character: 0}, // Line doesn't exist
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).To(BeNil())
		})

		It("should return nil for closed document", func() {
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.arc"},
					Position:     protocol.Position{Line: 0, Character: 0},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).To(BeNil())
		})

		It("should handle hovering at end of word", func() {
			content := "func"
			openDocument(server, ctx, uri, content)

			// Hover at last character
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 3}, // func|
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## func"))
		})

		It("should handle hovering at start of word", func() {
			content := "func"
			openDocument(server, ctx, uri, content)

			// Hover at first character
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 0}, // |func
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("## func"))
		})

		It("should handle empty lines", func() {
			content := "\n\nfunc test() {}"
			openDocument(server, ctx, uri, content)

			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 0}, // Empty line
				},
			})

			Expect(err).ToNot(HaveOccurred())
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
			var err error
			server, err = lsp.New(lsp.Config{GlobalResolver: globalResolver})
			Expect(err).ToNot(HaveOccurred())
			server.SetClient(&mockClient{})

			content := "func test() i32 {\n    return myGlobal\n}"
			openDocument(server, ctx, uri, content)

			// Hover over myGlobal
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 12}, // myGl|obal
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("myGlobal"))
			Expect(hover.Contents.Value).To(ContainSubstring("i32"))
		})
	})
})

// Helper function to open a document in the server
func openDocument(server *lsp.Server, ctx context.Context, uri protocol.DocumentURI, content string) {
	err := server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
		TextDocument: protocol.TextDocumentItem{
			URI:        uri,
			LanguageID: "arc",
			Version:    1,
			Text:       content,
		},
	})
	Expect(err).ToNot(HaveOccurred())
}

// Mock client for testing (satisfies protocol.Client interface)
type mockClient struct{}

func (m *mockClient) ShowMessage(ctx context.Context, params *protocol.ShowMessageParams) error {
	return nil
}

func (m *mockClient) ShowMessageRequest(ctx context.Context, params *protocol.ShowMessageRequestParams) (*protocol.MessageActionItem, error) {
	return nil, nil
}

func (m *mockClient) LogMessage(ctx context.Context, params *protocol.LogMessageParams) error {
	return nil
}

func (m *mockClient) Telemetry(ctx context.Context, params interface{}) error {
	return nil
}

func (m *mockClient) RegisterCapability(ctx context.Context, params *protocol.RegistrationParams) error {
	return nil
}

func (m *mockClient) UnregisterCapability(ctx context.Context, params *protocol.UnregistrationParams) error {
	return nil
}

func (m *mockClient) WorkspaceFolders(ctx context.Context) ([]protocol.WorkspaceFolder, error) {
	return nil, nil
}

func (m *mockClient) Configuration(ctx context.Context, params *protocol.ConfigurationParams) ([]interface{}, error) {
	return nil, nil
}

func (m *mockClient) ApplyEdit(ctx context.Context, params *protocol.ApplyWorkspaceEditParams) (bool, error) {
	return false, nil
}

func (m *mockClient) PublishDiagnostics(ctx context.Context, params *protocol.PublishDiagnosticsParams) error {
	// Silently ignore diagnostics in tests
	return nil
}

func (m *mockClient) Progress(ctx context.Context, params *protocol.ProgressParams) error {
	return nil
}

func (m *mockClient) WorkDoneProgressCreate(ctx context.Context, params *protocol.WorkDoneProgressCreateParams) error {
	return nil
}

func (m *mockClient) ShowDocument(ctx context.Context, params *protocol.ShowDocumentParams) (*protocol.ShowDocumentResult, error) {
	return nil, nil
}

func (m *mockClient) Request(ctx context.Context, method string, params interface{}) (interface{}, error) {
	return nil, nil
}
