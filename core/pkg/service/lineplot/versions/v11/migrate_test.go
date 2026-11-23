// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v11_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v10 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v10"
	v11 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v11"
	rangerv1 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v1"
	rangerv4 "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v4"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	It(
		"Should keep each range and axis with no T0, color, or time mode",
		func(ctx SpecContext) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			StartRun(db.Run)
			MustOpen(
				gorp.OpenTable(ctx, gorp.TableConfig[v10.Key, v10.LinePlot]{DB: db}),
			)
			rng := uuid.New()
			window := telem.Window{Variant: telem.DynamicWindow{TimeSpan: telem.Second}}
			old := v10.LinePlot{
				Key:  uuid.New(),
				Name: "Plot",
				Ranges: []rangerv1.Ref{
					{Variant: rangerv1.KeyRef{Key: rng}},
					{Variant: rangerv1.WindowRef{Window: window}},
				},
				Axes: v10.Axes{X1: v10.Axis{Key: v10.AxisKeyX1, Label: "Time"}},
			}
			Expect(gorp.NewCreate[v10.Key, v10.LinePlot]().
				Entry(&old).
				Exec(ctx, db)).To(Succeed())
			Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
				DB:         db,
				Namespace:  "LinePlot",
				Migrations: []gorp.Migration{v11.Migration},
			})).To(Succeed())
			var stored v11.LinePlot
			Expect(gorp.NewRetrieve[v11.Key, v11.LinePlot]().
				Where(gorp.MatchKeys[v11.Key, v11.LinePlot](old.Key)).
				Entry(&stored).
				Exec(ctx, db)).To(Succeed())
			Expect(stored.Ranges).To(Equal([]rangerv4.Ref{
				{Variant: rangerv4.KeyRef{Key: rng}},
				{Variant: rangerv4.WindowRef{Window: window}},
			}))
			Expect(stored.Axes.X1.Label).To(Equal("Time"))
			Expect(stored.Axes.X1.TimeMode).To(BeNil())
		},
	)

	It("Should drop a repeated range reference", func(ctx SpecContext) {
		rng := uuid.New()
		window := telem.Window{Variant: telem.DynamicWindow{TimeSpan: telem.Second}}
		key := rangerv1.Ref{Variant: rangerv1.KeyRef{Key: rng}}
		win := rangerv1.Ref{Variant: rangerv1.WindowRef{Window: window}}
		lp := MustSucceed(v11.MigrateLinePlot(ctx, v10.LinePlot{
			Ranges: []rangerv1.Ref{key, win, key, win},
		}))
		Expect(lp.Ranges).To(Equal([]rangerv4.Ref{
			{Variant: rangerv4.KeyRef{Key: rng}},
			{Variant: rangerv4.WindowRef{Window: window}},
		}))
	})

	It("Should reject a range reference with no variant", func(ctx SpecContext) {
		Expect(rangerv4.MigrateRef(ctx, rangerv1.Ref{})).Error().
			To(MatchError(ContainSubstring("unknown range reference variant")))
	})
})

var _ = Describe("MigrateAxis", func() {
	DescribeTable("Should keep bounds only where a manual edge gives them meaning",
		func(
			ctx SpecContext,
			bounds spatial.Bounds,
			manual v10.ManualBounds,
			want *spatial.ClosedBounds,
			wantManual v11.ManualBounds,
		) {
			a := MustSucceed(v11.MigrateAxis(ctx, v10.Axis{
				Bounds:       bounds,
				ManualBounds: manual,
			}))
			Expect(a.Bounds).To(Equal(want))
			Expect(a.ManualBounds).To(Equal(wantManual))
		},
		Entry(
			"dropping the bounds of an auto axis",
			spatial.Bounds{
				Lower: 0,
				Upper: 0,
			},
			v10.ManualBounds{},
			nil,
			v11.ManualBounds{},
		),
		Entry("keeping two manual edges in order",
			spatial.Bounds{Lower: -1, Upper: 1},
			v10.ManualBounds{Lower: true, Upper: true},
			&spatial.ClosedBounds{Lower: -1, Upper: 1},
			v11.ManualBounds{Lower: true, Upper: true}),
		Entry("swapping two reversed manual edges",
			spatial.Bounds{Lower: 4, Upper: -4},
			v10.ManualBounds{Lower: true, Upper: true},
			&spatial.ClosedBounds{Lower: -4, Upper: 4},
			v11.ManualBounds{Lower: true, Upper: true}),
		Entry("returning two equal manual edges to auto",
			spatial.Bounds{Lower: 3, Upper: 3},
			v10.ManualBounds{Lower: true, Upper: true},
			nil,
			v11.ManualBounds{}),
		Entry("moving the auto upper above a manual lower",
			spatial.Bounds{Lower: 5, Upper: 0},
			v10.ManualBounds{Lower: true},
			&spatial.ClosedBounds{Lower: 5, Upper: 6},
			v11.ManualBounds{Lower: true}),
		Entry("moving the auto upper above a manual timestamp lower",
			spatial.Bounds{Lower: 1.7e18, Upper: 0},
			v10.ManualBounds{Lower: true},
			&spatial.ClosedBounds{Lower: 1.7e18, Upper: 1.7e18 + 1.7e18*0x1p-51},
			v11.ManualBounds{Lower: true}),
		Entry("moving the auto lower below a manual upper",
			spatial.Bounds{Lower: 0, Upper: -2},
			v10.ManualBounds{Upper: true},
			&spatial.ClosedBounds{Lower: -3, Upper: -2},
			v11.ManualBounds{Upper: true}),
	)
})
