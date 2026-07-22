// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/types/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v2"
	"github.com/synnaxlabs/x/color"
)

// nonZeroV1 builds a v1.Data with every model field populated so passthrough
// regressions in v2.Migrate surface.
func nonZeroV1() v1.Data {
	label := "pressure"
	return v1.Data{
		Version:       v1.Version,
		Key:           "plot-1",
		RemoteCreated: true,
		Title:         v0.Title{Level: "h4", Visible: true},
		Legend:        v1.ZeroLegend,
		Channels: v0.Channels{
			X1: 1, X2: 2,
			Y1: []channel.Key{10}, Y2: []channel.Key{}, Y3: []channel.Key{}, Y4: []channel.Key{},
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
		},
		Rules: []v0.Rule{
			{Key: "r1", Label: "max", Color: color.MustFromHex("#0000ff"), Axis: "y1", LineWidth: 1, LineDash: 0, Units: "psi", Position: 4.5},
		},
	}
}

var _ = Describe("Migrate", func() {
	It("Should set x-axis Type to 'time'", func() {
		out := v2.Migrate(nonZeroV1())
		Expect(out.Version).To(Equal(v2.Version))
		Expect(out.Axes.Axes.X1.Type).To(Equal("time"))
		Expect(out.Axes.Axes.X2.Type).To(Equal("time"))
	})

	It("Should leave y-axis Type unset", func() {
		out := v2.Migrate(nonZeroV1())
		Expect(out.Axes.Axes.Y1.Type).To(BeEmpty())
		Expect(out.Axes.Axes.Y4.Type).To(BeEmpty())
	})

	It("Should flip y-axis labelDirection from 'x' to 'y'", func() {
		out := v2.Migrate(nonZeroV1())
		Expect(out.Axes.Axes.Y1.LabelDirection).To(Equal("y"))
		Expect(out.Axes.Axes.Y4.LabelDirection).To(Equal("y"))
	})

	It("Should leave x-axis labelDirection as 'x'", func() {
		out := v2.Migrate(nonZeroV1())
		Expect(out.Axes.Axes.X1.LabelDirection).To(Equal("x"))
	})

	It("Should pass every v1 non-axes model field through unchanged", func() {
		in := nonZeroV1()
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
		in := nonZeroV1()
		out := v2.Migrate(in)
		Expect(out.Axes.Axes.Y1.Bounds).To(Equal(in.Axes.Axes.Y1.Bounds))
		Expect(out.Axes.Axes.Y1.AutoBounds).To(Equal(in.Axes.Axes.Y1.AutoBounds))
		Expect(out.Axes.Axes.Y1.TickSpacing).To(Equal(in.Axes.Axes.Y1.TickSpacing))
		Expect(out.Axes.Axes.Y1.Label).To(Equal(in.Axes.Axes.Y1.Label))
	})
})
