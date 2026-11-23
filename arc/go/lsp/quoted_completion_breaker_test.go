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

// editStarts returns the labels of the items whose text edit starts at pos.
func editStarts(items []protocol.CompletionItem, pos protocol.Position) []string {
	var labels []string
	for _, item := range items {
		if edit := ItemTextEdit(item); edit != nil && edit.Range.Start == pos {
			labels = append(labels, item.Label)
		}
	}
	return labels
}

var _ = Describe("Quoted channel name completion (breaker)", func() {
	var (
		server *lsp.Server
		docURI uri.URI = "file:///breaker.arc"
	)

	BeforeEach(func() {
		channels := []symbol.Symbol{{
			Name: "ox-pt-1",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   1,
		}, {
			Name: "pt-2",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   2,
		}}
		server = MustSucceed(lsp.New(lsp.Config{
			NewRoot: func() *symbol.Symbol { return NewRoot(nil, channels...) },
		}))
		server.SetClient(&MockClient{})
	})

	It("should not open a quoted name at an apostrophe inside a string", func(
		ctx SpecContext,
	) {
		OpenArcDocument(server, ctx, docURI, "func f() {\n    x := \"it 'ox\n}")
		completions := Completion(server, ctx, docURI, 1, 16)
		Expect(editStarts(
			completions.Items,
			protocol.Position{Line: 1, Character: 13},
		)).To(BeEmpty())
	})

	It("should not open a quoted name at an apostrophe inside a comment", func(
		ctx SpecContext,
	) {
		OpenArcDocument(server, ctx, docURI, "func f() {\n    /* it 'ox\n}")
		completions := Completion(server, ctx, docURI, 1, 13)
		Expect(editStarts(
			completions.Items,
			protocol.Position{Line: 1, Character: 10},
		)).To(BeEmpty())
	})

	It("should not read the closing quote of an empty pair as an open quote", func(
		ctx SpecContext,
	) {
		OpenArcDocument(server, ctx, docURI, "func f() {\n    x := ''\n}")
		completions := Completion(server, ctx, docURI, 1, 11)
		Expect(editStarts(
			completions.Items,
			protocol.Position{Line: 1, Character: 10},
		)).To(BeEmpty())
	})

	It("should keep normal completion after an empty quote pair", func(
		ctx SpecContext,
	) {
		OpenArcDocument(
			server,
			ctx,
			docURI,
			"func f() {\n    o_local f64 := 1.0\n    x := 1 + o\n}",
		)
		completions := Completion(server, ctx, docURI, 2, 14)
		Expect(HasCompletion(completions.Items, "o_local")).To(BeTrue())
		pairURI := uri.URI("file:///pair.arc")
		OpenArcDocument(
			server,
			ctx,
			pairURI,
			"func f() {\n    o_local f64 := 1.0\n    x := '' + o\n}",
		)
		completions = Completion(server, ctx, pairURI, 2, 15)
		Expect(HasCompletion(completions.Items, "o_local")).To(BeTrue())
	})

	It("should complete a quoted name in a format string placeholder", func(
		ctx SpecContext,
	) {
		OpenArcDocument(server, ctx, docURI, "func f() {\n    x := f\"v {'ox-p'}\"\n}")
		completions := Completion(server, ctx, docURI, 1, 19)
		Expect(HasCompletion(completions.Items, "pt-2")).To(BeFalse())
		Expect(HasCompletion(completions.Items, "ox-pt-1")).To(BeTrue())
	})
})
