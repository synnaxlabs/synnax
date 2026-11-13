// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v10_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	v10 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v10"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
)

var plotKey = uuid.MustParse("11111111-2222-4333-8444-555555555555")

func defaultLine(yAxis v10.YAxisKey, yChannel, xChannel channel.Key) v10.Line {
	return v10.Line{
		Key:         v8.LineKey(plotKey, yAxis, yChannel, xChannel),
		YAxis:       yAxis,
		YChannel:    yChannel,
		XChannel:    xChannel,
		StrokeWidth: 2,
		Aggregation: v10.AggregationMinMax,
		Detail:      v10.DetailMedium,
	}
}

var _ = Describe("ReconcileLines", func() {
	It("Should create one default line per bound y-channel", func() {
		channels := v10.Channels{X1: 10, Y1: []channel.Key{5, 6}, Y2: []channel.Key{7}}
		Expect(v10.ReconcileLines(plotKey, channels, nil)).To(Equal([]v10.Line{
			defaultLine("y1", 5, 10),
			defaultLine("y1", 6, 10),
			defaultLine("y2", 7, 10),
		}))
	})

	It("Should create one line for a channel bound twice to an axis", func() {
		channels := v10.Channels{Y1: []channel.Key{5, 5}}
		Expect(v10.ReconcileLines(plotKey, channels, nil)).To(HaveLen(1))
	})

	It("Should keep an existing line's styling and re-key it to the plot", func() {
		existing := v10.Line{
			Key:         "old",
			YAxis:       "y1",
			YChannel:    5,
			XChannel:    10,
			StrokeWidth: 7,
			Color:       &colorv2.Color{R: 1, A: 128},
		}
		lines := v10.ReconcileLines(
			plotKey,
			v10.Channels{X1: 10, Y1: []channel.Key{5}},
			[]v10.Line{existing},
		)
		Expect(lines).To(HaveLen(1))
		Expect(lines[0].StrokeWidth).To(Equal(7.0))
		Expect(lines[0].Color).To(HaveValue(Equal(colorv2.Color{R: 1, A: 128})))
		Expect(lines[0].Key).To(Equal(v8.LineKey(plotKey, "y1", 5, 10)))
	})

	It("Should drop a line whose channel is no longer bound", func() {
		existing := v10.Line{YAxis: "y1", YChannel: 9, XChannel: 10}
		Expect(v10.ReconcileLines(plotKey, v10.Channels{X1: 10}, []v10.Line{existing})).
			To(BeEmpty())
	})

	It("Should replace a line plotted against a different x-channel", func() {
		existing := v10.Line{YAxis: "y1", YChannel: 5, XChannel: 20, StrokeWidth: 7}
		Expect(v10.ReconcileLines(
			plotKey,
			v10.Channels{X1: 10, Y1: []channel.Key{5}},
			[]v10.Line{existing},
		)).To(Equal([]v10.Line{defaultLine("y1", 5, 10)}))
	})
})
