// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v9_test

import (
	"context"

	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	v0 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v0"
	v7 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v7"
	v8 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v8"
	v9 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v9"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/gorp"
	"github.com/synnaxlabs/x/kv/memkv"
	"github.com/synnaxlabs/x/migrate"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Config typing", func() {
	// typed runs a single v8 config entry through the v9 migration and returns the
	// decoded union variant, failing the spec when the entry was dropped.
	typed := func(ctx SpecContext, raw msgpack.EncodedJSON) v9.ElementConfigVariant {
		GinkgoHelper()
		out := MustSucceed(v9.MigrateSchematic(ctx, v8.Schematic{
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
		})).To(Equal(v9.ValveElementConfig{
			ToggleSymbolConfig: v9.ToggleSymbolConfig{
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
		})).To(Equal(v9.PipeElementConfig{
			SegmentedEdgeConfig: v9.SegmentedEdgeConfig{
				Color:    new(MustSucceed(color.FromHex("#0000ff"))),
				Segments: []v9.Segment{{Direction: "x", Length: 10}},
			},
		}))
	})

	// The Console wrote its configs verbatim, so stored entries carry camelCase keys
	// and camelCase variant discriminators.
	It("Should normalize the camelCase the Console wrote", func(ctx SpecContext) {
		Expect(typed(ctx, msgpack.EncodedJSON{
			"variant":   "stringDisplay",
			"textColor": "#00ff00",
		})).To(Equal(v9.StringDisplayElementConfig{
			TextColor: new(MustSucceed(color.FromHex("#00ff00"))),
		}))
	})

	// v8 stored a whole telem pipeline spec; v9 stores the channel key the pipeline
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
		}).(v9.ValueElementConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Channel).To(HaveValue(BeEquivalentTo(65537)))
		Expect(cfg.RollingAverage).To(HaveValue(BeEquivalentTo(5)))
	})

	It("Should drop an entry naming no known variant", func(ctx SpecContext) {
		out := MustSucceed(v9.MigrateSchematic(ctx, v8.Schematic{
			Configs: map[string]msgpack.EncodedJSON{
				"n1": {"variant": "not-a-symbol"},
				"n2": {"variant": "valve"},
			},
		}))
		Expect(out.Configs).To(SatisfyAll(HaveLen(1), HaveKey("n2")))
	})
})

var _ = Describe("ImportSchematic", func() {
	It("Should decode every config the union accepts", func(ctx SpecContext) {
		out := MustSucceed(v9.ImportSchematic(ctx, v8.Schematic{
			Nodes:   []v8.Node{{Key: "n1", Position: spatial.XY{X: 1, Y: 2}}},
			Configs: map[string]msgpack.EncodedJSON{"n1": {"variant": "valve"}},
		}))
		Expect(out.Nodes).To(Equal([]v9.Node{
			{Key: "n1", Position: spatial.XY{X: 1, Y: 2}},
		}))
		Expect(out.Configs).To(Equal(map[string]v9.ElementConfig{
			"n1": {Variant: v9.ValveElementConfig{}},
		}))
	})

	It("Should reject the schematic when a config names no known variant", func(
		ctx SpecContext,
	) {
		Expect(v9.ImportSchematic(ctx, v8.Schematic{
			Configs: map[string]msgpack.EncodedJSON{
				"n1": {"variant": "not-a-symbol"},
				"n2": {"variant": "valve"},
			},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`node n1`)),
			MatchError(ContainSubstring(`unknown variant "not-a-symbol"`)),
		))
	})

	It("Should name every rejected node in one error", func(ctx SpecContext) {
		Expect(v9.ImportSchematic(ctx, v8.Schematic{
			Configs: map[string]msgpack.EncodedJSON{
				"n1": {"variant": "not-a-symbol"},
				"n2": {"variant": "also-not-a-symbol"},
			},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("node n1")),
			MatchError(ContainSubstring("node n2")),
		))
	})

	It("Should reject a config carrying a field the variant cannot hold", func(
		ctx SpecContext,
	) {
		Expect(v9.ImportSchematic(ctx, v8.Schematic{
			Configs: map[string]msgpack.EncodedJSON{
				"n1": {"variant": "circle", "radius": "wide"},
			},
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("node n1")),
		))
	})
})

var _ = Describe("DecodeElementConfig", func() {
	It("Should decode a payload naming a known variant", func() {
		Expect(MustSucceed(v9.DecodeElementConfig(msgpack.EncodedJSON{
			"variant": "valve",
		}))).To(Equal(v9.ElementConfig{Variant: v9.ValveElementConfig{}}))
	})

	// A null payload decodes to a nil variant without erroring, so the guard against it
	// is the only thing keeping an unreadable entry out of the configs map.
	It("Should reject a nil payload", func() {
		Expect(v9.DecodeElementConfig(nil)).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring("names no variant")),
		))
	})

	It("Should reject a payload carrying no variant", func() {
		Expect(v9.DecodeElementConfig(msgpack.EncodedJSON{})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`unknown variant ""`)),
		))
	})

	It("Should reject a payload naming an unknown variant", func() {
		Expect(v9.DecodeElementConfig(msgpack.EncodedJSON{
			"variant": "not-a-symbol",
		})).Error().To(SatisfyAll(
			MatchError(validate.ErrValidation),
			MatchError(ContainSubstring(`unknown variant "not-a-symbol"`)),
		))
	})
})

var _ = Describe("Migration", func() {
	// A Core that ran v0.57 has v8's key in its applied set, so the upgrade must reach
	// stored configs through a key of its own or leave every schematic untyped.
	It("Should type configs a Core already lifted to v8", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		seed := v7.Schematic{
			Key:     uuid.New(),
			Name:    "Stored",
			Nodes:   []v7.Node{{Key: "a", Position: spatial.XY{X: 1, Y: 2}}},
			Configs: map[string]msgpack.EncodedJSON{"a": {"variant": "valve"}},
		}
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v7.Key, v7.Schematic]{DB: db}))
		Expect(gorp.NewCreate[v7.Key, v7.Schematic]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())
		noop := func(context.Context, gorp.Tx, alamos.Instrumentation) error {
			return nil
		}
		chain := []migrate.Migration{
			gorp.NewMigration(v0.Migration.Key(), noop),
			gorp.NewMigration(v7.Migration.Key(), noop),
			v8.Migration,
		}
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB: db, Namespace: "Schematic", Migrations: chain,
		})).To(Succeed())
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			DB:         db,
			Namespace:  "Schematic",
			Migrations: append(chain, v9.Migration),
		})).To(Succeed())
		var got v9.Schematic
		Expect(gorp.NewRetrieve[v9.Key, v9.Schematic]().
			Where(gorp.MatchKeys[v9.Key, v9.Schematic](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Nodes).To(Equal([]v9.Node{
			{Key: "a", Position: spatial.XY{X: 1, Y: 2}},
		}))
		Expect(got.Configs).To(Equal(map[string]v9.ElementConfig{
			"a": {Variant: v9.ValveElementConfig{}},
		}))
	})

	It("Should not reuse v8's migration key", func() {
		Expect(v9.Migration.Key()).ToNot(Equal(v8.Migration.Key()))
	})
})
