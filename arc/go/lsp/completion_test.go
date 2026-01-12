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
	"github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
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
		server = MustSucceed(lsp.New())
		server.SetClient(&testutil.MockClient{})
		uri = "file:///test.arc"
	})

	Describe("Basic Completion", func() {
		It("should return built-in completions", func() {
			content := "func test() {\n    i\n}"
			testutil.OpenDocument(server, ctx, uri, content)

			// Request completion at "i|" - should match i8, i16, i32, i64, if
			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 5}, // after "i"
				},
			}))
			Expect(completions).ToNot(BeNil())
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
			server = MustSucceed(lsp.New(lsp.Config{GlobalResolver: globalResolver}))
			server.SetClient(&testutil.MockClient{})

			// Use the same pattern as hover test - valid Arc code
			content := "func test() i32 {\n    return myGlobal\n}"
			testutil.OpenDocument(server, ctx, uri, content)

			// Request completion in the middle of typing "myGlobal" -> "myG|"
			// Simulating user typing "myG" and requesting completion
			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 14}, // after "myG" in "return myGlobal"
				},
			}))
			Expect(completions).ToNot(BeNil())

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
			server.SetClient(&testutil.MockClient{})

			content := "func test() i32 {\n    return xyz\n}"
			testutil.OpenDocument(server, ctx, uri, content)

			// Request completion at "xyz|"
			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 14}, // xyz|
				},
			}))
			Expect(completions).ToNot(BeNil())

			// Check that myGlobal is NOT in the completion list (prefix doesn't match)
			_, found := lo.Find(completions.Items, func(item protocol.CompletionItem) bool {
				return item.Label == "myGlobal"
			})
			Expect(found).To(BeFalse(), "Expected NOT to find 'myGlobal' in completion items when prefix doesn't match")
		})
	})
})
