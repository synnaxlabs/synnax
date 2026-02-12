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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var _ = Describe("SetupTestServer", func() {
	It("should create a server and URI with default config", func() {
		server, uri := SetupTestServer()
		Expect(server).ToNot(BeNil())
		Expect(uri).To(Equal(protocol.DocumentURI("file:///test.arc")))
	})

	It("should create a functional server that handles document operations", func() {
		server, uri := SetupTestServer()
		ctx := context.Background()
		OpenArcDocument(server, ctx, uri, "func test() {}")
		hover := Hover(server, ctx, uri, 0, 2)
		Expect(hover).ToNot(BeNil())
		Expect(hover.Contents.Value).To(ContainSubstring("func"))
	})

	It("should accept a custom GlobalResolver config", func() {
		resolver := symbol.MapResolver{
			"sensor": symbol.Symbol{
				Name: "sensor",
				Type: types.Chan(types.F32()),
				Kind: symbol.KindChannel,
				ID:   1,
			},
		}
		server, uri := SetupTestServer(lsp.Config{GlobalResolver: resolver})
		ctx := context.Background()
		OpenArcDocument(server, ctx, uri, "func test() { x := sensor }")
		completions := Completion(server, ctx, uri, 0, 24)
		Expect(completions).ToNot(BeNil())
		Expect(HasCompletion(completions.Items, "sensor")).To(BeTrue())
	})
})

var _ = Describe("SetupTestServerWithClient", func() {
	It("should return a server, URI, and a non-nil MockClient", func() {
		server, uri, client := SetupTestServerWithClient()
		Expect(server).ToNot(BeNil())
		Expect(uri).To(Equal(protocol.DocumentURI("file:///test.arc")))
		Expect(client).ToNot(BeNil())
	})

	It("should wire the client to receive diagnostics from server operations", func() {
		server, uri, client := SetupTestServerWithClient()
		ctx := context.Background()
		OpenArcDocument(server, ctx, uri, "func test() {\n\tx := undefined_var\n}")
		Expect(client.Diagnostics()).To(HaveLen(1))
		Expect(client.Diagnostics()[0].Message).To(ContainSubstring("undefined symbol"))
	})

	It("should accept a custom config and propagate diagnostics", func() {
		resolver := symbol.MapResolver{
			"sensor": symbol.Symbol{
				Name: "sensor",
				Type: types.Chan(types.F32()),
				Kind: symbol.KindChannel,
			},
		}
		server, uri, client := SetupTestServerWithClient(lsp.Config{GlobalResolver: resolver})
		ctx := context.Background()
		OpenArcDocument(server, ctx, uri, "func test() { x := sensor }")
		Expect(client.Diagnostics()).To(BeEmpty())
	})
})

var _ = Describe("OpenArcDocument", func() {
	It("should open a document that subsequent LSP operations can query", func() {
		server, uri := SetupTestServer()
		ctx := context.Background()
		OpenArcDocument(server, ctx, uri, "func hello() { return 42 }")
		hover := Hover(server, ctx, uri, 0, 2)
		Expect(hover).ToNot(BeNil())
		Expect(hover.Contents.Value).To(ContainSubstring("func"))
	})

	It("should allow opening multiple documents on the same server", func() {
		server, uri, client := SetupTestServerWithClient()
		ctx := context.Background()
		uri2 := protocol.DocumentURI("file:///second.arc")
		OpenArcDocument(server, ctx, uri, "func a() {}")
		OpenArcDocument(server, ctx, uri2, "func b() { x := undefined }")
		Expect(client.Diagnostics()).To(HaveLen(1))
		Expect(client.Diagnostics()[0].Message).To(ContainSubstring("undefined"))
	})
})

var _ = Describe("Hover", func() {
	It("should return hover information for a known keyword", func() {
		server, uri := SetupTestServer()
		ctx := context.Background()
		OpenArcDocument(server, ctx, uri, "func test() {}")
		hover := Hover(server, ctx, uri, 0, 2)
		Expect(hover).ToNot(BeNil())
		Expect(hover.Contents.Value).To(ContainSubstring("func"))
	})

	It("should return nil for an unknown position", func() {
		server, uri := SetupTestServer()
		ctx := context.Background()
		OpenArcDocument(server, ctx, uri, "func test() {}")
		Expect(Hover(server, ctx, uri, 10, 0)).To(BeNil())
	})

	It("should return hover for a type annotation", func() {
		server, uri := SetupTestServer()
		ctx := context.Background()
		OpenArcDocument(server, ctx, uri, "x i32 := 42")
		hover := Hover(server, ctx, uri, 0, 3)
		Expect(hover).ToNot(BeNil())
		Expect(hover.Contents.Value).To(ContainSubstring("i32"))
	})
})

var _ = Describe("Definition", func() {
	It("should return definition locations for a variable reference", func() {
		server := MustSucceed(lsp.New())
		server.SetClient(&MockClient{})
		uri := protocol.DocumentURI("file:///test.arc")
		ctx := context.Background()
		OpenArcDocument(server, ctx, uri, "func test() {\n    x i32 := 42\n    y := x + 1\n}")
		locations := Definition(server, ctx, uri, 2, 9)
		Expect(locations).To(HaveLen(1))
		Expect(locations[0].Range.Start.Line).To(Equal(uint32(1)))
	})

	It("should return nil for a non-existent document", func() {
		server := MustSucceed(lsp.New())
		server.SetClient(&MockClient{})
		ctx := context.Background()
		locations := Definition(server, ctx, "file:///missing.arc", 0, 0)
		Expect(locations).To(BeNil())
	})
})

var _ = Describe("Completion", func() {
	It("should return completion items for a partial identifier", func() {
		server, uri := SetupTestServer()
		ctx := context.Background()
		OpenArcDocument(server, ctx, uri, "func test() {\n    i\n}")
		completions := Completion(server, ctx, uri, 1, 5)
		Expect(completions).ToNot(BeNil())
		Expect(len(completions.Items)).To(BeNumerically(">", 0))
	})

	It("should return completions including global resolver symbols", func() {
		resolver := symbol.MapResolver{
			"pressure": symbol.Symbol{
				Name: "pressure",
				Type: types.Chan(types.F64()),
				Kind: symbol.KindChannel,
				ID:   1,
			},
		}
		server, uri := SetupTestServer(lsp.Config{GlobalResolver: resolver})
		ctx := context.Background()
		OpenArcDocument(server, ctx, uri, "func test() { x := pres }")
		completions := Completion(server, ctx, uri, 0, 24)
		Expect(completions).ToNot(BeNil())
		Expect(HasCompletion(completions.Items, "pressure")).To(BeTrue())
	})
})

var _ = Describe("SemanticTokens", func() {
	It("should return semantic tokens for a document", func() {
		server, uri := SetupTestServer()
		ctx := context.Background()
		OpenArcDocument(server, ctx, uri, "func test() {}")
		tokens := SemanticTokens(server, ctx, uri)
		Expect(tokens).ToNot(BeNil())
		Expect(len(tokens.Data)).To(BeNumerically(">=", 5))
	})

	It("should return tokens with correct encoding", func() {
		server, uri := SetupTestServer()
		ctx := context.Background()
		OpenArcDocument(server, ctx, uri, "x := 42")
		tokens := SemanticTokens(server, ctx, uri)
		Expect(tokens).ToNot(BeNil())
		Expect(len(tokens.Data) % 5).To(Equal(0))
	})
})
