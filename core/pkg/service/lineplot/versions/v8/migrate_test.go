// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v8_test

import (
	"embed"
	"encoding/hex"
	"strings"
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/samber/lo"
	channel "github.com/synnaxlabs/synnax/pkg/service/channel/versions/v0"
	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v6"
	v7 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v7"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	ranger "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v1"
	"github.com/synnaxlabs/x/encoding/orc"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	. "github.com/synnaxlabs/x/testutil"
)

//go:embed testdata/*.hex
var fixtures embed.FS

// loadWire returns the Orc payload stored as hex in testdata. v6_released.hex is a
// stored v7 entry captured from a running Core.
func loadWire(name string) []byte {
	GinkgoHelper()
	raw := MustSucceed(fixtures.ReadFile("testdata/" + name))
	return MustSucceed(hex.DecodeString(strings.ReplaceAll(string(raw), "\n", "")))
}

var plotKey = uuid.MustParse("11111111-2222-4333-8444-555555555555")

// ka, kb, and kc are saved range keys.
const (
	ka = "00000000-0000-4000-8000-00000000000a"
	kb = "00000000-0000-4000-8000-00000000000b"
	kc = "00000000-0000-4000-8000-00000000000c"
)

func keyRef(key string) ranger.Ref {
	return ranger.Ref{Variant: ranger.KeyRef{Key: uuid.MustParse(key)}}
}

func dynamicRef(span telem.TimeSpan) ranger.Ref {
	return ranger.Ref{Variant: ranger.WindowRef{
		Window: telem.Window{Variant: telem.DynamicWindow{TimeSpan: span}},
	}}
}

func defaultLine(yAxis v8.YAxisKey, yChannel, xChannel channel.Key) v8.Line {
	return v8.Line{
		Key:         v8.LineKey(plotKey, yAxis, yChannel, xChannel),
		YAxis:       yAxis,
		YChannel:    yChannel,
		XChannel:    xChannel,
		StrokeWidth: 2,
		Aggregation: v8.AggregationMinMax,
		Detail:      v8.DetailMedium,
	}
}

var _ = Describe("LineKey", func() {
	It("Should derive a name-based UUID shared with the Console", func() {
		// Computed independently as uuid5(lineNamespace, "<plot>---y1---5---10").
		Expect(v8.LineKey(plotKey, "y1", 5, 10)).
			To(Equal("9a828c1e-611f-5db4-bc47-9bdc99c8f1ab"))
	})

	DescribeTable("Should change with each part of the line's identity",
		func(plot v8.Key, yAxis v8.YAxisKey, yChannel, xChannel channel.Key) {
			Expect(v8.LineKey(plot, yAxis, yChannel, xChannel)).
				ToNot(Equal(v8.LineKey(plotKey, "y1", 5, 10)))
		},
		Entry("plot", uuid.New(), v8.YAxisKey("y1"), channel.Key(5), channel.Key(10)),
		Entry("y-axis", plotKey, v8.YAxisKey("y2"), channel.Key(5), channel.Key(10)),
		Entry("y-channel", plotKey, v8.YAxisKey("y1"), channel.Key(6), channel.Key(10)),
		Entry("x-channel", plotKey, v8.YAxisKey("y1"), channel.Key(5), channel.Key(0)),
	)
})

var _ = Describe("ReconcileLines", func() {
	It("Should create one default line per bound y-channel", func() {
		channels := v8.Channels{X1: 10, Y1: []channel.Key{5, 6}, Y2: []channel.Key{7}}
		Expect(v8.ReconcileLines(plotKey, channels, nil)).To(Equal([]v8.Line{
			defaultLine("y1", 5, 10),
			defaultLine("y1", 6, 10),
			defaultLine("y2", 7, 10),
		}))
	})

	It("Should create one line for a channel bound twice to an axis", func() {
		channels := v8.Channels{Y1: []channel.Key{5, 5}}
		Expect(v8.ReconcileLines(plotKey, channels, nil)).To(HaveLen(1))
	})

	It("Should keep an existing line's styling and re-key it to the plot", func() {
		existing := v8.Line{
			Key:         "old",
			YAxis:       "y1",
			YChannel:    5,
			XChannel:    10,
			StrokeWidth: 7,
		}
		lines := v8.ReconcileLines(
			plotKey,
			v8.Channels{X1: 10, Y1: []channel.Key{5}},
			[]v8.Line{existing},
		)
		Expect(lines).To(HaveLen(1))
		Expect(lines[0].StrokeWidth).To(Equal(7.0))
		Expect(lines[0].Key).To(Equal(v8.LineKey(plotKey, "y1", 5, 10)))
	})

	It("Should drop a line whose channel is no longer bound", func() {
		existing := v8.Line{YAxis: "y1", YChannel: 9, XChannel: 10}
		Expect(v8.ReconcileLines(plotKey, v8.Channels{X1: 10}, []v8.Line{existing})).
			To(BeEmpty())
	})

	It("Should replace a line plotted against a different x-channel", func() {
		existing := v8.Line{YAxis: "y1", YChannel: 5, XChannel: 20, StrokeWidth: 7}
		Expect(v8.ReconcileLines(
			plotKey,
			v8.Channels{X1: 10, Y1: []channel.Key{5}},
			[]v8.Line{existing},
		)).To(Equal([]v8.Line{defaultLine("y1", 5, 10)}))
	})
})

var _ = Describe("MigrateLinePlot", func() {
	migrate := func(ctx SpecContext, old v7.LinePlot) v8.LinePlot {
		GinkgoHelper()
		return MustSucceed(v8.MigrateLinePlot(ctx, old))
	}

	Describe("Ranges", func() {
		It("Should merge the x1 and x2 keys in order without duplicates",
			func(ctx SpecContext) {
				out := migrate(ctx, v7.LinePlot{Ranges: v7.Ranges{
					X1: []string{ka, kb},
					X2: []string{kb, kc},
				}})
				Expect(out.Ranges).To(Equal([]ranger.Ref{
					keyRef(ka), keyRef(kb), keyRef(kc),
				}))
			})

		It("Should keep keys bound only to x2", func(ctx SpecContext) {
			out := migrate(ctx, v7.LinePlot{Ranges: v7.Ranges{X2: []string{ka}}})
			Expect(out.Ranges).To(Equal([]ranger.Ref{keyRef(ka)}))
		})

		DescribeTable("Should turn a built-in rolling key into a window with its span",
			func(ctx SpecContext, key string, span telem.TimeSpan) {
				out := migrate(ctx, v7.LinePlot{Ranges: v7.Ranges{X1: []string{key}}})
				Expect(out.Ranges).To(Equal([]ranger.Ref{dynamicRef(span)}))
			},
			Entry("recent", "recent", 30*telem.Second),
			Entry("rolling1m", "rolling1m", telem.Minute),
			Entry("rolling5m", "rolling5m", 5*telem.Minute),
			Entry("rolling15m", "rolling15m", 15*telem.Minute),
			Entry("rolling30m", "rolling30m", 30*telem.Minute),
			Entry("rolling1h", "rolling1h", telem.Hour),
			Entry("rolling6h", "rolling6h", 6*telem.Hour),
			Entry("rolling12h", "rolling12h", 12*telem.Hour),
			Entry("rolling1d", "rolling1d", telem.Day),
			Entry("rolling7d", "rolling7d", 7*telem.Day),
			Entry("rolling30d", "rolling30d", 30*telem.Day),
		)

		It("Should turn a dynamic custom range into a window", func(ctx SpecContext) {
			out := migrate(ctx, v7.LinePlot{Ranges: v7.Ranges{
				X1: []string{"custom"},
				Custom: &v7.CustomRange{
					Variant: v7.DynamicCustomRange{Span: telem.TimeSpan(telem.Hour)},
				},
			}})
			Expect(out.Ranges).To(Equal([]ranger.Ref{dynamicRef(telem.Hour)}))
		})

		It(
			"Should turn a static custom range into a fixed window",
			func(ctx SpecContext) {
				out := migrate(ctx, v7.LinePlot{Ranges: v7.Ranges{
					X1: []string{"custom"},
					Custom: &v7.CustomRange{
						Variant: v7.StaticCustomRange{Start: 10, End: 20},
					},
				}})
				Expect(out.Ranges).To(Equal([]ranger.Ref{{Variant: ranger.WindowRef{
					Window: telem.Window{Variant: telem.StaticWindow{
						TimeRange: telem.TimeRange{Start: 10, End: 20},
					}},
				}}}))
			},
		)

		It("Should drop a key that is not a UUID", func(ctx SpecContext) {
			out := migrate(
				ctx,
				v7.LinePlot{Ranges: v7.Ranges{X1: []string{"local", ka}}},
			)
			Expect(out.Ranges).To(Equal([]ranger.Ref{keyRef(ka)}))
		})

		It("Should drop the custom key when no window is set", func(ctx SpecContext) {
			out := migrate(ctx, v7.LinePlot{
				Ranges: v7.Ranges{X1: []string{"custom", ka}},
			})
			Expect(out.Ranges).To(Equal([]ranger.Ref{keyRef(ka)}))
		})
	})

	Describe("Lines", func() {
		It("Should merge per-range lines into one line per channel, keeping the "+
			"styling of the first in key order", func(ctx SpecContext) {
			red, blue := "red", "blue"
			out := migrate(ctx, v7.LinePlot{
				Key:      plotKey,
				Channels: v7.Channels{X1: 10, Y1: []channel.Key{5}},
				Ranges:   v7.Ranges{X1: []string{"a", "b"}},
				Lines: []v7.Line{
					{Key: "y1---x1---b---10---5", Label: &blue, StrokeWidth: 4},
					{Key: "y1---x1---a---10---5", Label: &red, StrokeWidth: 3},
				},
			})
			Expect(out.Lines).To(Equal([]v8.Line{{
				Key:         v8.LineKey(plotKey, "y1", 5, 10),
				YAxis:       "y1",
				YChannel:    5,
				XChannel:    10,
				Label:       &red,
				StrokeWidth: 3,
			}}))
		})

		It("Should keep a channel's line when none of its ranges remain",
			func(ctx SpecContext) {
				out := migrate(ctx, v7.LinePlot{
					Key:      plotKey,
					Channels: v7.Channels{Y1: []channel.Key{5}},
					Ranges:   v7.Ranges{X1: []string{"custom"}},
					Lines: []v7.Line{
						{Key: "y1---x1---custom---0---5", StrokeWidth: 3},
					},
				})
				Expect(out.Ranges).To(BeEmpty())
				Expect(out.Lines).To(HaveLen(1))
				Expect(out.Lines[0].StrokeWidth).To(Equal(3.0))
			})

		It("Should drop a line whose channel is on no axis", func(ctx SpecContext) {
			out := migrate(ctx, v7.LinePlot{
				Key:    plotKey,
				Ranges: v7.Ranges{X1: []string{"a"}},
				Lines:  []v7.Line{{Key: "y1---x1---a---0---5", StrokeWidth: 3}},
			})
			Expect(out.Lines).To(BeEmpty())
		})

		It("Should add a default line for a bound channel with no v7 line",
			func(ctx SpecContext) {
				out := migrate(ctx, v7.LinePlot{
					Key:      plotKey,
					Channels: v7.Channels{X1: 10, Y1: []channel.Key{5}},
				})
				Expect(out.Lines).To(Equal([]v8.Line{defaultLine("y1", 5, 10)}))
			})

		DescribeTable("Should ignore the styling of a line whose v7 key does not parse",
			func(ctx SpecContext, key string) {
				out := migrate(ctx, v7.LinePlot{
					Key:      plotKey,
					Channels: v7.Channels{Y1: []channel.Key{5}},
					Lines:    []v7.Line{{Key: key, StrokeWidth: 9}},
				})
				Expect(out.Lines).To(Equal([]v8.Line{defaultLine("y1", 5, 0)}))
			},
			Entry("too few parts", "l1"),
			Entry("unknown y-axis", "y9---x1---a---0---5"),
			Entry("non-numeric x-channel", "y1---x1---a---x---5"),
			Entry("non-numeric y-channel", "y1---x1---a---0---y"),
		)
	})

	It("Should lift a stored v6 entry through v7", func(ctx SpecContext) {
		var stored v6.LinePlot
		Expect(orc.Codec.Decode(ctx, loadWire("v6_released.hex"), &stored)).
			To(Succeed())
		old := MustSucceed(v7.MigrateLinePlot(ctx, stored))
		out := migrate(ctx, old)
		Expect(out.Key).To(Equal(old.Key))
		Expect(out.Channels).To(Equal(old.Channels))
		Expect(out.Ranges).To(Equal([]ranger.Ref{
			dynamicRef(telem.TimeSpan(108000000000000)),
		}))
		Expect(out.Lines).To(HaveLen(2))
		for _, l := range out.Lines {
			Expect(
				l.Key,
			).To(Equal(v8.LineKey(out.Key, l.YAxis, l.YChannel, l.XChannel)))
		}
		Expect([]channel.Key{out.Lines[0].YChannel, out.Lines[1].YChannel}).
			To(ConsistOf(channel.Key(1048581), channel.Key(1048580)))
	})
})

var _ = Describe("MigrateLine", func() {
	It("Should decode the y-axis and channels and leave the key empty",
		func(ctx SpecContext) {
			l := MustSucceed(v8.MigrateLine(ctx, v7.Line{Key: "y2---x1---a---10---5"}))
			Expect(l.Key).To(BeEmpty())
			Expect(l.YAxis).To(Equal(v8.YAxisKey("y2")))
			Expect(l.YChannel).To(Equal(channel.Key(5)))
			Expect(l.XChannel).To(Equal(channel.Key(10)))
		})

	It("Should reject a key that does not parse", func(ctx SpecContext) {
		Expect(v8.MigrateLine(ctx, v7.Line{Key: "l1"})).Error().
			To(MatchError(ContainSubstring(`malformed v7 line key "l1"`)))
	})
})

var _ = Describe("RaiseStrokeWidths", func() {
	It("Should raise widths below 1 to 1 and keep the rest", func(ctx SpecContext) {
		in := v8.LinePlot{Lines: []v8.Line{
			{StrokeWidth: 0},
			{StrokeWidth: 0.5},
			{StrokeWidth: 1},
			{StrokeWidth: 3.5},
		}}
		out := MustSucceed(v8.RaiseStrokeWidths(ctx, in))
		Expect(lo.Map(out.Lines, func(l v8.Line, _ int) float64 {
			return l.StrokeWidth
		})).To(Equal([]float64{1, 1, 1, 3.5}))
	})
})
