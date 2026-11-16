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
})

var _ = Describe("Migration", func() {
	It("Should lift schematics that v0.61 stored to v12", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v11.Key, v11.Schematic]{
			DB: db,
		}))
		group, project := uuid.New(), uuid.New()
		numeric := func(p float64) v10.NumericTelemConfig {
			return v10.NumericTelemConfig{Precision: p}
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
			Edges: []v11.Edge{{
				Key:    "pipe",
				Source: v11.Handle{Node: "value", Param: "out"},
				Target: v11.Handle{Node: "tank", Param: "in"},
			}},
			Configs: map[string]v11.ElementConfig{
				"value": {Variant: v10.ValueElementConfig{
					NumericTelemConfig: numeric(2),
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
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v12.Key, v12.Schematic]{
			DB:         db,
			Migrations: []gorp.Migration{v12.SnapshotMigration, v12.Migration},
		}))
		var got v12.Schematic
		Expect(gorp.NewRetrieve[v12.Key, v12.Schematic]().
			Where(gorp.MatchKeys[v12.Key, v12.Schematic](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Name).To(Equal("Feed system"))
		Expect(got.Nodes).To(Equal(old.Nodes))
		Expect(got.Edges).To(Equal(old.Edges))
		Expect(got.Group).To(HaveValue(Equal(group)))
		Expect(got.Project).To(HaveValue(Equal(project)))
		Expect(got.Configs).To(HaveLen(5))
		Expect(got.Configs["value"].Variant.(v12.ValueElementConfig).Precision).
			To(HaveValue(BeEquivalentTo(2)))
		Expect(got.Configs["gauge"].Variant.(v12.GaugeElementConfig).Precision).
			To(HaveValue(BeEquivalentTo(3)))
		Expect(got.Configs["tank"].Variant.(v12.TankElementConfig).Precision).
			To(HaveValue(BeEquivalentTo(17)))
		Expect(got.Configs["scale"].Variant.(v12.ScaleElementConfig).Precision).
			To(HaveValue(BeZero()))
		Expect(got.Configs["pipe"].Variant).
			To(BeAssignableToTypeOf(v12.PipeElementConfig{}))
	})
})
