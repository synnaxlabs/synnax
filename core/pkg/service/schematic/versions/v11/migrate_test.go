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
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateNumericTelemConfig", func() {
	DescribeTable("Should round and clamp the precision", func(
		ctx SpecContext,
		in float64,
		out uint8,
	) {
		cfg := MustSucceed(v11.MigrateNumericTelemConfig(
			ctx,
			v10.NumericTelemConfig{Precision: in},
		))
		Expect(cfg.Precision).To(Equal(out))
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
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{"n1": {
				Variant: v10.ValueElementConfig{
					NumericTelemConfig: v10.NumericTelemConfig{Precision: 0},
				},
			}},
		}))
		value := out.Configs["n1"].Variant.(v11.ValueElementConfig)
		Expect(value.Precision).To(BeZero())
	})

	It("Should keep the precision of a gauge", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{"n1": {
				Variant: v10.GaugeElementConfig{
					NumericTelemConfig: v10.NumericTelemConfig{Precision: 3},
				},
			}},
		}))
		gauge := out.Configs["n1"].Variant.(v11.GaugeElementConfig)
		Expect(gauge.Precision).To(BeEquivalentTo(3))
	})

	It("Should keep the precision of a tank", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{
				"n1": {Variant: v10.TankElementConfig{Precision: 4.4}},
			},
		}))
		tank := out.Configs["n1"].Variant.(v11.TankElementConfig)
		Expect(tank.Precision).To(BeEquivalentTo(4))
	})

	It("Should keep the precision of a scale", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{"n1": {
				Variant: v10.ScaleElementConfig{
					ScaleIndicatorConfig: v10.ScaleIndicatorConfig{
						NumericTelemConfig: v10.NumericTelemConfig{Precision: 5},
					},
				},
			}},
		}))
		scale := out.Configs["n1"].Variant.(v11.ScaleElementConfig)
		Expect(scale.Precision).To(BeEquivalentTo(5))
	})
})
