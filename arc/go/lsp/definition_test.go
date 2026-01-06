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
	"go.lsp.dev/protocol"
)

var _ = Describe("Definition", func() {
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

		server.SetClient(&testutil.MockClient{})
		uri = "file:///test.arc"
	})

	Describe("Function Definitions", func() {
		It("should jump to function definition from call site", func() {
			content := `func add(x i32, y i32) i32 {
    return x + y
}

func main() {
    result := add(1, 2)
}`
			testutil.OpenDocument(server, ctx, uri, content)

			// Click on 'add' in the function call
			locations, err := server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 5, Character: 15}, // add|(1, 2)
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(locations).To(HaveLen(1))
			Expect(locations[0].URI).To(Equal(uri))
			Expect(locations[0].Range.Start.Line).To(Equal(uint32(0))) // Line 0: func add
			// Column should be at "func" keyword or function name
		})

		It("should jump to function definition when hovering over declaration", func() {
			content := `func multiply(x f64, y f64) f64 {
    return x * y
}`
			testutil.OpenDocument(server, ctx, uri, content)

			// Click on 'multiply' in the declaration itself
			locations, err := server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 7}, // func m|ultiply
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(locations).To(HaveLen(1))
			Expect(locations[0].URI).To(Equal(uri))
			Expect(locations[0].Range.Start.Line).To(Equal(uint32(0)))
		})
	})

	Describe("Stage Definitions", func() {
		It("should jump to stage definition", func() {
			content := `func max{} (value f32) f32 {
    max_val $= value
    if (value > max_val) {
        max_val = value
    }
    return max_val
}`
			testutil.OpenDocument(server, ctx, uri, content)

			// Click on 'max' in the declaration
			locations, err := server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 6}, // func m|ax
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(locations).To(HaveLen(1))
			Expect(locations[0].URI).To(Equal(uri))
			Expect(locations[0].Range.Start.Line).To(Equal(uint32(0)))
		})
	})

	Describe("Variable Definitions", func() {
		It("should jump to variable declaration from usage", func() {
			content := `func test() {
    x i32 := 42
    y := x + 10
}`
			testutil.OpenDocument(server, ctx, uri, content)

			// Click on 'x' in the expression
			locations, err := server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 2, Character: 9}, // x| + 10
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(locations).To(HaveLen(1))
			Expect(locations[0].URI).To(Equal(uri))
			Expect(locations[0].Range.Start.Line).To(Equal(uint32(1))) // Line 1: x i32 := 42
		})

		It("should jump to stateful variable declaration", func() {
			content := `func counter{} () u32 {
    count u32 $= 0
    count = count + 1
    return count
}`
			testutil.OpenDocument(server, ctx, uri, content)

			// Click on 'count' in the assignment
			locations, err := server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 2, Character: 5}, // count| = count + 1
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(locations).To(HaveLen(1))
			Expect(locations[0].URI).To(Equal(uri))
			Expect(locations[0].Range.Start.Line).To(Equal(uint32(1))) // Line 1: count u32 $= 0
		})
	})

	Describe("Parameter Definitions", func() {
		It("should jump to parameter declaration from function body", func() {
			content := `func multiply(x f64, y f64) f64 {
    return x * y
}`
			testutil.OpenDocument(server, ctx, uri, content)

			// Click on 'x' in the return statement
			locations, err := server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 11}, // x| * y
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(locations).To(HaveLen(1))
			Expect(locations[0].URI).To(Equal(uri))
			Expect(locations[0].Range.Start.Line).To(Equal(uint32(0))) // Line 0: func multiply(x f64, y f64)
		})
	})

	Describe("Edge Cases", func() {
		It("should return nil for keywords", func() {
			content := `func test() {
    return 42
}`
			testutil.OpenDocument(server, ctx, uri, content)

			// Click on 'return' keyword
			locations, err := server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 5}, // ret|urn
				},
			})

			Expect(err).ToNot(HaveOccurred())
			// Keywords don't have definitions, so should return nil
			Expect(locations).To(BeNil())
		})

		It("should return nil for undefined symbols", func() {
			content := `func test() {
    x := undefined_symbol
}`
			testutil.OpenDocument(server, ctx, uri, content)

			// Click on 'undefined_symbol'
			locations, err := server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 13}, // undefined_symbol|
				},
			})

			Expect(err).ToNot(HaveOccurred())
			// Undefined symbols should return nil
			Expect(locations).To(BeNil())
		})

		It("should return nil when document not found", func() {
			locations, err := server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.arc"},
					Position:     protocol.Position{Line: 0, Character: 0},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(locations).To(BeNil())
		})

		It("should return nil for empty word", func() {
			content := `func test() {

}`
			testutil.OpenDocument(server, ctx, uri, content)

			// Click on empty line
			locations, err := server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 0}, // Empty line
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(locations).To(BeNil())
		})
	})

	Describe("GlobalResolver", func() {
		It("should return nil for global variables from GlobalResolver (no AST)", func() {
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
			server.SetClient(&testutil.MockClient{})

			content := "func test() i32 {\n    return myGlobal\n}"
			testutil.OpenDocument(server, ctx, uri, content)

			// Try to jump to definition of myGlobal
			locations, err := server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 12}, // myGl|obal
				},
			})

			Expect(err).ToNot(HaveOccurred())
			// GlobalResolver symbols have no AST, so should return nil
			Expect(locations).To(BeNil())
		})
	})

})
