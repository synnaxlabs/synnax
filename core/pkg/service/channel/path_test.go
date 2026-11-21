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
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/node"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/filter"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Paths", Ordered, func() {
	var (
		db                *gorp.DB
		groupSvc          *group.Service
		svc               *channel.Service
		stand, sensors    string
		root, top, nested channel.Key
	)
	folder := func(tx gorp.Tx, ctx SpecContext, name string, parent *group.Key) group.Key {
		GinkgoHelper()
		g := group.Group{Name: name, Scope: resource.TypeChannel}
		if parent != nil {
			g.Parent = new(group.ResourceID(*parent))
		}
		Expect(groupSvc.NewWriter(tx).Create(ctx, &g)).To(Succeed())
		return g.Key
	}
	create := func(tx gorp.Tx, ctx SpecContext, name string, g *group.Key) channel.Key {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     name,
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    g,
		}
		Expect(svc.NewWriter(tx).Create(ctx, &ch)).To(Succeed())
		return ch.Key()
	}
	keysAt := func(ctx SpecContext, snap gorp.Snapshot, f channel.Filter) []channel.Key {
		GinkgoHelper()
		var chs []channel.Channel
		Expect(svc.NewRetrieve().Where(f).Entries(&chs).Exec(ctx, snap)).To(Succeed())
		return channel.KeysFromChannels(chs)
	}

	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		cfg, gs := serviceConfig(ctx, node, node.DB)
		groupSvc = gs
		svc = MustOpen(channel.OpenService(ctx, cfg))
		stand, sensors = UniqueChannelName(), UniqueChannelName()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			standKey := folder(tx, ctx, stand, nil)
			sensorsKey := folder(tx, ctx, sensors, &standKey)
			root = create(tx, ctx, UniqueChannelName(), nil)
			top = create(tx, ctx, UniqueChannelName(), &standKey)
			nested = create(tx, ctx, UniqueChannelName(), &sensorsKey)
			return nil
		})).To(Succeed())
	})
	nameOf := func(ctx SpecContext, k channel.Key) string {
		GinkgoHelper()
		var ch channel.Channel
		Expect(svc.NewRetrieve().
			Where(channel.MatchKeys(k)).
			Entry(&ch).
			Exec(ctx, db)).To(Succeed())
		return ch.Name
	}

	It("Should find a channel at the top by its name alone", func(ctx SpecContext) {
		Expect(keysAt(ctx, db, channel.MatchPaths(nameOf(ctx, root)))).
			To(ConsistOf(root))
	})

	It(
		"Should find a channel in nested folders by its full path",
		func(ctx SpecContext) {
			Expect(keysAt(ctx, db, channel.MatchPaths(
				stand+"."+nameOf(ctx, top),
				stand+"."+sensors+"."+nameOf(ctx, nested),
			))).To(ConsistOf(top, nested))
		},
	)

	It("Should match no channel in a folder by its name alone", func(ctx SpecContext) {
		Expect(keysAt(ctx, db, channel.MatchPaths(nameOf(ctx, top)))).To(BeEmpty())
	})

	It("Should match no channel by a path that skips a folder", func(ctx SpecContext) {
		Expect(keysAt(ctx, db, channel.MatchPaths(sensors+"."+nameOf(ctx, nested)))).
			To(BeEmpty())
	})

	It("Should match nothing at a folder, under a channel, or at no record", func(
		ctx SpecContext,
	) {
		Expect(keysAt(ctx, db, channel.MatchPaths(
			stand,
			stand+"."+sensors,
			nameOf(ctx, root)+".x",
			stand+".missing",
			"",
		))).To(BeEmpty())
	})

	It("Should find a channel and a folder that its transaction staged", func(
		ctx SpecContext,
	) {
		name, dir := UniqueChannelName(), UniqueChannelName()
		tx := db.OpenTx()
		defer func() { Expect(tx.Close()).To(Succeed()) }()
		k := create(tx, ctx, name, new(folder(tx, ctx, dir, nil)))
		Expect(keysAt(ctx, tx, channel.MatchPaths(dir+"."+name))).To(ConsistOf(k))
		Expect(keysAt(ctx, db, channel.MatchPaths(dir+"."+name))).To(BeEmpty())
	})

	It("Should match the eq and in of the path line of a Where", func(ctx SpecContext) {
		topPath := stand + "." + nameOf(ctx, top)
		eq := MustSucceed(channel.MatchWhere(channel.Where{
			Path: &filter.Set[string]{Eq: &topPath},
		}))
		Expect(keysAt(ctx, db, eq)).To(ConsistOf(top))
		in := MustSucceed(channel.MatchWhere(channel.Where{
			Path: &filter.Set[string]{In: []string{topPath, nameOf(ctx, root)}},
		}))
		Expect(keysAt(ctx, db, in)).To(ConsistOf(top, root))
		Expect(channel.MatchWhere(channel.Where{Path: &filter.Set[string]{}})).
			Error().To(MatchError(ContainSubstring("path")))
	})

	It("Should find a path sent in another normal form", func(ctx SpecContext) {
		var name string
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			dir := folder(tx, ctx, "Å_"+stand, nil)
			name = UniqueChannelName()
			create(tx, ctx, name, &dir)
			return nil
		})).To(Succeed())
		decomposed := "Å_" + stand + "." + name
		w := channel.NormalizeWhere(channel.Where{
			Path: &filter.Set[string]{In: []string{decomposed}},
		})
		Expect(keysAt(ctx, db, MustSucceed(channel.MatchWhere(w)))).To(HaveLen(1))
	})
})

var _ = Describe("Paths over stored duplicates", func() {
	It("Should find the top channel by its name and the other by its path", func(
		ctx SpecContext,
	) {
		fresh := testutil.NewNode(ctx)
		cfg, groupSvc := serviceConfig(ctx, fresh, fresh.DB)
		dir := group.Group{Name: "stand", Scope: resource.TypeChannel}
		Expect(fresh.DB.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &dir)
		})).To(Succeed())
		atTop := channel.Channel{
			Name:        "pt_01",
			Leaseholder: node.KeyFree,
			LocalKey:    7,
			DataType:    telem.Float32T,
			Virtual:     true,
		}
		inFolder := atTop
		inFolder.LocalKey, inFolder.Group = 8, &dir.Key
		createStored(ctx, fresh.DB, atTop, inFolder)
		svc := MustOpen(channel.OpenService(ctx, cfg))
		at := func(path string) []channel.Key {
			var chs []channel.Channel
			Expect(svc.NewRetrieve().
				Where(channel.MatchPaths(path)).
				Entries(&chs).
				Exec(ctx, fresh.DB)).To(Succeed())
			return channel.KeysFromChannels(chs)
		}
		Expect(at("pt_01")).To(ConsistOf(atTop.Key()))
		Expect(at("stand.pt_01")).To(ConsistOf(inFolder.Key()))
	})
})
