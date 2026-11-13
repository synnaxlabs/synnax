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
	v9 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v9"
	v10 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v10"
	color "github.com/synnaxlabs/x/color/versions/v0"
	colorv2 "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
)

var alphaEntries = []TableEntry{
	Entry("a transparent alpha", 0.0, uint8(0)),
	Entry("a faint alpha", 0.003, uint8(1)),
	Entry("a half alpha", 0.5, uint8(128)),
	Entry("an opaque alpha", 1.0, uint8(255)),
}

var _ = Describe("Migrate", func() {
	DescribeTable("Should store the line color alpha as a byte",
		func(ctx SpecContext, a float64, expected uint8) {
			l := MustSucceed(v10.MigrateLine(ctx, v9.Line{
				Key: "l1", Color: &color.Color{B: 3, A: a},
			}))
			Expect(l.Key).To(Equal("l1"))
			Expect(l.Color).To(HaveValue(Equal(colorv2.Color{B: 3, A: expected})))
		},
		alphaEntries,
	)

	DescribeTable("Should store the rule color alpha as a byte",
		func(ctx SpecContext, a float64, expected uint8) {
			r := MustSucceed(v10.MigrateRule(ctx, v9.Rule{
				Key: "r", Color: &color.Color{R: 9, A: a},
			}))
			Expect(r.Key).To(Equal("r"))
			Expect(r.Color).To(HaveValue(Equal(colorv2.Color{R: 9, A: expected})))
		},
		alphaEntries,
	)

	It(
		"Should convert the colors of every line and rule of a plot",
		func(ctx SpecContext) {
			key := uuid.New()
			group := uuid.New()
			old := v9.LinePlot{
				Key:   key,
				Name:  "plot",
				Lines: []v9.Line{{Key: "l1", Color: &color.Color{R: 1, A: 0.5}}},
				Rules: []v9.Rule{{Key: "r1", Color: &color.Color{G: 2, A: 1}}},
				Group: &group,
			}
			lp := MustSucceed(v10.MigrateLinePlot(ctx, old))
			Expect(lp.Key).To(Equal(key))
			Expect(lp.Name).To(Equal("plot"))
			Expect(lp.Group).To(HaveValue(Equal(group)))
			Expect(lp.Lines).To(HaveLen(1))
			Expect(lp.Lines[0].Color).To(HaveValue(Equal(colorv2.Color{R: 1, A: 128})))
			Expect(lp.Rules).To(HaveLen(1))
			Expect(lp.Rules[0].Color).To(HaveValue(Equal(colorv2.Color{G: 2, A: 255})))
		},
	)
})
