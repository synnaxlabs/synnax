// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color_test

import (
	"fmt"
	"slices"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xcolor "github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/query"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Built-in named colors", func() {
	var (
		db     *gorp.DB
		groups *group.Service
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		groups = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
	})
	open := func(ctx SpecContext) *color.Service {
		GinkgoHelper()
		return MustOpen(
			color.OpenService(ctx, color.ServiceConfig{DB: db, Group: groups}),
		)
	}
	palette := func(ctx SpecContext) (group.Group, error) {
		var g group.Group
		err := groups.NewRetrieve().
			Where(group.MatchNames("Series")).
			Where(group.MatchScopes(resource.TypeColor)).
			Entry(&g).
			Exec(ctx, db)
		return g, err
	}
	series := func(ctx SpecContext, svc *color.Service) []color.Named {
		GinkgoHelper()
		g := MustSucceed(palette(ctx))
		var named []color.Named
		Expect(svc.NewRetrieve().
			Where(MustSucceed(color.MatchWhere(color.Where{Groups: []group.Key{g.Key}}))).
			Entries(&named).
			Exec(ctx, db)).To(Succeed())
		return named
	}
	byName := func(named []color.Named, name string) color.Named {
		GinkgoHelper()
		n, ok := lo.Find(named, func(n color.Named) bool { return n.Name == name })
		Expect(ok).To(BeTrue())
		return n
	}
	hex := func(c xcolor.Color) string {
		return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
	}

	It("Should list the values that a new Core stores", func(ctx SpecContext) {
		Expect(color.Builtins()).To(ConsistOf(series(ctx, open(ctx))))
	})

	It("Should not let a change to one list reach the next", func() {
		first := color.Builtins()
		first[0].Name = "Changed"
		*first[0].Dark = xcolor.Color{}
		next := color.Builtins()
		Expect(next[0].Name).To(Equal("Series 1"))
		Expect(*next[0].Dark).ToNot(Equal(xcolor.Color{}))
	})

	Describe("Series", func() {
		It("Should list the series colors by position", func(ctx SpecContext) {
			svc := open(ctx)
			keys := lo.Map(color.Builtins(), func(n color.Named, _ int) color.Key {
				return n.Key
			})
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Apply(ctx, keys[3], color.NewSetAction(
					color.SetPayload{Path: []string{"position"}, Value: "Zz"},
				))
			})).To(Succeed())
			moved := slices.Concat(keys[3:4], keys[:3], keys[4:])
			Expect(svc.Series(ctx, db)).To(Equal(moved))
		})

		It("Should list a series color with no position last", func(ctx SpecContext) {
			svc := open(ctx)
			g := MustSucceed(palette(ctx))
			extra := color.Named{
				Name:  "Extra",
				Light: MustSucceed(xcolor.Parse("#123456")),
				Group: &g.Key,
			}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Create(ctx, &extra)
			})).To(Succeed())
			keys := MustSucceed(svc.Series(ctx, db))
			Expect(keys).To(HaveLen(17))
			Expect(keys[16]).To(Equal(extra.Key))
		})
	})

	It("Should create 16 series colors in the Series palette", func(ctx SpecContext) {
		named := series(ctx, open(ctx))
		Expect(named).To(HaveLen(16))
		Expect(lo.Map(named, func(n color.Named, _ int) string { return n.Name })).
			To(ConsistOf(lo.Times(16, func(i int) string {
				return fmt.Sprintf("Series %d", i+1)
			})))
	})

	It("Should place the series colors in order", func(ctx SpecContext) {
		named := series(ctx, open(ctx))
		Expect(*byName(named, "Series 1").Position).To(Equal("a0"))
		Expect(*byName(named, "Series 10").Position).To(Equal("a9"))
		Expect(*byName(named, "Series 11").Position).To(Equal("aA"))
		Expect(*byName(named, "Series 16").Position).To(Equal("aF"))
	})

	It("Should give each series color a light and a dark value", func(ctx SpecContext) {
		named := series(ctx, open(ctx))
		blue := byName(named, "Series 2")
		Expect(hex(blue.Light)).To(Equal("#099BE9"))
		Expect(blue.Dark).ToNot(BeNil())
		Expect(hex(*blue.Dark)).To(Equal("#20A4F3"))
		last := byName(named, "Series 16")
		Expect(hex(last.Light)).To(Equal("#7C5F06"))
		Expect(hex(*last.Dark)).To(Equal("#B99A51"))
	})

	It(
		"Should keep an edited built-in when the Core opens again",
		func(ctx SpecContext) {
			svc := open(ctx)
			first := byName(series(ctx, svc), "Series 1")
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				return svc.NewWriter(tx).Apply(ctx, first.Key, color.NewSetAction(
					color.SetPayload{Path: []string{"name"}, Value: "Coolant"},
				))
			})).To(Succeed())
			Expect(svc.Close()).To(Succeed())
			named := series(ctx, open(ctx))
			Expect(named).To(HaveLen(16))
			Expect(byName(named, "Coolant").Key).To(Equal(first.Key))
		},
	)

	It("Should refuse to delete a built-in", func(ctx SpecContext) {
		svc := open(ctx)
		first := byName(series(ctx, svc), "Series 1")
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).Delete(ctx, first.Key)
		})).To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("cannot delete built-in color Series 1")),
		))
		Expect(series(ctx, svc)).To(HaveLen(16))
	})

	It("Should delete nothing when one key names a built-in", func(ctx SpecContext) {
		svc := open(ctx)
		first := byName(series(ctx, svc), "Series 1")
		tx := DeferClose(db.OpenTx())
		w := svc.NewWriter(tx)
		mine := color.Named{Name: "Breaker", Light: first.Light}
		Expect(w.Create(ctx, &mine)).To(Succeed())
		Expect(w.Delete(ctx, mine.Key, first.Key)).
			To(MatchError(ContainSubstring("cannot delete built-in color Series 1")))
		Expect(svc.NewRetrieve().
			Where(color.MatchKeys(mine.Key)).
			Exists(ctx, tx)).To(BeTrue())
	})

	It("Should create only the missing built-ins and their palette", func(
		ctx SpecContext,
	) {
		svc := open(ctx)
		named := series(ctx, svc)
		first := byName(named, "Series 1")
		second := byName(named, "Series 2")
		g := MustSucceed(palette(ctx))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := svc.NewWriter(tx)
			for _, n := range named {
				if err := w.Apply(ctx, n.Key, color.NewSetAction(
					color.SetPayload{Path: []string{"group"}, Value: nil},
				)); err != nil {
					return err
				}
			}
			if err := w.Apply(ctx, second.Key, color.NewSetAction(
				color.SetPayload{Path: []string{"name"}, Value: "Coolant"},
			)); err != nil {
				return err
			}
			if err := groups.NewWriter(tx).Delete(ctx, g.Key); err != nil {
				return err
			}
			// A release that adds a built-in finds it missing, as here.
			return gorp.NewDelete[color.Key, color.Named]().
				Where(gorp.MatchKeys[color.Key, color.Named](first.Key)).
				Exec(ctx, tx)
		})).To(Succeed())
		Expect(svc.Close()).To(Succeed())
		svc = open(ctx)
		Expect(series(ctx, svc)).To(Equal([]color.Named{first}))
		var stored color.Named
		Expect(svc.NewRetrieve().
			Where(color.MatchKeys(second.Key)).
			Entry(&stored).
			Exec(ctx, db)).To(Succeed())
		Expect(stored.Name).To(Equal("Coolant"))
		Expect(stored.Group).To(BeNil())
	})

	It("Should not create a deleted palette again", func(ctx SpecContext) {
		svc := open(ctx)
		g := MustSucceed(palette(ctx))
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			w := svc.NewWriter(tx)
			for _, n := range series(ctx, svc) {
				if err := w.Apply(ctx, n.Key, color.NewSetAction(
					color.SetPayload{Path: []string{"group"}, Value: nil},
				)); err != nil {
					return err
				}
			}
			return groups.NewWriter(tx).Delete(ctx, g.Key)
		})).To(Succeed())
		Expect(svc.Close()).To(Succeed())
		open(ctx)
		Expect(palette(ctx)).Error().To(MatchError(query.ErrNotFound))
	})
})
