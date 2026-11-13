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
	"github.com/synnaxlabs/x/validate"
)

var _ = Describe("Color", func() {
	Describe("Parse", func() {
		DescribeTable(
			"Should parse a color string",
			func(input string, expected color.Color) {
				Expect(color.Parse(input)).To(Equal(expected))
			},
			Entry("a name", "red", color.Color{R: 255, A: 255}),
			Entry("a mixed case name", " RebeccaPurple ",
				color.Color{R: 102, G: 51, B: 153, A: 255}),
			Entry("the transparent keyword", "transparent", color.Color{}),
			Entry("6 digit hex", "#ff8000", color.Color{R: 255, G: 128, A: 255}),
			Entry("6 digit hex without a hash", "00ff00", color.Color{G: 255, A: 255}),
			Entry("8 digit hex", "#ff000080", color.Color{R: 255, A: 128}),
			Entry("upper case hex", "#FF0000", color.Color{R: 255, A: 255}),
			Entry("3 digit hex", "#f80", color.Color{R: 255, G: 136, A: 255}),
			Entry("4 digit hex", "#f808", color.Color{R: 255, G: 136, A: 136}),
			Entry("rgb", "rgb(59,196,84)", color.Color{R: 59, G: 196, B: 84, A: 255}),
			Entry("rgb with alpha", "rgb(59, 196, 84, 0.5)",
				color.Color{R: 59, G: 196, B: 84, A: 128}),
			Entry("rgba", "rgba(59,196,84,0.5)",
				color.Color{R: 59, G: 196, B: 84, A: 128}),
			Entry(
				"rgba without alpha",
				"rgba(1, 2, 3)",
				color.Color{R: 1, G: 2, B: 3, A: 255},
			),
			Entry(
				"rgb space syntax",
				"rgb(1 2 3 / 0.5)",
				color.Color{R: 1, G: 2, B: 3, A: 128},
			),
			Entry(
				"rgb percentages",
				"rgb(100% 0% 0% / 50%)",
				color.Color{R: 255, A: 128},
			),
			Entry(
				"surrounding whitespace",
				"  RGB(1,2,3)  ",
				color.Color{R: 1, G: 2, B: 3, A: 255},
			),
			Entry("hsl red", "hsl(0, 100%, 50%)", color.Color{R: 255, A: 255}),
			Entry("hsl green", "hsl(120,100%,50%)", color.Color{G: 255, A: 255}),
			Entry("hsl blue", "hsl(240deg 100% 50%)", color.Color{B: 255, A: 255}),
			Entry("hsl without percent signs", "hsl(0, 0, 100)",
				color.Color{R: 255, G: 255, B: 255, A: 255}),
			Entry("hsl wraps hue", "hsl(480, 100%, 50%)", color.Color{G: 255, A: 255}),
			Entry(
				"hsl wraps negative hue",
				"hsl(-120, 100%, 50%)",
				color.Color{B: 255, A: 255},
			),
			Entry("hsla", "hsla(0, 55%, 43%, 0)", color.Color{R: 170, G: 49, B: 49}),
		)
		DescribeTable("Should reject",
			func(input string) {
				Expect(
					color.Parse(input),
				).Error().
					To(MatchError(validate.ErrValidation))
			},
			Entry("an empty string", ""),
			Entry("an unknown name", "reddish"),
			Entry("a 5 digit hex", "#12345"),
			Entry("a non-hex digit", "#xyzxyz"),
			Entry("too few values", "rgb(1, 2)"),
			Entry("too many values", "rgb(1, 2, 3, 0.5, 1)"),
			Entry("a channel above 255", "rgb(300, 0, 0)"),
			Entry("a negative channel", "rgb(-1, 0, 0)"),
			Entry("an alpha above 1", "rgba(1, 2, 3, 1.5)"),
			Entry("a malformed alpha", "rgba(1, 2, 3, 1.2.3)"),
			Entry("a malformed hue", "hsl(1.2.3, 0%, 0%)"),
			Entry("a NaN channel", "rgb(NaN, 0, 0)"),
			Entry("an infinite hue", "hsl(Inf, 0%, 0%)"),
			Entry("an infinite alpha", "rgba(1, 2, 3, inf)"),
			Entry("a saturation above 100", "hsl(0, 101%, 0%)"),
			Entry("a lightness above 100", "hsl(0, 0%, 101%)"),
		)
	})

	Describe("Hex", func() {
		DescribeTable(
			"Should write lower case hex",
			func(c color.Color, expected string) { Expect(c.Hex()).To(Equal(expected)) },
			Entry(
				"6 digits when opaque",
				color.Color{R: 255, G: 128, B: 1, A: 255},
				"#ff8001",
			),
			Entry("8 digits when translucent", color.Color{R: 10, G: 11, B: 12, A: 128},
				"#0a0b0c80"),
			Entry("8 digits for the zero color", color.Color{}, "#00000000"),
		)
	})

	Describe("FromUint32", func() {
		It("Should unpack a 0xRRGGBBAA color", func() {
			Expect(color.FromUint32(0x12345678)).
				To(Equal(color.Color{R: 0x12, G: 0x34, B: 0x56, A: 0x78}))
		})
	})
})
