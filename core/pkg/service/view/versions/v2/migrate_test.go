// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	rangerv1 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v1"
	statusv1 "github.com/synnaxlabs/synnax/pkg/service/status/versions/v1"
	v1 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/view/versions/v2"
	filter "github.com/synnaxlabs/x/filter/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	migrateViews := func(ctx SpecContext, views ...v1.View) map[v2.Key]v2.View {
		GinkgoHelper()
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v1.Key, v1.View]{DB: db}))
		Expect(gorp.NewCreate[v1.Key, v1.View]().Entries(&views).Exec(ctx, db)).
			To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "View",
			Migrations: []gorp.Migration{v2.Migration},
		})).To(Succeed())
		out := make(map[v2.Key]v2.View, len(views))
		for _, old := range views {
			var v v2.View
			Expect(gorp.NewRetrieve[v2.Key, v2.View]().
				Where(gorp.MatchKeys[v2.Key, v2.View](old.GorpKey())).
				Entry(&v).Exec(ctx, db)).To(Succeed())
			out[old.GorpKey()] = v
		}
		return out
	}

	It("Should turn the query of range and status views into a where", func(
		ctx SpecContext,
	) {
		label := uuid.New()
		ranges := v1.Base{Key: uuid.New(), Name: "Ranges"}
		statuses := v1.Base{Key: uuid.New(), Name: "Statuses"}
		variants := v1.Base{Key: uuid.New(), Name: "Variants"}
		out := migrateViews(
			ctx,
			v1.View{Variant: v1.RangeView{
				Base:  ranges,
				Query: v1.RangeQuery{HasLabels: []uuid.UUID{label}},
			}},
			v1.View{Variant: v1.StatusView{
				Base: statuses,
				Query: v1.StatusQuery{
					HasLabels: []uuid.UUID{label},
					Variants: []statusv1.Variant{
						statusv1.VariantError,
						statusv1.VariantWarning,
					},
				},
			}},
			v1.View{Variant: v1.StatusView{
				Base: variants,
				Query: v1.StatusQuery{
					HasLabels: []uuid.UUID{},
					Variants:  []statusv1.Variant{statusv1.VariantError},
				},
			}},
		)
		Expect(out[ranges.Key]).To(Equal(v2.View{Variant: v2.RangeView{
			Base:  ranges,
			Where: &rangerv1.Where{Labels: []uuid.UUID{label}},
		}}))
		Expect(out[statuses.Key]).To(Equal(v2.View{Variant: v2.StatusView{
			Base: statuses,
			Where: &statusv1.Where{
				Labels: []uuid.UUID{label},
				Variant: &filter.Set[statusv1.Variant]{In: []statusv1.Variant{
					statusv1.VariantError,
					statusv1.VariantWarning,
				}},
			},
		}}))
		Expect(out[variants.Key]).To(Equal(v2.View{Variant: v2.StatusView{
			Base: variants,
			Where: &statusv1.Where{Variant: &filter.Set[statusv1.Variant]{
				In: []statusv1.Variant{statusv1.VariantError},
			}},
		}}))
	})

	It("Should turn an empty query into an absent where", func(ctx SpecContext) {
		ranges := v1.Base{Key: uuid.New(), Name: "Ranges"}
		statuses := v1.Base{Key: uuid.New(), Name: "Statuses"}
		out := migrateViews(
			ctx,
			v1.View{Variant: v1.RangeView{
				Base:  ranges,
				Query: v1.RangeQuery{HasLabels: []uuid.UUID{}},
			}},
			v1.View{Variant: v1.StatusView{
				Base: statuses,
				Query: v1.StatusQuery{
					HasLabels: []uuid.UUID{},
					Variants:  []statusv1.Variant{},
				},
			}},
		)
		Expect(out[ranges.Key]).
			To(Equal(v2.View{Variant: v2.RangeView{Base: ranges}}))
		Expect(out[statuses.Key]).
			To(Equal(v2.View{Variant: v2.StatusView{Base: statuses}}))
	})
})
