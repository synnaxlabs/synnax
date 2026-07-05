// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package testutil_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/uri"
)

var _ = Describe("SetupTestServer", func() {
	It("should create a server and URI with default config", func(ctx SpecContext) {
		server, docURI := SetupTestServer()
		Expect(server).ToNot(BeNil())
		Expect(docURI).To(Equal(uri.URI("file:///test.arc")))
	})

	It("should create a functional server that handles document operations", func(ctx SpecContext) {
		server, docURI := SetupTestServer()
		OpenArcDocument(server, ctx, docURI, "func test() {}")
		hover := Hover(server, ctx, docURI, 0, 2)
		Expect(hover).ToNot(BeNil())
		Expect(HoverContents(hover)).To(ContainSubstring("func"))
	})

	It("should expose custom symbols attached to the ambient prelude", func(ctx SpecContext) {
		server, docURI := SetupTestServer(lsp.Config{NewRoot: func() *symbol.Symbol {
			return NewRoot(nil, symbol.Symbol{
				Name: "sensor",
				Type: types.Chan(types.F32()),
				Kind: symbol.KindChannel,
				ID:   1,
			})
		}})
		OpenArcDocument(server, ctx, docURI, "func test() { x := sensor }")
		completions := Completion(server, ctx, docURI, 0, 24)
		Expect(completions).ToNot(BeNil())
		Expect(HasCompletion(completions.Items, "sensor")).To(BeTrue())
	})
})

var _ = Describe("SetupTestServerWithClient", func() {
	It("should return a server, URI, and a non-nil MockClient", func(ctx SpecContext) {
		server, docURI, client := SetupTestServerWithClient()
		Expect(server).ToNot(BeNil())
		Expect(docURI).To(Equal(uri.URI("file:///test.arc")))
		Expect(client).ToNot(BeNil())
	})

	It("should wire the client to receive diagnostics from server operations", func(ctx SpecContext) {
		server, docURI, client := SetupTestServerWithClient()
		OpenArcDocument(server, ctx, docURI, "func test() {\n\tx := undefined_var\n}")
		Expect(client.Diagnostics()).To(HaveLen(1))
		Expect(DiagnosticMessage(client.Diagnostics()[0])).To(ContainSubstring("undefined symbol"))
	})

	It("should accept a custom config and propagate diagnostics", func(ctx SpecContext) {
		server, docURI, client := SetupTestServerWithClient(lsp.Config{NewRoot: func() *symbol.Symbol {
			return NewRoot(nil, symbol.Symbol{
				Name: "sensor",
				Type: types.Chan(types.F32()),
				Kind: symbol.KindChannel,
			})
		}})
		OpenArcDocument(server, ctx, docURI, "func test() { x := sensor }")
		Expect(client.Diagnostics()).To(BeEmpty())
	})
})

var _ = Describe("OpenArcDocument", func() {
	It("should open a document that subsequent LSP operations can query", func(ctx SpecContext) {
		server, docURI := SetupTestServer()
		OpenArcDocument(server, ctx, docURI, "func hello() { return 42 }")
		hover := Hover(server, ctx, docURI, 0, 2)
		Expect(hover).ToNot(BeNil())
		Expect(HoverContents(hover)).To(ContainSubstring("func"))
	})

	It("should allow opening multiple documents on the same server", func(ctx SpecContext) {
		server, docURI, client := SetupTestServerWithClient()
		docURI2 := uri.URI("file:///second.arc")
		OpenArcDocument(server, ctx, docURI, "func a() {}")
		OpenArcDocument(server, ctx, docURI2, "func b() { x := undefined }")
		Expect(client.Diagnostics()).To(HaveLen(1))
		Expect(DiagnosticMessage(client.Diagnostics()[0])).To(ContainSubstring("undefined"))
	})
})

var _ = Describe("Hover", func() {
	It("should return hover information for a known keyword", func(ctx SpecContext) {
		server, docURI := SetupTestServer()
		OpenArcDocument(server, ctx, docURI, "func test() {}")
		hover := Hover(server, ctx, docURI, 0, 2)
		Expect(hover).ToNot(BeNil())
		Expect(HoverContents(hover)).To(ContainSubstring("func"))
	})

	It("should return nil for an unknown position", func(ctx SpecContext) {
		server, docURI := SetupTestServer()
		OpenArcDocument(server, ctx, docURI, "func test() {}")
		Expect(Hover(server, ctx, docURI, 10, 0)).To(BeNil())
	})

	It("should return hover for a type annotation", func(ctx SpecContext) {
		server, docURI := SetupTestServer()
		OpenArcDocument(server, ctx, docURI, "x i32 := 42")
		hover := Hover(server, ctx, docURI, 0, 3)
		Expect(hover).ToNot(BeNil())
		Expect(HoverContents(hover)).To(ContainSubstring("i32"))
	})
})

var _ = Describe("Definition", func() {
	It("should return definition locations for a variable reference", func(ctx SpecContext) {
		server := MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
		server.SetClient(&MockClient{})
		docURI := uri.URI("file:///test.arc")
		OpenArcDocument(server, ctx, docURI, "func test() {\n    x i32 := 42\n    y := x + 1\n}")
		locations := Definition(server, ctx, docURI, 2, 9)
		Expect(locations).To(HaveLen(1))
		Expect(locations[0].Range.Start.Line).To(Equal(uint32(1)))
	})

	It("should return nil for a non-existent document", func(ctx SpecContext) {
		server := MustSucceed(lsp.New(lsp.Config{NewRoot: func() *symbol.Symbol { return NewRoot(nil) }}))
		server.SetClient(&MockClient{})
		locations := Definition(server, ctx, "file:///missing.arc", 0, 0)
		Expect(locations).To(BeNil())
	})
})

var _ = Describe("Completion", func() {
	It("should return completion items for a partial identifier", func(ctx SpecContext) {
		server, docURI := SetupTestServer()
		OpenArcDocument(server, ctx, docURI, "func test() {\n    i\n}")
		completions := Completion(server, ctx, docURI, 1, 5)
		Expect(completions).ToNot(BeNil())
		Expect(completions.Items).ToNot(BeEmpty())
	})

	It("should return completions including symbols attached to the ambient prelude", func(ctx SpecContext) {
		server, docURI := SetupTestServer(lsp.Config{NewRoot: func() *symbol.Symbol {
			return NewRoot(nil, symbol.Symbol{
				Name: "pressure",
				Type: types.Chan(types.F64()),
				Kind: symbol.KindChannel,
				ID:   1,
			})
		}})
		OpenArcDocument(server, ctx, docURI, "func test() { x := pres }")
		completions := Completion(server, ctx, docURI, 0, 24)
		Expect(completions).ToNot(BeNil())
		Expect(HasCompletion(completions.Items, "pressure")).To(BeTrue())
	})
})

var _ = Describe("SemanticTokens", func() {
	It("should return semantic tokens for a document", func(ctx SpecContext) {
		server, docURI := SetupTestServer()
		OpenArcDocument(server, ctx, docURI, "func test() {}")
		tokens := SemanticTokens(server, ctx, docURI)
		Expect(tokens).ToNot(BeNil())
		Expect(len(tokens.Data)).To(BeNumerically(">=", 5))
	})

	It("should return tokens with correct encoding", func(ctx SpecContext) {
		server, docURI := SetupTestServer()
		OpenArcDocument(server, ctx, docURI, "x := 42")
		tokens := SemanticTokens(server, ctx, docURI)
		Expect(tokens).ToNot(BeNil())
		Expect(len(tokens.Data) % 5).To(Equal(0))
	})
})
