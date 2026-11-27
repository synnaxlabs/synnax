// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package channel_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/query"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Ambiguity breaker 3", func() {
	var (
		db             *gorp.DB
		svc            *channel.Service
		stand, rig     group.Group
		inStand, inRig channel.Channel
	)
	all := func(context.Context) (func(resource.ID) bool, error) {
		return func(resource.ID) bool { return true }, nil
	}
	open := func(ctx SpecContext, name string) {
		fresh := testutil.NewNode(ctx)
		db = fresh.DB
		cfg, groupSvc := serviceConfig(ctx, fresh, fresh.DB)
		stand = group.Group{Name: "stand", Scope: resource.TypeChannel}
		rig = group.Group{Name: "rig", Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			for _, g := range []*group.Group{&stand, &rig} {
				if err := groupSvc.NewWriter(tx).Create(ctx, g); err != nil {
					return err
				}
			}
			return nil
		})).To(Succeed())
		stored := func(key channel.LocalKey, g *group.Key) channel.Channel {
			return channel.Channel{
				Name:        name,
				Leaseholder: node.KeyFree,
				LocalKey:    key,
				DataType:    telem.Float32T,
				Virtual:     true,
				Group:       g,
			}
		}
		inStand, inRig = stored(7, &stand.Key), stored(8, &rig.Key)
		createStored(ctx, db, inStand, inRig)
		svc = MustOpen(channel.OpenService(ctx, cfg))
	}

	It("Should refuse a name that several channels held when a commit deletes one "+
		"holder during the lookup, and never report the name as not found", func(
		ctx SpecContext,
	) {
		open(ctx, "pt")
		// The LSP resolves through the DB, so each read sees the latest commit. The
		// access read runs once before the lookup and once before the description.
		calls := 0
		deleting := func(ctx context.Context) (func(resource.ID) bool, error) {
			if calls++; calls == 2 {
				if err := gorp.WrapWriter[channel.Key, channel.Channel](db).
					Delete(ctx, inRig.Key()); err != nil {
					return nil, err
				}
			}
			return all(ctx)
		}
		_, err := svc.NewSubjectResolver(db, deleting).Resolve(ctx, "pt")
		Expect(calls).To(Equal(2))
		Expect(err).ToNot(MatchError(query.ErrNotFound))
	})

	It("Should escape a quote in the ambiguous name as it does in each quoted path",
		func(ctx SpecContext) {
			open(ctx, "it's")
			Expect(svc.DescribeAmbiguity(ctx, db, channel.AmbiguousNameError{
				Name:    "it's",
				Holders: channel.Keys{inStand.Key(), inRig.Key()},
			}, all)).To(MatchError(
				`2 channels are named 'it\'s': 'rig.it\'s', 'stand.it\'s'`,
			))
		})

	It("Should take the stored channel of each group for a retrieve-if-exists create "+
		"that names one name in two groups", func(ctx SpecContext) {
		open(ctx, "pt")
		chs := []channel.Channel{
			{Name: "pt", DataType: telem.Float32T, Virtual: true, Group: &stand.Key},
			{Name: "pt", DataType: telem.Float32T, Virtual: true, Group: &rig.Key},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).
				CreateMany(ctx, &chs, channel.RetrieveIfNameExists())
		})).To(Succeed())
		Expect(channel.KeysFromChannels(chs)).
			To(Equal(channel.Keys{inStand.Key(), inRig.Key()}))
	})

	It("Should refuse an ambiguous name in a calculated channel when the same create "+
		"names one holder by key", func(ctx SpecContext) {
		open(ctx, "pt")
		cx := channel.Channel{
			Name:       "cx",
			DataType:   telem.Float32T,
			Virtual:    true,
			Expression: "return 1.0",
			Group:      &stand.Key,
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &cx)
		})).To(Succeed())
		// A v0.62 store can hold one name in two folders.
		other := cx
		other.LocalKey, other.Group = cx.LocalKey+100, &rig.Key
		Expect(gorp.WrapWriter[channel.Key, channel.Channel](db).Set(ctx, other)).
			To(Succeed())
		Expect(svc.NewArcSymbolResolver(db).Resolve(ctx, "cx")).Error().
			To(MatchError(query.ErrQuery))
		cy := channel.Channel{
			Name:       "cy",
			DataType:   telem.Float32T,
			Virtual:    true,
			Expression: "return cx",
		}
		create := func(chs []channel.Channel) error {
			return db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).
					CreateMany(ctx, &chs, channel.RetrieveIfNameExists())
			})
		}
		ambiguous := MatchError(ContainSubstring("2 channels are named 'cx'"))
		Expect(create([]channel.Channel{cy})).To(ambiguous)
		Expect(create([]channel.Channel{cx, cy})).To(ambiguous)
	})
})
