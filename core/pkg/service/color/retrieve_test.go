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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/color"
	"github.com/synnaxlabs/synnax/pkg/service/group"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xcolor "github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Retrieve", func() {
	var (
		db     *gorp.DB
		groups *group.Service
		svc    *color.Service
		amber  color.Named
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		groups = MustOpen(group.OpenService(ctx, group.ServiceConfig{DB: db}))
		svc = MustOpen(
			color.OpenService(ctx, color.ServiceConfig{DB: db, Group: groups}),
		)
		named := []color.Named{{Name: "Valve"}, {Name: "amber"}, {Name: "Breaker"}}
		for i := range named {
			named[i].Light = xcolor.Color{R: 255, A: 255}
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).CreateMany(ctx, named)
		})).To(Succeed())
		amber = named[1]
	})
	// loose drops the built-in named colors, which sit in a palette.
	loose := func() color.Filter {
		return MustSucceed(color.MatchWhere(color.Where{Grouped: new(false)}))
	}
	names := func(ctx SpecContext, q color.Retrieve) []string {
		GinkgoHelper()
		var res []color.Named
		Expect(q.Entries(&res).Exec(ctx, db)).To(Succeed())
		return lo.Map(res, func(n color.Named, _ int) string { return n.Name })
	}

	Describe("MatchNames", func() {
		It("Should keep the named colors with the given names", func(ctx SpecContext) {
			q := svc.NewRetrieve().Where(color.MatchNames("Valve", "amber"))
			Expect(names(ctx, q)).To(ConsistOf("Valve", "amber"))
		})
	})

	Describe("OrderBy", func() {
		DescribeTable(
			"Should walk the named colors by name, ignoring case",
			func(ctx SpecContext, dir gorp.Direction, expected []string) {
				q := svc.NewRetrieve().Where(loose()).OrderBy(color.OrderByName(dir))
				Expect(names(ctx, q)).To(Equal(expected))
			},
			Entry(
				"ascending",
				gorp.DirectionAsc,
				[]string{"amber", "Breaker", "Valve"},
			),
			Entry(
				"descending",
				gorp.DirectionDesc,
				[]string{"Valve", "Breaker", "amber"},
			),
		)

		It("Should start strictly past the name and key", func(ctx SpecContext) {
			o := color.OrderByNameAfter(gorp.DirectionAsc, amber.Name, amber.Key)
			Expect(names(ctx, svc.NewRetrieve().Where(loose()).OrderBy(o))).
				To(Equal([]string{"Breaker", "Valve"}))
		})
	})

	Describe("SkipMissingKeys", func() {
		It("Should treat a key with no named color as a non-match", func(
			ctx SpecContext,
		) {
			Expect(names(ctx, svc.NewRetrieve().
				Where(color.MatchKeys(amber.Key, uuid.New())).
				SkipMissingKeys())).To(Equal([]string{"amber"}))
		})
	})

	Describe("MatchWhere", func() {
		DescribeTable(
			"Should keep the named colors that the tree selects",
			func(ctx SpecContext, w color.Where, expected []string) {
				f := MustSucceed(color.MatchWhere(w))
				Expect(names(ctx, svc.NewRetrieve().Where(f))).
					To(ConsistOf(expected))
			},
			Entry(
				"a name",
				color.Where{Name: &filter.String{Eq: new("Valve")}},
				[]string{"Valve"},
			),
			Entry(
				"a name in any case",
				color.Where{
					Name: &filter.String{Eq: new("valve"), CaseInsensitive: true},
				},
				[]string{"Valve"},
			),
			Entry(
				"any of two names",
				color.Where{Or: []color.Where{
					{Name: &filter.String{Eq: new("Valve")}},
					{Name: &filter.String{StartsWith: new("Br")}},
				}},
				[]string{"Valve", "Breaker"},
			),
			Entry(
				"no name that a tree selects",
				color.Where{
					Grouped: new(false),
					Not:     &color.Where{Name: &filter.String{Eq: new("Valve")}},
				},
				[]string{"amber", "Breaker"},
			),
		)

		It("Should keep the named colors by their palette", func(ctx SpecContext) {
			g := group.Group{Name: "Gauges", Scope: resource.TypeColor}
			Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
				if err := groups.NewWriter(tx).Create(ctx, &g); err != nil {
					return err
				}
				return svc.NewWriter(tx).Create(ctx, &color.Named{
					Name:  "Gauge",
					Light: xcolor.Color{G: 255, A: 255},
					Group: &g.Key,
				})
			})).To(Succeed())
			f := MustSucceed(color.MatchWhere(color.Where{Groups: []group.Key{g.Key}}))
			Expect(names(ctx, svc.NewRetrieve().Where(f))).To(Equal([]string{"Gauge"}))
			f = MustSucceed(color.MatchWhere(color.Where{Grouped: new(false)}))
			Expect(names(ctx, svc.NewRetrieve().Where(f))).
				To(ConsistOf("Valve", "amber", "Breaker"))
		})

		It("Should return a path error for a line that sets no operator", func() {
			Expect(color.MatchWhere(color.Where{Name: &filter.String{}})).
				Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError("name: no operator set: validation error"),
			))
		})

		It("Should return a path error for an invalid regex", func() {
			Expect(color.MatchWhere(color.Where{
				Not: &color.Where{Name: &filter.String{Regex: new("(a")}},
			})).Error().To(SatisfyAll(
				BeAssignableToTypeOf(validate.PathError{}),
				MatchError(
					"not.name.regex: error parsing regexp: missing closing ): `(a`: "+
						"validation error",
				),
			))
		})
	})
})
