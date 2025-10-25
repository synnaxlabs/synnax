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

var _ = Describe("Completion", func() {
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

		server.SetClient(&mockClient{})
		uri = "file:///test.arc"
	})

	Describe("Basic Completion", func() {
		It("should return built-in completions", func() {
			content := "func test() {\n    i\n}"
			openDocument(server, ctx, uri, content)

			// Request completion at "i|" - should match i8, i16, i32, i64, if
			completions, err := server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 5}, // after "i"
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(completions).ToNot(BeNil())

			GinkgoWriter.Printf("Built-in completion items count: %d\n", len(completions.Items))
			for i, item := range completions.Items {
				GinkgoWriter.Printf("  [%d] %s (kind=%d)\n", i, item.Label, item.Kind)
			}

			// Should have at least i8, i16, i32, i64, if
			Expect(len(completions.Items)).To(BeNumerically(">", 0))
		})
	})

	Describe("GlobalResolver", func() {
		It("should include global variables from GlobalResolver in completion", func() {
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

			// Use the same pattern as hover test - valid Arc code
			content := "func test() i32 {\n    return myGlobal\n}"
			openDocument(server, ctx, uri, content)

			// Request completion in the middle of typing "myGlobal" -> "myG|"
			// Simulating user typing "myG" and requesting completion
			completions, err := server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 14}, // after "myG" in "return myGlobal"
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(completions).ToNot(BeNil())

			// Debug: print all completion items
			GinkgoWriter.Printf("Completion items count: %d\n", len(completions.Items))
			for i, item := range completions.Items {
				GinkgoWriter.Printf("  [%d] %s (kind=%d, detail=%s)\n", i, item.Label, item.Kind, item.Detail)
			}

			// Check that myGlobal is in the completion list
			found := false
			for _, item := range completions.Items {
				if item.Label == "myGlobal" {
					found = true
					Expect(item.Kind).To(Equal(protocol.CompletionItemKindVariable))
					Expect(item.Detail).To(Equal("i32"))
					break
				}
			}
			Expect(found).To(BeTrue(), "Expected to find 'myGlobal' in completion items")
		})

		It("should not show GlobalResolver symbols when prefix doesn't match", func() {
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

			content := "func test() i32 {\n    return xyz\n}"
			openDocument(server, ctx, uri, content)

			// Request completion at "xyz|"
			completions, err := server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 14}, // xyz|
				},
			})

			Expect(err).ToNot(HaveOccurred())
			Expect(completions).ToNot(BeNil())

			// Check that myGlobal is NOT in the completion list (prefix doesn't match)
			found := false
			for _, item := range completions.Items {
				if item.Label == "myGlobal" {
					found = true
					break
				}
			}
			Expect(found).To(BeFalse(), "Expected NOT to find 'myGlobal' in completion items when prefix doesn't match")
		})
	})
})
