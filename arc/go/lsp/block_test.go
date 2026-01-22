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
	"github.com/samber/lo"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

// generateBlockURI creates a URI with block metadata
func generateBlockURI(id string) protocol.DocumentURI {
	// Base64 encode of {"is_block":true}: eyJpc19ibG9jayI6dHJ1ZX0=
	encoded := "eyJpc19ibG9jayI6dHJ1ZX0="
	return protocol.DocumentURI("arc://block/" + id + "#" + encoded)
}

var _ = Describe("Block Expressions with GlobalResolver", func() {
	var (
		server         *lsp.Server
		ctx            context.Context
		globalResolver symbol.MapResolver
	)

	BeforeEach(func() {
		ctx = context.Background()
		globalResolver = symbol.MapResolver{
			"sensor": symbol.Symbol{
				Name: "sensor",
				Type: types.Chan(types.F32()),
				Kind: symbol.KindChannel,
				ID:   1,
			},
			"temp_c": symbol.Symbol{
				Name: "temp_c",
				Type: types.Chan(types.F32()),
				Kind: symbol.KindChannel,
				ID:   2,
			},
			"pressure": symbol.Symbol{
				Name: "pressure",
				Type: types.Chan(types.F64()),
				Kind: symbol.KindChannel,
				ID:   3,
			},
		}
		server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
		server.SetClient(&MockClient{})
	})

	Describe("Hover", func() {
		It("Should provide hover for channel from GlobalResolver in block expression", func() {
			uri := generateBlockURI("hover-test-1")
			content := "return sensor * 1.8 + 32"
			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			// Hover over "sensor" (position 7)
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 7},
				},
			}))
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("chan f32"))
		})

		It("Should handle multi-line block expression", func() {
			uri := generateBlockURI("hover-test-2")
			content := "let temp_f = temp_c * 1.8 + 32\nreturn temp_f"
			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			// Hover over "temp_c" on line 0, position 13
			hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 13},
				},
			}))
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("chan f32"))
		})
	})

	Describe("Completion", func() {
		It("Should complete channel names from GlobalResolver", func() {
			uri := generateBlockURI("completion-test-1")
			content := "return sen"
			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			// Request completion at "sen|"
			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 10}, // After "sen"
				},
			}))
			Expect(completions).ToNot(BeNil())

			// Should find "sensor"
			found := false
			for _, item := range completions.Items {
				if item.Label == "sensor" {
					found = true
					Expect(item.Detail).To(Equal("chan f32"))
					break
				}
			}
			Expect(found).To(BeTrue(), "Expected to find 'sensor' in completion items")
		})

		It("Should complete with prefix matching", func() {
			uri := generateBlockURI("completion-test-2")
			content := "return temp"
			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			// Request completion at "temp|"
			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 11}, // After "temp"
				},
			}))
			Expect(completions).ToNot(BeNil())

			// Should find "temp_c"
			found := false
			for _, item := range completions.Items {
				if item.Label == "temp_c" {
					found = true
					Expect(item.Detail).To(Equal("chan f32"))
					break
				}
			}
			Expect(found).To(BeTrue(), "Expected to find 'temp_c' in completion items")
		})

		It("Should complete in multi-line block", func() {
			uri := generateBlockURI("completion-test-3")
			content := "let x = sensor\nreturn pres"

			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			// Request completion at "pres|" on line 1
			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 11}, // After "pres"
				},
			}))
			Expect(completions).ToNot(BeNil())

			// Should find "pressure"
			found := false
			for _, item := range completions.Items {
				if item.Label == "pressure" {
					found = true
					Expect(item.Detail).To(Equal("chan f64"))
					break
				}
			}
			Expect(found).To(BeTrue(), "Expected to find 'pressure' in completion items")
		})

		It("Should complete local variables and globals together", func() {
			uri := generateBlockURI("completion-test-4")
			content := "let sensor_value = sensor * 2\nreturn sens"
			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			// Request completion at "sens|" on line 1
			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 11}, // After "sens"
				},
			}))
			Expect(completions).ToNot(BeNil())

			foundSensor := false
			for _, item := range completions.Items {
				if item.Label == "sensor" {
					foundSensor = true
				}
			}
			Expect(foundSensor).To(BeTrue(), "Expected to find 'sensor' from GlobalResolver")
		})
	})

	Describe("Definition", func() {
		It("Should return nil for GlobalResolver symbols (no AST)", func() {
			uri := generateBlockURI("definition-test-1")
			content := "return sensor * 2"
			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			// Go to definition on "sensor"
			locations := MustSucceed(server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 7},
				},
			}))
			// GlobalResolver symbols have no AST, so should return nil
			Expect(locations).To(BeNil())
		})
	})

	Describe("DidChange", func() {
		It("Should update completions after content change", func() {
			uri := generateBlockURI("didchange-test-1")
			initialContent := "return sen"

			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       initialContent,
				},
			})).To(Succeed())

			// Update content
			newContent := "return pres"
			Expect(server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
				TextDocument: protocol.VersionedTextDocumentIdentifier{
					TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
					Version:                2,
				},
				ContentChanges: []protocol.TextDocumentContentChangeEvent{
					{Text: newContent},
				},
			})).To(Succeed())

			// Request completion - should now complete "pressure"
			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 11},
				},
			}))

			Expect(completions).ToNot(BeNil())

			_, found := lo.Find(completions.Items, func(item protocol.CompletionItem) bool {
				return item.Label == "pressure"
			})
			Expect(found).To(BeTrue(), "Expected to find 'pressure' after content change")
		})
	})

	Describe("Rename", func() {
		It("Should rename local variable in block expression", func() {
			uri := generateBlockURI("rename-test-1")
			content := "x := 10\nreturn x * 2 + x"
			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			edit := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 0},
				},
				NewName: "value",
			}))
			Expect(edit).ToNot(BeNil())
			Expect(edit.Changes).To(HaveKey(uri))
			Expect(edit.Changes[uri]).To(HaveLen(3))
			for _, textEdit := range edit.Changes[uri] {
				Expect(textEdit.NewText).To(Equal("value"))
			}
		})

		It("Should return nil when renaming GlobalResolver symbol in block", func() {
			uri := generateBlockURI("rename-test-2")
			content := "return sensor * 2"
			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			edit := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 7},
				},
				NewName: "newSensor",
			}))
			Expect(edit).To(BeNil())
		})
	})

	Describe("Definition", func() {
		It("Should return nil for GlobalResolver symbols (no AST)", func() {
			uri := generateBlockURI("definition-test-1")
			content := "return sensor * 2"
			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			// Go to definition on "sensor"
			locations := MustSucceed(server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 7},
				},
			}))
			// GlobalResolver symbols have no AST, so should return nil
			Expect(locations).To(BeNil())
		})

		It("Should find definition of local variable in block expression", func() {
			uri := generateBlockURI("definition-test-2")
			content := "x := 10\nreturn x * 2"
			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			locations := MustSucceed(server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 7},
				},
			}))
			Expect(locations).ToNot(BeNil())
			Expect(locations).To(HaveLen(1))
			Expect(locations[0].Range.Start.Line).To(Equal(uint32(0)))
			Expect(locations[0].Range.Start.Character).To(Equal(uint32(0)))
		})
	})

	Describe("Semantic Tokens", func() {
		It("Should return semantic tokens for block expression", func() {
			uri := generateBlockURI("semantic-test-1")
			content := "x := sensor * 2"
			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			tokens := MustSucceed(server.SemanticTokensFull(ctx, &protocol.SemanticTokensParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			}))
			Expect(tokens).ToNot(BeNil())
			Expect(len(tokens.Data)).To(BeNumerically(">=", 5*5))
		})
	})

	Describe("Multi-statement Completion", func() {
		It("Should provide completion for GlobalResolver symbols with prefix in multi-statement block", func() {
			uri := generateBlockURI("completion-test-5")
			content := "let x = sensor\nlet y = x * 2\nreturn y + sen"
			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 2, Character: 14},
				},
			}))
			Expect(completions).ToNot(BeNil())

			labels := lo.Map(completions.Items, func(item protocol.CompletionItem, _ int) string {
				return item.Label
			})
			Expect(labels).To(ContainElement("sensor"))
		})

		It("Should provide completion for both local and GlobalResolver symbols in multi-statement block", func() {
			uri := generateBlockURI("completion-test-6")
			content := "myVar := sensor\nmyOther := myVar * 2\nreturn my"
			Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})).To(Succeed())

			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 2, Character: 9},
				},
			}))
			Expect(completions).ToNot(BeNil())

			labels := lo.Map(completions.Items, func(item protocol.CompletionItem, _ int) string {
				return item.Label
			})
			Expect(labels).To(ContainElement("myVar"))
			Expect(labels).To(ContainElement("myOther"))
		})
	})
})
