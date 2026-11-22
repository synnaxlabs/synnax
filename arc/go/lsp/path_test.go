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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

var _ = Describe("Bare channel paths in the LSP", func() {
	var (
		server *lsp.Server
		docURI uri.URI
	)
	BeforeEach(func() {
		server, docURI = SetupTestServer(lsp.Config{
			NewRoot: func() *symbol.Symbol {
				return NewRoot(nil,
					symbol.Symbol{
						Name: "stand.sensors.pt_01", Type: types.Chan(types.F64()),
						Kind: symbol.KindChannel, ID: 7,
					},
					symbol.Symbol{
						Name: "stand.trig", Type: types.Chan(types.U8()),
						Kind: symbol.KindChannel, ID: 8,
					},
				)
			},
		})
	})
	hover := func(ctx SpecContext, line, char uint32) *protocol.Hover {
		GinkgoHelper()
		return MustSucceed(server.Hover(ctx, &protocol.HoverParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: line, Character: char},
		}))
	}

	DescribeTable("Should hover the channel of the whole path from any segment",
		func(ctx SpecContext, char uint32) {
			OpenArcDocument(
				server,
				ctx,
				docURI,
				"func f() {\n    y := stand.sensors.pt_01\n}\nstand.trig -> f()\n",
			)
			h := hover(ctx, 1, char)
			Expect(h).ToNot(BeNil())
			Expect(HoverContents(h)).To(ContainSubstring("stand.sensors.pt_01"))
			Expect(HoverContents(h)).To(ContainSubstring("chan"))
		},
		Entry("the first segment", uint32(10)),
		Entry("a middle segment", uint32(17)),
		Entry("the last segment", uint32(25)),
	)

	It("Should hover a module member of a dotted name as before", func(
		ctx SpecContext,
	) {
		OpenArcDocument(server, ctx, docURI, "import math\n\nx := math.inf\n")
		h := hover(ctx, 2, 11)
		Expect(h).ToNot(BeNil())
		Expect(HoverContents(h)).ToNot(ContainSubstring("chan"))
	})

	It("Should color each segment of a bare channel path as a channel", func(
		ctx SpecContext,
	) {
		OpenArcDocument(server, ctx, docURI, "stand.trig -> stand.sensors.pt_01")
		all := decodeSemanticTokens(SemanticTokens(server, ctx, docURI).Data)
		Expect(filterByType(all, tokenTypeChannel)).To(Equal([]decodedToken{
			{Line: 0, StartChar: 0, Length: 5, TokenType: tokenTypeChannel},
			{Line: 0, StartChar: 6, Length: 4, TokenType: tokenTypeChannel},
			{Line: 0, StartChar: 14, Length: 5, TokenType: tokenTypeChannel},
			{Line: 0, StartChar: 20, Length: 7, TokenType: tokenTypeChannel},
			{Line: 0, StartChar: 28, Length: 5, TokenType: tokenTypeChannel},
		}))
	})

	It("Should not color a dotted name that names no channel", func(ctx SpecContext) {
		OpenArcDocument(server, ctx, docURI, "stand.nope -> stand.trig")
		all := decodeSemanticTokens(SemanticTokens(server, ctx, docURI).Data)
		Expect(filterByType(all, tokenTypeChannel)).To(Equal([]decodedToken{
			{Line: 0, StartChar: 14, Length: 5, TokenType: tokenTypeChannel},
			{Line: 0, StartChar: 20, Length: 4, TokenType: tokenTypeChannel},
		}))
	})
})
