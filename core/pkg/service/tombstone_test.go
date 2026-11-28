// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package service_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Channel tombstones", Ordered, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
		proj  project.Project
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
		proj = project.Project{Name: "Tombstones"}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Project.NewWriter(tx).Create(ctx, &proj)
		})).To(Succeed())
	})

	createFolder := func(ctx SpecContext) group.Group {
		GinkgoHelper()
		g := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}

	// deleteShown creates a channel in folder g, shows it in a schematic, and deletes
	// it.
	deleteShown := func(ctx SpecContext, g group.Group) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
			Group:    &g.Key,
		}
		var value schematic.ValueNodeConfig
		value.SetDefaults()
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			if err := layer.Channel.NewWriter(tx).Create(ctx, &ch); err != nil {
				return err
			}
			value.Channel = ch.Key()
			return layer.Schematic.NewWriter(tx).Create(ctx, &schematic.Schematic{
				Name:    "pid",
				Project: &proj.Key,
				Nodes: []schematic.Node{{
					Key:    "value",
					Config: schematic.NodeConfig{Variant: value},
				}},
			})
		})).To(Succeed())
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Delete(ctx, ch.Key(), false)
		})).To(Succeed())
		return ch
	}

	tombstones := func(ctx SpecContext) []channel.Tombstone {
		GinkgoHelper()
		var out []channel.Tombstone
		Expect(gorp.NewRetrieve[channel.Key, channel.Tombstone]().
			Entries(&out).
			Exec(ctx, db)).To(Succeed())
		return out
	}

	It(
		"Should delete a channel folder and its tombstones through the group writer",
		func(
			ctx SpecContext,
		) {
			g := createFolder(ctx)
			dead := deleteShown(ctx, g)
			Expect(tombstones(ctx)).To(ContainElement(
				HaveField("Key", dead.Key()),
			))
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Group.NewWriter(tx).Delete(ctx, g.Key)
			})).To(Succeed())
			Expect(tombstones(ctx)).ToNot(ContainElement(
				HaveField("Key", dead.Key()),
			))
			Expect(MustSucceed(layer.Group.NewRetrieve().
				Where(group.MatchKeys(g.Key)).
				Exists(ctx, db))).To(BeFalse())
		},
	)

	It("Should keep the tombstones of the folders that the delete does not name", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		other := createFolder(ctx)
		kept := deleteShown(ctx, other)
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Group.NewWriter(tx).Delete(ctx, g.Key)
		})).To(Succeed())
		Expect(tombstones(ctx)).To(ContainElement(HaveField("Key", kept.Key())))
	})
})
