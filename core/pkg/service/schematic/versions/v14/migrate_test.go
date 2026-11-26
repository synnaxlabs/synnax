// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v14_test

import (
	"embed"
	"encoding/hex"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	"github.com/synnaxlabs/synnax/pkg/service/enumeration"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	v14 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v14"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/defaults"
	"github.com/synnaxlabs/x/encoding/orc"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

//go:embed testdata/*.hex
var fixtures embed.FS

// loadWire returns the Orc payload stored as hex in testdata. v13_released.hex is a v13
// schematic with a select and a state indicator, encoded by the core/v0.62.0 codec.
// v13_released_custom.hex is a v13 schematic with a custom_actuator and a custom_static
// node that carry state overrides, stored by a core/v0.62.0 Core.
func loadWire(name string) []byte {
	GinkgoHelper()
	raw := MustSucceed(fixtures.ReadFile("testdata/" + name))
	return MustSucceed(hex.DecodeString(strings.ReplaceAll(string(raw), "\n", "")))
}

func shade(r uint8) *xcolor.Color { return &xcolor.Color{R: r, A: 255} }

func fixed(r uint8) *servicecolor.Ref {
	return &servicecolor.Ref{Variant: servicecolor.FixedRef{Value: *shade(r)}}
}

// migrate opens the v14 table on db, which runs the v13 to v14 migration with the
// interner of a real enumeration service.
func migrate(ctx SpecContext, db *gorp.DB) *enumeration.Service {
	GinkgoHelper()
	enums := MustOpen(enumeration.OpenService(ctx, enumeration.ServiceConfig{DB: db}))
	MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v14.Key, v14.Schematic]{
		DB: db,
		Migrations: []gorp.Migration{v14.NewMigration(v14.MigrationConfig{
			Intern: func(tx gorp.Tx) symbol.Intern {
				return enums.NewWriter(tx).NewInterner().Intern
			},
		})},
	}))
	return enums
}

var _ = Describe("Migration", func() {
	It("Should lift a schematic that v0.62 stored to v14", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		StartRun(db.Run)
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v13.Key, v13.Schematic]{DB: db}))
		old := v13.Schematic{
			Key:  uuid.New(),
			Name: "Engine",
			Nodes: []v13.Node{
				{Key: "box", Config: v13.NodeConfig{Variant: defaults.Fill(
					v13.BoxNodeConfig{StrokeColor: shade(1)},
				)}},
				{Key: "value", Config: v13.NodeConfig{Variant: defaults.Fill(
					v13.ValueNodeConfig{
						StrokeColor:      shade(2),
						TextColor:        shade(3),
						StalenessColor:   shade(4),
						StalenessTimeout: 3 * telem.Second,
					},
				)}},
			},
			Edges: []v13.Edge{{
				Key:    "pipe",
				Source: v13.Handle{Node: "box", Param: "top"},
				Target: v13.Handle{Node: "value", Param: "bottom"},
				Config: v13.EdgeConfig{Variant: v13.PipeEdgeConfig{
					SegmentedEdgeConfig: v13.SegmentedEdgeConfig{
						StrokeColor: shade(5),
						Segments:    []v13.Segment{},
						StrokeWidth: 3,
					},
				}},
			}},
		}
		Expect(gorp.NewCreate[v13.Key, v13.Schematic]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		migrate(ctx, db)
		var got v14.Schematic
		Expect(gorp.NewRetrieve[v14.Key, v14.Schematic]().
			Where(gorp.MatchKeys[v14.Key, v14.Schematic](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(got.Name).To(Equal("Engine"))
		Expect(got.Nodes).To(HaveLen(2))
		box, ok := got.Nodes[0].Config.Variant.(v14.BoxNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(box.StrokeColor).To(Equal(fixed(1)))
		Expect(box.FillColor).To(BeNil())
		value, ok := got.Nodes[1].Config.Variant.(v14.ValueNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(value.StrokeColor).To(Equal(fixed(2)))
		Expect(value.TextColor).To(Equal(fixed(3)))
		Expect(value.FillColor).To(BeNil())
		Expect(value.StalenessTimeout).To(Equal(3 * telem.Second))
		Expect(got.Edges).To(HaveLen(1))
		pipe, ok := got.Edges[0].Config.Variant.(v14.PipeEdgeConfig)
		Expect(ok).To(BeTrue())
		Expect(pipe.StrokeColor).To(Equal(fixed(5)))
		Expect(pipe.StrokeWidth).To(BeEquivalentTo(3))
		Expect(got.Edges[0].Source).To(Equal(v14.Handle{Node: "box", Param: "top"}))
	})
})

var _ = Describe("MigrateNodeConfig", func() {
	// Each entry gives its nth field the red value n, so a swapped field fails.
	DescribeTable(
		"Should store each look color as a fixed color reference",
		func(ctx SpecContext, old v13.NodeConfigVariant, fields ...string) {
			out := MustSucceed(v14.MigrateNodeConfig(ctx, v13.NodeConfig{Variant: old}))
			for i, f := range fields {
				Expect(out.Variant).To(HaveField(f, Equal(fixed(uint8(i+1)))))
			}
		},
		Entry(
			"a box",
			v13.BoxNodeConfig{StrokeColor: shade(1), FillColor: shade(2)},
			"StrokeColor",
			"FillColor",
		),
		Entry(
			"a circle",
			v13.CircleNodeConfig{StrokeColor: shade(1), FillColor: shade(2)},
			"StrokeColor",
			"FillColor",
		),
		Entry(
			"a cylinder",
			v13.CylinderNodeConfig{StrokeColor: shade(1), FillColor: shade(2)},
			"StrokeColor",
			"FillColor",
		),
		Entry("a line", v13.LineNodeConfig{StrokeColor: shade(1)}, "StrokeColor"),
		Entry(
			"a polygon",
			v13.PolygonNodeConfig{StrokeColor: shade(1), FillColor: shade(2)},
			"StrokeColor",
			"FillColor",
		),
		Entry("a text box", v13.TextBoxNodeConfig{TextColor: shade(1)}, "TextColor"),
		Entry(
			"an off page reference",
			v13.OffPageReferenceNodeConfig{FillColor: shade(1)},
			"FillColor",
		),
		Entry("a button", v13.ButtonNodeConfig{FillColor: shade(1)}, "FillColor"),
		Entry("an input", v13.InputNodeConfig{FillColor: shade(1)}, "FillColor"),
		Entry("a select", v13.SelectNodeConfig{FillColor: shade(1)}, "FillColor"),
		Entry("a setpoint", v13.SetpointNodeConfig{FillColor: shade(1)}, "FillColor"),
		Entry(
			"a state indicator",
			v13.StateIndicatorNodeConfig{StrokeColor: shade(1)},
			"StrokeColor",
		),
		Entry(
			"a tank",
			v13.TankNodeConfig{
				StrokeColor: shade(1),
				FillColor:   shade(2),
				LevelColor:  shade(3),
				AxisColor:   shade(4),
				TextColor:   shade(5),
			},
			"StrokeColor",
			"FillColor",
			"LevelColor",
			"AxisColor",
			"TextColor",
		),
		Entry("a gauge", v13.GaugeNodeConfig{StrokeColor: shade(1)}, "StrokeColor"),
		Entry(
			"a string display",
			v13.StringDisplayNodeConfig{StrokeColor: shade(1), TextColor: shade(2)},
			"StrokeColor",
			"TextColor",
		),
		Entry(
			"a value",
			v13.ValueNodeConfig{
				StrokeColor: shade(1),
				TextColor:   shade(2),
				FillColor:   shade(3),
			},
			"StrokeColor",
			"TextColor",
			"FillColor",
		),
		Entry(
			"a scale",
			v13.ScaleNodeConfig{
				LevelColor: shade(1),
				AxisColor:  shade(2),
				TextColor:  shade(3),
			},
			"LevelColor",
			"AxisColor",
			"TextColor",
		),
	)

	It("Should keep the ends of gauge and tank bounds", func(ctx SpecContext) {
		b := spatial.Bounds{Lower: -5, Upper: 5}
		want := spatial.ClosedBounds{Lower: -5, Upper: 5}
		gauge := MustSucceed(v14.MigrateNodeConfig(ctx, v13.NodeConfig{
			Variant: v13.GaugeNodeConfig{Bounds: b},
		}))
		Expect(gauge.Variant).To(HaveField("Bounds", want))
		tank := MustSucceed(v14.MigrateNodeConfig(ctx, v13.NodeConfig{
			Variant: v13.TankNodeConfig{Bounds: b},
		}))
		Expect(tank.Variant).To(HaveField("Bounds", want))
	})

	It("Should keep an absent look color absent", func(ctx SpecContext) {
		out := MustSucceed(v14.MigrateNodeConfig(ctx, v13.NodeConfig{
			Variant: v13.BoxNodeConfig{},
		}))
		box, ok := out.Variant.(v14.BoxNodeConfig)
		Expect(ok).To(BeTrue())
		Expect(box.StrokeColor).To(BeNil())
		Expect(box.FillColor).To(BeNil())
	})
})

var _ = Describe("MigrateSegmentedEdgeConfig", func() {
	It("Should store the stroke color as a fixed color and keep the width",
		func(ctx SpecContext) {
			out := MustSucceed(
				v14.MigrateSegmentedEdgeConfig(ctx, v13.SegmentedEdgeConfig{
					StrokeColor: shade(1),
					StrokeWidth: 3,
				}),
			)
			Expect(out.StrokeColor).To(Equal(fixed(1)))
			Expect(out.StrokeWidth).To(BeEquivalentTo(3))
		},
	)

	It("Should keep an absent stroke color absent", func(ctx SpecContext) {
		out := MustSucceed(
			v14.MigrateSegmentedEdgeConfig(
				ctx,
				v13.SegmentedEdgeConfig{StrokeWidth: 3},
			),
		)
		Expect(out.StrokeColor).To(BeNil())
		Expect(out.StrokeWidth).To(BeEquivalentTo(3))
	})
})

var _ = Describe("MigrateScaleIndicatorConfig", func() {
	It("Should store the level, axis, and text colors as fixed colors",
		func(ctx SpecContext) {
			out := MustSucceed(v14.MigrateScaleIndicatorConfig(
				ctx,
				v13.ScaleIndicatorConfig{
					LevelColor: shade(1),
					AxisColor:  shade(2),
					TextColor:  shade(3),
				},
			))
			Expect(out.LevelColor).To(Equal(fixed(1)))
			Expect(out.AxisColor).To(Equal(fixed(2)))
			Expect(out.TextColor).To(Equal(fixed(3)))
		},
	)

	DescribeTable("Should lift the bounds into valid closed bounds",
		func(ctx SpecContext, old spatial.Bounds, want spatial.ClosedBounds) {
			out := MustSucceed(v14.MigrateScaleIndicatorConfig(
				ctx,
				v13.ScaleIndicatorConfig{Bounds: old},
			))
			Expect(out.Bounds).To(Equal(want))
		},
		Entry(
			"keeping bounds in order",
			spatial.Bounds{
				Lower: 1,
				Upper: 9,
			},
			spatial.ClosedBounds{Lower: 1, Upper: 9},
		),
		Entry(
			"swapping reversed bounds",
			spatial.Bounds{
				Lower: 9,
				Upper: 1,
			},
			spatial.ClosedBounds{Lower: 1, Upper: 9},
		),
		Entry("giving equal bounds the default",
			spatial.Bounds{Lower: 4, Upper: 4},
			spatial.ClosedBounds{Lower: 0, Upper: 100}),
	)
})

var _ = Describe("MigrateStalenessConfig", func() {
	It("Should keep the timeout and drop the color", func(ctx SpecContext) {
		out := MustSucceed(v14.MigrateStalenessConfig(ctx, v13.StalenessConfig{
			StalenessTimeout: 3 * telem.Second,
			StalenessColor:   shade(1),
		}))
		Expect(out).To(Equal(v14.StalenessConfig{StalenessTimeout: 3 * telem.Second}))
	})
})

var _ = Describe("MigrateToggleConfig", func() {
	It("Should keep the staleness timeout and drop the color", func(ctx SpecContext) {
		out := MustSucceed(v14.MigrateToggleConfig(ctx, v13.ToggleConfig{
			StalenessTimeout: 3 * telem.Second,
			StalenessColor:   shade(1),
			StateChannel:     7,
		}))
		Expect(out.StalenessConfig).To(Equal(
			v14.StalenessConfig{StalenessTimeout: 3 * telem.Second},
		))
		Expect(out.StateChannel).To(BeEquivalentTo(7))
	})
})

var _ = Describe("MigrateNode", func() {
	It("Should keep the node and store its colors as fixed colors",
		func(ctx SpecContext) {
			out := MustSucceed(v14.MigrateNode(ctx, v13.Node{
				Key:    "n",
				ZIndex: 2,
				Config: v13.NodeConfig{
					Variant: v13.BoxNodeConfig{StrokeColor: shade(1)},
				},
			}))
			Expect(out.Key).To(Equal("n"))
			Expect(out.ZIndex).To(BeEquivalentTo(2))
			box, ok := out.Config.Variant.(v14.BoxNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(box.StrokeColor).To(Equal(fixed(1)))
		},
	)
})

var _ = Describe("MigrateEdge", func() {
	It("Should keep the edge and store its stroke color as a fixed color",
		func(ctx SpecContext) {
			out := MustSucceed(v14.MigrateEdge(ctx, v13.Edge{
				Key: "e",
				Config: v13.EdgeConfig{Variant: v13.PipeEdgeConfig{
					SegmentedEdgeConfig: v13.SegmentedEdgeConfig{StrokeColor: shade(1)},
				}},
			}))
			Expect(out.Key).To(Equal("e"))
			pipe, ok := out.Config.Variant.(v14.PipeEdgeConfig)
			Expect(ok).To(BeTrue())
			Expect(pipe.StrokeColor).To(Equal(fixed(1)))
		},
	)
})

var _ = Describe("MigrateSchematic", func() {
	It("Should store the colors of each node and edge as fixed colors",
		func(ctx SpecContext) {
			out := MustSucceed(v14.MigrateSchematic(ctx, v13.Schematic{
				Name: "Engine",
				Nodes: []v13.Node{
					{
						Key: "n",
						Config: v13.NodeConfig{
							Variant: v13.BoxNodeConfig{FillColor: shade(1)},
						},
					},
				},
				Edges: []v13.Edge{{
					Key: "e",
					Config: v13.EdgeConfig{Variant: v13.PipeEdgeConfig{
						SegmentedEdgeConfig: v13.SegmentedEdgeConfig{
							StrokeColor: shade(2),
						},
					}},
				}},
			}))
			Expect(out.Name).To(Equal("Engine"))
			box, ok := out.Nodes[0].Config.Variant.(v14.BoxNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(box.FillColor).To(Equal(fixed(1)))
			pipe, ok := out.Edges[0].Config.Variant.(v14.PipeEdgeConfig)
			Expect(ok).To(BeTrue())
			Expect(pipe.StrokeColor).To(Equal(fixed(2)))
		},
	)
	It(
		"Should turn a switch that faced right or bottom upright and keep its edges",
		func(ctx SpecContext) {
			switchNode := func(key string, o spatial.OuterLocation) v13.Node {
				var cfg v13.SwitchNodeConfig
				cfg.Orientation = o
				return v13.Node{Key: key, Config: v13.NodeConfig{Variant: cfg}}
			}
			var valve v13.ValveNodeConfig
			valve.Orientation = spatial.OuterLocationRight
			end := func(node, param string) v13.Handle {
				return v13.Handle{Node: node, Param: param}
			}
			out := MustSucceed(v14.MigrateSchematic(ctx, v13.Schematic{
				Nodes: []v13.Node{
					switchNode("right", spatial.OuterLocationRight),
					switchNode("bottom", spatial.OuterLocationBottom),
					switchNode("left", spatial.OuterLocationLeft),
					{Key: "valve", Config: v13.NodeConfig{Variant: valve}},
				},
				Edges: []v13.Edge{
					{Key: "a", Source: end("right", "1"), Target: end("valve", "1")},
					{Key: "b", Source: end("valve", "2"), Target: end("bottom", "2")},
					{Key: "c", Source: end("left", "1"), Target: end("bottom", "1")},
				},
			}))
			orientations := lo.Map(
				out.Nodes,
				func(n v14.Node, _ int) spatial.OuterLocation {
					return n.Config.Variant.(v14.SymbolNodeConfig).Orientation
				},
			)
			Expect(orientations).To(Equal([]spatial.OuterLocation{
				spatial.OuterLocationLeft,
				spatial.OuterLocationTop,
				spatial.OuterLocationLeft,
				spatial.OuterLocationRight,
			}))
			ends := lo.Map(out.Edges, func(e v14.Edge, _ int) [2]v14.Handle {
				return [2]v14.Handle{e.Source, e.Target}
			})
			Expect(ends).To(Equal([][2]v14.Handle{
				{end("right", "2"), end("valve", "1")},
				{end("valve", "2"), end("bottom", "1")},
				{end("left", "1"), end("bottom", "2")},
			}))
		},
	)
	It(
		"Should lift a schematic that 0.62.0 stored with no enumeration",
		func(ctx SpecContext) {
			var stored v13.Schematic
			Expect(
				orc.Codec.Decode(loadWire("v13_released.hex"), &stored),
			).To(Succeed())
			out := MustSucceed(v14.MigrateSchematic(ctx, stored))
			Expect(out.Key).To(Equal(stored.Key))
			Expect(out.Name).To(Equal("Valve panel"))
			Expect(out.Nodes).To(HaveLen(2))
			sel, ok := out.Nodes[0].Config.Variant.(v14.SelectNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(sel.Enumeration).To(BeNil())
			Expect(sel.CommandChannel).To(BeEquivalentTo(1048577))
			ind, ok := out.Nodes[1].Config.Variant.(v14.StateIndicatorNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(ind.Enumeration).To(BeNil())
			Expect(ind.Channel).To(BeEquivalentTo(1048578))
		},
	)

	It(
		"Should make the custom nodes that 0.62.0 stored symbol nodes",
		func(ctx SpecContext) {
			var stored v13.Schematic
			Expect(
				orc.Codec.Decode(loadWire("v13_released_custom.hex"), &stored),
			).To(Succeed())
			out := MustSucceed(v14.MigrateSchematic(ctx, stored))
			Expect(out.Nodes).To(HaveLen(2))
			act, ok := out.Nodes[0].Config.Variant.(v14.SymbolNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(act.Spec.Variant).To(Equal(symbol.CustomRef{
				Key: uuid.MustParse("3e46d160-a00b-4224-8119-ffabc45cdc24"),
			}))
			old, ok := stored.Nodes[1].Config.Variant.(v13.CustomStaticNodeConfig)
			Expect(ok).To(BeTrue())
			static, ok := out.Nodes[1].Config.Variant.(v14.SymbolNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(static.Spec.Variant).
				To(Equal(symbol.CustomRef{Key: uuid.MustParse(old.SpecKey)}))
		},
	)
})
