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
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Sibling names", Ordered, func() {
	var (
		db       *gorp.DB
		groupSvc *group.Service
		svc      *channel.Service
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		cfg, gs := serviceConfig(ctx, node, node.DB)
		groupSvc = gs
		svc = MustOpen(channel.OpenService(ctx, cfg))
	})
	createChannel := func(ctx SpecContext, name string, g *group.Key) error {
		ch := channel.Channel{
			Name:     name,
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    g,
		}
		return db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})
	}
	createFolder := func(ctx SpecContext, name string, parent *group.Key) (group.Group, error) {
		g := group.Group{Name: name, Scope: resource.TypeChannel}
		if parent != nil {
			id := group.OntologyID(*parent)
			g.Parent = &id
		}
		err := db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})
		return g, err
	}
	taken := func(table, name string) types.GomegaMatcher {
		return MatchError(MatchRegexp(
			"%s \\S+ already holds the name '%s' in this group", table, name,
		))
	}

	It("Should refuse a channel with the name of a folder beside it", func(
		ctx SpecContext,
	) {
		name := UniqueChannelName()
		MustSucceed(createFolder(ctx, name, nil))
		Expect(createChannel(ctx, name, nil)).To(taken("group", name))
	})

	It("Should refuse a folder with the name of a channel beside it", func(
		ctx SpecContext,
	) {
		name := UniqueChannelName()
		Expect(createChannel(ctx, name, nil)).To(Succeed())
		Expect(createFolder(ctx, name, nil)).Error().To(taken("channel", name))
	})

	It("Should refuse two folders with one name in one parent only", func(
		ctx SpecContext,
	) {
		parent := MustSucceed(createFolder(ctx, UniqueChannelName(), nil))
		other := MustSucceed(createFolder(ctx, UniqueChannelName(), nil))
		MustSucceed(createFolder(ctx, "Stand A", &parent.Key))
		Expect(createFolder(ctx, "Stand A", &parent.Key)).Error().
			To(taken("group", "Stand A"))
		MustSucceed(createFolder(ctx, "Stand A", &other.Key))
	})

	It("Should refuse a channel move next to a folder with its name", func(
		ctx SpecContext,
	) {
		name := UniqueChannelName()
		parent := MustSucceed(createFolder(ctx, UniqueChannelName(), nil))
		MustSucceed(createFolder(ctx, name, &parent.Key))
		ch := channel.Channel{Name: name, DataType: telem.Float32T, Virtual: true}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).SetGroup(ctx, &parent.Key, ch.Key())
		})).To(taken("group", name))
	})

	It("Should refuse a folder move or rename onto a sibling's name", func(
		ctx SpecContext,
	) {
		name := UniqueChannelName()
		parent := MustSucceed(createFolder(ctx, UniqueChannelName(), nil))
		Expect(createChannel(ctx, name, &parent.Key)).To(Succeed())
		moved := MustSucceed(createFolder(ctx, name, nil))
		id := group.OntologyID(parent.Key)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).SetParent(ctx, &id, moved.Key)
		})).To(taken("channel", name))
		renamed := MustSucceed(createFolder(ctx, UniqueChannelName(), &parent.Key))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Rename(ctx, renamed.Key, name)
		})).To(taken("channel", name))
	})

	It("Should check a folder name against the name rule", func(ctx SpecContext) {
		Expect(createFolder(ctx, "stand.a", nil)).Error().
			To(MatchError(ContainSubstring("no '.' or invisible character")))
		g := MustSucceed(createFolder(ctx, "Stand A\u030A", nil))
		Expect(g.Name).To(Equal("Stand \u00C5"))
	})

	It("Should leave the names of other scopes alone", func(ctx SpecContext) {
		g := group.Group{Name: "plots.v2", Scope: resource.TypeProject}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return groupSvc.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
	})
})
