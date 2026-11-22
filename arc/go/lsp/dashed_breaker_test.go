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
	"github.com/antlr4-go/antlr/v4"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

var _ = Describe("Dashed names breaker", func() {
	const docURI uri.URI = "file:///dashed.arc"
	open := func(ctx SpecContext, content string) *lsp.Server {
		GinkgoHelper()
		channels := []symbol.Symbol{
			{
				Name: "pt-01",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F32()),
				ID:   1,
			},
			{
				Name: "pt_02",
				Kind: symbol.KindChannel,
				Type: types.Chan(types.F32()),
				ID:   2,
			},
		}
		server := MustSucceed(lsp.New(lsp.Config{
			NewRoot: func() *symbol.Symbol { return NewRoot(nil, channels...) },
		}))
		server.SetClient(&MockClient{})
		OpenArcDocument(server, ctx, docURI, content)
		return server
	}
	// inserted returns the text that a client inserts for item.
	inserted := func(item protocol.CompletionItem) string {
		if edit, ok := item.TextEdit.(*protocol.TextEdit); ok {
			return edit.NewText
		}
		if text, ok := item.InsertText.Get(); ok && text != "" {
			return text
		}
		return item.Label
	}
	// tokens returns the text of each visible token of src.
	tokens := func(src string) []string {
		lexer := parser.NewArcLexer(antlr.NewInputStream(src))
		lexer.RemoveErrorListeners()
		var out []string
		for _, t := range lexer.GetAllTokens() {
			if t.GetChannel() == antlr.TokenDefaultChannel {
				out = append(out, t.GetText())
			}
		}
		return out
	}

	DescribeTable(
		"Should complete a dashed channel name as one quoted token",
		func(ctx SpecContext, content string, line, char uint32) {
			server := open(ctx, content)
			list := Completion(server, ctx, docURI, line, char)
			Expect(list).ToNot(BeNil())
			item, ok := FindCompletion(list.Items, "pt-01")
			Expect(ok).To(BeTrue())
			text := inserted(item)
			Expect(tokens(text)).To(HaveLen(1), "inserted %q", text)
			Expect(text).To(Equal("'pt-01'"))
		},
		Entry(
			"in a function body",
			"func test() f32 {\n    return p\n}",
			uint32(1),
			uint32(12),
		),
		Entry(
			"in an authority entry",
			"authority (\n    200\n    p\n)",
			uint32(2),
			uint32(5),
		),
	)

	It("Should complete an identifier channel name bare", func(ctx SpecContext) {
		server := open(ctx, "func test() f32 {\n    return p\n}")
		list := Completion(server, ctx, docURI, 1, 12)
		item, ok := FindCompletion(list.Items, "pt_02")
		Expect(ok).To(BeTrue())
		Expect(inserted(item)).To(Equal("pt_02"))
	})

	It("Should not offer a channel that a quoted authority entry lists", func(
		ctx SpecContext,
	) {
		server := open(ctx, "authority (\n    200\n    'pt-01' 100\n    \n)")
		list := Completion(server, ctx, docURI, 3, 4)
		Expect(HasCompletion(list.Items, "pt_02")).To(BeTrue())
		Expect(HasCompletion(list.Items, "pt-01")).To(BeFalse())
	})

	It("Should hover a quoted dashed channel name", func(ctx SpecContext) {
		server := open(ctx, "func test() f32 {\n    return 'pt-01' * 2.0\n}")
		Expect(
			HoverContents(Hover(server, ctx, docURI, 1, 13)),
		).To(ContainSubstring("pt-01"))
	})
})
