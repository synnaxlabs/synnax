// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v6_test

import (
	"github.com/google/uuid"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v6 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v6"
	"github.com/synnaxlabs/synnax/pkg/service/ontology"
	text "github.com/synnaxlabs/x/text/versions/v0"
)

var _ = Describe("LinePlot", func() {
	Describe("GorpKey", func() {
		It("Should return the plot's key", func() {
			k := uuid.New()
			Expect(v6.LinePlot{Key: k}.GorpKey()).To(Equal(k))
		})
	})

	Describe("SetOptions", func() {
		It("Should return no options", func() {
			Expect(v6.LinePlot{}.SetOptions()).To(BeNil())
		})
	})

	Describe("OntologyID", func() {
		It("Should return the line plot ontology identifier", func() {
			k := uuid.New()
			Expect(v6.LinePlot{Key: k}.OntologyID()).To(Equal(ontology.ID{
				Type: ontology.ResourceTypeLineplot, Key: k.String(),
			}))
		})
	})

	Describe("ApplyDefaults", func() {
		It("Should fill schema-declared defaults", func() {
			lp := v6.LinePlot{Lines: []v6.Line{{}}, Rules: []v6.Rule{{}}}
			lp.ApplyDefaults()
			Expect(lp.Title.Level).To(Equal(text.LevelH4))
			Expect(lp.Axes.X1.Key).To(Equal(v6.AxisKeyX1))
			Expect(lp.Axes.X1.TickSpacing).To(Equal(75.0))
			Expect(lp.Lines[0].StrokeWidth).To(Equal(2.0))
			Expect(lp.Lines[0].DownsampleMode).To(Equal(v6.DownsampleModeDecimate))
			Expect(lp.Rules[0].LineWidth).To(Equal(1.0))
		})
	})

	Describe("Validate", func() {
		It("Should accept a defaulted plot", func() {
			lp := v6.LinePlot{
				Name:  "valid",
				Lines: []v6.Line{{}},
				Rules: []v6.Rule{{Axis: v6.AxisKeyY1}},
			}
			lp.ApplyDefaults()
			Expect(lp.Validate()).To(Succeed())
		})

		It("Should reject an empty name", func() {
			lp := v6.LinePlot{}
			lp.ApplyDefaults()
			Expect(lp.Validate()).To(MatchError(ContainSubstring("name: required")))
		})

		It("Should path errors to the offending line and rule", func() {
			lp := v6.LinePlot{
				Name:  "valid",
				Lines: []v6.Line{{DownsampleMode: "bogus"}},
				Rules: []v6.Rule{{Axis: "bogus"}},
			}
			lp.ApplyDefaults()
			Expect(lp.Validate()).To(SatisfyAll(
				MatchError(ContainSubstring("lines.0")),
				MatchError(ContainSubstring("rules.0")),
			))
		})
	})
})
