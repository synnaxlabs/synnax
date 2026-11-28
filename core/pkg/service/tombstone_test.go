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
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/project"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
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

	createFolderIn := func(ctx SpecContext, parent *group.Group) group.Group {
		GinkgoHelper()
		g := group.Group{Name: UniqueChannelName(), Scope: resource.TypeChannel}
		if parent != nil {
			g.Parent = new(parent.ResourceID())
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	createFolder := func(ctx SpecContext) group.Group {
		GinkgoHelper()
		return createFolderIn(ctx, nil)
	}

	// createShown creates a channel named name in folder g and shows it in a schematic.
	createShown := func(ctx SpecContext, g group.Group, name string) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     name,
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
		return ch
	}

	// deleteShown creates a channel in folder g, shows it in a schematic, and deletes
	// it.
	deleteShown := func(ctx SpecContext, g group.Group) channel.Channel {
		GinkgoHelper()
		ch := createShown(ctx, g, UniqueChannelName())
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

	Describe("A folder delete plan", func() {
		It("Should list each tombstone of the folder with its path", func(
			ctx SpecContext,
		) {
			parent := createFolder(ctx)
			g := createFolderIn(ctx, &parent)
			dead := deleteShown(ctx, g)
			plan := MustSucceed(layer.Tree.PlanDelete(ctx, db, []group.Key{g.Key}, nil))
			Expect(plan.Records).To(Equal([]resource.ID{g.ResourceID()}))
			Expect(plan.Entries).To(Equal([]impact.Entry{{
				Record: channel.ResourceID(dead.Key()),
				Effect: impact.EffectDelete,
				Path:   new(parent.Name + "." + g.Name + "." + dead.Name),
			}}))
			Expect(plan.Token).ToNot(BeEmpty())
		})

		It("Should count a tombstone whose channel the caller cannot see", func(
			ctx SpecContext,
		) {
			g := createFolder(ctx)
			dead := deleteShown(ctx, g)
			hidden := channel.ResourceID(dead.Key())
			plan := MustSucceed(layer.Tree.PlanDelete(
				ctx,
				db,
				[]group.Key{g.Key},
				func(id resource.ID) bool { return id != hidden },
			))
			Expect(plan.Entries).To(BeEmpty())
			Expect(plan.Hidden.Delete).To(BeEquivalentTo(1))
		})

		It(
			"Should refuse a folder that is not a channel folder",
			func(ctx SpecContext) {
				g := group.Group{
					Name:  UniqueChannelName(),
					Scope: resource.TypeSchematic,
				}
				Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
					return layer.Group.NewWriter(tx).Create(ctx, &g)
				})).To(Succeed())
				Expect(layer.Tree.PlanDelete(ctx, db, []group.Key{g.Key}, nil)).
					Error().To(MatchError(validate.ErrValidation))
			},
		)

		It("Should apply the delete with the token of its plan", func(ctx SpecContext) {
			g := createFolder(ctx)
			dead := deleteShown(ctx, g)
			plan := MustSucceed(layer.Tree.PlanDelete(ctx, db, []group.Key{g.Key}, nil))
			var applied impact.Plan
			Expect(db.WithTx(ctx, func(tx gorp.Tx) (err error) {
				applied, err = layer.Tree.ApplyDelete(
					ctx,
					tx,
					[]group.Key{g.Key},
					plan.Token,
					nil,
				)
				return err
			})).To(Succeed())
			Expect(applied).To(Equal(plan))
			Expect(tombstones(ctx)).ToNot(ContainElement(HaveField("Key", dead.Key())))
			Expect(MustSucceed(layer.Group.NewRetrieve().
				Where(group.MatchKeys(g.Key)).
				Exists(ctx, db))).To(BeFalse())
		})

		It("Should fail with the new plan when a tombstone joins the folder", func(
			ctx SpecContext,
		) {
			g := createFolder(ctx)
			plan := MustSucceed(layer.Tree.PlanDelete(ctx, db, []group.Key{g.Key}, nil))
			dead := deleteShown(ctx, g)
			err := db.WithTx(ctx, func(tx gorp.Tx) error {
				_, err := layer.Tree.ApplyDelete(
					ctx,
					tx,
					[]group.Key{g.Key},
					plan.Token,
					nil,
				)
				return err
			})
			changed := MustBeOk(errors.AsType[impact.PlanChangedError](err))
			Expect(changed.Plan.Entries).To(ConsistOf(
				HaveField("Record", channel.ResourceID(dead.Key())),
			))
			Expect(MustSucceed(layer.Group.NewRetrieve().
				Where(group.MatchKeys(g.Key)).
				Exists(ctx, db))).To(BeTrue())
		})
	})

	Describe("A channel delete plan", func() {
		It("Should list the tombstone that the delete replaces", func(ctx SpecContext) {
			g := createFolder(ctx)
			dead := deleteShown(ctx, g)
			live := createShown(ctx, g, dead.Name)
			plan := MustSucceed(layer.Impact.Plan(
				ctx,
				db,
				[]resource.ID{channel.ResourceID(live.Key())},
				impact.PlanOptions{},
			))
			Expect(plan.Entries).To(ContainElement(impact.Entry{
				Record: channel.ResourceID(dead.Key()),
				Effect: impact.EffectDelete,
				Path:   new(g.Name + "." + dead.Name),
			}))
		})

		It("Should list no tombstone when nothing shows the channel", func(
			ctx SpecContext,
		) {
			g := createFolder(ctx)
			dead := deleteShown(ctx, g)
			live := channel.Channel{
				Name:     dead.Name,
				DataType: telem.Float32T,
				Virtual:  true,
				Group:    &g.Key,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return layer.Channel.NewWriter(tx).Create(ctx, &live)
			})).To(Succeed())
			plan := MustSucceed(layer.Impact.Plan(
				ctx,
				db,
				[]resource.ID{channel.ResourceID(live.Key())},
				impact.PlanOptions{},
			))
			Expect(plan.Entries).To(BeEmpty())
		})
	})
})
