// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package arc_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/lsp"
	arclsptestutil "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/errors"
	lsptestutil "github.com/synnaxlabs/x/lsp/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

var _ = Describe("LSP Rename", func() {
	type renamed struct {
		key  channel.Key
		name string
	}
	open := func(
		ctx SpecContext,
		rename func(context.Context, channel.Key, string) error,
	) (*lsp.Server, uri.URI, channel.Channel) {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     "lsp_rename_ch",
			Virtual:  true,
			DataType: telem.Float32T,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return channelSvc.NewWriter(tx).Delete(ctx, ch.Key(), false)
			})).To(Succeed())
		})
		server := MustSucceed(svc.NewLSP(arc.LSPConfig{
			Allowed: readAll,
			Rename:  rename,
		}))
		server.SetClient(&lsptestutil.MockClient{})
		DeferCleanup(
			func(ctx SpecContext) { Expect(server.Shutdown(ctx)).To(Succeed()) },
		)
		docURI := uri.URI("file:///rename.arc")
		arclsptestutil.OpenArcDocument(server, ctx, docURI, `func test() {
    x f32 := lsp_rename_ch + 1.0
    y := lsp_rename_ch * 2.0
}`)
		return server, docURI, ch
	}
	rename := func(
		ctx SpecContext,
		server *lsp.Server,
		docURI uri.URI,
	) (*protocol.WorkspaceEdit, error) {
		return server.Rename(ctx, &protocol.RenameParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: docURI},
			Position:     protocol.Position{Line: 1, Character: 14},
			NewName:      "lsp_renamed_ch",
		})
	}

	It(
		"Should rename the channel through Rename and update every reference",
		func(ctx SpecContext) {
			var calls []renamed
			server, docURI, ch := open(ctx, func(
				_ context.Context,
				key channel.Key,
				name string,
			) error {
				calls = append(calls, renamed{key: key, name: name})
				return nil
			})
			result := MustSucceed(rename(ctx, server, docURI))
			Expect(result.Changes[docURI]).To(HaveLen(2))
			Expect(calls).To(Equal([]renamed{{key: ch.Key(), name: "lsp_renamed_ch"}}))
		},
	)

	It("Should return the error of Rename", func(ctx SpecContext) {
		server, docURI, _ := open(
			ctx,
			func(context.Context, channel.Key, string) error {
				return errors.New("rename failed")
			},
		)
		Expect(rename(ctx, server, docURI)).Error().To(MatchError("rename failed"))
	})

	It("Should reject rename on internal channels", func(ctx SpecContext) {
		ch := &channel.Channel{
			Name:     "lsp_rename_internal",
			Virtual:  true,
			Internal: true,
			DataType: telem.Float32T,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return channelSvc.NewWriter(tx).Create(ctx, ch)
		})).To(Succeed())

		server := MustSucceed(svc.NewLSP(arc.LSPConfig{Allowed: readAll}))
		server.SetClient(&lsptestutil.MockClient{})
		DeferCleanup(
			func(ctx SpecContext) { Expect(server.Shutdown(ctx)).To(Succeed()) },
		)
		uri := uri.URI("file:///rename_internal.arc")
		arclsptestutil.OpenArcDocument(server, ctx, uri, `func test() {
    x f32 := lsp_rename_internal + 1.0
}`)

		prepared := MustSucceed(server.PrepareRename(ctx, &protocol.PrepareRenameParams{
			TextDocument: protocol.TextDocumentIdentifier{URI: uri},
			Position:     protocol.Position{Line: 1, Character: 14},
		}))
		Expect(prepared).To(BeNil())

		var original channel.Channel
		Expect(channelSvc.NewRetrieve().
			Where(channel.MatchKeys(ch.Key())).
			Entry(&original).Exec(ctx, db)).To(Succeed())
		Expect(original.Name).To(Equal("lsp_rename_internal"))
	})
})
