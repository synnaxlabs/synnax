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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/alamos"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v2"
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
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

// labeled holds the LabeledConfig values ApplyDefaults fills in, including the nested
// label defaults every symbol carries.
var labeled = v9.LabeledConfig{
	Label: v9.LabelConfig{
		Level:         "h5",
		Orientation:   "top",
		Direction:     "x",
		MaxInlineSize: 150,
		Align:         "center",
	},
	Orientation: "left",
	Scale:       1,
}

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
			LabeledConfig:    labeled,
			StalenessTimeout: 5,
			Color:            new(MustSucceed(color.FromHex("#ff0000"))),
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
			Color:    new(MustSucceed(color.FromHex("#0000ff"))),
			Segments: []v9.Segment{{Direction: "x", Length: 10}},
		}))
	})

	// The Console wrote its configs verbatim, so stored entries carry camelCase keys
	// and camelCase variant discriminators.
	It("Should normalize the camelCase the Console wrote", func(ctx SpecContext) {
		Expect(typed(ctx, msgpack.EncodedJSON{
			"variant":   "stringDisplay",
			"textColor": "#00ff00",
		})).To(Equal(v9.StringDisplayElementConfig{
			LabeledConfig:    labeled,
			Level:            "h4",
			InlineSize:       100,
			StalenessTimeout: 5,
			TextColor:        new(MustSucceed(color.FromHex("#00ff00"))),
		}))
	})

	// v8 stored a whole telem pipeline spec; v9 stores the arguments the pipeline
	// was built from.
	pipeline := func(segments map[string]map[string]any) map[string]any {
		segs := make(map[string]any, len(segments))
		for name, props := range segments {
			segs[name] = map[string]any{"props": props}
		}
		return map[string]any{"props": map[string]any{"segments": segs}}
	}
	valueStream := map[string]map[string]any{
		"valueStream":    {"channel": 7.0},
		"rollingAverage": {"windowSize": 5.0},
	}
	setter := map[string]map[string]any{"setter": {"channel": 8.0}}
	DescribeTable("Should rewrite a stored telem pipeline into its arguments",
		func(
			ctx SpecContext,
			raw msgpack.EncodedJSON,
			check func(v9.ElementConfigVariant),
		) {
			check(typed(ctx, raw))
		},
		Entry("value",
			msgpack.EncodedJSON{"variant": "value", "telem": pipeline(valueStream)},
			func(v v9.ElementConfigVariant) {
				cfg := v.(v9.ValueElementConfig)
				Expect(cfg.Channel).To(HaveValue(BeEquivalentTo(7)))
				Expect(cfg.RollingAverage).To(HaveValue(BeEquivalentTo(5)))
			}),
		Entry("gauge",
			msgpack.EncodedJSON{"variant": "gauge", "telem": pipeline(valueStream)},
			func(v v9.ElementConfigVariant) {
				cfg := v.(v9.GaugeElementConfig)
				Expect(cfg.Channel).To(HaveValue(BeEquivalentTo(7)))
				Expect(cfg.RollingAverage).To(HaveValue(BeEquivalentTo(5)))
			}),
		Entry("string_display",
			msgpack.EncodedJSON{
				"variant": "string_display",
				"telem":   pipeline(valueStream),
			},
			func(v v9.ElementConfigVariant) {
				cfg := v.(v9.StringDisplayElementConfig)
				Expect(cfg.Channel).To(HaveValue(BeEquivalentTo(7)))
			}),
		Entry("light", msgpack.EncodedJSON{
			"variant": "light",
			"source": pipeline(map[string]map[string]any{
				"valueStream": {"channel": 7.0},
				"threshold": {
					"trueBound": map[string]any{"lower": 1.0, "upper": 2.0},
				},
			}),
		}, func(v v9.ElementConfigVariant) {
			cfg := v.(v9.LightElementConfig)
			Expect(cfg.Channel).To(HaveValue(BeEquivalentTo(7)))
			Expect(cfg.Threshold).To(
				HaveValue(Equal(spatial.Bounds{Lower: 1, Upper: 2})),
			)
		}),
		Entry("state_indicator",
			msgpack.EncodedJSON{
				"variant": "state_indicator",
				"source":  pipeline(valueStream),
			},
			func(v v9.ElementConfigVariant) {
				cfg := v.(v9.StateIndicatorElementConfig)
				Expect(cfg.Channel).To(HaveValue(BeEquivalentTo(7)))
			}),
		Entry("setpoint", msgpack.EncodedJSON{
			"variant": "setpoint",
			"source":  pipeline(valueStream),
			"sink":    pipeline(setter),
		}, func(v v9.ElementConfigVariant) {
			cfg := v.(v9.SetpointElementConfig)
			Expect(cfg.CommandChannel).To(HaveValue(BeEquivalentTo(8)))
		}),
		Entry("button",
			msgpack.EncodedJSON{"variant": "button", "sink": pipeline(setter)},
			func(v v9.ElementConfigVariant) {
				cfg := v.(v9.ButtonElementConfig)
				Expect(cfg.CommandChannel).To(HaveValue(BeEquivalentTo(8)))
			}),
		Entry("input",
			msgpack.EncodedJSON{"variant": "input", "sink": pipeline(setter)},
			func(v v9.ElementConfigVariant) {
				cfg := v.(v9.InputElementConfig)
				Expect(cfg.CommandChannel).To(HaveValue(BeEquivalentTo(8)))
			}),
		Entry("toggle", msgpack.EncodedJSON{
			"variant": "valve",
			"source":  pipeline(valueStream),
			"sink":    pipeline(setter),
		}, func(v v9.ElementConfigVariant) {
			cfg := v.(v9.ValveElementConfig)
			Expect(cfg.StateChannel).To(HaveValue(BeEquivalentTo(7)))
			Expect(cfg.CommandChannel).To(HaveValue(BeEquivalentTo(8)))
		}),
		Entry("control chip authority", msgpack.EncodedJSON{
			"variant": "valve",
			"control": map[string]any{
				"chip": map[string]any{"sink": map[string]any{
					"props": map[string]any{"authority": 200.0},
				}},
				"indicator": map[string]any{},
			},
		}, func(v v9.ElementConfigVariant) {
			cfg := v.(v9.ValveElementConfig)
			Expect(cfg.Control).ToNot(BeNil())
			Expect(cfg.Control.Authority).To(HaveValue(BeEquivalentTo(200)))
		}),
	)

	DescribeTable("Should drop a zero channel whatever width msgpack decoded it to",
		func(ctx SpecContext, zero any) {
			cfg, ok := typed(ctx, msgpack.EncodedJSON{
				"variant": "value",
				"telem": pipeline(map[string]map[string]any{
					"valueStream": {"channel": zero},
				}),
			}).(v9.ValueElementConfig)
			Expect(ok).To(BeTrue())
			Expect(cfg.Channel).To(BeNil())
		},
		Entry("float64", 0.0),
		Entry("int8", int8(0)),
		Entry("int64", int64(0)),
		Entry("uint64", uint64(0)),
	)

	// v8 stored an off-page reference's target as a bare schematic key; v9 stores a
	// typed page reference, and an empty key meant no page.
	It("Should lift a legacy page key into a schematic page reference", func(
		ctx SpecContext,
	) {
		cfg, ok := typed(ctx, msgpack.EncodedJSON{
			"variant": "offPageReference",
			"page":    "abc",
		}).(v9.OffPageReferenceElementConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Page).To(HaveValue(Equal(v9.Page{Type: "schematic", Key: "abc"})))
	})

	It("Should drop an empty legacy page key", func(ctx SpecContext) {
		cfg, ok := typed(ctx, msgpack.EncodedJSON{
			"variant": "offPageReference",
			"page":    "",
		}).(v9.OffPageReferenceElementConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Page).To(BeNil())
	})

	// Consoles before v9 stored transparent black for an unchosen color. v9 stores no
	// color at all, so the theme picks it.
	DescribeTable("Should drop a stored zero color",
		func(ctx SpecContext, stored any) {
			cfg, ok := typed(ctx, msgpack.EncodedJSON{
				"variant": "valve",
				"color":   stored,
			}).(v9.ValveElementConfig)
			Expect(ok).To(BeTrue())
			Expect(cfg.Color).To(BeNil())
		},
		Entry("array", []any{0.0, 0.0, 0.0, 0.0}),
		Entry("hex", "#00000000"),
		Entry("object", map[string]any{"r": 0.0, "g": 0.0, "b": 0.0, "a": 0.0}),
		Entry("null", nil),
	)

	It("Should keep a chosen color", func(ctx SpecContext) {
		cfg, ok := typed(ctx, msgpack.EncodedJSON{
			"variant": "valve",
			"color":   "#ff000080",
		}).(v9.ValveElementConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Color).To(HaveValue(Equal(MustSucceed(color.FromHex("#ff000080")))))
	})

	It("Should drop zero colors nested in a symbol's indicator", func(ctx SpecContext) {
		cfg, ok := typed(ctx, msgpack.EncodedJSON{
			"variant":         "tank",
			"backgroundColor": []any{0.0, 0.0, 0.0, 0.0},
			"fill": map[string]any{
				"color":     []any{0.0, 0.0, 0.0, 0.0},
				"axisColor": "#00ff00",
			},
		}).(v9.TankElementConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.BackgroundColor).To(BeNil())
		Expect(cfg.Fill.Color).To(BeNil())
		green := MustSucceed(color.FromHex("#00ff00"))
		Expect(cfg.Fill.AxisColor).To(HaveValue(Equal(green)))
	})

	It("Should keep a zero color on a gradient stop", func(ctx SpecContext) {
		cfg, ok := typed(ctx, msgpack.EncodedJSON{
			"variant": "value",
			"redline": map[string]any{
				"bounds": map[string]any{"lower": 0.0, "upper": 1.0},
				"gradient": []any{
					map[string]any{"key": "a", "color": "#00000000", "position": 0.0},
				},
			},
		}).(v9.ValueElementConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Redline.Gradient).To(HaveLen(1))
		Expect(cfg.Redline.Gradient[0].Color).To(Equal(color.Color{}))
	})

	It("Should drop a stored zero color on an edge", func(ctx SpecContext) {
		cfg, ok := typed(ctx, msgpack.EncodedJSON{
			"variant":  "pipe",
			"color":    []any{0.0, 0.0, 0.0, 0.0},
			"segments": []any{},
		}).(v9.PipeElementConfig)
		Expect(ok).To(BeTrue())
		Expect(cfg.Color).To(BeNil())
	})

	// A v8 config predates every schema default, so the lift is the only place the
	// stored entry can pick them up.
	It("Should fill schema defaults the stored config never carried", func(
		ctx SpecContext,
	) {
		Expect(typed(ctx, msgpack.EncodedJSON{"variant": "value"})).To(
			Equal(v9.ValueElementConfig{
				LabeledConfig:    labeled,
				Level:            "h4",
				InlineSize:       70,
				StalenessTimeout: 5,
				Notation:         "standard",
				Precision:        2,
				Units:            "psi",
				Redline: v9.Redline{
					Bounds: spatial.Bounds{Lower: 0, Upper: 1},
				},
				Location: spatial.LocationXY{X: "left", Y: "center"},
			}),
		)
	})

	It("Should type a custom symbol's state overrides", func(ctx SpecContext) {
		Expect(typed(ctx, msgpack.EncodedJSON{
			"variant": "customStatic",
			"specKey": "spec",
			"stateOverrides": []any{map[string]any{
				"key":  "on",
				"name": "On",
				"regions": []any{map[string]any{
					"key":         "body",
					"name":        "Body",
					"selectors":   []any{"#body"},
					"strokeColor": "#ff0000",
					"fillColor":   []any{0, 0, 0, 0},
				}},
			}},
		})).To(Equal(v9.CustomStaticElementConfig{
			LabeledConfig: labeled,
			SpecKey:       "spec",
			StateOverrides: []symbol.State{{
				Key:  "on",
				Name: "On",
				Regions: []symbol.Region{{
					Key:         "body",
					Name:        "Body",
					Selectors:   []string{"#body"},
					StrokeColor: new(MustSucceed(color.FromHex("#ff0000"))),
				}},
			}},
		}))
	})

	It("Should reset a config its variant cannot hold to the variant's defaults", func(
		ctx SpecContext,
	) {
		Expect(typed(ctx, msgpack.EncodedJSON{
			"variant": "circle", "radius": "wide",
		})).To(Equal(typed(ctx, msgpack.EncodedJSON{"variant": "circle"})))
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
			"n1": {Variant: v9.ValveElementConfig{
				ToggleSymbolConfig: v9.ToggleSymbolConfig{
					ToggleConfig: v9.ToggleConfig{
						LabeledConfig:    labeled,
						StalenessTimeout: 5,
					},
				},
			}},
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

	It("Should reject a config carrying a value the variant cannot decode", func(
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
	It("Should log every config it resets or drops", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		seed := v8.Schematic{
			Key:   uuid.New(),
			Name:  "Stored",
			Nodes: []v8.Node{{Key: "a"}, {Key: "b"}, {Key: "c"}},
			Configs: map[string]msgpack.EncodedJSON{
				"a": {"variant": "circle", "radius": "wide"},
				"b": {"variant": "not-a-symbol"},
				"c": {"variant": "valve"},
			},
		}
		MustSucceed(gorp.OpenTable(ctx, gorp.TableConfig[v8.Key, v8.Schematic]{DB: db}))
		Expect(gorp.NewCreate[v8.Key, v8.Schematic]().
			Entry(&seed).Exec(ctx, db)).To(Succeed())
		core, logs := observer.New(zapcore.WarnLevel)
		logger := MustSucceed(alamos.NewLogger(alamos.LoggerConfig{
			ZapLogger: zap.New(core),
		}))
		Expect(gorp.Migrate(ctx, gorp.MigrateConfig{
			Instrumentation: alamos.New("test", alamos.WithLogger(logger)),
			DB:              db,
			Namespace:       "Schematic",
			Migrations:      []migrate.Migration{v9.Migration},
		})).To(Succeed())
		var got v9.Schematic
		Expect(gorp.NewRetrieve[v9.Key, v9.Schematic]().
			Where(gorp.MatchKeys[v9.Key, v9.Schematic](seed.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		fresh := MustSucceed(v9.MigrateSchematic(ctx, v8.Schematic{
			Configs: map[string]msgpack.EncodedJSON{"a": {"variant": "circle"}},
		}))
		Expect(got.Configs).To(HaveLen(2))
		Expect(got.Configs["a"]).To(Equal(fresh.Configs["a"]))
		Expect(got.Configs).To(HaveKey("c"))
		Expect(logs.Len()).To(Equal(2))
		reset := logs.FilterMessage(
			"reset a rejected schematic config to its variant's defaults",
		).All()
		Expect(reset).To(HaveLen(1))
		Expect(reset[0].ContextMap()).To(HaveKeyWithValue("node", "a"))
		Expect(reset[0].ContextMap()).To(
			HaveKeyWithValue("schematic", seed.Key.String()),
		)
		dropped := logs.FilterMessage(
			"dropped a schematic config naming no known variant",
		).All()
		Expect(dropped).To(HaveLen(1))
		Expect(dropped[0].ContextMap()).To(HaveKeyWithValue("node", "b"))
	})

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
			"a": {Variant: v9.ValveElementConfig{
				ToggleSymbolConfig: v9.ToggleSymbolConfig{
					ToggleConfig: v9.ToggleConfig{
						LabeledConfig:    labeled,
						StalenessTimeout: 5,
					},
				},
			}},
		}))
	})

	It("Should not reuse v8's migration key", func() {
		Expect(v9.Migration.Key()).ToNot(Equal(v8.Migration.Key()))
	})
})
