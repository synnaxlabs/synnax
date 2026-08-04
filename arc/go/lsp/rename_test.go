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
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	"github.com/synnaxlabs/x/errors"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

var _ = Describe("Rename", func() {
	var (
		server *lsp.Server
		uri    uri.URI
	)

	BeforeEach(func() {
		server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol {
			return NewRoot(nil)
		}}))
		server.SetClient(&MockClient{})
		uri = "file:///test.arc"
	})

	Describe("PrepareRename", func() {
		It("should return range for renameable variable", func(ctx SpecContext) {
			content := `func test() {
    x i32 := 42
    y := x + 10
}`
			OpenArcDocument(server, ctx, uri, content)

			result := MustSucceed(server.PrepareRename(ctx, &protocol.PrepareRenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 4}, // x| i32
				},
			}))
			Expect(result).ToNot(BeNil())
			r, ok := result.(*protocol.Range)
			Expect(ok).To(BeTrue(), "expected a *protocol.Range, got %T", result)
			Expect(r.Start.Line).To(Equal(uint32(1)))
			Expect(r.Start.Character).To(Equal(uint32(4)))
		})

		It("should return nil for keywords", func(ctx SpecContext) {
			content := `func test() {
    return 42
}`
			OpenArcDocument(server, ctx, uri, content)

			result := MustSucceed(server.PrepareRename(ctx, &protocol.PrepareRenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 5}, // ret|urn
				},
			}))
			Expect(result).To(BeNil())
		})

		It("should return nil for global/builtin symbols", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol {
				return NewRoot(nil, symbol.Symbol{Name: "myGlobal",
					Type: types.I32(),
					Kind: symbol.KindVariable})
			}}))
			server.SetClient(&MockClient{})

			content := "func test() i32 {\n    return myGlobal\n}"
			OpenArcDocument(server, ctx, uri, content)

			result := MustSucceed(server.PrepareRename(ctx, &protocol.PrepareRenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 12}, // myGl|obal
				},
			}))
			Expect(result).To(BeNil())
		})

		It("should return nil when document not found", func(ctx SpecContext) {
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
		It("should rename variable with multiple references", func(ctx SpecContext) {
			content := `func test() {
    x i32 := 42
    y := x + 10
    z := x * 2
}`
			OpenArcDocument(server, ctx, uri, content)

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

		It("should rename function parameter", func(ctx SpecContext) {
			content := `func multiply(x f64, y f64) f64 {
    return x * y
}`
			OpenArcDocument(server, ctx, uri, content)

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

		It("should not rename variables across different functions", func(ctx SpecContext) {
			content := `func first() {
    x := 10
    y := x + 1
}

func second() {
    x := 20
    z := x + 2
}`
			OpenArcDocument(server, ctx, uri, content)

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

		It("should rename stateful variable", func(ctx SpecContext) {
			content := `func counter{} () u32 {
    count u32 $= 0
    count = count + 1
    return count
}`
			OpenArcDocument(server, ctx, uri, content)

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

		It("should rename function", func(ctx SpecContext) {
			content := `func add(x i32, y i32) i32 {
    return x + y
}

func main() {
    result := add(1, 2)
}`
			OpenArcDocument(server, ctx, uri, content)

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

		It("should return nil for global/builtin symbols", func(ctx SpecContext) {
			server = MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol {
				return NewRoot(nil, symbol.Symbol{Name: "myGlobal",
					Type: types.I32(),
					Kind: symbol.KindVariable})
			}}))
			server.SetClient(&MockClient{})

			content := "func test() i32 {\n    return myGlobal\n}"
			OpenArcDocument(server, ctx, uri, content)

			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 12}, // myGl|obal
				},
				NewName: "renamed",
			}))
			Expect(result).To(BeNil())
		})

		It("should return nil when document not found", func(ctx SpecContext) {
			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: "file:///nonexistent.arc"},
					Position:     protocol.Position{Line: 0, Character: 0},
				},
				NewName: "newName",
			}))
			Expect(result).To(BeNil())
		})

		It("should return nil for empty word", func(ctx SpecContext) {
			content := `func test() {

}`
			OpenArcDocument(server, ctx, uri, content)

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

	Describe("OnRename", func() {
		channelSymbols := []symbol.Symbol{
			{
				Name:       "sensor",
				Type:       types.Chan(types.F32()),
				Kind:       symbol.KindChannel,
				ID:         42,
				Renameable: true,
			},
			{
				Name: "internal_sensor",
				Type: types.Chan(types.F32()),
				Kind: symbol.KindChannel,
				ID:   43,
			},
		}
		newRootWithChannels := func() *symbol.Symbol {
			return NewRoot(nil, channelSymbols...)
		}
		channelContent := `func test() {
    x f32 := sensor + 1.0
    y := sensor * 2.0
}`

		It("should invoke OnRename with the resolved symbol and names", func(ctx SpecContext) {
			var (
				gotSym     *symbol.Symbol
				gotOldName string
				gotNewName string
				callCount  int
			)
			onRename := func(_ context.Context, sym *symbol.Symbol, oldName, newName string) error {
				callCount++
				gotSym = sym
				gotOldName = oldName
				gotNewName = newName
				return nil
			}
			server = MustSucceed(lsp.New(lsp.Config{
				NewRoot:  newRootWithChannels,
				OnRename: onRename,
			}))
			server.SetClient(&MockClient{})
			OpenArcDocument(server, ctx, uri, channelContent)

			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 14}, // sensor
				},
				NewName: "renamed_sensor",
			}))
			Expect(result).ToNot(BeNil())
			Expect(callCount).To(Equal(1))
			Expect(gotSym).ToNot(BeNil())
			Expect(gotSym.Kind).To(Equal(symbol.KindChannel))
			Expect(gotSym.ID).To(Equal(42))
			Expect(gotOldName).To(Equal("sensor"))
			Expect(gotNewName).To(Equal("renamed_sensor"))
		})

		It("should produce text edits for every channel reference in the document", func(ctx SpecContext) {
			onRename := func(context.Context, *symbol.Symbol, string, string) error { return nil }
			server = MustSucceed(lsp.New(lsp.Config{
				NewRoot:  newRootWithChannels,
				OnRename: onRename,
			}))
			server.SetClient(&MockClient{})
			OpenArcDocument(server, ctx, uri, channelContent)

			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 14},
				},
				NewName: "renamed_sensor",
			}))
			Expect(result).ToNot(BeNil())
			Expect(result.Changes).To(HaveKey(uri))
			edits := result.Changes[uri]
			Expect(edits).To(HaveLen(2))
			for _, edit := range edits {
				Expect(edit.NewText).To(Equal("renamed_sensor"))
			}
		})

		It("should abort the rename when OnRename returns an error", func(ctx SpecContext) {
			renameErr := errors.New("channel rename rejected")
			onRename := func(context.Context, *symbol.Symbol, string, string) error {
				return renameErr
			}
			server = MustSucceed(lsp.New(lsp.Config{
				NewRoot:  newRootWithChannels,
				OnRename: onRename,
			}))
			server.SetClient(&MockClient{})
			OpenArcDocument(server, ctx, uri, channelContent)

			Expect(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 14},
				},
				NewName: "renamed_sensor",
			})).Error().To(MatchError(renameErr))
		})

		It("should invoke OnRename for source-defined symbols too", func(ctx SpecContext) {
			callCount := 0
			onRename := func(_ context.Context, _ *symbol.Symbol, _, _ string) error {
				callCount++
				return nil
			}
			server = MustSucceed(lsp.New(lsp.Config{
				NewRoot:  func() *symbol.Symbol { return NewRoot(nil) },
				OnRename: onRename,
			}))
			server.SetClient(&MockClient{})
			OpenArcDocument(server, ctx, uri, `func test() {
    x i32 := 42
    y := x + 10
}`)

			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 4}, // x
				},
				NewName: "value",
			}))
			Expect(result).ToNot(BeNil())
			Expect(callCount).To(Equal(1))
		})

		It("should reject rename on symbols not marked Renameable", func(ctx SpecContext) {
			onRename := func(context.Context, *symbol.Symbol, string, string) error { return nil }
			server = MustSucceed(lsp.New(lsp.Config{
				NewRoot:  newRootWithChannels,
				OnRename: onRename,
			}))
			server.SetClient(&MockClient{})
			OpenArcDocument(server, ctx, uri, `func test() {
    x f32 := internal_sensor + 1.0
}`)

			prepared := MustSucceed(server.PrepareRename(ctx, &protocol.PrepareRenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 14}, // internal_sensor
				},
			}))
			Expect(prepared).To(BeNil())

			result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 14},
				},
				NewName: "renamed",
			}))
			Expect(result).To(BeNil())
		})
	})
})
