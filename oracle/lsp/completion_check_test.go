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
	"github.com/samber/lo"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
)

var _ = Describe("Completion check", func() {
	It("Should offer the request domain after the @ sigil", func(ctx SpecContext) {
		server := serve()
		Expect(server.DidOpen(ctx, &protocol.DidOpenTextDocumentParams{
			TextDocument: protocol.TextDocumentItem{
				URI: "file:///request.oracle", Version: 1, Text: "  @\n",
			},
		})).To(Succeed())
		result := MustSucceed(server.Completion(ctx, &protocol.CompletionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: "file:///request.oracle",
			},
			Position: protocol.Position{Line: 0, Character: 3},
		}))
		list, ok := result.(*protocol.CompletionList)
		Expect(ok).To(BeTrue())
		Expect(lo.Map(list.Items, func(item protocol.CompletionItem, _ int) string {
			return item.Label
		})).To(ContainElement("request"))
	})
})
