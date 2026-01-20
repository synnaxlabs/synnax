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

			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 5},
				},
			}))
			Expect(completions).ToNot(BeNil())
			Expect(len(completions.Items)).To(BeNumerically(">", 0))
		})
	})

	Describe("Context-Aware Completion", func() {
		It("should return empty completions in single-line comment", func() {
			content := "// comment here"
			testutil.OpenDocument(server, ctx, uri, content)

			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 10},
				},
			}))
			Expect(completions).ToNot(BeNil())
			Expect(completions.Items).To(BeEmpty())
		})

		It("should return empty completions in multi-line comment", func() {
			content := "/* multi\nline\ncomment */"
			testutil.OpenDocument(server, ctx, uri, content)

			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 1, Character: 2},
				},
			}))
			Expect(completions).ToNot(BeNil())
			Expect(completions.Items).To(BeEmpty())
		})

		It("should return only types in type annotation position", func() {
			content := "func foo(x "
			testutil.OpenDocument(server, ctx, uri, content)

			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 11},
				},
			}))
			Expect(completions).ToNot(BeNil())
			Expect(len(completions.Items)).To(BeNumerically(">", 0))

			for _, item := range completions.Items {
				Expect(item.Kind).To(Equal(protocol.CompletionItemKindClass),
					"Expected only type completions, got: %s (kind: %v)", item.Label, item.Kind)
			}

			_, foundFunc := lo.Find(completions.Items, func(item protocol.CompletionItem) bool {
				return item.Label == "func"
			})
			Expect(foundFunc).To(BeFalse(), "Should not show 'func' keyword in type annotation context")

			_, foundIf := lo.Find(completions.Items, func(item protocol.CompletionItem) bool {
				return item.Label == "if"
			})
			Expect(foundIf).To(BeFalse(), "Should not show 'if' keyword in type annotation context")
		})

		It("should return types matching prefix in type annotation position", func() {
			content := "func foo(x i"
			testutil.OpenDocument(server, ctx, uri, content)

			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 12},
				},
			}))
			Expect(completions).ToNot(BeNil())
			Expect(len(completions.Items)).To(BeNumerically(">", 0))

			for _, item := range completions.Items {
				Expect(item.Label).To(HavePrefix("i"), "Expected items with 'i' prefix, got: %s", item.Label)
			}
		})

		It("should not show keywords in expression context", func() {
			content := "x := "
			testutil.OpenDocument(server, ctx, uri, content)

			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 5},
				},
			}))
			Expect(completions).ToNot(BeNil())

			_, foundFunc := lo.Find(completions.Items, func(item protocol.CompletionItem) bool {
				return item.Label == "func"
			})
			Expect(foundFunc).To(BeFalse(), "Should not show 'func' keyword in expression context")

			_, foundIf := lo.Find(completions.Items, func(item protocol.CompletionItem) bool {
				return item.Label == "if"
			})
			Expect(foundIf).To(BeFalse(), "Should not show 'if' keyword in expression context")
		})

		It("should show functions and values in expression context", func() {
			content := "x := "
			testutil.OpenDocument(server, ctx, uri, content)

			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 5},
				},
			}))
			Expect(completions).ToNot(BeNil())

			_, foundLen := lo.Find(completions.Items, func(item protocol.CompletionItem) bool {
				return item.Label == "len"
			})
			Expect(foundLen).To(BeTrue(), "Should show 'len' function in expression context")

			_, foundNow := lo.Find(completions.Items, func(item protocol.CompletionItem) bool {
				return item.Label == "now"
			})
			Expect(foundNow).To(BeTrue(), "Should show 'now' function in expression context")
		})

		It("should show keywords at statement start", func() {
			content := "func foo() { "
			testutil.OpenDocument(server, ctx, uri, content)

			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 13},
				},
			}))
			Expect(completions).ToNot(BeNil())

			_, foundIf := lo.Find(completions.Items, func(item protocol.CompletionItem) bool {
				return item.Label == "if"
			})
			Expect(foundIf).To(BeTrue(), "Should show 'if' keyword at statement start")

			_, foundReturn := lo.Find(completions.Items, func(item protocol.CompletionItem) bool {
				return item.Label == "return"
			})
			Expect(foundReturn).To(BeTrue(), "Should show 'return' keyword at statement start")
		})

		It("should not show types at statement start", func() {
			content := "func foo() { "
			testutil.OpenDocument(server, ctx, uri, content)

			completions := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
				TextDocumentPositionParams: protocol.TextDocumentPositionParams{
					TextDocument: protocol.TextDocumentIdentifier{URI: uri},
					Position:     protocol.Position{Line: 0, Character: 13},
				},
			}))
			Expect(completions).ToNot(BeNil())

			_, foundI32 := lo.Find(completions.Items, func(item protocol.CompletionItem) bool {
				return item.Label == "i32"
			})
			Expect(foundI32).To(BeFalse(), "Should not show 'i32' type at statement start")
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
