// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v11_test

import (
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	colorv0 "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v11 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v11"
	ranger "github.com/synnaxlabs/synnax/pkg/service/ranger/versions/v4"
	"github.com/synnaxlabs/synnax/pkg/service/resource"
	xcolor "github.com/synnaxlabs/x/color"
	"github.com/synnaxlabs/x/defaults"
	telem "github.com/synnaxlabs/x/telem/versions/v0"
	text "github.com/synnaxlabs/x/text/versions/v0"
)

var _ = Describe("LinePlot", func() {
	Describe("GorpKey", func() {
		It("Should return the plot's key", func() {
			k := uuid.New()
			Expect(v11.LinePlot{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("ResourceID", func() {
		It("Should return the line plot resource identifier", func() {
			k := uuid.New()
			Expect(v11.LinePlot{Key: k}.ResourceID()).To(Equal(resource.ID{
				Type: resource.TypeLineplot, Key: k.String(),
			}))
		})
	})

	Describe("Defaults", func() {
		It("Should fill schema-declared defaults", func() {
			lp := v11.LinePlot{Lines: []v11.Line{{}}, Rules: []v11.Rule{{}}}
			lp = defaults.Fill(lp)
			Expect(lp.Title.Level).To(Equal(text.LevelH4))
			Expect(lp.Axes.X1.Key).To(Equal(v11.AxisKeyX1))
			Expect(lp.Axes.X1.TickSpacing).To(Equal(75.0))
			Expect(lp.Lines[0].StrokeWidth).To(Equal(2.0))
			Expect(lp.Lines[0].Aggregation).To(Equal(v11.AggregationMinMax))
			Expect(lp.Lines[0].Detail).To(Equal(v11.DetailMedium))
			Expect(lp.Rules[0].LineWidth).To(Equal(1.0))
			Expect(lp.Axes.X1.TimeMode).To(BeNil())
		})
	})

	Describe("Validate", func() {
		It("Should accept a defaulted plot", func() {
			lp := v11.LinePlot{
				Name:  "valid",
				Lines: []v11.Line{{YAxis: v11.YAxisKeyY1}},
				Rules: []v11.Rule{{Axis: v11.AxisKeyY1}},
			}
			lp = defaults.Fill(lp)
			Expect(lp.Validate()).To(Succeed())
		})

		It("Should reject a line without a y-axis", func() {
			lp := v11.LinePlot{Name: "valid", Lines: []v11.Line{{}}}
			lp = defaults.Fill(lp)
			Expect(lp.Validate()).
				To(MatchError(ContainSubstring("lines.0.y_axis: invalid y_axis")))
		})

		It("Should reject a line stroke width below 1", func() {
			lp := v11.LinePlot{
				Name:  "valid",
				Lines: []v11.Line{{YAxis: v11.YAxisKeyY1, StrokeWidth: 0.5}},
			}
			lp = defaults.Fill(lp)
			Expect(lp.Validate()).
				To(MatchError(ContainSubstring("lines.0.stroke_width")))
		})

		It("Should reject an empty name", func() {
			lp := v11.LinePlot{}
			lp = defaults.Fill(lp)
			Expect(lp.Validate()).To(MatchError(ContainSubstring("name: required")))
		})

		It("Should path errors to the offending line and rule", func() {
			lp := v11.LinePlot{
				Name:  "valid",
				Lines: []v11.Line{{YAxis: v11.YAxisKeyY1, Detail: "bogus"}},
				Rules: []v11.Rule{{Axis: "bogus"}},
			}
			lp = defaults.Fill(lp)
			Expect(lp.Validate()).To(SatisfyAll(
				MatchError(ContainSubstring("lines.0")),
				MatchError(ContainSubstring("rules.0")),
			))
		})
	})

	Describe("Range references", func() {
		k1, k2 := uuid.New(), uuid.New()
		key := func(k uuid.UUID) ranger.Ref { return ranger.Ref{Variant: ranger.KeyRef{Key: k}} }
		window := func(span telem.TimeSpan) ranger.Ref {
			return ranger.Ref{Variant: ranger.WindowRef{
				Window: telem.Window{Variant: telem.DynamicWindow{TimeSpan: span}},
			}}
		}
		t0 := telem.TimeStamp(5)
		red := &colorv0.Ref{
			Variant: colorv0.FixedRef{Value: xcolor.Color{R: 255, A: 255}},
		}

		DescribeTable("SameRange",
			func(a, b ranger.Ref, same bool) {
				Expect(v11.SameRange(a, b)).To(Equal(same))
			},
			Entry("one key", key(k1), key(k1), true),
			Entry("two keys", key(k1), key(k2), false),
			Entry("one key with another T0 and color",
				key(k1),
				ranger.Ref{Variant: ranger.KeyRef{Key: k1, T0: &t0, Color: red}},
				true,
			),
			Entry("one window", window(telem.Second), window(telem.Second), true),
			Entry("two windows", window(telem.Second), window(telem.Minute), false),
			Entry("one window with another color",
				window(telem.Second),
				ranger.Ref{Variant: ranger.WindowRef{
					Window: telem.Window{
						Variant: telem.DynamicWindow{TimeSpan: telem.Second},
					},
					Color: red,
				}},
				true,
			),
			Entry("a key and a window", key(k1), window(telem.Second), false),
			Entry("a window and a key", window(telem.Second), key(k1), false),
		)

		Describe("UniqueRanges", func() {
			It("Should keep the first reference to each range, in order", func() {
				colored := ranger.Ref{Variant: ranger.KeyRef{Key: k1, Color: red}}
				Expect(v11.UniqueRanges([]ranger.Ref{
					colored,
					window(telem.Second),
					key(k2),
					key(k1),
					window(telem.Second),
				})).To(Equal([]ranger.Ref{colored, window(telem.Second), key(k2)}))
			})

			It("Should return nil for nil", func() {
				Expect(v11.UniqueRanges(nil)).To(BeNil())
			})
		})
	})
})
