// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v12_test

import (
	"context"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	symbolv2 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v2"
	symbolv4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	v12 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v12"
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
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v11.Key, v11.Schematic]{
				DB: db,
			}))
			c := func(r uint8) *color.Color { return &color.Color{R: r, A: a} }
			old := v11.Schematic{
				Key:  uuid.New(),
				Name: "s",
				Configs: map[string]v11.ElementConfig{
					"cap": {Variant: v11.CapElementConfig{
						StaticSymbolConfig: v11.StaticSymbolConfig{StrokeColor: c(1)},
					}},
					"pipe": {Variant: v11.PipeElementConfig{
						SegmentedEdgeConfig: v11.SegmentedEdgeConfig{StrokeColor: c(2)},
					}},
					"value": {Variant: v11.ValueElementConfig{
						StrokeColor: c(3),
						TextColor:   c(4),
						FillColor:   c(5),
						Redline: colorv1.Scale{Bands: []colorv1.Band{
							{Key: "b", Color: *c(6)},
						}},
					}},
					"custom": {Variant: v11.CustomStaticElementConfig{
						StateOverrides: []symbolv2.State{
							{Key: "on", Regions: []symbolv2.Region{
								{Key: "r", FillColor: c(7)},
							}},
						},
					}},
				},
			}
			Expect(gorp.NewCreate[v11.Key, v11.Schematic]().Entry(&old).Exec(ctx, db)).
				To(Succeed())
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v12.Key, v12.Schematic]{
				DB: db, Migrations: []gorp.Migration{v12.Migration},
			}))
			var got v12.Schematic
			Expect(gorp.NewRetrieve[v12.Key, v12.Schematic]().
				Where(gorp.MatchKeys[v12.Key, v12.Schematic](old.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			n := func(r uint8) *colorv2.Color { return &colorv2.Color{R: r, A: expected} }
			Expect(got.Configs["cap"].Variant.(v12.CapElementConfig).StrokeColor).
				To(Equal(n(1)))
			Expect(got.Configs["pipe"].Variant.(v12.PipeElementConfig).StrokeColor).
				To(Equal(n(2)))
			value := got.Configs["value"].Variant.(v12.ValueElementConfig)
			Expect(value.StrokeColor).To(Equal(n(3)))
			Expect(value.TextColor).To(Equal(n(4)))
			Expect(value.FillColor).To(Equal(n(5)))
			Expect(
				value.Redline.Bands,
			).To(Equal([]colorv2.Band{{Key: "b", Color: *n(6)}}))
			custom := got.Configs["custom"].Variant.(v12.CustomStaticElementConfig)
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
			out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
				Configs: map[string]v11.ElementConfig{
					"n": {Variant: v11.CapElementConfig{
						StaticSymbolConfig: v11.StaticSymbolConfig{StrokeColor: half()},
					}},
				},
			}))
			return out.Configs["n"].Variant.(v12.CapElementConfig).StrokeColor
		}),
		Entry("MigrateElementConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			out := MustSucceed(v12.MigrateElementConfig(ctx, v11.ElementConfig{
				Variant: v11.PipeElementConfig{
					SegmentedEdgeConfig: v11.SegmentedEdgeConfig{StrokeColor: half()},
				},
			}))
			return out.Variant.(v12.PipeElementConfig).StrokeColor
		}),
		Entry("MigrateStaticSymbolConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v12.MigrateStaticSymbolConfig(
				ctx, v11.StaticSymbolConfig{StrokeColor: half()},
			)).StrokeColor
		}),
		Entry("MigrateToggleSymbolConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v12.MigrateToggleSymbolConfig(
				ctx, v11.ToggleSymbolConfig{StrokeColor: half()},
			)).StrokeColor
		}),
		Entry("MigrateStalenessConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v12.MigrateStalenessConfig(
				ctx, v11.StalenessConfig{StalenessColor: half()},
			)).StalenessColor
		}),
		Entry("MigrateStateMapping", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v12.MigrateStateMapping(
				ctx, v11.StateMapping{Color: half()},
			)).Color
		}),
		Entry("MigrateScaleIndicatorConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v12.MigrateScaleIndicatorConfig(
				ctx, v11.ScaleIndicatorConfig{LevelColor: half()},
			)).LevelColor
		}),
		Entry(
			"MigrateDummyToggleSymbolConfig",
			func(ctx context.Context) *colorv2.Color {
				GinkgoHelper()
				return MustSucceed(v12.MigrateDummyToggleSymbolConfig(
					ctx, v11.DummyToggleSymbolConfig{StrokeColor: half()},
				)).StrokeColor
			},
		),
		Entry("MigrateToggleConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v12.MigrateToggleConfig(
				ctx, v11.ToggleConfig{StalenessColor: half()},
			)).StalenessColor
		}),
		Entry("MigrateSegmentedEdgeConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v12.MigrateSegmentedEdgeConfig(
				ctx, v11.SegmentedEdgeConfig{StrokeColor: half()},
			)).StrokeColor
		}),
	)
	It("Should make an optional zero color absent", func(ctx SpecContext) {
		zero := &color.Color{}
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Configs: map[string]v11.ElementConfig{
				"zero": {Variant: v11.CapElementConfig{
					StaticSymbolConfig: v11.StaticSymbolConfig{StrokeColor: zero},
				}},
				"set": {Variant: v11.CapElementConfig{
					StaticSymbolConfig: v11.StaticSymbolConfig{StrokeColor: half()},
				}},
				"e": {Variant: v11.PipeElementConfig{
					SegmentedEdgeConfig: v11.SegmentedEdgeConfig{StrokeColor: zero},
				}},
			},
		}))
		Expect(
			out.Configs["zero"].Variant.(v12.CapElementConfig).StrokeColor,
		).To(BeNil())
		Expect(out.Configs["set"].Variant.(v12.CapElementConfig).StrokeColor).
			To(Equal(want))
		Expect(out.Configs["e"].Variant.(v12.PipeElementConfig).StrokeColor).To(BeNil())
	})
})
