// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package legacy_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/distribution/channel"
	"github.com/synnaxlabs/synnax/pkg/service/lineplot/migrations/legacy"
	v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/migrations/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/lineplot/migrations/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/lineplot/migrations/legacy/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/lineplot/migrations/legacy/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/lineplot/migrations/legacy/v4"
	"github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/encoding/msgpack"
	. "github.com/synnaxlabs/x/testutil"
)

func jsonMap(raw string) msgpack.EncodedJSON {
	var m map[string]any
	Expect(json.Unmarshal([]byte(raw), &m)).To(Succeed())
	return m
}

// nonZeroV0 builds a v0.Data with every model field populated to a non-zero
// value so step-migrate passthrough regressions surface.
func nonZeroV0() v0.Data {
	label := "pressure"
	return v0.Data{
		Version:       v0.Version,
		Key:           "plot-1",
		RemoteCreated: true,
		Title:         v0.Title{Level: "h4", Visible: true},
		Legend:        v0.Legend{Visible: false},
		Channels: v0.Channels{
			X1: 1, X2: 2,
			Y1: []channel.Key{10, 11}, Y2: []channel.Key{12}, Y3: []channel.Key{}, Y4: []channel.Key{},
		},
		Ranges: v0.Ranges{X1: []string{"00000000-0000-0000-0000-000000000010"}},
		Axes: v0.AxesContainer{
			RenderTrigger:    7,
			HasHadChannelSet: true,
			Axes: v0.Axes{
				X1: v0.Axis{Key: "x1", Label: "time", LabelDirection: "x", LabelLevel: "small", Bounds: v0.Bounds{Lower: 0, Upper: 100}, AutoBounds: v0.AutoBounds{Lower: true, Upper: true}, TickSpacing: 75},
				X2: v0.Axis{Key: "x2", Label: "", LabelDirection: "x", LabelLevel: "small", Bounds: v0.Bounds{Lower: 0, Upper: 0}, AutoBounds: v0.AutoBounds{Lower: true, Upper: true}, TickSpacing: 75},
				Y1: v0.Axis{Key: "y1", Label: "p", LabelDirection: "x", LabelLevel: "small", Bounds: v0.Bounds{Lower: -1, Upper: 5}, AutoBounds: v0.AutoBounds{Lower: false, Upper: false}, TickSpacing: 60},
				Y2: v0.Axis{Key: "y2", Label: "", LabelDirection: "x", LabelLevel: "small", Bounds: v0.Bounds{Lower: 0, Upper: 0}, AutoBounds: v0.AutoBounds{Lower: true, Upper: true}, TickSpacing: 75},
				Y3: v0.Axis{Key: "y3", Label: "", LabelDirection: "x", LabelLevel: "small", Bounds: v0.Bounds{Lower: 0, Upper: 0}, AutoBounds: v0.AutoBounds{Lower: true, Upper: true}, TickSpacing: 75},
				Y4: v0.Axis{Key: "y4", Label: "", LabelDirection: "x", LabelLevel: "small", Bounds: v0.Bounds{Lower: 0, Upper: 0}, AutoBounds: v0.AutoBounds{Lower: true, Upper: true}, TickSpacing: 75},
			},
		},
		Lines: []v0.Line{
			{Key: "y1-rng1-ch10", Label: &label, Color: color.MustFromHex("#ff0000"), StrokeWidth: 2, Downsample: 1, DownsampleMode: "decimate"},
			{Key: "y1-rng1-ch11", Color: color.MustFromHex("#00ff00"), StrokeWidth: 1, Downsample: 2, DownsampleMode: "average"},
		},
		Rules: []v0.Rule{
			{Key: "r1", Label: "max", Color: color.MustFromHex("#0000ff"), Axis: "y1", LineWidth: 1, LineDash: 0, Units: "psi", Position: 4.5},
		},
	}
}

var _ = Describe("MigrateData", func() {
	Describe("version dispatch", func() {
		It("Should walk a v0 blob through every step to v4.Data", func() {
			out := MustSucceed(legacy.MigrateData(jsonMap(`{
				"version": "0.0.0",
				"axes": {"renderTrigger": 0, "hasHadChannelSet": false, "axes": {
					"x1": {"key": "x1", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"x2": {"key": "x2", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y1": {"key": "y1", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y2": {"key": "y2", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y3": {"key": "y3", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y4": {"key": "y4", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75}
				}}
			}`)))
			Expect(out.Version).To(Equal(v4.Version))
			Expect(out.Legend.Visible).To(BeTrue())
			Expect(out.Legend.Position.X).To(Equal(50.0))
			Expect(out.Axes.Axes.X1.Type).NotTo(BeNil())
			Expect(*out.Axes.Axes.X1.Type).To(Equal("time"))
			Expect(out.Axes.Axes.Y1.LabelDirection).To(Equal("y"))
		})

		It("Should fall back to v0 when the blob has no version field", func() {
			out := MustSucceed(legacy.MigrateData(jsonMap(`{"title": {"level": "h4", "visible": false}}`)))
			Expect(out.Version).To(Equal(v4.Version))
			Expect(out.Title.Level).To(Equal("h4"))
		})

		It("Should walk a nil blob to a zero v4.Data", func() {
			out := MustSucceed(legacy.MigrateData(nil))
			Expect(out.Version).To(Equal(v4.Version))
		})

		It("Should error on an unknown declared version", func() {
			Expect(legacy.MigrateData(jsonMap(`{"version": "99.0.0"}`))).Error().
				To(MatchError(ContainSubstring("unknown line plot data version")))
		})

		It("Should produce a v4-shaped Data when given a v2 input", func() {
			out := MustSucceed(legacy.MigrateData(jsonMap(`{
				"version": "2.0.0",
				"title": {"level": "h3", "visible": true},
				"legend": {"visible": false, "position": {"x": 1, "y": 2}},
				"axes": {"renderTrigger": 0, "hasHadChannelSet": false, "axes": {
					"x1": {"key": "x1", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75, "type": "time"},
					"x2": {"key": "x2", "label": "", "labelDirection": "x", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75, "type": "time"},
					"y1": {"key": "y1", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y2": {"key": "y2", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75},
					"y3": {"key": "y3", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75,"type": null},
					"y4": {"key": "y4", "label": "", "labelDirection": "y", "labelLevel": "small", "bounds": {"lower": 0, "upper": 0}, "autoBounds": {"lower": true, "upper": true}, "tickSpacing": 75}
				}}
			}`)))
			Expect(out.Version).To(Equal(v4.Version))
			Expect(out.Title.Level).To(Equal("h3"))
			Expect(out.Legend.Visible).To(BeFalse())
			Expect(out.Legend.Position.X).To(Equal(1.0))
		})
	})
})

// Each step is fed nonZeroV0() chained up to its input version. Tests assert
// the step's *new* fields and that every prior field passes through unchanged.

var _ = Describe("Step migrations", func() {
	Describe("v1.Migrate (v0 -> v1)", func() {
		It("Should attach the default legend (visible, sticky 50/50 px)", func() {
			out := v1.Migrate(nonZeroV0())
			Expect(out.Version).To(Equal(v1.Version))
			Expect(out.Legend).To(Equal(v1.ZeroLegend))
			Expect(out.Legend.Visible).To(BeTrue())
			Expect(out.Legend.Position.X).To(Equal(50.0))
			Expect(out.Legend.Position.Units.X).To(Equal("px"))
		})

		It("Should reset the user-set legend.visible to the default (mirroring the console v1 migration)", func() {
			in := nonZeroV0()
			in.Legend.Visible = false
			out := v1.Migrate(in)
			Expect(out.Legend.Visible).To(BeTrue())
		})

		It("Should pass every v0 model field through unchanged", func() {
			in := nonZeroV0()
			out := v1.Migrate(in)
			Expect(out.Key).To(Equal(in.Key))
			Expect(out.RemoteCreated).To(Equal(in.RemoteCreated))
			Expect(out.Title).To(Equal(in.Title))
			Expect(out.Channels).To(Equal(in.Channels))
			Expect(out.Ranges).To(Equal(in.Ranges))
			Expect(out.Axes).To(Equal(in.Axes))
			Expect(out.Lines).To(Equal(in.Lines))
			Expect(out.Rules).To(Equal(in.Rules))
		})
	})

	Describe("v2.Migrate (v1 -> v2)", func() {
		It("Should set x-axis Type to 'time'", func() {
			out := v2.Migrate(v1.Migrate(nonZeroV0()))
			Expect(out.Version).To(Equal(v2.Version))
			Expect(out.Axes.Axes.X1.Type).NotTo(BeNil())
			Expect(*out.Axes.Axes.X1.Type).To(Equal("time"))
			Expect(out.Axes.Axes.X2.Type).NotTo(BeNil())
			Expect(*out.Axes.Axes.X2.Type).To(Equal("time"))
		})

		It("Should leave y-axis Type unset", func() {
			out := v2.Migrate(v1.Migrate(nonZeroV0()))
			Expect(out.Axes.Axes.Y1.Type).To(BeNil())
			Expect(out.Axes.Axes.Y4.Type).To(BeNil())
		})

		It("Should flip y-axis labelDirection from 'x' to 'y'", func() {
			out := v2.Migrate(v1.Migrate(nonZeroV0()))
			Expect(out.Axes.Axes.Y1.LabelDirection).To(Equal("y"))
			Expect(out.Axes.Axes.Y4.LabelDirection).To(Equal("y"))
		})

		It("Should leave x-axis labelDirection as 'x'", func() {
			out := v2.Migrate(v1.Migrate(nonZeroV0()))
			Expect(out.Axes.Axes.X1.LabelDirection).To(Equal("x"))
		})

		It("Should pass every v1 non-axes model field through unchanged", func() {
			in := v1.Migrate(nonZeroV0())
			out := v2.Migrate(in)
			Expect(out.Key).To(Equal(in.Key))
			Expect(out.RemoteCreated).To(Equal(in.RemoteCreated))
			Expect(out.Title).To(Equal(in.Title))
			Expect(out.Legend).To(Equal(in.Legend))
			Expect(out.Channels).To(Equal(in.Channels))
			Expect(out.Ranges).To(Equal(in.Ranges))
			Expect(out.Lines).To(Equal(in.Lines))
			Expect(out.Rules).To(Equal(in.Rules))
		})

		It("Should preserve per-axis bounds, tickSpacing, label, and autoBounds", func() {
			in := v1.Migrate(nonZeroV0())
			out := v2.Migrate(in)
			Expect(out.Axes.Axes.Y1.Bounds).To(Equal(in.Axes.Axes.Y1.Bounds))
			Expect(out.Axes.Axes.Y1.AutoBounds).To(Equal(in.Axes.Axes.Y1.AutoBounds))
			Expect(out.Axes.Axes.Y1.TickSpacing).To(Equal(in.Axes.Axes.Y1.TickSpacing))
			Expect(out.Axes.Axes.Y1.Label).To(Equal(in.Axes.Axes.Y1.Label))
		})
	})

	Describe("v3.Migrate (v2 -> v3)", func() {
		It("Should rewrite the version string and pass every other field through", func() {
			in := v2.Migrate(v1.Migrate(nonZeroV0()))
			out := v3.Migrate(in)
			Expect(out.Version).To(Equal(v3.Version))
			Expect(out.Title).To(Equal(in.Title))
			Expect(out.Legend).To(Equal(in.Legend))
			Expect(out.Channels).To(Equal(in.Channels))
			Expect(out.Ranges).To(Equal(in.Ranges))
			Expect(out.Axes).To(Equal(in.Axes))
			Expect(out.Lines).To(Equal(in.Lines))
			Expect(out.Rules).To(Equal(in.Rules))
		})
	})

	Describe("v4.Migrate (v3 -> v4)", func() {
		It("Should rewrite the version string and pass every other field through", func() {
			in := v3.Migrate(v2.Migrate(v1.Migrate(nonZeroV0())))
			out := v4.Migrate(in)
			Expect(out.Version).To(Equal(v4.Version))
			Expect(out.Title).To(Equal(in.Title))
			Expect(out.Legend).To(Equal(in.Legend))
			Expect(out.Channels).To(Equal(in.Channels))
			Expect(out.Ranges).To(Equal(in.Ranges))
			Expect(out.Axes).To(Equal(in.Axes))
			Expect(out.Lines).To(Equal(in.Lines))
			Expect(out.Rules).To(Equal(in.Rules))
		})
	})
})
