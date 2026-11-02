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
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	symbolv2 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v2"
	symbolv3 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v3"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	color "github.com/synnaxlabs/x/color/versions/v0"
	colorv1 "github.com/synnaxlabs/x/color/versions/v1"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("Migration", func() {
	DescribeTable("Should store every element color alpha as a byte",
		func(ctx SpecContext, a float64, expected uint8) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v10.Key, v10.Schematic]{
				DB: db,
			}))
			c := func(r uint8) *color.Color { return &color.Color{R: r, A: a} }
			old := v10.Schematic{
				Key:  uuid.New(),
				Name: "s",
				Configs: map[string]v10.ElementConfig{
					"cap": {Variant: v10.CapElementConfig{
						StaticSymbolConfig: v10.StaticSymbolConfig{StrokeColor: c(1)},
					}},
					"pipe": {Variant: v10.PipeElementConfig{
						SegmentedEdgeConfig: v10.SegmentedEdgeConfig{StrokeColor: c(2)},
					}},
					"value": {Variant: v10.ValueElementConfig{
						StrokeColor: c(3),
						TextColor:   c(4),
						FillColor:   c(5),
						Redline: colorv1.Scale{Bands: []colorv1.Band{
							{Key: "b", Color: *c(6)},
						}},
					}},
					"custom": {Variant: v10.CustomStaticElementConfig{
						StateOverrides: []symbolv2.State{
							{Key: "on", Regions: []symbolv2.Region{
								{Key: "r", FillColor: c(7)},
							}},
						},
					}},
				},
			}
			Expect(gorp.NewCreate[v10.Key, v10.Schematic]().Entry(&old).Exec(ctx, db)).
				To(Succeed())
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v11.Key, v11.Schematic]{
				DB: db, Migrations: []gorp.Migration{v11.Migration},
			}))
			var got v11.Schematic
			Expect(gorp.NewRetrieve[v11.Key, v11.Schematic]().
				Where(gorp.MatchKeys[v11.Key, v11.Schematic](old.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			n := func(r uint8) *colorv2.Color { return &colorv2.Color{R: r, A: expected} }
			Expect(got.Configs["cap"].Variant.(v11.CapElementConfig).StrokeColor).
				To(Equal(n(1)))
			Expect(got.Configs["pipe"].Variant.(v11.PipeElementConfig).StrokeColor).
				To(Equal(n(2)))
			value := got.Configs["value"].Variant.(v11.ValueElementConfig)
			Expect(value.StrokeColor).To(Equal(n(3)))
			Expect(value.TextColor).To(Equal(n(4)))
			Expect(value.FillColor).To(Equal(n(5)))
			Expect(
				value.Redline.Bands,
			).To(Equal([]colorv2.Band{{Key: "b", Color: *n(6)}}))
			custom := got.Configs["custom"].Variant.(v11.CustomStaticElementConfig)
			Expect(custom.StateOverrides).To(Equal([]symbolv3.State{{
				Key: "on", Regions: []symbolv3.Region{{Key: "r", FillColor: n(7)}},
			}}))
		},
		Entry("a transparent alpha", 0.0, uint8(0)),
		Entry("a faint alpha", 0.003, uint8(1)),
		Entry("a half alpha", 0.5, uint8(128)),
		Entry("an opaque alpha", 1.0, uint8(255)),
	)
})

var _ = Describe("Migrate functions", func() {
	half := func() *color.Color { return &color.Color{R: 1, A: 0.5} }
	want := &colorv2.Color{R: 1, A: 128}
	DescribeTable(
		"Should store the color alpha as a byte",
		func(ctx SpecContext, migrate func(context.Context) *colorv2.Color) {
			Expect(migrate(ctx)).To(Equal(want))
		},
		Entry("MigrateSchematic", func(ctx context.Context) *colorv2.Color {
			out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
				Configs: map[string]v10.ElementConfig{
					"n": {Variant: v10.CapElementConfig{
						StaticSymbolConfig: v10.StaticSymbolConfig{StrokeColor: half()},
					}},
				},
			}))
			return out.Configs["n"].Variant.(v11.CapElementConfig).StrokeColor
		}),
		Entry("MigrateElementConfig", func(ctx context.Context) *colorv2.Color {
			out := MustSucceed(v11.MigrateElementConfig(ctx, v10.ElementConfig{
				Variant: v10.PipeElementConfig{
					SegmentedEdgeConfig: v10.SegmentedEdgeConfig{StrokeColor: half()},
				},
			}))
			return out.Variant.(v11.PipeElementConfig).StrokeColor
		}),
		Entry("MigrateStaticSymbolConfig", func(ctx context.Context) *colorv2.Color {
			return MustSucceed(v11.MigrateStaticSymbolConfig(
				ctx, v10.StaticSymbolConfig{StrokeColor: half()},
			)).StrokeColor
		}),
		Entry("MigrateToggleSymbolConfig", func(ctx context.Context) *colorv2.Color {
			return MustSucceed(v11.MigrateToggleSymbolConfig(
				ctx, v10.ToggleSymbolConfig{StrokeColor: half()},
			)).StrokeColor
		}),
		Entry("MigrateStalenessConfig", func(ctx context.Context) *colorv2.Color {
			return MustSucceed(v11.MigrateStalenessConfig(
				ctx, v10.StalenessConfig{StalenessColor: half()},
			)).StalenessColor
		}),
		Entry("MigrateStateMapping", func(ctx context.Context) *colorv2.Color {
			return MustSucceed(v11.MigrateStateMapping(
				ctx, v10.StateMapping{Color: half()},
			)).Color
		}),
		Entry("MigrateScaleIndicatorConfig", func(ctx context.Context) *colorv2.Color {
			return MustSucceed(v11.MigrateScaleIndicatorConfig(
				ctx, v10.ScaleIndicatorConfig{LevelColor: half()},
			)).LevelColor
		}),
		Entry(
			"MigrateDummyToggleSymbolConfig",
			func(ctx context.Context) *colorv2.Color {
				return MustSucceed(v11.MigrateDummyToggleSymbolConfig(
					ctx, v10.DummyToggleSymbolConfig{StrokeColor: half()},
				)).StrokeColor
			},
		),
		Entry("MigrateToggleConfig", func(ctx context.Context) *colorv2.Color {
			return MustSucceed(v11.MigrateToggleConfig(
				ctx, v10.ToggleConfig{StalenessColor: half()},
			)).StalenessColor
		}),
		Entry("MigrateSegmentedEdgeConfig", func(ctx context.Context) *colorv2.Color {
			return MustSucceed(v11.MigrateSegmentedEdgeConfig(
				ctx, v10.SegmentedEdgeConfig{StrokeColor: half()},
			)).StrokeColor
		}),
	)
})
