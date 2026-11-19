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
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	v12 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v12"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	color "github.com/synnaxlabs/x/color/versions/v0"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

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
		value, ok := out.Configs["n1"].Variant.(v12.ValueElementConfig)
		Expect(ok).To(BeTrue())
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
		gauge, ok := out.Configs["n1"].Variant.(v12.GaugeElementConfig)
		Expect(ok).To(BeTrue())
		Expect(gauge.Precision).To(HaveValue(BeEquivalentTo(3)))
	})

	It("Should keep the precision of a tank", func(ctx SpecContext) {
		out := MustSucceed(v12.MigrateSchematic(ctx, v11.Schematic{
			Configs: map[string]v11.ElementConfig{
				"n1": {Variant: v10.TankElementConfig{Precision: 4.4}},
			},
		}))
		tank, ok := out.Configs["n1"].Variant.(v12.TankElementConfig)
		Expect(ok).To(BeTrue())
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
		scale, ok := out.Configs["n1"].Variant.(v12.ScaleElementConfig)
		Expect(ok).To(BeTrue())
		Expect(scale.Precision).To(HaveValue(BeEquivalentTo(5)))
	})
})

var _ = Describe("Migration", func() {
	It("Should lift schematics that v0.61 stored to v13", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v11.Key, v11.Schematic]{
			DB: db,
		}))
		group, project := uuid.New(), uuid.New()
		numeric := func(p float64) v10.NumericTelemConfig {
			return v10.NumericTelemConfig{Precision: p}
		}
		pipe := v11.Edge{
			Key:    "pipe",
			Source: v11.Handle{Node: "value", Param: "out"},
			Target: v11.Handle{Node: "tank", Param: "in"},
		}
		old := v11.Schematic{
			Key:  uuid.New(),
			Name: "Feed system",
			Nodes: []v11.Node{
				{Key: "value", Position: spatial.XY{X: 10, Y: 20}, ZIndex: 2},
				{Key: "gauge"},
				{Key: "tank"},
				{Key: "scale"},
			},
			Edges: []v11.Edge{pipe},
			Configs: map[string]v11.ElementConfig{
				"value": {Variant: v10.ValueElementConfig{
					NumericTelemConfig: numeric(2),
					StrokeColor:        &color.Color{R: 1, G: 2, B: 3, A: 0.5},
				}},
				"gauge": {Variant: v10.GaugeElementConfig{
					NumericTelemConfig: numeric(2.6),
				}},
				"tank": {Variant: v10.TankElementConfig{Precision: 40}},
				"scale": {Variant: v10.ScaleElementConfig{
					ScaleIndicatorConfig: v10.ScaleIndicatorConfig{
						NumericTelemConfig: numeric(-1),
					},
				}},
				"pipe": {Variant: v10.PipeElementConfig{}},
			},
			Group:   &group,
			Project: &project,
		}
		Expect(gorp.NewCreate[v11.Key, v11.Schematic]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v13.Key, v13.Schematic]{
			DB: db,
			Migrations: []gorp.Migration{
				v12.SnapshotMigration,
				v12.Migration,
				v13.Migration,
			},
		}))
		var got v13.Schematic
		Expect(gorp.NewRetrieve[v13.Key, v13.Schematic]().
			Where(gorp.MatchKeys[v13.Key, v13.Schematic](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Name).To(Equal("Feed system"))
		Expect(got.Group).To(HaveValue(Equal(group)))
		Expect(got.Project).To(HaveValue(Equal(project)))
		Expect(got.Nodes).To(HaveLen(4))
		Expect(got.Nodes[0].Key).To(Equal("value"))
		Expect(got.Nodes[0].Position).To(Equal(spatial.XY{X: 10, Y: 20}))
		Expect(got.Nodes[0].ZIndex).To(BeEquivalentTo(2))
		value, ok := got.Nodes[0].Config.Variant.(v13.ValueNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(value.Precision).To(HaveValue(BeEquivalentTo(2)))
		Expect(value.StrokeColor).
			To(HaveValue(Equal(colorv2.Color{R: 1, G: 2, B: 3, A: 128})))
		gauge, ok := got.Nodes[1].Config.Variant.(v13.GaugeNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(gauge.Precision).To(HaveValue(BeEquivalentTo(3)))
		tank, ok := got.Nodes[2].Config.Variant.(v13.TankNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(tank.Precision).To(HaveValue(BeEquivalentTo(17)))
		scale, ok := got.Nodes[3].Config.Variant.(v13.ScaleNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(scale.Precision).To(HaveValue(BeZero()))
		Expect(got.Edges).To(HaveLen(1))
		Expect(got.Edges[0].Key).To(Equal("pipe"))
		Expect(got.Edges[0].Source).To(Equal(pipe.Source))
		Expect(got.Edges[0].Target).To(Equal(pipe.Target))
		Expect(got.Edges[0].Config.Variant).
			To(BeAssignableToTypeOf(v13.PipeEdgeConfig{}))
	})
})
