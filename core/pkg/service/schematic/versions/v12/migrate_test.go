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

// migrateStored writes old to an in-memory table, runs the v12 migration, and reads the
// migrated schematic back.
func migrateStored(ctx SpecContext, old v11.Schematic) v12.Schematic {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	old.Key = uuid.New()
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
	return got
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
		value := out.Configs["n1"].Variant.(v12.ValueNodeConfig)
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
		gauge := out.Configs["n1"].Variant.(v12.GaugeNodeConfig)
		Expect(gauge.Precision).To(HaveValue(BeEquivalentTo(3)))
	})

	It("Should keep the precision of a tank", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Configs: map[string]v11.ElementConfig{
				"n1": {Variant: v10.TankElementConfig{Precision: 4.4}},
			},
		}))
		tank := out.Configs["n1"].Variant.(v12.TankNodeConfig)
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
		scale := out.Configs["n1"].Variant.(v12.ScaleNodeConfig)
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

	It("Should move the style of an edge onto the edge", func(ctx SpecContext) {
		stroke := color.Color{R: 255, A: 1}
		segments := []v11.Segment{{Length: 40}}
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Edges: []v11.Edge{{Key: "e1"}},
			Configs: map[string]v11.ElementConfig{"e1": {
				Variant: v10.HydraulicElementConfig{
					SegmentedEdgeConfig: v10.SegmentedEdgeConfig{
						StrokeColor: &stroke,
						Segments:    segments,
					},
				},
			}},
		}))
		Expect(out.Edges).To(HaveLen(1))
		Expect(out.Edges[0].Variant).To(Equal(v12.EdgeVariantHydraulic))
		Expect(out.Edges[0].StrokeColor).To(HaveValue(Equal(stroke)))
		Expect(out.Edges[0].StrokeWidth).To(BeEquivalentTo(2))
		Expect(out.Edges[0].Segments).To(Equal(segments))
		Expect(out.Configs).To(BeEmpty())
	})

	It("Should give an edge with no config the defaults", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Edges: []v11.Edge{{Key: "e1"}},
		}))
		Expect(out.Edges).To(HaveLen(1))
		Expect(out.Edges[0].Variant).To(Equal(v12.EdgeVariantPipe))
		Expect(out.Edges[0].StrokeColor).To(BeNil())
		Expect(out.Edges[0].StrokeWidth).To(BeEquivalentTo(2))
		Expect(out.Edges[0].Segments).To(BeEmpty())
	})

	It("Should drop an edge config that names no edge", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Configs: map[string]v11.ElementConfig{
				"e1": {Variant: v10.PipeElementConfig{}},
			},
		}))
		Expect(out.Configs).To(BeEmpty())
	})

	It("Should keep a node config that names no node", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Configs: map[string]v11.ElementConfig{
				"n1": {Variant: v10.BoxElementConfig{StrokeWidth: 3}},
			},
		}))
		Expect(out.Configs["n1"].Variant).To(HaveField("StrokeWidth", Equal(uint8(3))))
	})

	It("Should keep a node config stored under an edge key", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Edges: []v11.Edge{{Key: "e1"}},
			Configs: map[string]v11.ElementConfig{
				"e1": {Variant: v10.CapElementConfig{}},
			},
		}))
		Expect(out.Edges[0].Variant).To(Equal(v12.EdgeVariantPipe))
		Expect(out.Configs["e1"].Variant).To(BeAssignableToTypeOf(v12.CapNodeConfig{}))
	})

	It("Should keep a config that names no variant", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Configs: map[string]v11.ElementConfig{"n1": {}},
		}))
		Expect(out.Configs).To(HaveKeyWithValue("n1", v12.NodeConfig{}))
	})
})

var _ = Describe("MigrateEdge", func() {
	It("Should keep the handles and fill the default style", func(ctx SpecContext) {
		source := v11.Handle{Node: "n1", Param: "out"}
		target := v11.Handle{Node: "n2", Param: "in"}
		e := MustSucceed(v12.MigrateEdge(ctx, v11.Edge{
			Key:    "e1",
			Source: source,
			Target: target,
		}))
		Expect(e).To(Equal(v12.Edge{
			Key:         "e1",
			Source:      source,
			Target:      target,
			Variant:     v12.EdgeVariantPipe,
			StrokeWidth: 2,
		}))
	})
})

var _ = Describe("MigrateNodeConfig", func() {
	DescribeTable("Should round and clamp a symbol stroke width",
		func(ctx SpecContext, old v11.NodeConfigVariant, want uint8) {
			out := MustSucceed(v12.MigrateNodeConfig(
				ctx,
				v11.NodeConfig{Variant: old},
			))
			Expect(out.Variant).To(HaveField("StrokeWidth", Equal(want)))
		},
		Entry("box", v11.BoxNodeConfig{StrokeWidth: 2.6}, uint8(3)),
		Entry("box at 0", v11.BoxNodeConfig{}, uint8(0)),
		Entry("circle", v11.CircleNodeConfig{StrokeWidth: 0.4}, uint8(1)),
		Entry("line at 0", v11.LineNodeConfig{}, uint8(1)),
		Entry("box at NaN", v11.BoxNodeConfig{StrokeWidth: math.NaN()}, uint8(0)),
		Entry("line at NaN", v11.LineNodeConfig{StrokeWidth: math.NaN()}, uint8(1)),
		Entry("polygon", v11.PolygonNodeConfig{StrokeWidth: 30}, uint8(20)),
	)

	It("Should round the precision of a tank", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateNodeConfig(ctx, v11.NodeConfig{
			Variant: v11.TankNodeConfig{Precision: 2.6},
		}))
		Expect(out.Variant).To(HaveField("Precision", HaveValue(BeEquivalentTo(3))))
	})

	It("Should keep the other fields of a symbol", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateNodeConfig(ctx, v11.NodeConfig{
			Variant: v11.BoxNodeConfig{BorderRadius: 7, StrokeWidth: 4},
		}))
		box := out.Variant.(v12.BoxNodeConfig)
		Expect(box.BorderRadius).To(BeEquivalentTo(7))
		Expect(box.StrokeWidth).To(BeEquivalentTo(4))
	})
})

var _ = Describe("Stored migration", func() {
	DescribeTable("Should keep a stroke width of 0 through storage and JSON",
		func(ctx SpecContext, old v10.ElementConfigVariant) {
			cfg := migrateStored(ctx, v11.Schematic{
				Configs: map[string]v11.ElementConfig{"n1": {Variant: old}},
			}).Configs["n1"]
			Expect(cfg.Variant).To(HaveField("StrokeWidth", BeZero()))
			var decoded v12.NodeConfig
			encoded := MustSucceed(json.Marshal(cfg))
			Expect(json.Unmarshal(encoded, &decoded)).To(Succeed())
			Expect(decoded.Variant).To(HaveField("StrokeWidth", BeZero()))
		},
		Entry("box", v10.BoxElementConfig{}),
		Entry("circle", v10.CircleElementConfig{}),
		Entry("polygon", v10.PolygonElementConfig{}),
	)

	It("Should move a stored edge config onto its edge", func(ctx SpecContext) {
		got := migrateStored(ctx, v11.Schematic{
			Edges: []v11.Edge{{Key: "e1"}},
			Configs: map[string]v11.ElementConfig{
				"e1": {Variant: v10.DataElementConfig{}},
			},
		})
		Expect(got.Edges).To(HaveLen(1))
		Expect(got.Edges[0].Variant).To(Equal(v12.EdgeVariantData))
		Expect(got.Edges[0].StrokeWidth).To(BeEquivalentTo(2))
		Expect(got.Configs).To(BeEmpty())
	})
})
