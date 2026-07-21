// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v1_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/types/v0"
	v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v1"
	"github.com/synnaxlabs/x/color"
)

// nonZeroV0 builds a v0.Data with every model field populated so passthrough
// regressions in v1.Migrate surface.
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
			Y1: []channel.Key{10, 11},
			Y2: []channel.Key{12},
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

var _ = Describe("v1.Migrate (v0 -> v1)", func() {
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
