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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	arclsptestutil "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/x/lsp/protocol"
	lsptestutil "github.com/synnaxlabs/x/lsp/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("LSP Rename", func() {
	It("Should rename the backing channel and update every reference", func(ctx SpecContext) {
		ch := &channel.Channel{
			Name:     "lsp_rename_ch",
			Virtual:  true,
			DataType: telem.Float32T,
		}
		Expect(channelSvc.Create(ctx, ch)).To(Succeed())

		server := MustSucceed(svc.NewLSP())
		server.SetClient(&lsptestutil.MockClient{})
		DeferCleanup(func(ctx SpecContext) { Expect(server.Shutdown(ctx)).To(Succeed()) })
		uri := protocol.DocumentURI("file:///rename.arc")
		arclsptestutil.OpenArcDocument(server, ctx, uri, `func test() {
    x f32 := lsp_rename_ch + 1.0
    y := lsp_rename_ch * 2.0
}`)

		result := MustSucceed(server.Rename(ctx, &protocol.RenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 1, Character: 14}, // lsp_rename_ch
			},
			NewName: "lsp_renamed_ch",
		}))
		Expect(result).ToNot(BeNil())
		Expect(result.Changes[uri]).To(HaveLen(2))

		var renamed channel.Channel
		Expect(node.Channel.NewRetrieve().
			Where(channel.MatchKeys(ch.Key())).
			Entry(&renamed).Exec(ctx, nil)).To(Succeed())
		Expect(renamed.Name).To(Equal("lsp_renamed_ch"))
	})

	It("Should reject rename on internal channels", func(ctx SpecContext) {
		ch := &channel.Channel{
			Name:     "lsp_rename_internal",
			Virtual:  true,
			Internal: true,
			DataType: telem.Float32T,
		}
		Expect(channelSvc.Create(ctx, ch)).To(Succeed())

		server := MustSucceed(svc.NewLSP())
		server.SetClient(&lsptestutil.MockClient{})
		DeferCleanup(func(ctx SpecContext) { Expect(server.Shutdown(ctx)).To(Succeed()) })
		uri := protocol.DocumentURI("file:///rename_internal.arc")
		arclsptestutil.OpenArcDocument(server, ctx, uri, `func test() {
    x f32 := lsp_rename_internal + 1.0
}`)

		prepared := MustSucceed(server.PrepareRename(ctx, &protocol.PrepareRenameParams{
			TextDocumentPositionParams: protocol.TextDocumentPositionParams{
				TextDocument: protocol.TextDocumentIdentifier{URI: uri},
				Position:     protocol.Position{Line: 1, Character: 14},
			},
		}))
		Expect(prepared).To(BeNil())

		var original channel.Channel
		Expect(node.Channel.NewRetrieve().
			Where(channel.MatchKeys(ch.Key())).
			Entry(&original).Exec(ctx, nil)).To(Succeed())
		Expect(original.Name).To(Equal("lsp_rename_internal"))
	})
})
