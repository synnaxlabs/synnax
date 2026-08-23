// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v8_test

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v0"
	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	v8 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v8"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("MigrateNode", func() {
	It("Should drop the node's measured dimensions", func(ctx SpecContext) {
		got := MustSucceed(v8.MigrateNode(ctx, v7.Node{
			Key:      "n1",
			Position: spatial.XY{X: 1.5, Y: -2.5},
			ZIndex:   3,
			Measured: spatial.Dimensions{Width: 125, Height: 250},
		}))
		Expect(got).To(Equal(v8.Node{
			Key:      "n1",
			Position: spatial.XY{X: 1.5, Y: -2.5},
			ZIndex:   3,
		}))
	})
})

var _ = Describe("MigrateSchematic", func() {
	It("Should drop measured from every node", func(ctx SpecContext) {
		key := uuid.New()
		got := MustSucceed(v8.MigrateSchematic(ctx, v7.Schematic{
			Key:      key,
			Name:     "Press Panel",
			Snapshot: true,
			Nodes: []v7.Node{
				{
					Key:      "valve",
					Position: spatial.XY{X: 10, Y: 20},
					ZIndex:   1,
					Measured: spatial.Dimensions{Width: 100, Height: 50},
				},
				{
					Key:      "tank",
					Position: spatial.XY{X: 30, Y: 40},
					Measured: spatial.Dimensions{Width: 200, Height: 300},
				},
			},
			Edges: []v7.Edge{{
				Key:    "e1",
				Source: v7.Handle{Node: "valve", Param: "outlet"},
				Target: v7.Handle{Node: "tank", Param: "inlet"},
			}},
			Configs: map[string]msgpack.EncodedJSON{
				"valve": {"variant": "valve"},
			},
		}))
		Expect(got).To(Equal(v8.Schematic{
			Key:      key,
			Name:     "Press Panel",
			Snapshot: true,
			Nodes: []v8.Node{
				{Key: "valve", Position: spatial.XY{X: 10, Y: 20}, ZIndex: 1},
				{Key: "tank", Position: spatial.XY{X: 30, Y: 40}},
			},
			Edges: []v7.Edge{{
				Key:    "e1",
				Source: v7.Handle{Node: "valve", Param: "outlet"},
				Target: v7.Handle{Node: "tank", Param: "inlet"},
			}},
			Configs: map[string]v8.ElementConfig{
				"valve": {Variant: v8.ValveElementConfig{}},
			},
		}))
	})
})

var _ = Describe("Migration", func() {
	It("Should rewrite v7-encoded entries through the v8 codec", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		seed := v7.Schematic{
			Key:  uuid.New(),
			Name: "Stored",
			Nodes: []v7.Node{
				{
					Key:      "a",
					Position: spatial.XY{X: 1, Y: 2},
					ZIndex:   4,
					Measured: spatial.Dimensions{Width: 80, Height: 40},
				},
				{
					Key:      "b",
					Position: spatial.XY{X: 3, Y: 4},
					Measured: spatial.Dimensions{Width: 60, Height: 30},
				},
			},
			Edges: []v7.Edge{{
				Key:    "e1",
				Source: v7.Handle{Node: "a", Param: "outlet"},
				Target: v7.Handle{Node: "b", Param: "inlet"},
			}},
			Configs: map[string]msgpack.EncodedJSON{"a": {"variant": "valve"}},
		}
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v7.Key, v7.Schematic]{DB: db}))
		Expect(gorp.NewCreate[v7.Key, v7.Schematic]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())
		noop := func(context.Context, gorp.Tx, alamos.Instrumentation) error {
			return nil
		}
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:        db,
			Namespace: "Schematic",
			Migrations: []migrate.Migration{
				gorp.NewMigration(v0.Migration.Key(), noop),
				gorp.NewMigration(v7.Migration.Key(), noop),
				v8.Migration,
			},
		})).To(Succeed())
		var got v8.Schematic
		Expect(gorp.NewRetrieve[v8.Key, v8.Schematic]().
			Where(gorp.MatchKeys[v8.Key, v8.Schematic](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Key).To(Equal(seed.Key))
		Expect(got.Name).To(Equal(seed.Name))
		Expect(got.Nodes).To(Equal([]v8.Node{
			{Key: "a", Position: spatial.XY{X: 1, Y: 2}, ZIndex: 4},
			{Key: "b", Position: spatial.XY{X: 3, Y: 4}},
		}))
		Expect(got.Edges).To(Equal(seed.Edges))
		Expect(got.Configs).To(Equal(map[string]v8.ElementConfig{
			"a": {Variant: v8.ValveElementConfig{}},
		}))
	})
})

var _ = Describe("Config typing", func() {
	// typed runs a single v7 config entry through the v8 migration and returns the
	// decoded union variant, failing the spec when the entry was dropped.
	typed := func(ctx SpecContext, raw msgpack.EncodedJSON) v8.ElementConfigVariant {
		GinkgoHelper()
		out := MustSucceed(v8.MigrateSchematic(ctx, v7.Schematic{
			Configs: map[string]msgpack.EncodedJSON{"n1": raw},
		}))
		cfg, ok := out.Configs["n1"]
		Expect(ok).To(BeTrue(), "config was dropped")
		return cfg.Variant
	}

	It("Should decode a node config into its variant", func(ctx SpecContext) {
		Expect(typed(ctx, msgpack.EncodedJSON{
			"variant": "valve",
			"color":   "#ff0000",
		})).To(Equal(v8.ValveElementConfig{
			ToggleSymbolConfig: v8.ToggleSymbolConfig{
				Color: new(MustSucceed(color.FromHex("#ff0000"))),
			},
		}))
	})

	It("Should decode a segmented edge config", func(ctx SpecContext) {
		Expect(typed(ctx, msgpack.EncodedJSON{
			"variant": "pipe",
			"color":   "#0000ff",
			"segments": []any{
				map[string]any{"direction": "x", "length": 10.0},
			},
		})).To(Equal(v8.PipeElementConfig{
			SegmentedEdgeConfig: v8.SegmentedEdgeConfig{
				Color:    new(MustSucceed(color.FromHex("#0000ff"))),
				Segments: []v8.Segment{{Direction: "x", Length: 10}},
			},
		}))
	})

	// The Console wrote its configs verbatim, so stored entries carry camelCase keys
	// and camelCase variant discriminators.
	It("Should normalize the camelCase the Console wrote", func(ctx SpecContext) {
		Expect(typed(ctx, msgpack.EncodedJSON{
			"variant":   "stringDisplay",
			"textColor": "#00ff00",
		})).To(Equal(v8.StringDisplayElementConfig{
			TextColor: new(MustSucceed(color.FromHex("#00ff00"))),
		}))
	})

	// v7 stored a whole telem pipeline spec; v8 stores the channel key the pipeline
	// was built from.
	It("Should rewrite a stored telem pipeline into its arguments", func(
		ctx SpecContext,
	) {
		cfg, ok := typed(ctx, msgpack.EncodedJSON{
			"variant": "value",
			"telem": map[string]any{"props": map[string]any{
				"segments": map[string]any{
					"valueStream": map[string]any{
						"props": map[string]any{"channel": 65537.0},
					},
					"rollingAverage": map[string]any{
						"props": map[string]any{"windowSize": 5.0},
					},
				},
			}},
		}).(v8.ValueElementConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Channel).To(HaveValue(BeEquivalentTo(65537)))
		Expect(cfg.RollingAverage).To(HaveValue(BeEquivalentTo(5)))
	})

	It("Should drop an entry naming no known variant", func(ctx SpecContext) {
		out := MustSucceed(v8.MigrateSchematic(ctx, v7.Schematic{
			Configs: map[string]msgpack.EncodedJSON{
				"n1": {"variant": "not-a-symbol"},
				"n2": {"variant": "valve"},
			},
		}))
		Expect(out.Configs).To(SatisfyAll(HaveLen(1), HaveKey("n2")))
	})
})
