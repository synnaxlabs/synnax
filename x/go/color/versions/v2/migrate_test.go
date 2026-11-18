// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v2_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v1 "github.com/synnaxlabs/x/color/versions/v1"
	v2 "github.com/synnaxlabs/x/color/versions/v2"
)

var _ = Describe("Migrate", func() {
	DescribeTable("MigrateColor",
		func(ctx SpecContext, a float64, expected uint8) {
			c := v1.Color{R: 1, G: 2, B: 3, A: a}
			Expect(v2.MigrateColor(ctx, c)).
				To(Equal(v2.Color{R: 1, G: 2, B: 3, A: expected}))
		},
		Entry("zero alpha", 0.0, uint8(0)),
		Entry("a faint alpha stays visible", 0.001, uint8(1)),
		Entry("a small alpha", 0.003, uint8(1)),
		Entry("a half alpha", 0.5, uint8(128)),
		Entry("an opaque alpha", 1.0, uint8(255)),
		Entry("an alpha above 1 clamps", 1.5, uint8(255)),
		Entry("a negative alpha", -0.5, uint8(0)),
	)

	It("Should keep the zero color as the zero color", func(ctx SpecContext) {
		Expect(v2.MigrateColor(ctx, v1.Color{})).To(Equal(v2.Color{}))
	})

	It("Should migrate the color of a stop", func(ctx SpecContext) {
		old := v1.Stop{Key: "s", Color: v1.Color{R: 4, A: 0.5}, Position: 0.25}
		Expect(v2.MigrateStop(ctx, old)).To(Equal(v2.Stop{
			Key: "s", Color: v2.Color{R: 4, A: 128}, Position: 0.25,
		}))
	})

	It("Should migrate the band colors of a scale", func(ctx SpecContext) {
		old := v1.Scale{
			Bands: []v1.Band{
				{Key: "b", Threshold: 2, Color: v1.Color{G: 4, A: 1}, Flashing: true},
			},
			Smooth: true,
		}
		Expect(v2.MigrateScale(ctx, old)).To(Equal(v2.Scale{
			Bands: []v2.Band{
				{Key: "b", Threshold: 2, Color: v2.Color{G: 4, A: 255}, Flashing: true},
			},
			Smooth: true,
		}))
	})

	It("Should migrate the color of a band", func(ctx SpecContext) {
		old := v1.Band{Key: "b", Color: v1.Color{B: 4, A: 0.25}}
		Expect(v2.MigrateBand(ctx, old)).
			To(Equal(v2.Band{Key: "b", Color: v2.Color{B: 4, A: 64}}))
	})
})
