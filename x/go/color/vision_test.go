// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package color_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/color"
)

var _ = Describe("Vision", func() {
	Describe("Simulate", func() {
		// The values match color.simulate in x/ts, in the order of Deficiencies.
		DescribeTable(
			"Should match the TypeScript simulation",
			func(c color.Color, expected ...color.Color) {
				Expect(color.Deficiencies).To(HaveLen(len(expected)))
				for i, d := range color.Deficiencies {
					Expect(color.Simulate(c, d)).To(Equal(expected[i]))
				}
			},
			Entry("red",
				color.Color{R: 255, A: 255},
				color.Color{R: 109, G: 95, A: 255},
				color.Color{R: 163, G: 144, A: 255},
				color.Color{R: 255, B: 15, A: 255},
			),
			Entry("green",
				color.Color{G: 255, A: 255},
				color.Color{R: 255, G: 229, A: 255},
				color.Color{R: 239, G: 214, B: 58, A: 255},
				color.Color{G: 247, B: 217, A: 255},
			),
			Entry("blue",
				color.Color{R: 32, G: 164, B: 243, A: 255},
				color.Color{R: 126, G: 166, B: 246, A: 255},
				color.Color{R: 97, G: 148, B: 242, A: 255},
				color.Color{G: 184, B: 192, A: 255},
			),
			Entry("gray",
				color.Color{R: 128, G: 128, B: 128, A: 255},
				color.Color{R: 128, G: 128, B: 128, A: 255},
				color.Color{R: 128, G: 128, B: 128, A: 255},
				color.Color{R: 128, G: 128, B: 128, A: 255},
			),
		)

		It("Should keep the alpha", func() {
			c := color.Color{R: 255, A: 128}
			for _, d := range color.Deficiencies {
				Expect(color.Simulate(c, d).A).To(Equal(uint8(128)))
			}
		})
	})

	Describe("Distance", func() {
		It("Should put black and white 1 apart", func() {
			black := color.Color{A: 255}
			white := color.Color{R: 255, G: 255, B: 255, A: 255}
			Expect(color.Distance(black, white)).To(BeNumerically("~", 1, 1e-6))
		})

		It("Should match the TypeScript distance", func() {
			red := color.Color{R: 255, A: 255}
			pink := color.Color{R: 220, G: 19, B: 108, A: 255}
			Expect(color.Distance(red, pink)).To(BeNumerically("~", 0.1208930593, 1e-9))
		})

		It("Should ignore alpha", func() {
			Expect(color.Distance(
				color.Color{R: 255, A: 255},
				color.Color{R: 255, A: 0},
			)).To(BeZero())
		})
	})
})
