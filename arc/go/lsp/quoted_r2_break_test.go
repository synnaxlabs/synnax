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

var _ = Describe("Quoted channel names in the LSP (round 2 break)", func() {
	const docURI = uri.URI("file:///quoted.arc")
	open := func(ctx SpecContext, content string) *lsp.Server {
		GinkgoHelper()
		channel := func(name string, t types.Type, id int) symbol.Symbol {
			return symbol.Symbol{
				Name: name, Type: t, Kind: symbol.KindChannel, ID: id, Renameable: true,
			}
		}
		server := MustSucceed(lsp.New(lsp.Config{
			NewRoot: func() *symbol.Symbol {
				return NewRoot(nil,
					channel("x", types.Chan(types.F64()), 7),
					channel("trig", types.Chan(types.U8()), 9),
					channel("a->b", types.Chan(types.F64()), 10),
					channel("\U0001F525", types.Chan(types.F64()), 11),
					channel("é", types.Chan(types.F64()), 12),
					channel("log", types.Chan(types.String()), 13),
				)
			},
			OnRename: func(context.Context, *symbol.Symbol, string, string) error {
				return nil
			},
		}))
		server.SetClient(&MockClient{})
		OpenArcDocument(server, ctx, docURI, content)
		return server
	}
	position := func(line, char uint32) protocol.TextDocumentPositionParams {
		return protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: line, Character: char},
		}
	}
	hover := func(ctx SpecContext, content string, line, char uint32) string {
		GinkgoHelper()
		h := MustSucceed(open(ctx, content).Hover(ctx, &protocol.HoverParams{
			TextDocumentPositionParams: position(line, char),
		}))
		return HoverContents(h)
	}

	It("R2-H1 hovers a quoted channel whose name holds an operator", func(
		ctx SpecContext,
	) {
		Expect(hover(ctx, "func f() {\n    y := 'a->b'\n}\n'trig' -> f()\n", 1, 12)).
			To(HavePrefix("#### a->b\n##### Channel"))
	})

	It(
		"R2-H2 hovers a quoted channel in a placeholder beside a local of its name",
		func(
			ctx SpecContext,
		) {
			content := "func f() {\n    x := 1.0\n    'log' = f\"{'x'}\"\n}\n'trig' -> f()\n"
			Expect(hover(ctx, content, 2, 16)).To(HavePrefix("#### x\n##### Channel"))
		},
	)
})
