// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v13_test

import (
	"context"
	"encoding/json/v2"
	"math"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	symbolv2 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v2"
	symbolv4 "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v4"
	v12 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v12"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	color "github.com/synnaxlabs/x/color/versions/v0"
	colorv1 "github.com/synnaxlabs/x/color/versions/v1"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
	text "github.com/synnaxlabs/x/text/versions/v0"
)

// withElements gives each config key of s a node, or an edge between two capped anchor
// nodes when the config is a pipe, so the migration keeps every config.
func withElements(s v12.Schematic) v12.Schematic {
	anchor := v12.ElementConfig{Variant: v12.CapElementConfig{}}
	s.Configs["src"], s.Configs["dst"] = anchor, anchor
	for k, cfg := range s.Configs {
		if _, isPipe := cfg.Variant.(v12.PipeElementConfig); isPipe {
			s.Edges = append(s.Edges, v12.Edge{
				Key:    k,
				Source: v12.Handle{Node: "src"},
				Target: v12.Handle{Node: "dst"},
			})
			continue
		}
		s.Nodes = append(s.Nodes, v12.Node{Key: k})
	}
	return s
}

// configOf returns the config variant of the node or edge of s with the key.
func configOf(s v13.Schematic, key string) any {
	GinkgoHelper()
	for _, n := range s.Nodes {
		if n.Key == key {
			return n.Config.Variant
		}
	}
	for _, e := range s.Edges {
		if e.Key == key {
			return e.Config.Variant
		}
	}
	Fail("no element with key " + key)
	return nil
}

var _ = Describe("Migration", func() {
	DescribeTable("Should store every element color alpha as a byte",
		func(ctx SpecContext, a float64, expected uint8) {
			db := DeferClose(gorp.Wrap(memkv.New()))
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v12.Key, v12.Schematic]{
				DB: db,
			}))
			c := func(r uint8) *color.Color { return &color.Color{R: r, A: a} }
			old := withElements(v12.Schematic{
				Key:  uuid.New(),
				Name: "s",
				Configs: map[string]v12.ElementConfig{
					"cap": {Variant: v12.CapElementConfig{
						StaticSymbolConfig: v12.StaticSymbolConfig{StrokeColor: c(1)},
					}},
					"pipe": {Variant: v12.PipeElementConfig{
						SegmentedEdgeConfig: v12.SegmentedEdgeConfig{StrokeColor: c(2)},
					}},
					"value": {Variant: v12.ValueElementConfig{
						StrokeColor: c(3),
						TextColor:   c(4),
						FillColor:   c(5),
						Redline: colorv1.Scale{Bands: []colorv1.Band{
							{Key: "b", Color: *c(6)},
						}},
					}},
					"custom": {Variant: v12.CustomStaticElementConfig{
						StateOverrides: []symbolv2.State{
							{Key: "on", Regions: []symbolv2.Region{
								{Key: "r", FillColor: c(7)},
							}},
						},
					}},
				},
			})
			Expect(gorp.NewCreate[v12.Key, v12.Schematic]().Entry(&old).Exec(ctx, db)).
				To(Succeed())
			MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v13.Key, v13.Schematic]{
				DB: db, Migrations: []gorp.Migration{v13.Migration},
			}))
			var got v13.Schematic
			Expect(gorp.NewRetrieve[v13.Key, v13.Schematic]().
				Where(gorp.MatchKeys[v13.Key, v13.Schematic](old.Key)).
				Entry(&got).Exec(ctx, db)).To(Succeed())
			n := func(r uint8) *colorv2.Color { return &colorv2.Color{R: r, A: expected} }
			Expect(configOf(got, "cap").(v13.CapNodeConfig).StrokeColor).
				To(Equal(n(1)))
			Expect(configOf(got, "pipe").(v13.PipeEdgeConfig).StrokeColor).
				To(Equal(n(2)))
			value := configOf(got, "value").(v13.ValueNodeConfig)
			Expect(value.StrokeColor).To(Equal(n(3)))
			Expect(value.TextColor).To(Equal(n(4)))
			Expect(value.FillColor).To(Equal(n(5)))
			Expect(
				value.Redline.Bands,
			).To(Equal([]colorv2.Band{{Key: "b", Color: *n(6)}}))
			custom := configOf(got, "custom").(v13.CustomStaticNodeConfig)
			Expect(custom.StateOverrides).To(Equal([]symbolv4.State{{
				Key: "on",
				Regions: []symbolv4.Region{
					{Key: "r", Selectors: []string{}, FillColor: n(7)},
				},
			}}))
		},
		Entry("a transparent alpha", 0.0, uint8(0)),
		Entry("a faint alpha", 0.003, uint8(1)),
		Entry("a half alpha", 0.5, uint8(128)),
		Entry("an opaque alpha", 1.0, uint8(255)),
	)
})

var _ = Describe("Migrate functions", func() {
	half := func() *color.Color { return &color.Color{R: 1, A: 0.5} }
	want := &colorv2.Color{R: 1, A: 128}
	DescribeTable(
		"Should store the color alpha as a byte",
		func(ctx SpecContext, migrate func(context.Context) *colorv2.Color) {
			Expect(migrate(ctx)).To(Equal(want))
		},
		Entry("MigrateSchematic", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			out := MustSucceed(v13.MigrateSchematic(ctx, withElements(v12.Schematic{
				Configs: map[string]v12.ElementConfig{
					"n": {Variant: v12.CapElementConfig{
						StaticSymbolConfig: v12.StaticSymbolConfig{StrokeColor: half()},
					}},
				},
			})))
			return configOf(out, "n").(v13.CapNodeConfig).StrokeColor
		}),
		Entry("MigrateNodeConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			out := MustSucceed(v13.MigrateNodeConfig(ctx, v12.NodeConfig{
				Variant: v12.CapNodeConfig{
					StaticSymbolConfig: v12.StaticSymbolConfig{StrokeColor: half()},
				},
			}))
			return out.Variant.(v13.CapNodeConfig).StrokeColor
		}),
		Entry("MigrateEdgeConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			out := MustSucceed(v13.MigrateEdgeConfig(ctx, v12.EdgeConfig{
				Variant: v12.PipeEdgeConfig{
					SegmentedEdgeConfig: v12.SegmentedEdgeConfig{StrokeColor: half()},
				},
			}))
			return out.Variant.(v13.PipeEdgeConfig).StrokeColor
		}),
		Entry("MigrateStaticSymbolConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateStaticSymbolConfig(
				ctx, v12.StaticSymbolConfig{StrokeColor: half()},
			)).StrokeColor
		}),
		Entry("MigrateToggleSymbolConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateToggleSymbolConfig(
				ctx, v12.ToggleSymbolConfig{StrokeColor: half()},
			)).StrokeColor
		}),
		Entry("MigrateStalenessConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateStalenessConfig(
				ctx, v12.StalenessConfig{StalenessColor: half()},
			)).StalenessColor
		}),
		Entry("MigrateStateMapping", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateStateMapping(
				ctx, v12.StateMapping{Color: half()},
			)).Color
		}),
		Entry("MigrateScaleIndicatorConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateScaleIndicatorConfig(
				ctx, v12.ScaleIndicatorConfig{LevelColor: half()},
			)).LevelColor
		}),
		Entry(
			"MigrateDummyToggleSymbolConfig",
			func(ctx context.Context) *colorv2.Color {
				GinkgoHelper()
				return MustSucceed(v13.MigrateDummyToggleSymbolConfig(
					ctx, v12.DummyToggleSymbolConfig{StrokeColor: half()},
				)).StrokeColor
			},
		),
		Entry("MigrateToggleConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateToggleConfig(
				ctx, v12.ToggleConfig{StalenessColor: half()},
			)).StalenessColor
		}),
		Entry("MigrateSegmentedEdgeConfig", func(ctx context.Context) *colorv2.Color {
			GinkgoHelper()
			return MustSucceed(v13.MigrateSegmentedEdgeConfig(
				ctx, v12.SegmentedEdgeConfig{StrokeColor: half()},
			)).StrokeColor
		}),
	)
	DescribeTable("Should make a zero color that meant the theme absent",
		func(
			ctx SpecContext,
			variant v12.ElementConfigVariant,
			cleared func(any) []*colorv2.Color,
		) {
			out := MustSucceed(v13.MigrateSchematic(ctx, withElements(v12.Schematic{
				Configs: map[string]v12.ElementConfig{"n": {Variant: variant}},
			})))
			for _, c := range cleared(configOf(out, "n")) {
				Expect(c).To(BeNil())
			}
		},
		Entry("a value", v12.ValueElementConfig{
			StalenessColor: &color.Color{},
			TextColor:      &color.Color{},
		}, func(v any) []*colorv2.Color {
			c := v.(v13.ValueNodeConfig)
			return []*colorv2.Color{c.StalenessColor, c.TextColor}
		}),
		Entry("a gauge", v12.GaugeElementConfig{
			StalenessColor: &color.Color{},
			StrokeColor:    &color.Color{},
		}, func(v any) []*colorv2.Color {
			c := v.(v13.GaugeNodeConfig)
			return []*colorv2.Color{c.StalenessColor, c.StrokeColor}
		}),
		Entry("a scale", v12.ScaleElementConfig{
			StalenessColor: &color.Color{},
			LevelColor:     &color.Color{},
			AxisColor:      &color.Color{},
			TextColor:      &color.Color{},
		}, func(v any) []*colorv2.Color {
			c := v.(v13.ScaleNodeConfig)
			return []*colorv2.Color{
				c.StalenessColor,
				c.LevelColor,
				c.AxisColor,
				c.TextColor,
			}
		}),
		Entry("a tank", v12.TankElementConfig{
			StalenessColor: &color.Color{},
			LevelColor:     &color.Color{},
			AxisColor:      &color.Color{},
			TextColor:      &color.Color{},
		}, func(v any) []*colorv2.Color {
			c := v.(v13.TankNodeConfig)
			return []*colorv2.Color{
				c.StalenessColor,
				c.LevelColor,
				c.AxisColor,
				c.TextColor,
			}
		}),
	)
	DescribeTable("Should keep a black theme color with a non-finite alpha",
		func(ctx SpecContext, a float64) {
			out := MustSucceed(v13.MigrateSchematic(ctx, withElements(v12.Schematic{
				Configs: map[string]v12.ElementConfig{
					"n": {Variant: v12.ValueElementConfig{
						TextColor: &color.Color{A: a},
					}},
				},
			})))
			Expect(configOf(out, "n").(v13.ValueNodeConfig).TextColor).
				To(Equal(&colorv2.Color{A: 255}))
		},
		Entry("NaN", math.NaN()),
		Entry("positive infinity", math.Inf(1)),
		Entry("negative infinity", math.Inf(-1)),
	)
	It("Should keep the zero stroke of a value", func(ctx SpecContext) {
		out := MustSucceed(v13.MigrateSchematic(ctx, withElements(v12.Schematic{
			Configs: map[string]v12.ElementConfig{
				"n": {Variant: v12.ValueElementConfig{StrokeColor: &color.Color{}}},
			},
		})))
		Expect(configOf(out, "n").(v13.ValueNodeConfig).StrokeColor).
			To(HaveValue(Equal(colorv2.Color{})))
	})
	It("Should keep a zero color that painted transparent", func(ctx SpecContext) {
		zero := &color.Color{}
		out := MustSucceed(v13.MigrateSchematic(ctx, withElements(v12.Schematic{
			Configs: map[string]v12.ElementConfig{
				"cap": {Variant: v12.CapElementConfig{
					StaticSymbolConfig: v12.StaticSymbolConfig{StrokeColor: zero},
				}},
				"pipe": {Variant: v12.PipeElementConfig{
					SegmentedEdgeConfig: v12.SegmentedEdgeConfig{StrokeColor: zero},
				}},
			},
		})))
		Expect(configOf(out, "cap").(v13.CapNodeConfig).StrokeColor).
			To(HaveValue(Equal(colorv2.Color{})))
		Expect(configOf(out, "pipe").(v13.PipeEdgeConfig).StrokeColor).
			To(HaveValue(Equal(colorv2.Color{})))
	})
	It("Should keep a zero region color in a state override", func(ctx SpecContext) {
		out := MustSucceed(v13.MigrateSchematic(ctx, withElements(v12.Schematic{
			Configs: map[string]v12.ElementConfig{
				"n": {Variant: v12.CustomStaticElementConfig{
					StateOverrides: []symbolv2.State{
						{
							Key: "base",
							Regions: []symbolv2.Region{
								{Key: "r", FillColor: &color.Color{}},
							},
						},
					},
				}},
			},
		})))
		node := configOf(out, "n").(v13.CustomStaticNodeConfig)
		Expect(node.StateOverrides[0].Regions[0].FillColor).
			To(HaveValue(Equal(colorv2.Color{})))
	})
})

var _ = Describe("MigrateSchematic config move", func() {
	var (
		valve      = v12.ValveElementConfig{Orientation: "left", Scale: 2}
		pipe       = v12.PipeElementConfig{Segments: []v12.Segment{}}
		movedValve = v13.NodeConfig{
			Variant: v13.ValveNodeConfig{
				Label:       v13.LabelConfig{MaxInlineSize: 1},
				Orientation: "left",
				Scale:       2,
			},
		}
		movedPipe = v13.EdgeConfig{Variant: v13.PipeEdgeConfig{
			SegmentedEdgeConfig: v13.SegmentedEdgeConfig{
				Segments:    []v13.Segment{},
				StrokeWidth: 2,
			},
		}}
	)
	node := func(key string) v12.Node {
		return v12.Node{Key: key, Position: spatial.XY{X: 1, Y: 2}, ZIndex: 3}
	}
	edge := func(key, source, target string) v12.Edge {
		return v12.Edge{
			Key:    key,
			Source: v12.Handle{Node: source, Param: "out"},
			Target: v12.Handle{Node: target, Param: "in"},
		}
	}
	migrate := func(ctx SpecContext, old v12.Schematic) v13.Schematic {
		GinkgoHelper()
		return MustSucceed(v13.MigrateSchematic(ctx, old))
	}

	It("Should move each config onto its node or edge", func(ctx SpecContext) {
		key := uuid.New()
		s := migrate(ctx, v12.Schematic{
			Key:   key,
			Name:  "Engine",
			Nodes: []v12.Node{node("n1"), node("n2")},
			Edges: []v12.Edge{edge("e1", "n1", "n2")},
			Configs: map[string]v12.ElementConfig{
				"n1": {Variant: valve},
				"n2": {Variant: valve},
				"e1": {Variant: pipe},
			},
		})
		Expect(s.Key).To(Equal(key))
		Expect(s.Name).To(Equal("Engine"))
		Expect(s.Nodes).To(Equal([]v13.Node{
			{
				Key:      "n1",
				Position: spatial.XY{X: 1, Y: 2},
				ZIndex:   3,
				Config:   movedValve,
			},
			{
				Key:      "n2",
				Position: spatial.XY{X: 1, Y: 2},
				ZIndex:   3,
				Config:   movedValve,
			},
		}))
		Expect(s.Edges).To(Equal([]v13.Edge{{
			Key:    "e1",
			Source: v12.Handle{Node: "n1", Param: "out"},
			Target: v12.Handle{Node: "n2", Param: "in"},
			Config: movedPipe,
		}}))
	})

	It("Should drop a config that no node or edge has", func(ctx SpecContext) {
		s := migrate(ctx, v12.Schematic{
			Nodes: []v12.Node{node("n1")},
			Configs: map[string]v12.ElementConfig{
				"n1":     {Variant: valve},
				"orphan": {Variant: valve},
			},
		})
		Expect(s.Nodes).To(HaveLen(1))
		Expect(s.Nodes[0].Key).To(Equal("n1"))
	})

	DescribeTable("Should drop a node",
		func(ctx SpecContext, configs map[string]v12.ElementConfig) {
			s := migrate(ctx, v12.Schematic{
				Nodes:   []v12.Node{node("n1"), node("n2")},
				Configs: configs,
			})
			Expect(s.Nodes).To(HaveLen(1))
			Expect(s.Nodes[0].Key).To(Equal("n2"))
		},
		Entry("with no config", map[string]v12.ElementConfig{
			"n2": {Variant: valve},
		}),
		Entry("whose config is an edge config", map[string]v12.ElementConfig{
			"n1": {Variant: pipe},
			"n2": {Variant: valve},
		}),
	)

	DescribeTable("Should give the default pipe to an edge",
		func(ctx SpecContext, configs map[string]v12.ElementConfig) {
			configs["n1"] = v12.ElementConfig{Variant: valve}
			configs["n2"] = v12.ElementConfig{Variant: valve}
			s := migrate(ctx, v12.Schematic{
				Nodes:   []v12.Node{node("n1"), node("n2")},
				Edges:   []v12.Edge{edge("e1", "n1", "n2")},
				Configs: configs,
			})
			Expect(s.Edges).To(Equal([]v13.Edge{{
				Key:    "e1",
				Source: v12.Handle{Node: "n1", Param: "out"},
				Target: v12.Handle{Node: "n2", Param: "in"},
				Config: movedPipe,
			}}))
		},
		Entry("with no config", map[string]v12.ElementConfig{}),
		Entry(
			"whose config is a node config",
			map[string]v12.ElementConfig{"e1": {Variant: valve}},
		),
	)

	DescribeTable("Should drop an edge",
		func(ctx SpecContext, e v12.Edge, configs map[string]v12.ElementConfig) {
			s := migrate(ctx, v12.Schematic{
				Nodes:   []v12.Node{node("n1"), node("n2"), node("bare")},
				Edges:   []v12.Edge{e, edge("kept", "n1", "n2")},
				Configs: configs,
			})
			Expect(s.Edges).To(HaveLen(1))
			Expect(s.Edges[0].Key).To(Equal("kept"))
		},
		Entry(
			"whose source node is dropped",
			edge("e1", "bare", "n2"),
			map[string]v12.ElementConfig{
				"n1":   {Variant: valve},
				"n2":   {Variant: valve},
				"e1":   {Variant: pipe},
				"kept": {Variant: pipe},
			},
		),
		Entry(
			"whose target node is dropped",
			edge("e1", "n1", "bare"),
			map[string]v12.ElementConfig{
				"n1":   {Variant: valve},
				"n2":   {Variant: valve},
				"e1":   {Variant: pipe},
				"kept": {Variant: pipe},
			},
		),
		Entry(
			"whose target node does not exist",
			edge("e1", "n1", "ghost"),
			map[string]v12.ElementConfig{
				"n1":   {Variant: valve},
				"n2":   {Variant: valve},
				"e1":   {Variant: pipe},
				"kept": {Variant: pipe},
			},
		),
	)

	DescribeTable("Should give the schema default to a non-finite number",
		func(ctx SpecContext, f float64) {
			s := migrate(ctx, v12.Schematic{
				Nodes: []v12.Node{node("n1")},
				Edges: []v12.Edge{edge("e1", "n1", "n1")},
				Configs: map[string]v12.ElementConfig{
					"n1": {Variant: v12.PolygonElementConfig{
						NumSides:    f,
						SideLength:  30,
						Rotation:    f,
						StrokeColor: &color.Color{R: 1, A: f},
						StrokeWidth: f,
					}},
					"e1": {Variant: v12.PipeElementConfig{
						SegmentedEdgeConfig: v12.SegmentedEdgeConfig{
							Segments: []v12.Segment{{Length: f}},
						},
					}},
				},
			})
			polygon := configOf(s, "n1").(v13.PolygonNodeConfig)
			Expect(polygon.NumSides).To(Equal(6.0))
			Expect(polygon.SideLength).To(Equal(30.0))
			Expect(polygon.Rotation).To(BeZero())
			Expect(polygon.StrokeWidth).To(Equal(uint8(2)))
			Expect(polygon.StrokeColor).To(Equal(&colorv2.Color{R: 1, A: 255}))
			Expect(configOf(s, "e1").(v13.PipeEdgeConfig).Segments).
				To(Equal([]v13.Segment{{}}))
		},
		Entry("NaN", math.NaN()),
		Entry("positive infinity", math.Inf(1)),
		Entry("negative infinity", math.Inf(-1)),
	)

	It("Should not change the schematic it was given", func(ctx SpecContext) {
		old := v12.Schematic{
			Nodes:   []v12.Node{node("n1"), node("n2")},
			Configs: map[string]v12.ElementConfig{"n2": {Variant: valve}},
		}
		migrate(ctx, old)
		Expect(old.Nodes).To(Equal([]v12.Node{node("n1"), node("n2")}))
	})
})

var _ = Describe("MigrateSchematic group parents", func() {
	box := func(members ...string) v12.ElementConfig {
		return v12.ElementConfig{Variant: v12.GroupBoxElementConfig{Members: members}}
	}
	valve := v12.ElementConfig{Variant: v12.ValveElementConfig{Orientation: "left"}}
	// parents migrates a schematic with a node for each key of order and returns the
	// parent of each migrated node by key, with "" for no parent.
	parents := func(
		ctx SpecContext,
		order []string,
		configs map[string]v12.ElementConfig,
	) map[string]string {
		GinkgoHelper()
		old := v12.Schematic{Configs: configs}
		for _, k := range order {
			old.Nodes = append(old.Nodes, v12.Node{Key: k})
		}
		s := MustSucceed(v13.MigrateSchematic(ctx, old))
		out := make(map[string]string, len(s.Nodes))
		for _, n := range s.Nodes {
			out[n.Key] = ""
			if n.Parent != nil {
				out[n.Key] = *n.Parent
			}
		}
		return out
	}

	It("Should give each member its group as parent", func(ctx SpecContext) {
		Expect(
			parents(
				ctx,
				[]string{"outer", "inner", "a", "b"},
				map[string]v12.ElementConfig{
					"outer": box("inner", "b"),
					"inner": box("a"),
					"a":     valve,
					"b":     valve,
				},
			),
		).To(Equal(map[string]string{
			"outer": "",
			"inner": "outer",
			"a":     "inner",
			"b":     "outer",
		}))
	})

	It("Should skip a member that is not a node", func(ctx SpecContext) {
		Expect(parents(ctx, []string{"g", "a", "bare"}, map[string]v12.ElementConfig{
			"g": box("a", "bare", "ghost"),
			"a": valve,
		})).To(Equal(map[string]string{"g": "", "a": "g"}))
	})

	It("Should keep the first group in node order", func(ctx SpecContext) {
		Expect(parents(ctx, []string{"g2", "g1", "a"}, map[string]v12.ElementConfig{
			"g1": box("a"),
			"g2": box("a"),
			"a":  valve,
		})).To(Equal(map[string]string{"g1": "", "g2": "", "a": "g2"}))
	})

	DescribeTable("Should skip a link that closes a cycle",
		func(
			ctx SpecContext,
			order []string,
			configs map[string]v12.ElementConfig,
			expected map[string]string,
		) {
			Expect(parents(ctx, order, configs)).To(Equal(expected))
		},
		Entry(
			"a group that lists itself",
			[]string{"g"},
			map[string]v12.ElementConfig{"g": box("g")},
			map[string]string{"g": ""},
		),
		Entry(
			"two groups that list each other",
			[]string{"g1", "g2"},
			map[string]v12.ElementConfig{"g1": box("g2"), "g2": box("g1")},
			map[string]string{"g1": "", "g2": "g1"},
		),
		Entry(
			"three groups in a ring",
			[]string{"g1", "g2", "g3"},
			map[string]v12.ElementConfig{
				"g1": box("g2"),
				"g2": box("g3"),
				"g3": box("g1"),
			},
			map[string]string{"g1": "", "g2": "g1", "g3": "g2"},
		),
	)

	It("Should drop the members of a group box", func(ctx SpecContext) {
		s := MustSucceed(v13.MigrateSchematic(ctx, v12.Schematic{
			Nodes:   []v12.Node{{Key: "g"}, {Key: "a"}},
			Configs: map[string]v12.ElementConfig{"g": box("a"), "a": valve},
		}))
		Expect(s.Nodes[0].Config.Variant).To(Equal(v13.GroupBoxNodeConfig{}))
	})
})

var _ = Describe("MigrateNodeConfig stroke widths", func() {
	DescribeTable("Should round and clamp a symbol stroke width",
		func(ctx SpecContext, old v12.NodeConfigVariant, want uint8) {
			out := MustSucceed(v13.MigrateNodeConfig(ctx, v12.NodeConfig{Variant: old}))
			Expect(out.Variant).To(HaveField("StrokeWidth", Equal(want)))
		},
		Entry("box whole", v12.BoxNodeConfig{StrokeWidth: 4}, uint8(4)),
		Entry("box fraction", v12.BoxNodeConfig{StrokeWidth: 2.6}, uint8(3)),
		Entry("box at 0", v12.BoxNodeConfig{}, uint8(0)),
		Entry("box at NaN", v12.BoxNodeConfig{StrokeWidth: math.NaN()}, uint8(0)),
		Entry("box too large", v12.BoxNodeConfig{StrokeWidth: 30}, uint8(20)),
		Entry("circle thin", v12.CircleNodeConfig{StrokeWidth: 0.4}, uint8(1)),
		Entry("circle at 0", v12.CircleNodeConfig{}, uint8(0)),
		Entry("line fraction", v12.LineNodeConfig{StrokeWidth: 5.5}, uint8(6)),
		Entry("line at 0", v12.LineNodeConfig{}, uint8(1)),
		Entry("line negative", v12.LineNodeConfig{StrokeWidth: -3}, uint8(1)),
		Entry("line at NaN", v12.LineNodeConfig{StrokeWidth: math.NaN()}, uint8(1)),
		Entry("polygon too large", v12.PolygonNodeConfig{StrokeWidth: 1e9}, uint8(20)),
	)

	It("Should keep a stroke width of 0 through storage", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v12.Key, v12.Schematic]{DB: db}))
		old := withElements(v12.Schematic{
			Key: uuid.New(),
			Configs: map[string]v12.ElementConfig{
				"box": {Variant: v12.BoxElementConfig{}},
			},
		})
		Expect(gorp.NewCreate[v12.Key, v12.Schematic]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v13.Key, v13.Schematic]{
			DB: db, Migrations: []gorp.Migration{v13.Migration},
		}))
		var got v13.Schematic
		Expect(gorp.NewRetrieve[v13.Key, v13.Schematic]().
			Where(gorp.MatchKeys[v13.Key, v13.Schematic](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(configOf(got, "box")).To(HaveField("StrokeWidth", BeZero()))
	})
})

// migrateNode migrates a v12 node config decoded from raw.
func migrateNode(ctx SpecContext, raw map[string]any) v13.NodeConfigVariant {
	GinkgoHelper()
	var cfg v12.NodeConfig
	Expect(json.Unmarshal(MustSucceed(json.Marshal(raw)), &cfg)).To(Succeed())
	return MustSucceed(v13.MigrateNodeConfig(ctx, cfg)).Variant
}

var _ = Describe("MigrateNodeConfig text", func() {
	It("Should set a saved button's level and keep its wrap width", func(
		ctx SpecContext,
	) {
		Expect(migrateNode(ctx, map[string]any{
			"variant": "button",
			"size":    "large",
			"label":   map[string]any{"max_inline_size": 80.0},
		})).To(SatisfyAll(
			HaveField("Level", Equal(text.LevelH5)),
			HaveField("LabeledConfig.Label.MaxInlineSize", BeEquivalentTo(80)),
		))
	})

	DescribeTable("Should keep the wrap width of a saved symbol",
		func(ctx SpecContext, variant string, label map[string]any, width uint16) {
			Expect(migrateNode(ctx, map[string]any{
				"variant": variant,
				"label":   label,
			})).To(SatisfyAll(
				HaveField("LabeledConfig.Label.MaxInlineSize", Equal(width)),
				HaveField("LabeledConfig.Label.Label", Equal("P1")),
			))
		},
		Entry("a button with no width", "button",
			map[string]any{"label": "P1"}, uint16(65535)),
		Entry("a button with a width of 0", "button",
			map[string]any{"label": "P1", "max_inline_size": 0.0}, uint16(1)),
		Entry("a button with the default", "button",
			map[string]any{"label": "P1", "max_inline_size": 150.0}, uint16(65535)),
		Entry("a button with a negative width", "button",
			map[string]any{"label": "P1", "max_inline_size": -20.0}, uint16(65535)),
		Entry("a button with a chosen width", "button",
			map[string]any{"label": "P1", "max_inline_size": 80.0}, uint16(80)),
		Entry("a value with no width", "value",
			map[string]any{"label": "P1"}, uint16(150)),
		Entry("a value with a negative width", "value",
			map[string]any{"label": "P1", "max_inline_size": -20.0}, uint16(65535)),
		Entry("a cap with a fractional width", "cap",
			map[string]any{"label": "P1", "max_inline_size": 80.6}, uint16(81)),
		Entry("a valve with a width of 0", "valve",
			map[string]any{"label": "P1", "max_inline_size": 0.0}, uint16(1)),
		Entry("a relief valve with a fractional width", "relief_valve",
			map[string]any{"label": "P1", "max_inline_size": 80.6}, uint16(81)),
	)

	DescribeTable("Should keep the rendered text size of a saved state indicator",
		func(ctx SpecContext, size string, level text.Level) {
			Expect(migrateNode(ctx, map[string]any{
				"variant": "state_indicator",
				"size":    size,
			})).To(HaveField("Level", Equal(level)))
		},
		Entry("tiny", "tiny", text.LevelSmall),
		Entry("small", "small", text.LevelH5),
		Entry("medium", "medium", text.LevelH4),
		Entry("large", "large", text.LevelH3),
		Entry("huge", "huge", text.LevelH2),
	)

	DescribeTable("Should keep the rendered text size of a saved symbol",
		func(ctx SpecContext, size string, level text.Level) {
			for _, variant := range []string{"button", "input", "select", "setpoint"} {
				Expect(migrateNode(ctx, map[string]any{
					"variant": variant,
					"size":    size,
				})).To(HaveField("Level", Equal(level)), variant)
			}
		},
		Entry("tiny", "tiny", text.LevelTiny),
		Entry("small", "small", text.LevelSmall),
		Entry("medium", "medium", text.LevelP),
		Entry("large", "large", text.LevelH5),
		Entry("huge", "huge", text.LevelH2),
	)

	It("Should keep a level that a saved button stores", func(ctx SpecContext) {
		Expect(migrateNode(ctx, map[string]any{
			"variant": "button",
			"size":    "large",
			"level":   "h3",
		})).To(HaveField("Level", Equal(text.LevelH3)))
	})

	It("Should replace an invalid level that a saved button stores", func(
		ctx SpecContext,
	) {
		Expect(migrateNode(ctx, map[string]any{
			"variant": "button",
			"size":    "large",
			"level":   "",
		})).To(HaveField("Level", Equal(text.LevelH5)))
	})

	It("Should migrate a saved label through storage", func(ctx SpecContext) {
		db := DeferClose(gorp.Wrap(memkv.New()))
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v12.Key, v12.Schematic]{DB: db}))
		old := withElements(v12.Schematic{
			Key: uuid.New(),
			Configs: map[string]v12.ElementConfig{
				"b": {Variant: v12.ButtonElementConfig{
					LabeledConfig: v12.LabeledConfig{
						Label: v12.LabelConfig{Label: "P1", MaxInlineSize: 150},
					},
					Size: v12.ComponentSizeHuge,
				}},
			},
		})
		Expect(gorp.NewCreate[v12.Key, v12.Schematic]().Entry(&old).Exec(ctx, db)).
			To(Succeed())
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v13.Key, v13.Schematic]{
			DB: db, Migrations: []gorp.Migration{v13.Migration},
		}))
		var got v13.Schematic
		Expect(gorp.NewRetrieve[v13.Key, v13.Schematic]().
			Where(gorp.MatchKeys[v13.Key, v13.Schematic](old.Key)).
			Entry(&got).Exec(ctx, db)).To(Succeed())
		Expect(configOf(got, "b")).To(SatisfyAll(
			HaveField("Level", Equal(text.LevelH2)),
			HaveField("LabeledConfig.Label.MaxInlineSize", BeEquivalentTo(65535)),
		))
	})
})

var _ = Describe("MigrateLabelConfig", func() {
	DescribeTable("Should round and clamp the wrap width", func(
		ctx SpecContext,
		in float64,
		out uint16,
	) {
		cfg := MustSucceed(v13.MigrateLabelConfig(
			ctx,
			v12.LabelConfig{Label: "P1", MaxInlineSize: in},
		))
		Expect(cfg.MaxInlineSize).To(Equal(out))
		Expect(cfg.Label).To(Equal("P1"))
	},
		Entry("whole", 80.0, uint16(80)),
		Entry("round up", 80.6, uint16(81)),
		Entry("round down", 80.4, uint16(80)),
		Entry("below 1", 0.3, uint16(1)),
		Entry("negative", -20.0, uint16(65535)),
		Entry("negative infinite", math.Inf(-1), uint16(65535)),
		Entry("too large", 1e9, uint16(65535)),
		Entry("infinite", math.Inf(1), uint16(65535)),
		Entry("zero", 0.0, uint16(1)),
		Entry("NaN", math.NaN(), uint16(150)),
	)
})

var _ = Describe("MigrateLabeledConfig", func() {
	It("Should migrate the wrap width of the label", func(ctx SpecContext) {
		cfg := MustSucceed(v13.MigrateLabeledConfig(ctx, v12.LabeledConfig{
			Label: v12.LabelConfig{Label: "P1", MaxInlineSize: 80.6},
		}))
		Expect(cfg.Label.MaxInlineSize).To(BeEquivalentTo(81))
		Expect(cfg.Label.Label).To(Equal("P1"))
	})
})

var _ = Describe("MigrateSegmentedEdgeConfig", func() {
	It("Should give the edge the default stroke width", func(ctx SpecContext) {
		Expect(MustSucceed(v13.MigrateSegmentedEdgeConfig(
			ctx,
			v12.SegmentedEdgeConfig{},
		)).StrokeWidth).To(BeEquivalentTo(2))
	})
})

var _ = Describe("MigrateNode", func() {
	It("Should copy the node with no config", func(ctx SpecContext) {
		Expect(MustSucceed(v13.MigrateNode(ctx, v12.Node{
			Key:      "n1",
			Position: spatial.XY{X: 1, Y: 2},
			ZIndex:   3,
		}))).To(Equal(v13.Node{Key: "n1", Position: spatial.XY{X: 1, Y: 2}, ZIndex: 3}))
	})
})

var _ = Describe("MigrateEdge", func() {
	It("Should copy the edge with no config", func(ctx SpecContext) {
		source := v12.Handle{Node: "n1", Param: "out"}
		target := v12.Handle{Node: "n2", Param: "in"}
		Expect(MustSucceed(v13.MigrateEdge(ctx, v12.Edge{
			Key:    "e1",
			Source: source,
			Target: target,
		}))).To(Equal(v13.Edge{Key: "e1", Source: source, Target: target}))
	})
})
