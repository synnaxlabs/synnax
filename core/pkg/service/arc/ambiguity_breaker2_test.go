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
	"github.com/synnaxlabs/x/crdt"
	. "github.com/synnaxlabs/x/lsp/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"go.lsp.dev/uri"
)

// D-242 (a) and (b): the LSP refuses what the compile refuses, and the refusal counts a
// hidden holder.
var _ = Describe("Ambiguity breaker 2", func() {
	It("Should refuse in the LSP a name that the compile refuses, when the subject "+
		"cannot retrieve one holder", func(ctx SpecContext) {
		folders := []*group.Group{
			{Name: "breaker2_a", Scope: resource.TypeChannel},
			{Name: "breaker2_b", Scope: resource.TypeChannel},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			for _, g := range folders {
				if err := groupSvc.NewWriter(tx).Create(ctx, g); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
		holders := make([]channel.Channel, len(folders))
		for i, g := range folders {
			holders[i] = channel.Channel{
				Name:        "breaker2_dup",
				Leaseholder: node.KeyFree,
				LocalKey:    channel.LocalKey(1_000_221 + i),
				DataType:    telem.Float32T,
				Virtual:     true,
				Group:       &g.Key,
			}
		}
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, holders...)).
			To(Succeed())
		source := "func f() {\n\tx := breaker2_dup\n}"
		a := newAutomation(arc.Base{
			Name:   "breaker2",
			Source: arc.Source{Variant: arc.TextSource{}},
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &a)
		})).To(Succeed())
		DeferCleanup(func(ctx SpecContext) {
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Delete(ctx, a.Base().Key)
			})).To(Succeed())
			Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).
				Delete(ctx, channel.KeysFromChannels(holders)...)).To(Succeed())
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return groupSvc.NewWriter(tx).
					Delete(ctx, folders[0].Key, folders[1].Key)
			})).To(Succeed())
		})
		Expect(dispatch(
			ctx,
			a.Base().Key,
			"",
			insertCharActions(crdt.New(2).Insert(0, source)),
		)).To(Succeed())
		Expect(svc.CompileProgram(ctx, a.Base().Key)).Error().
			To(MatchError(ContainSubstring("2 channels are named 'breaker2_dup'")))

		hiddenCh := holders[1].ResourceID()
		server := MustSucceed(svc.NewLSP(arc.LSPConfig{
			Allowed: func(context.Context) (func(resource.ID) bool, error) {
				return func(id resource.ID) bool { return id != hiddenCh }, nil
			},
			Rename: noRename,
		}))
		DeferCleanup(func(ctx SpecContext) {
			Expect(server.Shutdown(ctx)).To(Succeed())
		})
		client := &MockClient{}
		server.SetClient(client)
		OpenArcDocument(server, ctx, uri.URI("file:///breaker2.arc"), source)
		Expect(client.Diagnostics()).To(ContainElement(WithTransform(
			DiagnosticMessage,
			ContainSubstring("and 1 more you cannot see"),
		)))
	})
})
