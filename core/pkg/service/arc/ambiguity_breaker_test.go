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
	. "github.com/synnaxlabs/arc/lsp/testutil"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	. "github.com/synnaxlabs/x/lsp/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/uri"
)

// D-214: a path through a folder that the subject cannot retrieve matches nothing for
// that subject, and no hidden folder's name shows.
var _ = Describe("Ambiguity breaker", func() {
	const docURI = uri.URI("file:///ambiguity_breaker.arc")
	var shown, hidden group.Group
	BeforeEach(func(ctx SpecContext) {
		shown = group.Group{Name: "breaker_shown", Scope: resource.TypeChannel}
		hidden = group.Group{Name: "breaker_hidden", Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := groupSvc.NewWriter(tx)
			for _, g := range []*group.Group{&shown, &hidden} {
				if err := w.Create(ctx, g); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).Delete(ctx, shown.Key, hidden.Key)
			})).To(Succeed())
		})
	})
	stored := func(ctx SpecContext, chs ...channel.Channel) {
		GinkgoHelper()
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, chs...)).
			To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
				Delete(ctx, channel.KeysFromChannels(chs)...)).To(Succeed())
		})
	}
	virtual := func(name string, key channel.LocalKey, g *group.Key) channel.Channel {
		return channel.Channel{
			Name:        name,
			Leaseholder: node.KeyFree,
			LocalKey:    key,
			DataType:    telem.Float32T,
			Virtual:     true,
			Group:       g,
		}
	}
	hiding := func(folder resource.ID) func(
		context.Context,
	) (func(resource.ID) bool, error) {
		return func(context.Context) (func(resource.ID) bool, error) {
			return func(id resource.ID) bool { return id != folder }, nil
		}
	}
	open := func(ctx SpecContext, source string) *MockClient {
		GinkgoHelper()
		server := MustSucceed(svc.NewLSP(arc.LSPConfig{
			Allowed: hiding(hidden.ResourceID()),
			Rename:  noRename,
		}))
		DeferCleanup(func(ctx SpecContext) {
			Expect(server.Shutdown(ctx)).To(Succeed())
		})
		client := &MockClient{}
		server.SetClient(client)
		OpenArcDocument(server, ctx, docURI, source)
		return client
	}

	It("Should not name a hidden folder in an ambiguous name's diagnostic", func(
		ctx SpecContext,
	) {
		stored(
			ctx,
			virtual("breaker_dup", 1_000_202, &shown.Key),
			virtual("breaker_dup", 1_000_203, &hidden.Key),
		)
		client := open(ctx, "func test() {\n\tx := breaker_dup\n}")
		Expect(client.Diagnostics()).To(HaveLen(1))
		Expect(DiagnosticMessage(client.Diagnostics()[0])).
			ToNot(ContainSubstring("breaker_hidden"))
	})
})
