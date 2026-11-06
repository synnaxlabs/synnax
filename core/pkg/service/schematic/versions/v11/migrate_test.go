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
	"github.com/synnaxlabs/gorp"
	v10 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v10"
	v11 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v11"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	"github.com/synnaxlabs/x/spatial"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

var _ = Describe("MigrateSchematic", func() {
	stored := func(raw msgpack.EncodedJSON) v10.ElementConfig {
		GinkgoHelper()
		return MustSucceed(v10.DecodeWithDefaults(raw))
	}

	It("Should fill the default stroke width of a lifted edge", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{
				"e1": stored(msgpack.EncodedJSON{"variant": "pipe"}),
			},
		}))
		pipe := out.Configs["e1"].Variant.(v11.PipeElementConfig)
		Expect(pipe.StrokeWidth).To(BeEquivalentTo(2))
	})

	It("Should keep the values a v10 edge carries", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{
				"e1": stored(msgpack.EncodedJSON{
					"variant":  "hydraulic",
					"segments": []any{map[string]any{"direction": "x", "length": 40.0}},
				}),
			},
		}))
		edge := out.Configs["e1"].Variant.(v11.HydraulicElementConfig)
		Expect(edge.Segments).To(HaveLen(1))
		Expect(edge.Segments[0].Length).To(BeEquivalentTo(40))
	})

	It("Should keep a node config", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{
				"n1": stored(msgpack.EncodedJSON{"variant": "setpoint", "units": "mV"}),
			},
		}))
		setpoint := out.Configs["n1"].Variant.(v11.SetpointElementConfig)
		Expect(setpoint.Units).To(Equal("mV"))
		Expect(setpoint.InlineSize).To(BeEquivalentTo(100))
	})

	DescribeTable("Should clear the wrap width of a saved button",
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
			Expect(button.Label.MaxInlineSize).To(BeNil())
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

	It("Should leave the text level of a field symbol absent", func(ctx SpecContext) {
		out := MustSucceed(v11.MigrateSchematic(ctx, v10.Schematic{
			Configs: map[string]v10.ElementConfig{
				"n1": stored(msgpack.EncodedJSON{"variant": "input", "size": "large"}),
				"n2": stored(msgpack.EncodedJSON{"variant": "select", "size": "large"}),
				"n3": stored(
					msgpack.EncodedJSON{"variant": "setpoint", "size": "large"},
				),
			},
		}))
		Expect(out.Configs["n1"].Variant.(v11.InputElementConfig).Level).To(BeNil())
		Expect(out.Configs["n2"].Variant.(v11.SelectElementConfig).Level).To(BeNil())
		Expect(out.Configs["n3"].Variant.(v11.SetpointElementConfig).Level).To(BeNil())
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

var _ = Describe("Migration", func() {
	It("Should lift stored schematics to v11", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		old := v10.Schematic{
			Key:  uuid.New(),
			Name: "Feed system",
			Configs: map[string]v10.ElementConfig{
				"e1": MustSucceed(v10.DecodeWithDefaults(msgpack.EncodedJSON{
					"variant": "pipe",
				})),
				"n1": MustSucceed(v10.DecodeWithDefaults(msgpack.EncodedJSON{
					"variant": "button",
				})),
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
		Expect(got.Name).To(Equal("Feed system"))
		pipe := got.Configs["e1"].Variant.(v11.PipeElementConfig)
		Expect(pipe.StrokeWidth).To(BeEquivalentTo(2))
		button := got.Configs["n1"].Variant.(v11.ButtonElementConfig)
		Expect(button.Label.MaxInlineSize).To(BeNil())
	})

	It("Should not reuse a v10 migration key", func() {
		Expect(v11.Migration.Key()).ToNot(Equal(v10.Migration.Key()))
	})
})
