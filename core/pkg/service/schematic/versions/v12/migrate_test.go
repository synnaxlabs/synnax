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
	"encoding/json/v2"
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	v12 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v12"
	color "github.com/synnaxlabs/x/color/versions/v1"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// migrateStored writes a v11 schematic holding cfg to an in-memory table, runs the v12
// migration, and reads the migrated config back.
func migrateStored(ctx SpecContext, cfg v10.ElementConfigVariant) v12.ElementConfig {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	old := v11.Schematic{
		Key:     uuid.New(),
		Configs: map[string]v11.ElementConfig{"n1": {Variant: cfg}},
	}
	MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v11.Key, v11.Schematic]{DB: db}))
	Expect(gorp.NewCreate[v11.Key, v11.Schematic]().
		Entry(&old).Exec(ctx, db)).To(Succeed())
	Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:         db,
		Namespace:  "Schematic",
		Migrations: []gorp.Migration{v12.Migration},
	})).To(Succeed())
	var got v12.Schematic
	Expect(gorp.NewRetrieve[v12.Key, v12.Schematic]().
		Where(gorp.MatchKeys[v12.Key, v12.Schematic](old.Key)).
		Entry(&got).Exec(ctx, db)).To(Succeed())
	return got.Configs["n1"]
}

var _ = Describe("MigrateNumericTelemConfig", func() {
	DescribeTable("Should round and clamp the precision", func(
		ctx SpecContext,
		in float64,
		out uint8,
	) {
		cfg := MustSucceed(v12.MigrateNumericTelemConfig(
			ctx,
			v11.NumericTelemConfig{Precision: in},
		))
		Expect(cfg.Precision).To(HaveValue(Equal(out)))
	},
		Entry("zero", 0.0, uint8(0)),
		Entry("whole", 4.0, uint8(4)),
		Entry("round up", 2.6, uint8(3)),
		Entry("round down", 1.4, uint8(1)),
		Entry("negative", -3.0, uint8(0)),
		Entry("above 17", 18.0, uint8(17)),
		Entry("above a byte", 1000.0, uint8(17)),
		Entry("NaN", math.NaN(), uint8(0)),
	)
})

var _ = Describe("MigrateSchematic", func() {
	It("Should keep the precision of a value", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Configs: map[string]v11.ElementConfig{"n1": {
				Variant: v10.ValueElementConfig{
					NumericTelemConfig: v10.NumericTelemConfig{Precision: 0},
				},
			}},
		}))
		value := out.Configs["n1"].Variant.(v12.ValueElementConfig)
		Expect(value.Precision).To(HaveValue(BeZero()))
	})

	It("Should keep the precision of a gauge", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Configs: map[string]v11.ElementConfig{"n1": {
				Variant: v10.GaugeElementConfig{
					NumericTelemConfig: v10.NumericTelemConfig{Precision: 3},
				},
			}},
		}))
		gauge := out.Configs["n1"].Variant.(v12.GaugeElementConfig)
		Expect(gauge.Precision).To(HaveValue(BeEquivalentTo(3)))
	})

	It("Should keep the precision of a tank", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Configs: map[string]v11.ElementConfig{
				"n1": {Variant: v10.TankElementConfig{Precision: 4.4}},
			},
		}))
		tank := out.Configs["n1"].Variant.(v12.TankElementConfig)
		Expect(tank.Precision).To(HaveValue(BeEquivalentTo(4)))
	})

	It("Should keep the precision of a scale", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Configs: map[string]v11.ElementConfig{"n1": {
				Variant: v10.ScaleElementConfig{
					ScaleIndicatorConfig: v10.ScaleIndicatorConfig{
						NumericTelemConfig: v10.NumericTelemConfig{Precision: 5},
					},
				},
			}},
		}))
		scale := out.Configs["n1"].Variant.(v12.ScaleElementConfig)
		Expect(scale.Precision).To(HaveValue(BeEquivalentTo(5)))
	})

	DescribeTable("Should round a symbol stroke width to whole pixels",
		func(ctx SpecContext, old v10.ElementConfigVariant, want uint8) {
			out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
				Configs: map[string]v11.ElementConfig{"n1": {Variant: old}},
			}))
			Expect(out.Configs["n1"].Variant).To(HaveField("StrokeWidth", Equal(want)))
		},
		Entry("box whole", v10.BoxElementConfig{StrokeWidth: 4}, uint8(4)),
		Entry("box fraction", v10.BoxElementConfig{StrokeWidth: 2.6}, uint8(3)),
		Entry("box zero", v10.BoxElementConfig{}, uint8(0)),
		Entry("box too large", v10.BoxElementConfig{StrokeWidth: 30}, uint8(20)),
		Entry("circle fraction", v10.CircleElementConfig{StrokeWidth: 1.4}, uint8(1)),
		Entry("circle zero", v10.CircleElementConfig{}, uint8(0)),
		Entry("circle too large", v10.CircleElementConfig{StrokeWidth: 300}, uint8(20)),
		Entry("line fraction", v10.LineElementConfig{StrokeWidth: 5.5}, uint8(6)),
		Entry("line zero", v10.LineElementConfig{}, uint8(1)),
		Entry("line negative", v10.LineElementConfig{StrokeWidth: -3}, uint8(1)),
		Entry("line too large", v10.LineElementConfig{StrokeWidth: 22}, uint8(20)),
		Entry("polygon thin", v10.PolygonElementConfig{StrokeWidth: 0.4}, uint8(1)),
		Entry("polygon zero", v10.PolygonElementConfig{}, uint8(0)),
		Entry("polygon too large",
			v10.PolygonElementConfig{StrokeWidth: 1e9}, uint8(20)),
	)

	It("Should fill the default stroke width of an edge", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Configs: map[string]v11.ElementConfig{
				"e1": {Variant: v10.PipeElementConfig{}},
			},
		}))
		Expect(out.Configs["e1"].Variant).To(
			HaveField("StrokeWidth", BeEquivalentTo(2)),
		)
	})
})

var _ = Describe("MigrateElementConfig", func() {
	DescribeTable("Should round and clamp a symbol stroke width",
		func(ctx SpecContext, old v10.ElementConfigVariant, want uint8) {
			out := MustSucceed(v12.MigrateElementConfig(
				ctx,
				v11.ElementConfig{Variant: old},
			))
			Expect(out.Variant).To(HaveField("StrokeWidth", Equal(want)))
		},
		Entry("box", v10.BoxElementConfig{StrokeWidth: 2.6}, uint8(3)),
		Entry("box at 0", v10.BoxElementConfig{}, uint8(0)),
		Entry("circle", v10.CircleElementConfig{StrokeWidth: 0.4}, uint8(1)),
		Entry("line at 0", v10.LineElementConfig{}, uint8(1)),
		Entry("box at NaN", v10.BoxElementConfig{StrokeWidth: math.NaN()}, uint8(0)),
		Entry("line at NaN", v10.LineElementConfig{StrokeWidth: math.NaN()}, uint8(1)),
		Entry("polygon", v10.PolygonElementConfig{StrokeWidth: 30}, uint8(20)),
	)

	It("Should round the precision of a tank", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateElementConfig(ctx, v11.ElementConfig{
			Variant: v10.TankElementConfig{Precision: 2.6},
		}))
		Expect(out.Variant).To(HaveField("Precision", HaveValue(BeEquivalentTo(3))))
	})

	It("Should keep the other fields of a symbol", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateElementConfig(ctx, v11.ElementConfig{
			Variant: v10.BoxElementConfig{BorderRadius: 7, StrokeWidth: 4},
		}))
		box := out.Variant.(v12.BoxElementConfig)
		Expect(box.BorderRadius).To(BeEquivalentTo(7))
		Expect(box.StrokeWidth).To(BeEquivalentTo(4))
	})
})

var _ = Describe("MigrateSegmentedEdgeConfig", func() {
	It("Should fill the default stroke width", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateSegmentedEdgeConfig(
			ctx,
			v11.SegmentedEdgeConfig{},
		))
		Expect(out.StrokeWidth).To(BeEquivalentTo(2))
	})

	It("Should keep the stroke color and segments", func(ctx SpecContext) {
		stroke := color.Color{R: 255, A: 1}
		segments := []v11.Segment{{Length: 40}}
		out := MustSucceed(v12.MigrateSegmentedEdgeConfig(
			ctx,
			v11.SegmentedEdgeConfig{StrokeColor: &stroke, Segments: segments},
		))
		Expect(out.StrokeColor).To(HaveValue(Equal(stroke)))
		Expect(out.Segments).To(Equal(segments))
	})
})

var _ = Describe("Stored migration", func() {
	DescribeTable("Should keep a stroke width of 0 through storage and JSON",
		func(ctx SpecContext, old v10.ElementConfigVariant) {
			cfg := migrateStored(ctx, old)
			Expect(cfg.Variant).To(HaveField("StrokeWidth", BeZero()))
			var decoded v12.ElementConfig
			encoded := MustSucceed(json.Marshal(cfg))
			Expect(json.Unmarshal(encoded, &decoded)).To(Succeed())
			Expect(decoded.Variant).To(HaveField("StrokeWidth", BeZero()))
		},
		Entry("box", v10.BoxElementConfig{}),
		Entry("circle", v10.CircleElementConfig{}),
		Entry("polygon", v10.PolygonElementConfig{}),
	)
})
