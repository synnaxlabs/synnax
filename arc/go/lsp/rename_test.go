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

var _ = Describe("Rename", func() {
	var (
		server *lsp.Server
		ctx    context.Context
		uri    protocol.DocumentURI
	)

	BeforeEach(func() {
		ctx = context.Background()
		server = MustSucceed(lsp.New())
		server.SetClient(&MockClient{})
		uri = "file:///test.arc"
	})

	Describe("PrepareRename", func() {
		It("should return range for renameable variable", func() {
			content := `func test() {
    x i32 := 42
    y := x + 10
}`
			OpenDocument(server, ctx, uri, content)

			result := MustSucceed(server.PrepareRename(ctx, &protocol.PrepareRenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 4}, // x| i32
				},
			}))
			Expect(result).ToNot(BeNil())
			Expect(result.Start.Line).To(Equal(uint32(1)))
			Expect(result.Start.Character).To(Equal(uint32(4)))
		})

		It("should return nil for keywords", func() {
			content := `func test() {
    return 42
}`
			OpenDocument(server, ctx, uri, content)

			result := MustSucceed(server.PrepareRename(ctx, &protocol.PrepareRenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 5}, // ret|urn
				},
			}))
			Expect(result).To(BeNil())
		})

		It("should return nil for global/builtin symbols", func() {
			globalResolver := symbol.MapResolver{
				"myGlobal": symbol.Symbol{
					Name: "myGlobal",
					Type: types.I32(),
					Kind: symbol.KindVariable,
				},
			}

			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			content := "func test() i32 {\n    return myGlobal\n}"
			OpenDocument(server, ctx, uri, content)

			result := MustSucceed(server.PrepareRename(ctx, &protocol.PrepareRenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 12}, // myGl|obal
				},
			}))
			Expect(result).To(BeNil())
		})

		It("should return nil when document not found", func() {
			result := MustSucceed(server.PrepareRename(ctx, &protocol.PrepareRenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.arc"},
					Position:     protocol.Position{Line: 0, Character: 0},
				},
			}))
			Expect(result).To(BeNil())
		})
	})

	Describe("Rename", func() {
		It("should rename variable with multiple references", func() {
			content := `func test() {
    x i32 := 42
    y := x + 10
    z := x * 2
}`
			OpenDocument(server, ctx, uri, content)

			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 4}, // x| i32
				},
				NewName: "value",
			}))
			Expect(result).ToNot(BeNil())
			Expect(result.Changes).To(HaveKey(uri))
			edits := result.Changes[uri]
			Expect(edits).To(HaveLen(3)) // declaration + 2 usages
			for _, edit := range edits {
				Expect(edit.NewText).To(Equal("value"))
			}
		})

		It("should rename function parameter", func() {
			content := `func multiply(x f64, y f64) f64 {
    return x * y
}`
			OpenDocument(server, ctx, uri, content)

			// Click on 'x' in return statement
			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 11}, // x| * y
				},
				NewName: "first",
			}))
			Expect(result).ToNot(BeNil())
			Expect(result.Changes).To(HaveKey(uri))
			edits := result.Changes[uri]
			Expect(edits).To(HaveLen(2)) // parameter + usage
			for _, edit := range edits {
				Expect(edit.NewText).To(Equal("first"))
			}
		})

		It("should not rename variables across different functions", func() {
			content := `func first() {
    x := 10
    y := x + 1
}

func second() {
    x := 20
    z := x + 2
}`
			OpenDocument(server, ctx, uri, content)

			// Rename x in first function
			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 4}, // x| in first()
				},
				NewName: "value",
			}))
			Expect(result).ToNot(BeNil())
			Expect(result.Changes).To(HaveKey(uri))
			edits := result.Changes[uri]
			// Should only rename x in first function (declaration + usage)
			// NOT the x in second function
			Expect(edits).To(HaveLen(2))
			// Verify edits are on lines 1 and 2 (first function only)
			for _, edit := range edits {
				Expect(edit.Range.Start.Line).To(BeNumerically("<=", 2))
				Expect(edit.NewText).To(Equal("value"))
			}
		})

		It("should rename stateful variable", func() {
			content := `func counter{} () u32 {
    count u32 $= 0
    count = count + 1
    return count
}`
			OpenDocument(server, ctx, uri, content)

			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 4}, // count| u32 $= 0
				},
				NewName: "total",
			}))
			Expect(result).ToNot(BeNil())
			Expect(result.Changes).To(HaveKey(uri))
			edits := result.Changes[uri]
			// count is used: declaration, assignment (left), addition (right), return
			Expect(edits).To(HaveLen(4))
			for _, edit := range edits {
				Expect(edit.NewText).To(Equal("total"))
			}
		})

		It("should rename function", func() {
			content := `func add(x i32, y i32) i32 {
    return x + y
}

func main() {
    result := add(1, 2)
}`
			OpenDocument(server, ctx, uri, content)

			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 6}, // func a|dd
				},
				NewName: "sum",
			}))
			Expect(result).ToNot(BeNil())
			Expect(result.Changes).To(HaveKey(uri))
			edits := result.Changes[uri]
			Expect(edits).To(HaveLen(2)) // definition + call site
			for _, edit := range edits {
				Expect(edit.NewText).To(Equal("sum"))
			}
		})

		It("should return nil for global/builtin symbols", func() {
			globalResolver := symbol.MapResolver{
				"myGlobal": symbol.Symbol{
					Name: "myGlobal",
					Type: types.I32(),
					Kind: symbol.KindVariable,
				},
			}

			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&MockClient{})

			content := "func test() i32 {\n    return myGlobal\n}"
			OpenDocument(server, ctx, uri, content)

			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 12}, // myGl|obal
				},
				NewName: "renamed",
			}))
			Expect(result).To(BeNil())
		})

		It("should return nil when document not found", func() {
			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.arc"},
					Position:     protocol.Position{Line: 0, Character: 0},
				},
				NewName: "newName",
			}))
			Expect(result).To(BeNil())
		})

		It("should return nil for empty word", func() {
			content := `func test() {

}`
			OpenDocument(server, ctx, uri, content)

			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 0}, // Empty line
				},
				NewName: "newName",
			}))
			Expect(result).To(BeNil())
		})
	})
})
