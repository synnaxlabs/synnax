// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v13_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	symbolv2 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v2"
	symbolv4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	v12 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v12"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
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
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v12.Key, v12.Schematic]{
				DB: db,
			}))
			c := func(r uint8) *color.Color { return &color.Color{R: r, A: a} }
			old := v12.Schematic{
				Key:   uuid.New(),
				Name:  "s",
				Edges: []v12.Edge{{Key: "pipe", StrokeColor: c(2)}},
				Configs: map[string]v12.NodeConfig{
					"cap": {Variant: v12.CapNodeConfig{
						StaticSymbolConfig: v12.StaticSymbolConfig{StrokeColor: c(1)},
					}},
					"value": {Variant: v12.ValueNodeConfig{
						StrokeColor: c(3),
						TextColor:   c(4),
						FillColor:   c(5),
						Redline: colorv1.Scale{Bands: []colorv1.Band{
							{Key: "b", Color: *c(6)},
						}},
					}},
					"custom": {Variant: v12.CustomStaticNodeConfig{
						StateOverrides: []symbolv2.State{
							{Key: "on", Regions: []symbolv2.Region{
								{Key: "r", FillColor: c(7)},
							}},
						},
					}},
				},
			}
			Expect(gorp.NewCreate[v12.Key, v12.Schematic]().Entry(&old).Exec(ctx, db)).
				To(Succeed())
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v13.Key, v13.Schematic]{
				DB: db, Migrations: []gorp.Migration{v13.Migration},
			}))
			var got v13.Schematic
			Expect(gorp.NewRetrieve[v13.Key, v13.Schematic]().
				Where(gorp.MatchKeys[v13.Key, v13.Schematic](old.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			n := func(r uint8) *colorv2.Color { return &colorv2.Color{R: r, A: expected} }
			Expect(got.Configs["cap"].Variant.(v13.CapNodeConfig).StrokeColor).
				To(Equal(n(1)))
			Expect(got.Edges[0].StrokeColor).To(Equal(n(2)))
			value := got.Configs["value"].Variant.(v13.ValueNodeConfig)
			Expect(value.StrokeColor).To(Equal(n(3)))
			Expect(value.TextColor).To(Equal(n(4)))
			Expect(value.FillColor).To(Equal(n(5)))
			Expect(
				value.Redline.Bands,
			).To(Equal([]colorv2.Band{{Key: "b", Color: *n(6)}}))
			custom := got.Configs["custom"].Variant.(v13.CustomStaticNodeConfig)
			Expect(custom.StateOverrides).To(Equal([]symbolv4.State{{
				Key: "on", Regions: []symbolv4.Region{{Key: "r", FillColor: n(7)}},
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
			GinkgoHelper()
			out := MustSucceed(v13.MigrateSchematic(ctx, v12.Schematic{
				Configs: map[string]v12.NodeConfig{
					"n": {Variant: v12.CapNodeConfig{
						StaticSymbolConfig: v12.StaticSymbolConfig{StrokeColor: half()},
					}},
				},
			}))
			return out.Configs["n"].Variant.(v13.CapNodeConfig).StrokeColor
		}),
		Entry("MigrateStaticSymbolConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateStaticSymbolConfig(
				ctx, v12.StaticSymbolConfig{StrokeColor: half()},
			)).StrokeColor
		}),
		Entry("MigrateToggleSymbolConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateToggleSymbolConfig(
				ctx, v12.ToggleSymbolConfig{StrokeColor: half()},
			)).StrokeColor
		}),
		Entry("MigrateStalenessConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateStalenessConfig(
				ctx, v12.StalenessConfig{StalenessColor: half()},
			)).StalenessColor
		}),
		Entry("MigrateStateMapping", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateStateMapping(
				ctx, v12.StateMapping{Color: half()},
			)).Color
		}),
		Entry("MigrateScaleIndicatorConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateScaleIndicatorConfig(
				ctx, v12.ScaleIndicatorConfig{LevelColor: half()},
			)).LevelColor
		}),
		Entry(
			"MigrateDummyToggleSymbolConfig",
			func(ctx context.Context) *colorv2.Color {
				GinkgoHelper()
				return MustSucceed(v13.MigrateDummyToggleSymbolConfig(
					ctx, v12.DummyToggleSymbolConfig{StrokeColor: half()},
				)).StrokeColor
			},
		),
		Entry("MigrateToggleConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateToggleConfig(
				ctx, v12.ToggleConfig{StalenessColor: half()},
			)).StalenessColor
		}),
		Entry("MigrateEdge", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateEdge(
				ctx, v12.Edge{StrokeColor: half()},
			)).StrokeColor
		}),
	)
	DescribeTable("Should make a zero color that meant the theme absent",
		func(
			ctx SpecContext,
			variant v12.NodeConfigVariant,
			cleared func(v13.NodeConfigVariant) []*colorv2.Color,
		) {
			out := MustSucceed(v13.MigrateSchematic(ctx, v12.Schematic{
				Configs: map[string]v12.NodeConfig{"n": {Variant: variant}},
			}))
			for _, c := range cleared(out.Configs["n"].Variant) {
				Expect(c).To(BeNil())
			}
		},
		Entry("a value", v12.ValueNodeConfig{
			StalenessColor: &color.Color{},
			TextColor:      &color.Color{},
		}, func(v v13.NodeConfigVariant) []*colorv2.Color {
			c := v.(v13.ValueNodeConfig)
			return []*colorv2.Color{c.StalenessColor, c.TextColor}
		}),
		Entry("a gauge", v12.GaugeNodeConfig{
			StalenessColor: &color.Color{},
			StrokeColor:    &color.Color{},
		}, func(v v13.NodeConfigVariant) []*colorv2.Color {
			c := v.(v13.GaugeNodeConfig)
			return []*colorv2.Color{c.StalenessColor, c.StrokeColor}
		}),
		Entry("a scale", v12.ScaleNodeConfig{
			StalenessColor: &color.Color{},
			LevelColor:     &color.Color{},
			AxisColor:      &color.Color{},
			TextColor:      &color.Color{},
		}, func(v v13.NodeConfigVariant) []*colorv2.Color {
			c := v.(v13.ScaleNodeConfig)
			return []*colorv2.Color{
				c.StalenessColor,
				c.LevelColor,
				c.AxisColor,
				c.TextColor,
			}
		}),
		Entry("a tank", v12.TankNodeConfig{
			StalenessColor: &color.Color{},
			LevelColor:     &color.Color{},
			AxisColor:      &color.Color{},
			TextColor:      &color.Color{},
		}, func(v v13.NodeConfigVariant) []*colorv2.Color {
			c := v.(v13.TankNodeConfig)
			return []*colorv2.Color{
				c.StalenessColor,
				c.LevelColor,
				c.AxisColor,
				c.TextColor,
			}
		}),
	)
	It("Should keep the zero stroke of a value", func(ctx SpecContext) {
		out := MustSucceed(v13.MigrateSchematic(ctx, v12.Schematic{
			Configs: map[string]v12.NodeConfig{
				"n": {Variant: v12.ValueNodeConfig{StrokeColor: &color.Color{}}},
			},
		}))
		Expect(out.Configs["n"].Variant.(v13.ValueNodeConfig).StrokeColor).
			To(HaveValue(Equal(colorv2.Color{})))
	})
	It("Should keep a zero color that painted transparent", func(ctx SpecContext) {
		zero := &color.Color{}
		out := MustSucceed(v13.MigrateSchematic(ctx, v12.Schematic{
			Edges: []v12.Edge{{Key: "pipe", StrokeColor: zero}},
			Configs: map[string]v12.NodeConfig{
				"cap": {Variant: v12.CapNodeConfig{
					StaticSymbolConfig: v12.StaticSymbolConfig{StrokeColor: zero},
				}},
			},
		}))
		Expect(out.Configs["cap"].Variant.(v13.CapNodeConfig).StrokeColor).
			To(HaveValue(Equal(colorv2.Color{})))
		Expect(out.Edges[0].StrokeColor).To(HaveValue(Equal(colorv2.Color{})))
	})
	It("Should keep a zero region color in a state override", func(ctx SpecContext) {
		out := MustSucceed(v13.MigrateSchematic(ctx, v12.Schematic{
			Configs: map[string]v12.NodeConfig{
				"n": {Variant: v12.CustomStaticNodeConfig{
					StateOverrides: []symbolv2.State{
						{
							Key: "base",
							Regions: []symbolv2.Region{
								{Key: "r", FillColor: &color.Color{}},
							},
						},
					},
				}},
			},
		}))
		node := out.Configs["n"].Variant.(v13.CustomStaticNodeConfig)
		Expect(node.StateOverrides[0].Regions[0].FillColor).
			To(HaveValue(Equal(colorv2.Color{})))
	})
})
