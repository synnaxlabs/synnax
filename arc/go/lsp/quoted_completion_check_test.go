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

var _ = Describe("Quoted channel name completion (check)", func() {
	var (
		server *lsp.Server
		docURI uri.URI = "file:///check.arc"
	)

	BeforeEach(func() {
		channels := []symbol.Symbol{{
			Name: "café-pt",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   1,
		}, {
			Name: "it's-pt",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   2,
		}, {
			Name: "pt-2",
			Kind: symbol.KindChannel,
			Type: types.Chan(types.F64()),
			ID:   3,
		}}
		server = MustSucceed(lsp.New(lsp.Config{
			NewRoot: func() *symbol.Symbol { return NewRoot(nil, channels...) },
		}))
		server.SetClient(&MockClient{})
	})

	It("should compare the prefix in NFC", func(ctx SpecContext) {
		OpenArcDocument(server, ctx, docURI, "func f() {\n    x := 'café\n}")
		completions := Completion(server, ctx, docURI, 1, 15)
		Expect(editStarts(
			completions.Items,
			protocol.Position{Line: 1, Character: 9},
		)).To(Equal([]string{"café-pt"}))
	})

	It("should decode an escaped quote in the prefix", func(ctx SpecContext) {
		OpenArcDocument(server, ctx, docURI, "func f() {\n    x := 'it\\'\n}")
		completions := Completion(server, ctx, docURI, 1, 14)
		Expect(editStarts(
			completions.Items,
			protocol.Position{Line: 1, Character: 9},
		)).To(Equal([]string{"it's-pt"}))
	})

	It(
		"should open the second quoted name on a line at its own quote",
		func(ctx SpecContext) {
			OpenArcDocument(server, ctx, docURI, "func f() {\n    x := 'pt-2' + 'pt\n}")
			completions := Completion(server, ctx, docURI, 1, 21)
			Expect(editStarts(
				completions.Items,
				protocol.Position{Line: 1, Character: 18},
			)).To(Equal([]string{"pt-2"}))
		},
	)
})
