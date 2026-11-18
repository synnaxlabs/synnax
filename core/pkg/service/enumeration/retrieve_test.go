// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package enumeration_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	"github.com/synnaxlabs/x/filter"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Retrieve", func() {
	var (
		db    *gorp.DB
		svc   *enumeration.Service
		mine  []enumeration.Enumeration
		names func(ctx SpecContext, q enumeration.Retrieve) []string
	)
	BeforeEach(func(ctx SpecContext) {
		db = DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		svc = MustOpen(enumeration.OpenService(ctx, enumeration.ServiceConfig{DB: db}))
		mine = []enumeration.Enumeration{
			{Name: "Breaker"},
			{Name: "alarm"},
			{Name: "Damper"},
		}
		Expect(db.WithTx(ctx, func(tx gorp.Tx) error {
			return svc.NewWriter(tx).CreateMany(ctx, mine)
		})).To(Succeed())
		names = func(ctx SpecContext, q enumeration.Retrieve) []string {
			GinkgoHelper()
			var res []enumeration.Enumeration
			Expect(q.Entries(&res).Exec(ctx, db)).To(Succeed())
			return lo.Map(res, func(e enumeration.Enumeration, _ int) string {
				return e.Name
			})
		}
	})
	keys := func() []enumeration.Key {
		return lo.Map(mine, func(e enumeration.Enumeration, _ int) enumeration.Key {
			return e.Key
		})
	}

	Describe("MatchNames", func() {
		It("Should keep the enumerations with the given names", func(ctx SpecContext) {
			q := svc.NewRetrieve().Where(enumeration.MatchNames("Breaker", "alarm"))
			Expect(names(ctx, q)).To(ConsistOf("Breaker", "alarm"))
		})
	})

	Describe("MatchWhere", func() {
		It("Should keep the enumerations that a Where selects", func(ctx SpecContext) {
			f := MustSucceed(enumeration.MatchWhere(enumeration.Where{
				Key: &filter.Set[enumeration.Key]{In: keys()[:2]},
			}))
			Expect(names(ctx, svc.NewRetrieve().Where(f))).
				To(ConsistOf("Breaker", "alarm"))
		})
	})

	Describe("OrderBy", func() {
		DescribeTable(
			"Should walk the enumerations by name, ignoring case",
			func(ctx SpecContext, dir gorp.Direction, expected []string) {
				q := svc.NewRetrieve().
					Where(enumeration.MatchKeys(keys()...)).
					OrderBy(enumeration.OrderByName(dir))
				Expect(names(ctx, q)).To(Equal(expected))
			},
			Entry(
				"ascending",
				gorp.DirectionAsc,
				[]string{"alarm", "Breaker", "Damper"},
			),
			Entry(
				"descending",
				gorp.DirectionDesc,
				[]string{"Damper", "Breaker", "alarm"},
			),
		)
	})

	Describe("Search", func() {
		It("Should find an enumeration by part of its name", func(ctx SpecContext) {
			Expect(names(ctx, svc.NewRetrieve().Search("dampe"))).
				To(Equal([]string{"Damper"}))
		})
	})
})
