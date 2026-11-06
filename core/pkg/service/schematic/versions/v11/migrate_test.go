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
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	text "github.com/synnaxlabs/x/text/versions/v0"
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

	It("Should fill the stroke width of a stored edge", func(ctx SpecContext) {
		pipe := migrateStored(ctx, stored(msgpack.EncodedJSON{"variant": "pipe"}))
		Expect(pipe).To(HaveField("StrokeWidth", BeEquivalentTo(2)))
	})

	It("Should keep the segments of a stored edge", func(ctx SpecContext) {
		edge := migrateStored(ctx, stored(msgpack.EncodedJSON{
			"variant":  "hydraulic",
			"segments": []any{map[string]any{"direction": "x", "length": 40.0}},
		}))
		Expect(edge).To(HaveField("Segments", HaveLen(1)))
	})

	It("Should keep a stored symbol 0 and fill an edge in one migration", func(
		ctx SpecContext,
	) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		old := v10.Schematic{
			Key:  uuid.New(),
			Name: "Feed system",
			Configs: map[string]v10.ElementConfig{
				"n1": stored(msgpack.EncodedJSON{
					"variant": "circle", "stroke_width": 0.0,
				}),
				"e1": stored(msgpack.EncodedJSON{"variant": "pipe"}),
			},
		}
		MustSucceed(
			gorp.OpenTable(ctx, gorp.TableConfig[v10.Key, v10.Schematic]{DB: db}),
		)
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
		Expect(got.Configs["n1"].Variant).To(HaveField("StrokeWidth", zero))
		Expect(got.Configs["e1"].Variant).To(
			HaveField("StrokeWidth", BeEquivalentTo(2)),
		)
	})

	It(
		"Should set a stored button's level and stop its wrapping",
		func(ctx SpecContext) {
			Expect(migrateStored(ctx, stored(msgpack.EncodedJSON{
				"variant": "button",
				"size":    "large",
				"label":   map[string]any{"max_inline_size": 80.0},
			}))).To(SatisfyAll(
				HaveField("Level", Equal(text.LevelH5)),
				HaveField("LabeledConfig.Label.MaxInlineSize", HaveValue(BeZero())),
			))
		},
	)

	It("Should keep a button's 0 through a defaults fill", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{
				"n1": stored(msgpack.EncodedJSON{"variant": "button"}),
			},
		}))
		out.ApplyDefaults()
		Expect(out.Configs["n1"].Variant).To(
			HaveField("LabeledConfig.Label.MaxInlineSize", HaveValue(BeZero())),
		)
	})

	It("Should keep a level that a stored button holds", func(ctx SpecContext) {
		Expect(migrateStored(ctx, stored(msgpack.EncodedJSON{
			"variant": "button",
			"size":    "large",
			"level":   "h3",
		}))).To(HaveField("Level", Equal(text.LevelH3)))
	})

	It("Should keep the wrap width of a stored field", func(ctx SpecContext) {
		Expect(migrateStored(ctx, stored(msgpack.EncodedJSON{
			"variant": "setpoint",
			"size":    "small",
			"label":   map[string]any{"max_inline_size": 80.0},
		}))).To(SatisfyAll(
			HaveField("Level", Equal(text.LevelSmall)),
			HaveField(
				"LabeledConfig.Label.MaxInlineSize",
				HaveValue(BeEquivalentTo(80)),
			),
		))
	})

	It("Should keep a stored default as a set value", func(ctx SpecContext) {
		box := migrateStored(ctx, stored(msgpack.EncodedJSON{"variant": "box"}))
		Expect(box).To(SatisfyAll(
			HaveField("BorderRadius", HaveValue(BeEquivalentTo(3))),
			HaveField("StrokeWidth", HaveValue(BeEquivalentTo(2))),
		))
	})

	It("Should fill the v10 defaults of an imported config", func(ctx SpecContext) {
		got := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{
				"n1": {Variant: v10.LineElementConfig{}},
			},
		}))
		Expect(got.Configs["n1"].Variant).To(SatisfyAll(
			HaveField("End", HaveValue(Equal(spatial.XY{X: 100}))),
			HaveField("StrokeWidth", HaveValue(BeEquivalentTo(2))),
		))
	})

	It("Should fill the stroke width of an imported edge", func(ctx SpecContext) {
		got := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{
				"e1": {Variant: v10.PipeElementConfig{}},
			},
		}))
		Expect(got.Configs["e1"].Variant).To(
			HaveField("StrokeWidth", BeEquivalentTo(2)),
		)
	})

	DescribeTable("Should stop the wrapping of a saved button",
		func(ctx SpecContext, label map[string]any) {
			out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
				Configs: map[string]v10.ElementConfig{
					"n1": stored(msgpack.EncodedJSON{
						"variant": "button",
						"label":   label,
					}),
				},
			}))
			button := out.Configs["n1"].Variant.(v11.ButtonElementConfig)
			Expect(button.Label.MaxInlineSize).To(HaveValue(BeZero()))
			Expect(button.Label.Label).To(Equal("Open"))
		},
		Entry("with the default", map[string]any{"label": "Open"}),
		Entry("with a chosen width", map[string]any{
			"label":           "Open",
			"max_inline_size": 80.0,
		}),
	)

	DescribeTable("Should keep the wrap width of a symbol that is not a button",
		func(ctx SpecContext, label map[string]any, width float64) {
			out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
				Configs: map[string]v10.ElementConfig{
					"n1": stored(
						msgpack.EncodedJSON{"variant": "value", "label": label},
					),
				},
			}))
			value := out.Configs["n1"].Variant.(v11.ValueElementConfig)
			Expect(value.Label.MaxInlineSize).To(HaveValue(BeEquivalentTo(width)))
		},
		Entry("with the default", map[string]any{"label": "P1"}, 150.0),
		Entry("with a chosen width", map[string]any{
			"label":           "P1",
			"max_inline_size": 80.0,
		}, 80.0),
	)

	DescribeTable("Should keep the rendered text size of a saved symbol",
		func(ctx SpecContext, size string, level text.Level) {
			sized := func(variant string) v10.ElementConfig {
				return stored(msgpack.EncodedJSON{"variant": variant, "size": size})
			}
			out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
				Configs: map[string]v10.ElementConfig{
					"button":   sized("button"),
					"input":    sized("input"),
					"select":   sized("select"),
					"setpoint": sized("setpoint"),
				},
			}))
			c := out.Configs
			Expect(c["button"].Variant).To(HaveField("Level", Equal(level)))
			Expect(c["input"].Variant).To(HaveField("Level", Equal(level)))
			Expect(c["select"].Variant).To(HaveField("Level", Equal(level)))
			Expect(c["setpoint"].Variant).To(HaveField("Level", Equal(level)))
		},
		Entry("tiny", "tiny", text.LevelTiny),
		Entry("small", "small", text.LevelSmall),
		Entry("medium", "medium", text.LevelP),
		Entry("large", "large", text.LevelH5),
		Entry("huge", "huge", text.LevelH2),
	)

	It("Should keep a level that a saved button stores", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{
				"n1": stored(msgpack.EncodedJSON{
					"variant": "button",
					"size":    "large",
					"level":   "h3",
				}),
			},
		}))
		button := out.Configs["n1"].Variant.(v11.ButtonElementConfig)
		Expect(button.Level).To(Equal(text.LevelH3))
	})

	It("Should keep the schematic's identity", func(ctx SpecContext) {
		key := uuid.New()
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Key:      key,
			Name:     "Feed system",
			Snapshot: true,
		}))
		Expect(out.Key).To(Equal(key))
		Expect(out.Name).To(Equal("Feed system"))
		Expect(out.Snapshot).To(BeTrue())
	})

	It("Should keep the zero fields of an element config", func(ctx SpecContext) {
		old := stored(msgpack.EncodedJSON{"variant": "polygon", "stroke_width": 0.0})
		out := MustSucceed(v11.MigrateElementConfig(ctx, old))
		variant := out.Variant.(v11.PolygonElementConfig)
		Expect(variant.StrokeWidth).To(HaveValue(BeZero()))
	})

	It("Should keep a zero numeric telemetry precision", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateNumericTelemConfig(ctx, v10.NumericTelemConfig{}))
		Expect(out.Precision).To(HaveValue(BeZero()))
	})

	DescribeTable("Should keep the wrap width of a label config",
		func(ctx SpecContext, width float64) {
			out := MustSucceed(v11.MigrateLabelConfig(
				ctx, v10.LabelConfig{MaxInlineSize: width},
			))
			Expect(out.MaxInlineSize).To(HaveValue(Equal(width)))
		},
		Entry("No wrapping", 0.0),
		Entry("A set width", 220.0),
	)

	DescribeTable("Should round a stored precision into a byte",
		func(ctx SpecContext, stored float64, want uint8) {
			out := MustSucceed(v11.MigrateNumericTelemConfig(
				ctx, v10.NumericTelemConfig{Precision: stored},
			))
			Expect(out.Precision).To(HaveValue(Equal(want)))
		},
		Entry("whole", 3.0, uint8(3)),
		Entry("fraction", 2.6, uint8(3)),
		Entry("negative", -1.0, uint8(0)),
		Entry("too large", 300.0, uint8(255)),
	)

	It("Should keep zero scale indicator bounds", func(ctx SpecContext) {
		out := MustSucceed(
			v11.MigrateScaleIndicatorConfig(ctx, v10.ScaleIndicatorConfig{}),
		)
		Expect(out.Bounds).To(HaveValue(BeZero()))
	})

	It("Should not reuse a v10 migration key", func() {
		Expect(v11.Migration.Key()).ToNot(Equal(v10.Migration.Key()))
	})
})

var _ = Describe("MigrateSegmentedEdgeConfig", func() {
	It("Should fill the default stroke width", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateSegmentedEdgeConfig(
			ctx, v10.SegmentedEdgeConfig{},
		))
		Expect(out.StrokeWidth).To(BeEquivalentTo(2))
	})

	It("Should keep the stroke color and segments", func(ctx SpecContext) {
		stroke := color.Color{R: 255, A: 1}
		segments := []v10.Segment{{Direction: spatial.DirectionX, Length: 40}}
		out := MustSucceed(v11.MigrateSegmentedEdgeConfig(
			ctx, v10.SegmentedEdgeConfig{StrokeColor: &stroke, Segments: segments},
		))
		Expect(out.StrokeColor).To(HaveValue(Equal(stroke)))
		Expect(out.Segments).To(Equal(segments))
	})
})
