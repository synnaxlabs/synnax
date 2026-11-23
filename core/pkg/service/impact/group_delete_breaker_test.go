// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package impact_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/distribution/testutil"
	"github.com/synnaxlabs/synnax/pkg/service"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	. "github.com/synnaxlabs/synnax/pkg/service/channel/testutil"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/impact"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	"github.com/synnaxlabs/synnax/pkg/service/schematic/symbol"
	svctestutil "github.com/synnaxlabs/synnax/pkg/service/testutil"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/errors"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Group delete breaker", Ordered, ContinueOnFailure, func() {
	var (
		layer *service.Layer
		db    *gorp.DB
	)
	BeforeAll(func(ctx SpecContext) {
		ShouldNotLeakGoroutines()
		node := testutil.NewNode(ctx)
		db = node.DB
		layer = MustOpen(svctestutil.OpenLayer(ctx, node))
	})

	createGroup := func(ctx SpecContext, scope resource.Type) group.Group {
		GinkgoHelper()
		g := group.Group{Name: uuid.New().String(), Scope: scope}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Group.NewWriter(tx).Create(ctx, &g)
		})).To(Succeed())
		return g
	}
	createFolder := func(ctx SpecContext) group.Group {
		GinkgoHelper()
		return createGroup(ctx, resource.TypeChannel)
	}
	createChannelIn := func(ctx SpecContext, g *group.Group) channel.Channel {
		GinkgoHelper()
		ch := channel.Channel{
			Name:     UniqueChannelName(),
			DataType: telem.Float32T,
			Virtual:  true,
		}
		if g != nil {
			ch.Group = &g.Key
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Channel.NewWriter(tx).Create(ctx, &ch)
		})).To(Succeed())
		return ch
	}
	exists := func(ctx SpecContext, k group.Key) bool {
		GinkgoHelper()
		return MustSucceed(layer.Group.NewRetrieve().
			Where(group.MatchKeys(k)).
			Exists(ctx, db))
	}
	groupsOf := func(keys ...group.Key) impact.Change {
		return impact.Change{Variant: impact.DeleteGroupsChange{Keys: keys}}
	}
	channelsOf := func(keys ...channel.Key) impact.Change {
		return impact.Change{Variant: impact.DeleteChannelsChange{Keys: keys}}
	}
	plan := func(
		ctx SpecContext,
		c impact.Change,
		opts impact.PlanOptions,
	) (impact.Plan, error) {
		return layer.Impact.Plan(ctx, db, c, opts)
	}
	apply := func(
		ctx SpecContext,
		c impact.Change,
		opts impact.ApplyOptions,
	) (impact.Plan, error) {
		return layer.Impact.Apply(ctx, db, c, opts)
	}
	planChanged := func(err error) impact.Plan {
		GinkgoHelper()
		Expect(err).To(MatchError("the records that the change touches changed"))
		return MustBeOk(errors.AsType[impact.PlanChangedError](err)).Plan
	}
	errOf := func(_ impact.Plan, err error) error { return err }

	It("Should bind the token to the change type", func(ctx SpecContext) {
		channels := MustSucceed(plan(ctx, channelsOf(), impact.PlanOptions{}))
		groups := MustSucceed(plan(ctx, groupsOf(), impact.PlanOptions{}))
		Expect(channels.Token).ToNot(Equal(groups.Token))
		planChanged(errOf(apply(ctx, groupsOf(), impact.ApplyOptions{
			Token: channels.Token,
		})))
		planChanged(errOf(apply(ctx, channelsOf(), impact.ApplyOptions{
			Token: groups.Token,
		})))
	})

	It("Should refuse the token of a subset or a superset of the groups", func(
		ctx SpecContext,
	) {
		a := createFolder(ctx)
		b := createFolder(ctx)
		subset := MustSucceed(plan(ctx, groupsOf(a.Key), impact.PlanOptions{}))
		now := planChanged(errOf(
			apply(
				ctx,
				groupsOf(a.Key, b.Key),
				impact.ApplyOptions{Token: subset.Token},
			),
		))
		Expect(now).To(Equal(
			MustSucceed(plan(ctx, groupsOf(a.Key, b.Key), impact.PlanOptions{})),
		))
		superset := MustSucceed(plan(ctx, groupsOf(a.Key, b.Key), impact.PlanOptions{}))
		planChanged(errOf(
			apply(ctx, groupsOf(b.Key), impact.ApplyOptions{Token: superset.Token}),
		))
		Expect(exists(ctx, a.Key)).To(BeTrue())
		Expect(exists(ctx, b.Key)).To(BeTrue())
	})

	It("Should take the token of the same groups listed twice", func(ctx SpecContext) {
		g := createFolder(ctx)
		p := MustSucceed(plan(ctx, groupsOf(g.Key), impact.PlanOptions{}))
		Expect(apply(ctx, groupsOf(g.Key, g.Key), impact.ApplyOptions{Token: p.Token})).
			To(HaveField("Token", p.Token))
		Expect(exists(ctx, g.Key)).To(BeFalse())
	})

	It("Should refuse a token that is not 16 hex characters", func(ctx SpecContext) {
		g := createFolder(ctx)
		p := MustSucceed(plan(ctx, groupsOf(g.Key), impact.PlanOptions{}))
		for _, t := range []string{
			p.Token[:15],
			p.Token + "0",
			"zz" + p.Token[2:],
			" " + p.Token[1:],
		} {
			planChanged(
				errOf(apply(ctx, groupsOf(g.Key), impact.ApplyOptions{Token: t})),
			)
		}
		Expect(exists(ctx, g.Key)).To(BeTrue())
	})

	It("Should plan and apply a group that does not exist", func(ctx SpecContext) {
		missing := uuid.New()
		p := MustSucceed(plan(ctx, groupsOf(missing), impact.PlanOptions{}))
		Expect(p.Entries).To(BeEmpty())
		Expect(apply(ctx, groupsOf(missing), impact.ApplyOptions{Token: p.Token})).
			To(Equal(p))
	})

	It("Should plan and apply an empty list of groups", func(ctx SpecContext) {
		g := createFolder(ctx)
		p := MustSucceed(plan(ctx, groupsOf(), impact.PlanOptions{}))
		Expect(apply(ctx, groupsOf(), impact.ApplyOptions{Token: p.Token})).To(Equal(p))
		Expect(exists(ctx, g.Key)).To(BeTrue())
	})

	It("Should refuse a group that holds a channel with a valid token", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		ch := createChannelIn(ctx, &g)
		p := MustSucceed(plan(ctx, groupsOf(g.Key), impact.PlanOptions{}))
		Expect(p.Entries).To(BeEmpty())
		Expect(apply(ctx, groupsOf(g.Key), impact.ApplyOptions{Token: p.Token})).
			Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("because it is not empty")),
		))
		Expect(exists(ctx, g.Key)).To(BeTrue())
		Expect(MustSucceed(layer.Channel.NewRetrieve().
			Where(channel.MatchKeys(ch.Key())).
			Exists(ctx, db))).To(BeTrue())
	})

	It("Should refuse a member added after the plan", func(ctx SpecContext) {
		g := createFolder(ctx)
		p := MustSucceed(plan(ctx, groupsOf(g.Key), impact.PlanOptions{}))
		createChannelIn(ctx, &g)
		Expect(apply(ctx, groupsOf(g.Key), impact.ApplyOptions{Token: p.Token})).
			Error().To(MatchError(ContainSubstring("because it is not empty")))
		Expect(exists(ctx, g.Key)).To(BeTrue())
	})

	It("Should keep the token of a group renamed after the plan", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		p := MustSucceed(plan(ctx, groupsOf(g.Key), impact.PlanOptions{}))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Group.NewWriter(tx).Rename(ctx, g.Key, "renamed-"+g.Name)
		})).To(Succeed())
		Expect(apply(ctx, groupsOf(g.Key), impact.ApplyOptions{Token: p.Token})).
			To(Equal(p))
		Expect(exists(ctx, g.Key)).To(BeFalse())
	})

	It("Should refuse a symbol group that holds a symbol", func(ctx SpecContext) {
		g := createGroup(ctx, resource.TypeSchematicSymbol)
		sym := defaults.Fill(symbol.Symbol{
			Name: "Inlet",
			Data: symbol.Spec{
				Drawing: symbol.Drawing{Variant: symbol.SvgDrawing{Markup: "<svg/>"}},
				Variant: "valve",
			},
			Group: &g.Key,
		})
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return layer.Schematic.Symbol.NewWriter(tx).Create(ctx, &sym)
		})).To(Succeed())
		p := MustSucceed(plan(ctx, groupsOf(g.Key), impact.PlanOptions{}))
		Expect(apply(ctx, groupsOf(g.Key), impact.ApplyOptions{Token: p.Token})).
			Error().To(MatchError(ContainSubstring("because it is not empty")))
		Expect(exists(ctx, g.Key)).To(BeTrue())
	})

	It("Should take a hidden pick that is a choice with a group delete", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		hidden := new(impact.ChoiceDelete)
		p := MustSucceed(plan(ctx, groupsOf(g.Key), impact.PlanOptions{Hidden: hidden}))
		Expect(p.Token).To(Equal(
			MustSucceed(plan(ctx, groupsOf(g.Key), impact.PlanOptions{})).Token,
		))
		Expect(apply(ctx, groupsOf(g.Key), impact.ApplyOptions{
			Token:  p.Token,
			Hidden: hidden,
		})).To(Equal(p))
		Expect(exists(ctx, g.Key)).To(BeFalse())
	})

	It("Should refuse a pick that names no record before the token check", func(
		ctx SpecContext,
	) {
		g := createFolder(ctx)
		Expect(apply(ctx, groupsOf(g.Key), impact.ApplyOptions{
			Token: "0000000000000000",
			Picks: map[string]impact.Choice{"folder": impact.ChoiceDelete},
		})).Error().To(And(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("folder is not a record")),
		))
		Expect(exists(ctx, g.Key)).To(BeTrue())
	})

	It("Should apply a group delete with no token as before", func(ctx SpecContext) {
		g := createFolder(ctx)
		Expect(apply(ctx, groupsOf(g.Key), impact.ApplyOptions{})).
			To(Equal(impact.Plan{Change: groupsOf(g.Key)}))
		Expect(exists(ctx, g.Key)).To(BeFalse())
	})

	It("Should fail a plan of no change with a validation error", func(
		ctx SpecContext,
	) {
		Expect(plan(ctx, impact.Change{}, impact.PlanOptions{})).
			Error().To(MatchError(validate.ErrValidation))
	})
})
