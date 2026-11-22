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

var _ = Describe("Bare channel paths in the LSP (breaker)", func() {
	var (
		server *lsp.Server
		docURI uri.URI
	)
	BeforeEach(func() {
		channel := func(name string, t types.Type, id int) symbol.Symbol {
			return symbol.Symbol{Name: name, Type: t, Kind: symbol.KindChannel, ID: id}
		}
		server, docURI = SetupTestServer(lsp.Config{
			NewRoot: func() *symbol.Symbol {
				return NewRoot(nil,
					channel("stand.sensors.pt_01", types.Chan(types.F64()), 7),
					channel("stand.trig", types.Chan(types.U8()), 8),
					channel("stand.log", types.Chan(types.String()), 9),
					channel("stand.for", types.Chan(types.F64()), 10),
				)
			},
		})
	})
	hover := func(ctx SpecContext, line, char uint32) string {
		GinkgoHelper()
		return HoverContents(MustSucceed(server.Hover(ctx, &protocol.HoverParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: line, Character: char},
		})))
	}
	const localHead = "func f() {\n    stand := 1.0\n    y := stand + stand.sensors.pt_01\n}\n" +
		"stand.trig -> f()\n"

	DescribeTable("L1 shows no channel on a bare path with the keyword for",
		func(ctx SpecContext, char uint32) {
			OpenArcDocument(
				server,
				ctx,
				docURI,
				"func f() {\n    y := stand.for\n}\nstand.trig -> f()\n",
			)
			Expect(hover(ctx, 1, char)).ToNot(ContainSubstring("##### Channel"))
		},
		Entry("the first segment", uint32(10)),
		Entry("the last segment", uint32(16)),
	)

	It("L2 colors no bare path with the keyword for as a channel", func(
		ctx SpecContext,
	) {
		OpenArcDocument(server, ctx, docURI, "stand.sensors.pt_01 -> stand.for")
		all := decodeSemanticTokens(SemanticTokens(server, ctx, docURI).Data)
		Expect(filterByType(all, tokenTypeChannel)).ToNot(ContainElement(decodedToken{
			Line: 0, StartChar: 23, Length: 5, TokenType: tokenTypeChannel,
		}))
	})

	It("L3 hovers a bare path in a format string placeholder", func(ctx SpecContext) {
		OpenArcDocument(
			server,
			ctx,
			docURI,
			"func f() {\n    stand.log = f\"{stand.sensors.pt_01}\"\n}\n"+
				"stand.trig -> f()\n",
		)
		Expect(hover(ctx, 1, 20)).
			To(HavePrefix("#### stand.sensors.pt_01\n##### Channel"))
	})

	It("L4 does not rewrite the head of a bare path when it renames a local", func(
		ctx SpecContext,
	) {
		OpenArcDocument(server, ctx, docURI, localHead)
		result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: 1, Character: 4},
			NewName:      "base",
		}))
		Expect(result).ToNot(BeNil())
		Expect(result.Changes[docURI]).ToNot(ContainElement(HaveField(
			"Range.Start",
			protocol.Position{Line: 2, Character: 17},
		)), "edits: %v", result.Changes[docURI])
	})

	It("L7 shows no other channel on a segment of a path that does not resolve",
		func(ctx SpecContext) {
			other, otherURI := SetupTestServer(lsp.Config{
				NewRoot: func() *symbol.Symbol {
					return NewRoot(nil,
						symbol.Symbol{
							Name: "pt_02", Type: types.Chan(types.F64()),
							Kind: symbol.KindChannel, ID: 12,
						},
						symbol.Symbol{
							Name: "stand.trig", Type: types.Chan(types.U8()),
							Kind: symbol.KindChannel, ID: 8,
						},
					)
				},
			})
			OpenArcDocument(
				other,
				ctx,
				otherURI,
				"func f() {\n    y := stand.pt_02\n}\nstand.trig -> f()\n",
			)
			h := MustSucceed(other.Hover(ctx, &protocol.HoverParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: otherURI},
				Position:     protocol.Position{Line: 1, Character: 16},
			}))
			Expect(HoverContents(h)).ToNot(ContainSubstring("##### Channel"))
		})

	It("L5 does not jump to a local from the head of a bare path", func(
		ctx SpecContext,
	) {
		OpenArcDocument(server, ctx, docURI, localHead)
		Expect(hover(ctx, 2, 18)).
			To(HavePrefix("#### stand.sensors.pt_01\n##### Channel"))
		Expect(Definition(server, ctx, docURI, 2, 18)).To(BeEmpty())
	})
})
