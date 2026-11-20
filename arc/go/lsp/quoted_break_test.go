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

var _ = Describe("Quoted channel names in the LSP (break)", func() {
	newServer := func() *lsp.Server {
		server, _ := SetupTestServer(lsp.Config{
			NewRoot: func() *symbol.Symbol {
				return NewRoot(nil,
					symbol.Symbol{
						Name: "x", Type: types.Chan(types.F64()),
						Kind: symbol.KindChannel, ID: 7,
					},
					symbol.Symbol{
						Name: "pt-01", Type: types.Chan(types.F64()),
						Kind: symbol.KindChannel, ID: 8,
					},
					symbol.Symbol{
						Name: "trig", Type: types.Chan(types.U8()),
						Kind: symbol.KindChannel, ID: 9,
					},
				)
			},
		})
		return server
	}

	It("L1 hovers a quoted channel beside a local of the same name", func(
		ctx SpecContext,
	) {
		server := newServer()
		uri := uri.URI("file:///quoted.arc")
		content := "func f() {\n    x := 1\n    'x' = 2.0\n}\n'trig' -> f()\n"
		OpenArcDocument(server, ctx, uri, content)
		hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 2, Character: 5},
		}))
		Expect(hover).ToNot(BeNil())
		Expect(HoverContents(hover)).To(ContainSubstring("chan"))
	})

	It("L2 hovers a dashed quoted channel", func(ctx SpecContext) {
		server := newServer()
		uri := uri.URI("file:///quoted.arc")
		content := "func f() {\n    y := 'pt-01'\n}\n'trig' -> f()\n"
		OpenArcDocument(server, ctx, uri, content)
		hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 11},
		}))
		Expect(hover).ToNot(BeNil())
		Expect(HoverContents(hover)).To(ContainSubstring("pt-01"))
	})

	It("L0 hovers a bare channel", func(ctx SpecContext) {
		server := newServer()
		uri := uri.URI("file:///quoted.arc")
		content := "func f() {\n    y := trig\n}\ntrig -> f()\n"
		OpenArcDocument(server, ctx, uri, content)
		hover := MustSucceed(server.Hover(ctx, &protocol.HoverParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 11},
		}))
		Expect(hover).ToNot(BeNil())
		Expect(HoverContents(hover)).To(ContainSubstring("chan"))
	})

	It("L4 a channel rename edits its quoted references too", func(ctx SpecContext) {
		var renamed []string
		server := MustSucceed(lsp.New(lsp.Config{
			NewRoot: func() *symbol.Symbol {
				return NewRoot(nil, symbol.Symbol{
					Name: "sensor", Type: types.Chan(types.F32()),
					Kind: symbol.KindChannel, ID: 42, Renameable: true,
				})
			},
			OnRename: func(_ context.Context, _ *symbol.Symbol, oldName, newName string) error {
				renamed = append(renamed, oldName+"->"+newName)
				return nil
			},
		}))
		server.SetClient(&MockClient{})
		uri := uri.URI("file:///quoted.arc")
		content := "func test() {\n    x f32 := sensor + 1.0\n    y := 'sensor' * 2.0\n}"
		OpenArcDocument(server, ctx, uri, content)
		result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 15},
			NewName:      "renamed",
		}))
		Expect(result).ToNot(BeNil())
		Expect(renamed).To(Equal([]string{"sensor->renamed"}))
		Expect(result.Changes[uri]).To(HaveLen(2))
	})

	It("L3 goes to the definition of a quoted channel beside a local", func(
		ctx SpecContext,
	) {
		server := newServer()
		uri := uri.URI("file:///quoted.arc")
		content := "func f() {\n    x := 1\n    'x' = 2.0\n}\n'trig' -> f()\n"
		OpenArcDocument(server, ctx, uri, content)
		locations := Definition(server, ctx, uri, 2, 5)
		Expect(locations).To(BeEmpty())
	})
})
