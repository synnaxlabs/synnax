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
	color "github.com/synnaxlabs/synnax/pkg/service/color/versions/v0"
	v10 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v10"
	v9 "github.com/synnaxlabs/synnax/pkg/service/lineplot/versions/v9"
	xcolorv0 "github.com/synnaxlabs/x/color/versions/v0"
	xcolor "github.com/synnaxlabs/x/color/versions/v2"
	. "github.com/synnaxlabs/x/testutil"
)

func fixed(c xcolor.Color) color.Ref {
	return color.Ref{Variant: color.FixedRef{Value: c}}
}

func fixedHex(hex string) color.Ref { return fixed(MustSucceed(xcolor.Parse(hex))) }

var alphaEntries = []TableEntry{
	Entry("a transparent alpha", 0.0, uint8(0)),
	Entry("a faint alpha", 0.003, uint8(1)),
	Entry("a half alpha", 0.5, uint8(128)),
	Entry("an opaque alpha", 1.0, uint8(255)),
}

var paletteEntries = []TableEntry{
	Entry("index 0", 0, "#DC136C"),
	Entry("index 1", 1, "#20A4F3"),
	Entry("index 2", 2, "#7AC74F"),
	Entry("index 3", 3, "#FFC43D"),
	Entry("index 4", 4, "#FE5F55"),
	Entry("index 5", 5, "#8075FF"),
	Entry("index 6", 6, "#D90429"),
	Entry("index 7", 7, "#F7AEF8"),
	Entry("index 8", 8, "#F18F01"),
	Entry("index 9", 9, "#791E94"),
	Entry("index 10", 10, "#279AF1"),
	Entry("index 11", 11, "#D9FFF5"),
	Entry("index 12", 12, "#FF6B6B"),
	Entry("index 13", 13, "#52FFB8"),
	Entry("index 14, which wraps to the first color", 14, "#DC136C"),
	Entry("index 29, which wraps twice", 29, "#20A4F3"),
}

var _ = Describe("Migrate", func() {
	DescribeTable("Should store the line color as a fixed color with a byte alpha",
		func(ctx SpecContext, a float64, expected uint8) {
			l := MustSucceed(v10.MigrateLine(ctx, v9.Line{
				Key: "l1", Color: &xcolorv0.Color{B: 3, A: a},
			}))
			Expect(l.Key).To(Equal("l1"))
			Expect(l.Color).To(HaveValue(Equal(fixed(xcolor.Color{B: 3, A: expected}))))
		},
		alphaEntries,
	)

	DescribeTable("Should store the rule color as a fixed color with a byte alpha",
		func(ctx SpecContext, a float64, expected uint8) {
			r := MustSucceed(v10.MigrateRule(ctx, v9.Rule{
				Key: "r", Color: &xcolorv0.Color{R: 9, A: a},
			}))
			Expect(r.Key).To(Equal("r"))
			Expect(r.Color).To(HaveValue(Equal(fixed(xcolor.Color{R: 9, A: expected}))))
		},
		alphaEntries,
	)

	It("Should leave a lone line and rule with no color uncolored", func(
		ctx SpecContext,
	) {
		Expect(MustSucceed(v10.MigrateLine(ctx, v9.Line{Key: "l"})).Color).To(BeNil())
		Expect(MustSucceed(v10.MigrateRule(ctx, v9.Rule{Key: "r"})).Color).To(BeNil())
	})

	DescribeTable("Should give an uncolored line the palette color at its index",
		func(ctx SpecContext, index int, hex string) {
			lp := MustSucceed(v10.MigrateLinePlot(ctx, v9.LinePlot{
				Lines: make([]v9.Line, index+1),
			}))
			Expect(lp.Lines[index].Color).To(HaveValue(Equal(fixedHex(hex))))
		},
		paletteEntries,
	)

	DescribeTable("Should give an uncolored rule the palette color at its index",
		func(ctx SpecContext, index int, hex string) {
			lp := MustSucceed(v10.MigrateLinePlot(ctx, v9.LinePlot{
				Rules: make([]v9.Rule, index+1),
			}))
			Expect(lp.Rules[index].Color).To(HaveValue(Equal(fixedHex(hex))))
		},
		paletteEntries,
	)

	It(
		"Should convert the stored colors and fill the missing ones of a plot",
		func(ctx SpecContext) {
			key := uuid.New()
			group := uuid.New()
			old := v9.LinePlot{
				Key:  key,
				Name: "plot",
				Lines: []v9.Line{
					{Key: "l0", Color: &xcolorv0.Color{R: 1, A: 0.5}},
					{Key: "l1"},
				},
				Rules: []v9.Rule{
					{Key: "r0"},
					{Key: "r1", Color: &xcolorv0.Color{G: 2, A: 1}},
				},
				Group: &group,
			}
			lp := MustSucceed(v10.MigrateLinePlot(ctx, old))
			Expect(lp.Key).To(Equal(key))
			Expect(lp.Name).To(Equal("plot"))
			Expect(lp.Group).To(HaveValue(Equal(group)))
			Expect(lp.Lines).To(HaveLen(2))
			Expect(lp.Lines[0].Key).To(Equal("l0"))
			Expect(lp.Lines[0].Color).
				To(HaveValue(Equal(fixed(xcolor.Color{R: 1, A: 128}))))
			Expect(lp.Lines[1].Color).To(HaveValue(Equal(fixedHex("#20A4F3"))))
			Expect(lp.Rules).To(HaveLen(2))
			Expect(lp.Rules[0].Color).To(HaveValue(Equal(fixedHex("#DC136C"))))
			Expect(lp.Rules[1].Color).
				To(HaveValue(Equal(fixed(xcolor.Color{G: 2, A: 255}))))
		},
	)
})
