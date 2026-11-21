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
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/lsp/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

var _ = Describe("Rename sites", func() {
	const docURI = uri.URI("file:///rename_sites.arc")
	It("Should edit only the tokens where a channel can be named", func(
		ctx SpecContext,
	) {
		channels := []symbol.Symbol{
			{
				Name: "trig",
				Type: types.Chan(types.U8()),
				Kind: symbol.KindChannel,
				ID:   9,
			},
			{
				Name:       "sensor",
				Type:       types.Chan(types.F64()),
				Kind:       symbol.KindChannel,
				ID:         14,
				Renameable: true,
			},
		}
		server := MustSucceed(lsp.New(lsp.Config{
			NewRoot: func() *symbol.Symbol { return NewRoot(nil, channels...) },
			OnRename: func(context.Context, *symbol.Symbol, string, string) error {
				return nil
			},
		}))
		server.SetClient(&MockClient{})
		content := "func g(sensor f64) f64 {\n    return sensor\n}\n" +
			"'trig' -> g(sensor=1.0)\n" +
			"func f() {\n    y := sensor * 2.0\n}\n" +
			"'trig' -> {sensor: f()}\n"
		OpenArcDocument(server, ctx, docURI, content)
		result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
				Position:     protocol.Position{Line: 5, Character: 9},
			},
			NewName: "PT 1",
		}))
		Expect(result.Changes[docURI]).To(ConsistOf(protocol.TextEdit{
			Range: protocol.Range{
				Start: protocol.Position{Line: 5, Character: 9},
				End:   protocol.Position{Line: 5, Character: 15},
			},
			NewText: "'PT 1'",
		}))
	})
})
