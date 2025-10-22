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

		// Create mock channels in GlobalResolver (simulating Synnax channels)
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

		// Create server with GlobalResolver
		var err error
		server, err = lsp.New(lsp.Config{GlobalResolver: globalResolver})
		Expect(err).ToNot(HaveOccurred())
		server.SetClient(&mockClient{})
	})

	Describe("Hover", func() {
		It("Should provide hover for channel from GlobalResolver in block expression", func() {
			uri := generateBlockURI("hover-test-1")
			content := "return sensor * 1.8 + 32"

			// Open document
			err := server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Hover over "sensor" (position 7)
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 7},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			if hover == nil {
				GinkgoWriter.Printf("Hover returned nil for position (0, 7) in content: %q\n", content)
			} else {
				GinkgoWriter.Printf("Hover returned: %s\n", hover.Contents.Value)
			}
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("chan f32"))
		})

		It("Should handle multi-line block expression", func() {
			uri := generateBlockURI("hover-test-2")
			content := "let temp_f = temp_c * 1.8 + 32\nreturn temp_f"

			err := server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Hover over "temp_c" on line 0, position 13
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 13},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			Expect(hover.Contents.Value).To(ContainSubstring("chan f32"))
		})

		It("Should hover over local variables in block", func() {
			uri := generateBlockURI("hover-test-3")
			content := "let result = sensor * 2\nreturn result"

			err := server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Hover over "result" on line 1
			hover, err := server.Hover(ctx, &protocol.HoverParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 7},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(hover).ToNot(BeNil())
			// Should show the local variable type
			Expect(hover.Contents.Value).To(ContainSubstring("series<f32>"))
		})
	})

	Describe("Completion", func() {
		It("Should complete channel names from GlobalResolver", func() {
			uri := generateBlockURI("completion-test-1")
			content := "return sen"

			err := server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Request completion at "sen|"
			completions, err := server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 10}, // After "sen"
				},
			})

			Expect(err).ToNot(HaveOccurred())
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

			err := server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Request completion at "temp|"
			completions, err := server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 11}, // After "temp"
				},
			})

			Expect(err).ToNot(HaveOccurred())
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

			err := server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Request completion at "pres|" on line 1
			completions, err := server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 11}, // After "pres"
				},
			})

			Expect(err).ToNot(HaveOccurred())
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

			err := server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Request completion at "sens|" on line 1
			completions, err := server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 11}, // After "sens"
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(completions).ToNot(BeNil())

			// Should find both "sensor" (global) and "sensor_value" (local)
			foundSensor := false
			foundSensorValue := false
			for _, item := range completions.Items {
				if item.Label == "sensor" {
					foundSensor = true
				}
				if item.Label == "sensor_value" {
					foundSensorValue = true
				}
			}
			Expect(foundSensor).To(BeTrue(), "Expected to find 'sensor' from GlobalResolver")
			Expect(foundSensorValue).To(BeTrue(), "Expected to find 'sensor_value' local variable")
		})
	})

	Describe("Definition", func() {
		It("Should return nil for GlobalResolver symbols (no AST)", func() {
			uri := generateBlockURI("definition-test-1")
			content := "return sensor * 2"

			err := server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Go to definition on "sensor"
			locations, err := server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 7},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			// GlobalResolver symbols have no AST, so should return nil
			Expect(locations).To(BeNil())
		})

		It("Should find definition for local variables", func() {
			uri := generateBlockURI("definition-test-2")
			content := "let result = sensor * 2\nreturn result"

			err := server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       content,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Go to definition on "result" on line 1
			locations, err := server.Definition(ctx, &protocol.DefinitionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 7},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(locations).ToNot(BeNil())
			Expect(len(locations)).To(Equal(1))
			// Should point to line 0 where "result" is declared
			Expect(locations[0].Range.Start.Line).To(Equal(uint32(0)))
		})
	})

	Describe("DidChange", func() {
		It("Should update completions after content change", func() {
			uri := generateBlockURI("didchange-test-1")
			initialContent := "return sen"

			err := server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
				TextDocument: protocol.TextDocumentItem{
					URI:        uri,
					LanguageID: "arc",
					Version:    1,
					Text:       initialContent,
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Update content
			newContent := "return pres"
			err = server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
				TextDocument: protocol.VersionedTextDocumentIdentifier{
					TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
					Version:                2,
				},
				ContentChanges: []protocol.TextDocumentContentChangeEvent{
					{Text: newContent},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			// Request completion - should now complete "pressure"
			completions, err := server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 11},
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(completions).ToNot(BeNil())

			found := false
			for _, item := range completions.Items {
				if item.Label == "pressure" {
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Expected to find 'pressure' after content change")
		})
	})
})
