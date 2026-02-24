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
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/lsp/testutil"
	"github.com/synnaxlabs/x/observe"
	"go.lsp.dev/protocol"
)

var _ = Describe("Server Diagnostics", func() {
	var (
		ctx    context.Context
		server *lsp.Server
		uri    protocol.DocumentURI
		client *MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		server, uri, client = SetupTestServerWithClient()
	})

	Describe("Diagnostic Range", func() {
		It("Should publish diagnostics with correct end position for undefined symbol", func() {
			OpenArcDocument(server, ctx, uri, "func test() {\n\tx := undefined_var\n}")

			Expect(client.Diagnostics()).To(HaveLen(1))
			diag := client.Diagnostics()[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: undefined_var"))
			Expect(diag.Range.Start.Line).To(Equal(uint32(1)))
			Expect(diag.Range.Start.Character).To(Equal(uint32(6)))
			Expect(diag.Range.End.Line).To(Equal(uint32(1)))
			Expect(diag.Range.End.Character).To(Equal(uint32(19)))
		})

		It("Should publish diagnostics with correct end position for short identifier", func() {
			OpenArcDocument(server, ctx, uri, "func test() {\n\tx := y\n}")

			Expect(client.Diagnostics()).To(HaveLen(1))
			diag := client.Diagnostics()[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: y"))
			Expect(diag.Range.Start.Line).To(Equal(uint32(1)))
			Expect(diag.Range.Start.Character).To(Equal(uint32(6)))
			Expect(diag.Range.End.Line).To(Equal(uint32(1)))
			Expect(diag.Range.End.Character).To(Equal(uint32(7)))
		})

		It("Should publish diagnostics with fallback end position when no stop token", func() {
			OpenArcDocument(server, ctx, uri, "func test() i32 {\n\tx := 1\n}")

			Expect(client.Diagnostics()).To(HaveLen(1))
			diag := client.Diagnostics()[0]
			Expect(diag.Message).To(ContainSubstring("must return"))
			Expect(diag.Range.End.Line).To(BeNumerically(">=", diag.Range.Start.Line))
			Expect(diag.Range.End.Character).To(BeNumerically(">=", diag.Range.Start.Character))
		})

		It("Should handle multiple diagnostics with correct ranges", func() {
			OpenArcDocument(server, ctx, uri, "func test() {\n\ta := undefined1\n\tb := undefined2\n}")

			Expect(client.Diagnostics()).To(HaveLen(2))

			diag1 := client.Diagnostics()[0]
			Expect(diag1.Message).To(ContainSubstring("undefined symbol: undefined1"))
			Expect(diag1.Range.Start.Line).To(Equal(uint32(1)))
			Expect(diag1.Range.End.Line).To(Equal(uint32(1)))
			Expect(diag1.Range.End.Character).To(Equal(uint32(16)))

			diag2 := client.Diagnostics()[1]
			Expect(diag2.Message).To(ContainSubstring("undefined symbol: undefined2"))
			Expect(diag2.Range.Start.Line).To(Equal(uint32(2)))
			Expect(diag2.Range.End.Line).To(Equal(uint32(2)))
			Expect(diag2.Range.End.Character).To(Equal(uint32(16)))
		})

		It("Should handle block URI diagnostics with correct ranges", func() {
			blockURI := protocol.DocumentURI("arc://block/test")
			OpenArcDocument(server, ctx, blockURI, "x := undefined_var")

			Expect(client.Diagnostics()).To(HaveLen(1))
			diag := client.Diagnostics()[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: undefined_var"))
			Expect(diag.Range.End.Character).To(BeNumerically(">", diag.Range.Start.Character))
		})
	})

	Describe("Diagnostic Severity", func() {
		It("Should set correct severity for errors", func() {
			OpenArcDocument(server, ctx, uri, "func test() {\n\tx := undefined\n}")

			Expect(client.Diagnostics()).To(HaveLen(1))
			Expect(client.Diagnostics()[0].Severity).To(Equal(protocol.DiagnosticSeverityError))
		})
	})

	Describe("Diagnostic Error Codes", func() {
		It("Should include error code for function argument count mismatch", func() {
			OpenArcDocument(server, ctx, uri, "func add(x i64, y i64) i64 { return x + y }\nfunc test() { z := add(1) }")

			Expect(client.Diagnostics()).To(HaveLen(1))
			Expect(client.Diagnostics()[0].Code).To(Equal("ARC3001"))
		})

		It("Should include error code for function argument type mismatch", func() {
			OpenArcDocument(server, ctx, uri, "func process(x i32) i32 { return x }\nfunc test() { z := process(\"hello\") }")

			Expect(client.Diagnostics()).To(HaveLen(1))
			Expect(client.Diagnostics()[0].Code).To(Equal("ARC3002"))
		})
	})

	Describe("Diagnostic Related Information", func() {
		It("Should include function signature in related information for argument errors", func() {
			OpenArcDocument(server, ctx, uri, "func add(x i64, y i64) i64 { return x + y }\nfunc test() { z := add(1) }")

			Expect(client.Diagnostics()).To(HaveLen(1))
			Expect(client.Diagnostics()[0].RelatedInformation).To(HaveLen(1))
			Expect(client.Diagnostics()[0].RelatedInformation[0].Message).To(ContainSubstring("add(x i64, y i64) i64"))
		})
	})
})

var _ = Describe("Debounced Diagnostics", func() {
	var (
		ctx    context.Context
		server *lsp.Server
		uri    protocol.DocumentURI
		client *MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		server, uri, client = SetupTestServerWithClient(lsp.Config{
			DebounceDelay:    20 * time.Millisecond,
			MaxDebounceDelay: 200 * time.Millisecond,
		})
	})

	It("Should publish diagnostics after debounce delay", func() {
		OpenArcDocument(server, ctx, uri, "func test() {}")
		baseline := client.PublishCount()

		ChangeDocument(server, ctx, uri, "func test() {\n\tx := undefined\n}", 2)

		Expect(client.WaitForDiagnostics(baseline, 500*time.Millisecond)).To(BeTrue())
		Expect(client.Diagnostics()).To(HaveLen(1))
		Expect(client.Diagnostics()[0].Message).To(ContainSubstring("undefined symbol"))
	})

	It("Should coalesce rapid changes into a single publish", func() {
		OpenArcDocument(server, ctx, uri, "func test() {}")
		baseline := client.PublishCount()

		for i := 2; i <= 6; i++ {
			ChangeDocument(server, ctx, uri, "func test() {\n\tx := undefined\n}", int32(i))
		}

		Expect(client.WaitForDiagnostics(baseline, 500*time.Millisecond)).To(BeTrue())
		time.Sleep(50 * time.Millisecond)
		// Should have far fewer publishes than changes
		Expect(client.PublishCount() - baseline).To(BeNumerically("<=", 2))
	})

	It("Should force-flush on DidSave", func() {
		OpenArcDocument(server, ctx, uri, "func test() {}")
		baseline := client.PublishCount()

		ChangeDocument(server, ctx, uri, "func test() {\n\tx := undefined\n}", 2)
		// Immediately save - should flush without waiting for debounce
		Expect(server.DidSave(ctx, &protocol.DidSaveTextDocumentParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
		})).To(Succeed())

		Expect(client.PublishCount()).To(BeNumerically(">", baseline))
		Expect(client.Diagnostics()).To(HaveLen(1))
	})

	It("Should refresh semantic tokens after debounced analysis", func() {
		OpenArcDocument(server, ctx, uri, "func test() {}")
		baseline := client.SemanticRefreshCount()

		ChangeDocument(server, ctx, uri, "func dog() {}", 2)

		Expect(client.WaitForSemanticRefresh(baseline, 500*time.Millisecond)).To(BeTrue())
	})

	It("Should cancel stale analysis when new change arrives", func() {
		OpenArcDocument(server, ctx, uri, "func test() {}")
		baseline := client.PublishCount()

		// Send invalid code, then quickly send valid code
		ChangeDocument(server, ctx, uri, "func test() {\n\tx := undefined\n}", 2)
		ChangeDocument(server, ctx, uri, "func test() {\n\tx := 42\n}", 3)

		Expect(client.WaitForDiagnostics(baseline, 500*time.Millisecond)).To(BeTrue())
		time.Sleep(50 * time.Millisecond)
		// The final diagnostics should be clean (from the valid code)
		Expect(client.Diagnostics()).To(BeEmpty())
	})
})

var _ = Describe("Incremental Sync", func() {
	var (
		ctx    context.Context
		server *lsp.Server
		uri    protocol.DocumentURI
		client *MockClient
	)

	BeforeEach(func() {
		ctx = context.Background()
		server, uri, client = SetupTestServerWithClient(lsp.Config{
			DebounceDelay:    5 * time.Millisecond,
			MaxDebounceDelay: 50 * time.Millisecond,
		})
	})

	It("Should apply incremental changes correctly", func() {
		OpenArcDocument(server, ctx, uri, "func test() {\n\tx := 42\n}")
		baseline := client.PublishCount()

		// Send an incremental change: replace "42" with "undefined"
		// In "\tx := 42", tab=0, x=1, ' '=2, :=3, ==4, ' '=5, 4=6, 2=7
		Expect(server.DidChange(ctx, &protocol.DidChangeTextDocumentParams{
			TextDocument: protocol.VersionedTextDocumentIdentifier{
				TextDocumentIdentifier: protocol.TextDocumentIdentifier{URI: uri},
				Version:                2,
			},
			ContentChanges: []protocol.TextDocumentContentChangeEvent{
				{
					Range: protocol.Range{
						Start: protocol.Position{Line: 1, Character: 6},
						End:   protocol.Position{Line: 1, Character: 8},
					},
					Text: "undefined",
				},
			},
		})).To(Succeed())

		Expect(client.WaitForDiagnostics(baseline, 500*time.Millisecond)).To(BeTrue())
		Expect(client.Diagnostics()).To(HaveLen(1))
		Expect(client.Diagnostics()[0].Message).To(ContainSubstring("undefined symbol"))
	})
})

var _ = Describe("External Change Notifications", func() {
	var (
		ctx      context.Context
		server   *lsp.Server
		uri      protocol.DocumentURI
		client   *MockClient
		resolver symbol.MapResolver
		observer observe.Observer[struct{}]
	)

	BeforeEach(func() {
		ctx = context.Background()
		resolver = make(symbol.MapResolver)
		observer = observe.New[struct{}]()
		server, uri, client = SetupTestServerWithClient(lsp.Config{
			GlobalResolver:   resolver,
			OnExternalChange: observer,
		})
	})

	It("Should republish diagnostics when external state changes", func() {
		OpenArcDocument(server, ctx, uri, "func test() {\n\tx := my_channel\n}")
		Expect(client.Diagnostics()).To(HaveLen(1))
		Expect(client.Diagnostics()[0].Message).To(ContainSubstring("undefined symbol: my_channel"))
		resolver["my_channel"] = symbol.Symbol{
			Name: "my_channel",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F32()),
		}
		observer.Notify(ctx, struct{}{})
		Eventually(func() []protocol.Diagnostic { return client.Diagnostics() }).Should(BeEmpty())
	})

	It("Should refresh semantic tokens when external state changes", func() {
		OpenArcDocument(server, ctx, uri, "func test() {\n\tx := my_channel\n}")
		baseline := client.SemanticRefreshCount()

		resolver["my_channel"] = symbol.Symbol{
			Name: "my_channel",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F32()),
		}
		observer.Notify(ctx, struct{}{})

		Eventually(func() int { return client.SemanticRefreshCount() }).Should(BeNumerically(">", baseline))
	})

	It("Should show errors when a previously valid symbol is removed", func() {
		resolver["sensor"] = symbol.Symbol{
			Name: "sensor",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
		}
		OpenArcDocument(server, ctx, uri, "func test() {\n\tx := sensor\n}")
		Expect(client.Diagnostics()).To(BeEmpty())
		delete(resolver, "sensor")
		observer.Notify(ctx, struct{}{})
		Eventually(func() int { return len(client.Diagnostics()) }).Should(Equal(1))
		Expect(client.Diagnostics()[0].Message).To(ContainSubstring("undefined symbol: sensor"))
	})

	It("Should republish diagnostics for multiple open documents", func() {
		uri2 := protocol.DocumentURI("file:///test2.arc")
		OpenArcDocument(server, ctx, uri, "func test1() {\n\tx := channel_a\n}")
		OpenArcDocument(server, ctx, uri2, "func test2() {\n\ty := channel_b\n}")
		Expect(client.Diagnostics()).To(HaveLen(1))
		resolver["channel_a"] = symbol.Symbol{
			Name: "channel_a",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.I32()),
		}
		resolver["channel_b"] = symbol.Symbol{
			Name: "channel_b",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.I64()),
		}
		observer.Notify(ctx, struct{}{})
		Eventually(func() []protocol.Diagnostic { return client.Diagnostics() }).Should(BeEmpty())
	})
})
