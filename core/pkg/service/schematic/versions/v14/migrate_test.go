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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/gorp"
	servicecolor "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	v14 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v14"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/defaults"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/synnaxlabs/x/testutil/memkv"
)

func shade(r uint8) *xcolor.Color { return &xcolor.Color{R: r, A: 255} }

func fixed(r uint8) *servicecolor.Ref {
	return &servicecolor.Ref{Variant: servicecolor.FixedRef{Value: *shade(r)}}
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
						StrokeColor:    shade(2),
						TextColor:      shade(3),
						StalenessColor: shade(4),
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
		MustOpen(gorp.OpenTable(ctx, gorp.TableConfig[v14.Key, v14.Schematic]{
			DB: db, Migrations: []gorp.Migration{v14.Migration},
		}))
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
		Expect(value.StalenessColor).To(Equal(shade(4)))
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
})
