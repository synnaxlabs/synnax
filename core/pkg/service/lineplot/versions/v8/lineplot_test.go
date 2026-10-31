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
	"uuid"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v8 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v8"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	text "github.com/synnaxlabs/x/text/versions/v0"
)

var _ = Describe("LinePlot", func() {
	Describe("GorpKey", func() {
		It("Should return the plot's key", func() {
			k := uuid.New()
			Expect(v8.LinePlot{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v8.LinePlot{}.SetOptions()).To(BeNil())
		})
	})

	Describe("OntologyID", func() {
		It("Should return the line plot ontology identifier", func() {
			k := uuid.New()
			Expect(v8.LinePlot{Key: k}.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeLineplot, Key: k.String(),
			}))
		})
	})

	Describe("ApplyDefaults", func() {
		It("Should fill schema-declared defaults", func() {
			lp := v8.LinePlot{Lines: []v8.Line{{}}, Rules: []v8.Rule{{}}}
			lp.ApplyDefaults()
			Expect(lp.Title.Level).To(Equal(text.LevelH4))
			Expect(lp.Axes.X1.Key).To(Equal(v8.AxisKeyX1))
			Expect(lp.Axes.X1.TickSpacing).To(Equal(75.0))
			Expect(lp.Lines[0].StrokeWidth).To(Equal(2.0))
			Expect(lp.Lines[0].Aggregation).To(Equal(v8.AggregationMinMax))
			Expect(lp.Lines[0].Detail).To(Equal(v8.DetailMedium))
			Expect(lp.Rules[0].LineWidth).To(Equal(1.0))
		})
	})

	Describe("Validate", func() {
		It("Should accept a defaulted plot", func() {
			lp := v8.LinePlot{
				Name:  "valid",
				Lines: []v8.Line{{YAxis: v8.YAxisKeyY1}},
				Rules: []v8.Rule{{Axis: v8.AxisKeyY1}},
			}
			lp.ApplyDefaults()
			Expect(lp.Validate()).To(Succeed())
		})

		It("Should reject a line without a y-axis", func() {
			lp := v8.LinePlot{Name: "valid", Lines: []v8.Line{{}}}
			lp.ApplyDefaults()
			Expect(lp.Validate()).
				To(MatchError(ContainSubstring("lines.0.y_axis: invalid y_axis")))
		})

		It("Should reject a line stroke width below 1", func() {
			lp := v8.LinePlot{
				Name:  "valid",
				Lines: []v8.Line{{YAxis: v8.YAxisKeyY1, StrokeWidth: 0.5}},
			}
			lp.ApplyDefaults()
			Expect(lp.Validate()).
				To(MatchError(ContainSubstring("lines.0.stroke_width")))
		})

		It("Should reject an empty name", func() {
			lp := v8.LinePlot{}
			lp.ApplyDefaults()
			Expect(lp.Validate()).To(MatchError(ContainSubstring("name: required")))
		})

		It("Should path errors to the offending line and rule", func() {
			lp := v8.LinePlot{
				Name:  "valid",
				Lines: []v8.Line{{YAxis: v8.YAxisKeyY1, Detail: "bogus"}},
				Rules: []v8.Rule{{Axis: "bogus"}},
			}
			lp.ApplyDefaults()
			Expect(lp.Validate()).To(SatisfyAll(
				MatchError(ContainSubstring("lines.0")),
				MatchError(ContainSubstring("rules.0")),
			))
		})
	})
})
