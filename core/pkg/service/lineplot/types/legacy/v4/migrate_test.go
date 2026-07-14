// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v4_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/synnax/pkg/service/channel"
	v0 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v0"
	v1 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v1"
	v2 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v2"
	v3 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v3"
	v4 "github.com/synnaxlabs/synnax/pkg/service/lineplot/types/legacy/v4"
	"github.com/synnaxlabs/x/color"
)

// nonZeroV3 builds a v3.Data with every model field populated so passthrough
// regressions in v4.Migrate surface.
func nonZeroV3() v3.Data {
	label := "pressure"
	return v3.Data{
		Version:       v3.Version,
		Key:           "plot-1",
		RemoteCreated: true,
		Title:         v0.Title{Level: "h4", Visible: true},
		Legend:        v1.ZeroLegend,
		Channels: v0.Channels{
			X1: 1, X2: 2,
			Y1: []channel.Key{10}, Y2: []channel.Key{}, Y3: []channel.Key{}, Y4: []channel.Key{},
		},
		Ranges: v0.Ranges{X1: []string{"00000000-0000-0000-0000-000000000010"}},
		Axes: v2.AxesContainer{
			RenderTrigger:    7,
			HasHadChannelSet: true,
			Axes: v2.Axes{
				X1: v2.Axis{Key: "x1", Label: "time", LabelDirection: "x", LabelLevel: "small", Bounds: v0.Bounds{Lower: 0, Upper: 100}, AutoBounds: v0.AutoBounds{Lower: true, Upper: true}, TickSpacing: 75, Type: "time"},
				X2: v2.Axis{Key: "x2", Label: "", LabelDirection: "x", LabelLevel: "small", Bounds: v0.Bounds{Lower: 0, Upper: 0}, AutoBounds: v0.AutoBounds{Lower: true, Upper: true}, TickSpacing: 75, Type: "time"},
				Y1: v2.Axis{Key: "y1", Label: "p", LabelDirection: "y", LabelLevel: "small", Bounds: v0.Bounds{Lower: -1, Upper: 5}, AutoBounds: v0.AutoBounds{Lower: false, Upper: false}, TickSpacing: 60},
				Y2: v2.Axis{Key: "y2", Label: "", LabelDirection: "y", LabelLevel: "small", Bounds: v0.Bounds{Lower: 0, Upper: 0}, AutoBounds: v0.AutoBounds{Lower: true, Upper: true}, TickSpacing: 75},
				Y3: v2.Axis{Key: "y3", Label: "", LabelDirection: "y", LabelLevel: "small", Bounds: v0.Bounds{Lower: 0, Upper: 0}, AutoBounds: v0.AutoBounds{Lower: true, Upper: true}, TickSpacing: 75},
				Y4: v2.Axis{Key: "y4", Label: "", LabelDirection: "y", LabelLevel: "small", Bounds: v0.Bounds{Lower: 0, Upper: 0}, AutoBounds: v0.AutoBounds{Lower: true, Upper: true}, TickSpacing: 75},
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

var _ = Describe("v4.Migrate (v3 -> v4)", func() {
	It("Should rewrite the version string and pass every other field through", func() {
		in := nonZeroV3()
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
