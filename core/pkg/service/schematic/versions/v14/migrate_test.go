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

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	symbol "github.com/synnaxlabs/synnax/pkg/service/schematic/symbol/versions/v5"
	v13 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v13"
	v14 "github.com/synnaxlabs/synnax/pkg/service/schematic/versions/v14"
	color "github.com/synnaxlabs/x/color/versions/v2"
	"github.com/synnaxlabs/x/encoding/orc"
	spatial "github.com/synnaxlabs/x/spatial/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
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

var _ = Describe("MigrateSchematic", func() {
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
			Expect(sel.Options).To(HaveLen(2))
			Expect(sel.Options[1].Name).To(Equal("Open"))
			Expect(sel.Options[1].Color).ToNot(BeNil())
			ind, ok := out.Nodes[1].Config.Variant.(v14.StateIndicatorNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(ind.Enumeration).To(BeNil())
			Expect(ind.Channel).To(BeEquivalentTo(1048578))
			Expect(ind.Options).To(HaveLen(2))
		},
	)

	It(
		"Should keep the state overrides of custom nodes that 0.62.0 stored",
		func(ctx SpecContext) {
			var stored v13.Schematic
			Expect(
				orc.Codec.Decode(loadWire("v13_released_custom.hex"), &stored),
			).To(Succeed())
			out := MustSucceed(v14.MigrateSchematic(ctx, stored))
			Expect(out.Nodes).To(HaveLen(2))
			region := func(stroke, fill *color.Color) symbol.Region {
				return symbol.Region{
					Key:         "body",
					Name:        "Body",
					Selectors:   []string{"#body"},
					StrokeColor: stroke,
					FillColor:   fill,
				}
			}
			act, ok := out.Nodes[0].Config.Variant.(v14.CustomActuatorNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(act.SpecKey).To(Equal("3e46d160-a00b-4224-8119-ffabc45cdc24"))
			Expect(act.StateOverrides).To(Equal([]symbol.StateOverride{
				{
					Key:  "base",
					Name: "Base",
					Regions: []symbol.Region{region(
						&color.Color{R: 220, G: 19, B: 108, A: 255},
						&color.Color{R: 255, G: 107, B: 107, A: 128},
					)},
				},
				{
					Key:  "active",
					Name: "Active",
					Regions: []symbol.Region{
						region(nil, &color.Color{R: 62, G: 142, B: 247, A: 255}),
					},
				},
			}))
			static, ok := out.Nodes[1].Config.Variant.(v14.CustomStaticNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(static.StateOverrides).To(Equal([]symbol.StateOverride{
				{
					Key:  "base",
					Name: "Base",
					Regions: []symbol.Region{
						region(&color.Color{R: 1, G: 2, B: 3, A: 4}, nil),
					},
				},
			}))
		},
	)
})

var options = []v13.StateMapping{{Key: "open", Name: "Open", Value: 1}}

var _ = Describe("MigrateNodeConfig", func() {
	DescribeTable("Should keep the options and name no enumeration", func(
		ctx SpecContext,
		old v13.NodeConfigVariant,
		expected v14.NodeConfigVariant,
	) {
		out := MustSucceed(v14.MigrateNodeConfig(ctx, v13.NodeConfig{Variant: old}))
		Expect(out.Variant).To(Equal(expected))
	},
		Entry(
			"select",
			v13.SelectNodeConfig{Options: options},
			v14.SelectNodeConfig{Options: options},
		),
		Entry(
			"state indicator",
			v13.StateIndicatorNodeConfig{Options: options},
			v14.StateIndicatorNodeConfig{Options: options},
		),
	)
})

var _ = Describe("MigrateNode", func() {
	It(
		"Should keep the placement of a node and lift its config",
		func(ctx SpecContext) {
			out := MustSucceed(v14.MigrateNode(ctx, v13.Node{
				Key:      "valve",
				Position: spatial.XY{X: 10, Y: 20},
				ZIndex:   3,
				Parent:   new("group"),
				Config: v13.NodeConfig{
					Variant: v13.SelectNodeConfig{Options: options},
				},
			}))
			Expect(out.Key).To(Equal("valve"))
			Expect(out.Position).To(Equal(spatial.XY{X: 10, Y: 20}))
			Expect(out.ZIndex).To(BeEquivalentTo(3))
			Expect(out.Parent).To(Equal(new("group")))
			sel, ok := out.Config.Variant.(v14.SelectNodeConfig)
			Expect(ok).To(BeTrue())
			Expect(sel.Options).To(Equal(options))
			Expect(sel.Enumeration).To(BeNil())
		},
	)
})
