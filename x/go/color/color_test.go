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
	Describe("MustFromHex", func() {
		It("Should parse a valid hex string", func() {
			c := color.MustFromHex("#0000ff")
			Expect(c).To(Equal(color.Color{R: 0, G: 0, B: 255, A: 1}))
		})
		It("Should panic on an invalid hex string", func() {
			Expect(func() { color.MustFromHex("invalid") }).To(Panic())
		})
	})

	Describe("FromCSS", func() {
		DescribeTable("Should parse valid color strings",
			func(input string, expected color.Color) {
				Expect(color.FromCSS(input)).To(Equal(expected))
			},
			Entry("hex", "#ff0000", color.Color{R: 255, G: 0, B: 0, A: 1}),
			Entry("rgb", "rgb(59,196,84)", color.Color{R: 59, G: 196, B: 84, A: 1}),
			Entry("rgb with spaces", "rgb(255, 0, 0)", color.Color{R: 255, G: 0, B: 0, A: 1}),
			Entry("rgba", "rgba(59,196,84,0.5)", color.Color{R: 59, G: 196, B: 84, A: 0.5}),
			Entry("rgba full alpha", "rgba(0,0,0,1)", color.Color{R: 0, G: 0, B: 0, A: 1}),
			Entry("surrounding whitespace", "  rgb(1,2,3)  ", color.Color{R: 1, G: 2, B: 3, A: 1}),
		)

		DescribeTable("Should reject invalid color strings",
			func(input, msg string) {
				Expect(color.FromCSS(input)).Error().To(SatisfyAll(
					MatchError(validate.ErrValidation),
					MatchError(ContainSubstring(msg)),
				))
			},
			Entry("rgb with a 4th alpha channel", "rgb(1,2,3,0.5)", "rgb() takes 3 channels"),
			Entry("rgba missing alpha", "rgba(1,2,3)", "rgba() requires"),
			Entry("alpha above 1", "rgba(1,2,3,1.5)", "alpha must be 0-1"),
			Entry("red channel above 255", "rgb(300,0,0)", "channels must be 0-255"),
			Entry("green channel above 255", "rgb(0,300,0)", "channels must be 0-255"),
			Entry("blue channel above 255", "rgb(0,0,300)", "channels must be 0-255"),
			Entry("malformed alpha", "rgba(1,2,3,1.2.3)", "invalid rgb alpha"),
			Entry("hex without a hash", "00ff00", "must be a hex value"),
			Entry("named color", "red", "must be a hex value"),
			Entry("too few channels", "rgb(1,2)", "must be a hex value"),
			Entry("empty string", "", "must be a hex value"),
			Entry("invalid hex", "#xyz", "invalid hex color"),
		)
	})
})
