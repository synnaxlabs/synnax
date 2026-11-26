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
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/service/arc"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/crdt"
	"github.com/synnaxlabs/x/telem"
)

// A store from before the sibling rule can hold a name in several folders.
var _ = Describe("Ambiguous names", func() {
	It("Should hold the failure of a compile that names a name that several "+
		"channels hold", func(ctx SpecContext) {
		folders := []*group.Group{
			{Name: "ambiguity_a", Scope: resource.TypeChannel},
			{Name: "ambiguity_b", Scope: resource.TypeChannel},
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
				Name:        "ambiguity_pt",
				Leaseholder: node.KeyFree,
				LocalKey:    channel.LocalKey(1_000_301 + i),
				DataType:    telem.Float32T,
				Virtual:     true,
				Group:       &g.Key,
			}
		}
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, holders...)).
			To(Succeed())
		a := newAutomation(arc.Base{
			Name:   "ambiguity",
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
		source := "func f() {\n\tx := ambiguity_pt\n}"
		Expect(dispatch(
			ctx,
			a.Base().Key,
			"",
			insertCharActions(crdt.New(2).Insert(0, source)),
		)).To(Succeed())
		Expect(svc.Build(ctx, a.Base().Key)).Error().ToNot(HaveOccurred())
		Expect(svc.CompileProgram(ctx, a.Base().Key)).Error().To(SatisfyAll(
			MatchError(ContainSubstring("2 channels are named 'ambiguity_pt'")),
			Not(MatchError(channel.ErrLookupsChanged)),
		))
	})
})
