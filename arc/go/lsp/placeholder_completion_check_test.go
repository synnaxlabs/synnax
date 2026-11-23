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

// itemText returns the text that accepting item inserts.
func itemText(item protocol.CompletionItem) string {
	if edit := ItemTextEdit(item); edit != nil {
		return edit.NewText
	}
	if text, ok := item.InsertText.Get(); ok && text != "" {
		return text
	}
	return item.Label
}

var _ = Describe("Format string placeholder completion (check)", func() {
	var (
		server *lsp.Server
		docURI uri.URI = "file:///placeholder_check.arc"
	)

	BeforeEach(func() {
		channels := []symbol.Symbol{{
			Name: "ox-pt-1",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   1,
		}, {
			Name: `say"hi`,
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   2,
		}}
		server = MustSucceed(lsp.New(lsp.Config{
			NewRoot: func() *symbol.Symbol { return NewRoot(nil, channels...) },
		}))
		server.SetClient(&MockClient{})
	})

	It("should escape a bare channel completion in a placeholder for its literal", func(
		ctx SpecContext,
	) {
		OpenArcDocument(server, ctx, docURI, "func f() {\n    x := f\"{sa}\"\n}")
		completions := Completion(server, ctx, docURI, 1, 14)
		var texts []string
		for _, item := range completions.Items {
			if item.Label == `say"hi` {
				texts = append(texts, itemText(item))
			}
		}
		Expect(texts).To(Equal([]string{`'say\"hi'`}))
	})

	It("should offer nothing in a later line of a backtick string", func(
		ctx SpecContext,
	) {
		OpenArcDocument(
			server,
			ctx,
			docURI,
			"func f() {\n    x := `first\nox\n`\n}",
		)
		completions := Completion(server, ctx, docURI, 2, 2)
		Expect(completions.Items).To(BeEmpty())
	})

	It("should offer nothing after an unclosed backtick string", func(
		ctx SpecContext,
	) {
		OpenArcDocument(server, ctx, docURI, "func f() {\n    x := `first\n    ox\n}")
		completions := Completion(server, ctx, docURI, 2, 6)
		Expect(completions.Items).To(BeEmpty())
	})

	It("should read an empty quote pair as closed", func(ctx SpecContext) {
		OpenArcDocument(server, ctx, docURI, "func f() {\n    x := ''\n}")
		completions := Completion(server, ctx, docURI, 1, 11)
		Expect(HasCompletion(completions.Items, "ox-pt-1")).To(BeTrue())
		Expect(editStarts(
			completions.Items,
			protocol.Position{Line: 1, Character: 9},
		)).To(BeEmpty())
	})

	It("should map a quoted edit past an escape before the placeholder", func(
		ctx SpecContext,
	) {
		OpenArcDocument(
			server,
			ctx,
			docURI,
			"func f() {\n    x := f\"a\\\"b{'ox-p}\"\n}",
		)
		completions := Completion(server, ctx, docURI, 1, 21)
		var edits []protocol.Range
		for _, item := range completions.Items {
			if item.Label == "ox-pt-1" {
				if edit := ItemTextEdit(item); edit != nil {
					edits = append(edits, edit.Range)
				}
			}
		}
		Expect(edits).To(Equal([]protocol.Range{{
			Start: protocol.Position{Line: 1, Character: 16},
			End:   protocol.Position{Line: 1, Character: 21},
		}}))
	})
})
