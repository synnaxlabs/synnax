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

	It("Should create 16 series colors in the Series palette", func(ctx SpecContext) {
		named := series(ctx, open(ctx))
		Expect(named).To(HaveLen(16))
		Expect(lo.Map(named, func(n color.Named, _ int) string { return n.Name })).
			To(ConsistOf(lo.Times(16, func(i int) string {
				return fmt.Sprintf("Series %d", i+1)
			})))
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
