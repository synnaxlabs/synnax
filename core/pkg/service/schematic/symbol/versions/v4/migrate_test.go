// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v3 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	color "github.com/synnaxlabs/x/color/versions/v0"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	DescribeTable("Should store the region color alphas as bytes",
		func(ctx SpecContext, a float64, expected uint8) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v3.Key, v3.Symbol]{DB: db}))
			old := v3.Symbol{Key: uuid.New(), Name: "s", Data: v3.Spec{
				Scale: 1,
				States: []v3.State{{Key: "on", Regions: []v3.Region{{
					Key:         "r",
					StrokeColor: &color.Color{R: 1, A: a},
					FillColor:   &color.Color{G: 2, A: a},
				}}}},
			}}
			Expect(gorp.NewCreate[v3.Key, v3.Symbol]().Entry(&old).Exec(ctx, db)).
				To(Succeed())
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v4.Key, v4.Symbol]{
				DB: db, Migrations: []gorp.Migration{v4.Migration},
			}))
			var got v4.Symbol
			Expect(gorp.NewRetrieve[v4.Key, v4.Symbol]().
				Where(gorp.MatchKeys[v4.Key, v4.Symbol](old.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			region := got.Data.States[0].Regions[0]
			Expect(
				region.StrokeColor,
			).To(HaveValue(Equal(colorv2.Color{R: 1, A: expected})))
			Expect(
				region.FillColor,
			).To(HaveValue(Equal(colorv2.Color{G: 2, A: expected})))
		},
		Entry("a transparent alpha", 0.0, uint8(0)),
		Entry("a faint alpha", 0.003, uint8(1)),
		Entry("a half alpha", 0.5, uint8(128)),
		Entry("an opaque alpha", 1.0, uint8(255)),
	)
})

var _ = Describe("Migrate functions", func() {
	region := func() v3.Region {
		return v3.Region{Key: "r", FillColor: &color.Color{R: 1, A: 0.5}}
	}
	want := &colorv2.Color{R: 1, A: 128}
	It("Should migrate a region", func(ctx SpecContext) {
		Expect(MustSucceed(v4.MigrateRegion(ctx, region())).FillColor).To(Equal(want))
	})
	It("Should migrate a state", func(ctx SpecContext) {
		out := MustSucceed(
			v4.MigrateState(ctx, v3.State{Regions: []v3.Region{region()}}),
		)
		Expect(out.Regions[0].FillColor).To(Equal(want))
	})
	It("Should migrate a spec", func(ctx SpecContext) {
		out := MustSucceed(v4.MigrateSpec(ctx, v3.Spec{
			States: []v3.State{{Regions: []v3.Region{region()}}},
		}))
		Expect(out.States[0].Regions[0].FillColor).To(Equal(want))
	})
	It("Should migrate a symbol", func(ctx SpecContext) {
		out := MustSucceed(v4.MigrateSymbol(ctx, v3.Symbol{Data: v3.Spec{
			States: []v3.State{{Regions: []v3.Region{region()}}},
		}}))
		Expect(out.Data.States[0].Regions[0].FillColor).To(Equal(want))
	})
})
