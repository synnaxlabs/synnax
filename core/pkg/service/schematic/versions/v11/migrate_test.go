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
	"github.com/onsi/gomega/types"
	"github.com/synnaxlabs/gorp"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	"github.com/synnaxlabs/x/border"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

// stored decodes a v10 config the way the v10 Core persists it.
func stored(raw msgpack.EncodedJSON) v10.ElementConfig {
	GinkgoHelper()
	return MustSucceed(v10.DecodeWithDefaults(raw))
}

// migrateStored writes a v10 schematic holding cfg to an in-memory table, runs the v11
// migration, and returns the migrated config's variant.
func migrateStored(ctx SpecContext, cfg v10.ElementConfig) v11.ElementConfigVariant {
	GinkgoHelper()
	db := DeferClose(gorp.Wrap(memkv.New()))
	old := v10.Schematic{
		Key:     uuid.New(),
		Name:    "Feed system",
		Configs: map[string]v10.ElementConfig{"n1": cfg},
	}
	MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v10.Key, v10.Schematic]{DB: db}))
	Expect(gorp.NewCreate[v10.Key, v10.Schematic]().
		Entry(&old).Exec(ctx, db)).To(Succeed())
	Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
		DB:         db,
		Namespace:  "Schematic",
		Migrations: []gorp.Migration{v11.Migration},
	})).To(Succeed())
	var got v11.Schematic
	Expect(gorp.NewRetrieve[v11.Key, v11.Schematic]().
		Where(gorp.MatchKeys[v11.Key, v11.Schematic](old.Key)).
		Entry(&got).Exec(ctx, db)).To(Succeed())
	Expect(got.Name).To(Equal("Feed system"))
	return got.Configs["n1"].Variant
}

var (
	zero        = HaveValue(BeZero())
	zeroBounds  = map[string]any{"lower": -10.0, "upper": 0.0}
	keptBounds  = HaveValue(Equal(spatial.Bounds{Lower: -10}))
	zeroCorner  = map[string]any{"x": 0.0, "y": 0.0}
	zeroCorners = map[string]any{
		"top_left":     zeroCorner,
		"top_right":    zeroCorner,
		"bottom_left":  zeroCorner,
		"bottom_right": zeroCorner,
	}
)

var _ = Describe("Migration", func() {
	DescribeTable("Should keep a stored 0 in every field v11 makes optional",
		func(ctx SpecContext, raw msgpack.EncodedJSON, matcher types.GomegaMatcher) {
			Expect(migrateStored(ctx, stored(raw))).To(matcher)
		},
		Entry("box", msgpack.EncodedJSON{
			"variant": "box", "border_radius": 0.0, "stroke_width": 0.0,
		}, SatisfyAll(HaveField("BorderRadius", zero), HaveField("StrokeWidth", zero))),
		Entry("circle", msgpack.EncodedJSON{"variant": "circle", "stroke_width": 0.0},
			HaveField("StrokeWidth", zero)),
		Entry("gauge", msgpack.EncodedJSON{
			"variant": "gauge", "precision": 0.0, "bounds": zeroBounds,
		}, SatisfyAll(
			HaveField("NumericTelemConfig.Precision", zero),
			HaveField("Bounds", keptBounds),
		)),
		Entry("line", msgpack.EncodedJSON{
			"variant":      "line",
			"end":          map[string]any{"x": 0.0, "y": 40.0},
			"stroke_width": 0.0,
		}, SatisfyAll(
			HaveField("End", HaveValue(Equal(spatial.XY{Y: 40}))),
			HaveField("StrokeWidth", zero),
		)),
		Entry("polygon", msgpack.EncodedJSON{"variant": "polygon", "stroke_width": 0.0},
			HaveField("StrokeWidth", zero)),
		Entry("scale", msgpack.EncodedJSON{
			"variant": "scale", "precision": 0.0, "bounds": zeroBounds,
		}, SatisfyAll(
			HaveField("ScaleIndicatorConfig.NumericTelemConfig.Precision", zero),
			HaveField("ScaleIndicatorConfig.Bounds", keptBounds),
		)),
		Entry("tank", msgpack.EncodedJSON{
			"variant":       "tank",
			"precision":     0.0,
			"bounds":        zeroBounds,
			"border_radius": zeroCorners,
		}, SatisfyAll(
			HaveField("Precision", zero),
			HaveField("Bounds", keptBounds),
			HaveField("BorderRadius", HaveValue(Equal(border.Radius{}))),
		)),
		Entry("value", msgpack.EncodedJSON{"variant": "value", "precision": 0.0},
			HaveField("NumericTelemConfig.Precision", zero)),
	)

	It("Should keep a stored default as a set value", func(ctx SpecContext) {
		box := migrateStored(ctx, stored(msgpack.EncodedJSON{"variant": "box"}))
		Expect(box).To(SatisfyAll(
			HaveField("BorderRadius", HaveValue(BeEquivalentTo(3))),
			HaveField("StrokeWidth", HaveValue(BeEquivalentTo(2))),
		))
	})

	It("Should not reuse a v10 migration key", func() {
		Expect(v11.Migration.Key()).ToNot(Equal(v10.Migration.Key()))
	})
})
