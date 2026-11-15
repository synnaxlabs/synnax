// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v3_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	rangerv2 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v2"
	rangerv3 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v3"
	statusv2 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v2"
	v2 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v3"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It("Should keep the base and where of each view", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v2.Key, v2.View]{DB: db}))
		label := uuid.New()
		ranges := v2.Base{Key: uuid.New(), Name: "Ranges"}
		statuses := v2.Base{Key: uuid.New(), Name: "Statuses"}
		views := []v2.View{
			{Variant: v2.RangeView{
				Base:  ranges,
				Where: &rangerv2.Where{Labels: []uuid.UUID{label}},
			}},
			{Variant: v2.StatusView{
				Base:  statuses,
				Where: &statusv2.Where{Labels: []uuid.UUID{label}},
			}},
		}
		Expect(gorp.NewCreate[v2.Key, v2.View]().Entries(&views).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "View",
			Migrations: []gorp.Migration{v3.Migration},
		})).To(Succeed())
		var out []v3.View
		Expect(gorp.NewRetrieve[v3.Key, v3.View]().Entries(&out).Exec(ctx, db)).
			To(Succeed())
		Expect(out).To(ConsistOf(
			v3.View{Variant: v3.RangeView{
				Base:   v3.Base{Key: ranges.Key, Name: "Ranges"},
				Where:  &rangerv3.Where{Labels: []uuid.UUID{label}},
				Sort:   v3.RangeSortNewest,
				Layout: v3.RangeLayoutTree,
			}},
			v3.View{Variant: v3.StatusView{
				Base:  v3.Base{Key: statuses.Key, Name: "Statuses"},
				Where: &statusv2.Where{Labels: []uuid.UUID{label}},
			}},
		))
	})
})

var _ = Describe("MigrateBase", func() {
	It("Should keep the key and name, with no icon or color", func(ctx SpecContext) {
		old := v2.Base{Key: uuid.New(), Name: "Ranges"}
		Expect(v3.MigrateBase(ctx, old)).
			To(Equal(v3.Base{Key: old.Key, Name: "Ranges"}))
	})
})

var _ = Describe("MigrateView", func() {
	base := v2.Base{Key: uuid.New(), Name: "View"}
	label := uuid.New()
	DescribeTable(
		"Should lift each variant",
		func(ctx SpecContext, old v2.View, expected v3.View) {
			Expect(v3.MigrateView(ctx, old)).To(Equal(expected))
		},
		Entry(
			"a range view, which takes the default sort and layout",
			v2.View{Variant: v2.RangeView{
				Base:  base,
				Where: &rangerv2.Where{Labels: []uuid.UUID{label}},
			}},
			v3.View{Variant: v3.RangeView{
				Base:   v3.Base{Key: base.Key, Name: "View"},
				Where:  &rangerv3.Where{Labels: []uuid.UUID{label}},
				Sort:   v3.RangeSortNewest,
				Layout: v3.RangeLayoutTree,
			}},
		),
		Entry(
			"a range view with no where",
			v2.View{Variant: v2.RangeView{Base: base}},
			v3.View{Variant: v3.RangeView{
				Base:   v3.Base{Key: base.Key, Name: "View"},
				Sort:   v3.RangeSortNewest,
				Layout: v3.RangeLayoutTree,
			}},
		),
		Entry(
			"a status view, which keeps its where",
			v2.View{Variant: v2.StatusView{
				Base:  base,
				Where: &statusv2.Where{Labels: []uuid.UUID{label}},
			}},
			v3.View{Variant: v3.StatusView{
				Base:  v3.Base{Key: base.Key, Name: "View"},
				Where: &statusv2.Where{Labels: []uuid.UUID{label}},
			}},
		),
		Entry("a view with no variant", v2.View{}, v3.View{}),
	)
})
