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
			OpenDocument(server, ctx, uri, "func test() {\n\tx := undefined_var\n}")

			Expect(client.Diagnostics).To(HaveLen(1))
			diag := client.Diagnostics[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: undefined_var"))
			Expect(diag.Range.Start.Line).To(Equal(uint32(1)))
			Expect(diag.Range.Start.Character).To(Equal(uint32(6)))
			Expect(diag.Range.End.Line).To(Equal(uint32(1)))
			Expect(diag.Range.End.Character).To(Equal(uint32(19)))
		})

		It("Should publish diagnostics with correct end position for short identifier", func() {
			OpenDocument(server, ctx, uri, "func test() {\n\tx := y\n}")

			Expect(client.Diagnostics).To(HaveLen(1))
			diag := client.Diagnostics[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: y"))
			Expect(diag.Range.Start.Line).To(Equal(uint32(1)))
			Expect(diag.Range.Start.Character).To(Equal(uint32(6)))
			Expect(diag.Range.End.Line).To(Equal(uint32(1)))
			Expect(diag.Range.End.Character).To(Equal(uint32(7)))
		})

		It("Should publish diagnostics with fallback end position when no stop token", func() {
			OpenDocument(server, ctx, uri, "func test() i32 {\n\tx := 1\n}")

			Expect(client.Diagnostics).To(HaveLen(1))
			diag := client.Diagnostics[0]
			Expect(diag.Message).To(ContainSubstring("must return"))
			Expect(diag.Range.End.Line).To(BeNumerically(">=", diag.Range.Start.Line))
			Expect(diag.Range.End.Character).To(BeNumerically(">=", diag.Range.Start.Character))
		})

		It("Should handle multiple diagnostics with correct ranges", func() {
			OpenDocument(server, ctx, uri, "func test() {\n\ta := undefined1\n\tb := undefined2\n}")

			Expect(client.Diagnostics).To(HaveLen(2))

			diag1 := client.Diagnostics[0]
			Expect(diag1.Message).To(ContainSubstring("undefined symbol: undefined1"))
			Expect(diag1.Range.Start.Line).To(Equal(uint32(1)))
			Expect(diag1.Range.End.Line).To(Equal(uint32(1)))
			Expect(diag1.Range.End.Character).To(Equal(uint32(16)))

			diag2 := client.Diagnostics[1]
			Expect(diag2.Message).To(ContainSubstring("undefined symbol: undefined2"))
			Expect(diag2.Range.Start.Line).To(Equal(uint32(2)))
			Expect(diag2.Range.End.Line).To(Equal(uint32(2)))
			Expect(diag2.Range.End.Character).To(Equal(uint32(16)))
		})

		It("Should handle block URI diagnostics with correct ranges", func() {
			blockURI := protocol.DocumentURI("arc://block/test")
			OpenDocument(server, ctx, blockURI, "x := undefined_var")

			Expect(client.Diagnostics).To(HaveLen(1))
			diag := client.Diagnostics[0]
			Expect(diag.Message).To(ContainSubstring("undefined symbol: undefined_var"))
			Expect(diag.Range.End.Character).To(BeNumerically(">", diag.Range.Start.Character))
		})
	})

	Describe("Diagnostic Severity", func() {
		It("Should set correct severity for errors", func() {
			OpenDocument(server, ctx, uri, "func test() {\n\tx := undefined\n}")

			Expect(client.Diagnostics).To(HaveLen(1))
			Expect(client.Diagnostics[0].Severity).To(Equal(protocol.DiagnosticSeverityError))
		})
	})

	Describe("Diagnostic Error Codes", func() {
		It("Should include error code for function argument count mismatch", func() {
			OpenDocument(server, ctx, uri, "func add(x i64, y i64) i64 { return x + y }\nfunc test() { z := add(1) }")

			Expect(client.Diagnostics).To(HaveLen(1))
			Expect(client.Diagnostics[0].Code).To(Equal("ARC3001"))
		})

		It("Should include error code for function argument type mismatch", func() {
			OpenDocument(server, ctx, uri, "func process(x i32) i32 { return x }\nfunc test() { z := process(\"hello\") }")

			Expect(client.Diagnostics).To(HaveLen(1))
			Expect(client.Diagnostics[0].Code).To(Equal("ARC3002"))
		})
	})

	Describe("Diagnostic Related Information", func() {
		It("Should include function signature in related information for argument errors", func() {
			OpenDocument(server, ctx, uri, "func add(x i64, y i64) i64 { return x + y }\nfunc test() { z := add(1) }")

			Expect(client.Diagnostics).To(HaveLen(1))
			Expect(client.Diagnostics[0].RelatedInformation).To(HaveLen(1))
			Expect(client.Diagnostics[0].RelatedInformation[0].Message).To(ContainSubstring("add(x i64, y i64) i64"))
		})
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
		OpenDocument(server, ctx, uri, "func test() {\n\tx := my_channel\n}")
		Expect(client.Diagnostics).To(HaveLen(1))
		Expect(client.Diagnostics[0].Message).To(ContainSubstring("undefined symbol: my_channel"))
		resolver["my_channel"] = symbol.Symbol{
			Name: "my_channel",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F32()),
		}
		observer.Notify(ctx, struct{}{})
		Eventually(func() []protocol.Diagnostic { return client.Diagnostics }).Should(BeEmpty())
	})

	It("Should show errors when a previously valid symbol is removed", func() {
		resolver["sensor"] = symbol.Symbol{
			Name: "sensor",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
		}
		OpenDocument(server, ctx, uri, "func test() {\n\tx := sensor\n}")
		Expect(client.Diagnostics).To(BeEmpty())
		delete(resolver, "sensor")
		observer.Notify(ctx, struct{}{})
		Eventually(func() int { return len(client.Diagnostics) }).Should(Equal(1))
		Expect(client.Diagnostics[0].Message).To(ContainSubstring("undefined symbol: sensor"))
	})

	It("Should republish diagnostics for multiple open documents", func() {
		uri2 := protocol.DocumentURI("file:///test2.arc")
		OpenDocument(server, ctx, uri, "func test1() {\n\tx := channel_a\n}")
		OpenDocument(server, ctx, uri2, "func test2() {\n\ty := channel_b\n}")
		Expect(client.Diagnostics).To(HaveLen(1))
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
		Eventually(func() []protocol.Diagnostic { return client.Diagnostics }).Should(BeEmpty())
	})
})
