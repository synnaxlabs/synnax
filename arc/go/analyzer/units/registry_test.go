// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package units_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/units"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("Registry", func() {
	Describe("Resolve", func() {
		type validEntry struct {
			name       string
			scale      float64
			approx     bool
			dimensions types.Dimensions
		}

		DescribeTable("Valid units",
			func(unitName string, expected validEntry) {
				u, ok := units.Resolve(unitName)
				Expect(ok).To(BeTrue())
				Expect(u.Name).To(Equal(expected.name))
				if expected.approx {
					Expect(u.Scale).To(BeNumerically("~", expected.scale, 0.01))
				} else {
					Expect(u.Scale).To(Equal(expected.scale))
				}
				Expect(u.Dimensions).To(Equal(expected.dimensions))
			},
			// Time units
			Entry("nanoseconds", "ns", validEntry{"ns", 1.0, false, types.DimTime}),
			Entry("milliseconds", "ms", validEntry{"ms", 1e6, false, types.DimTime}),
			Entry("seconds", "s", validEntry{"s", 1e9, false, types.DimTime}),
			Entry("minutes", "min", validEntry{"min", 60e9, false, types.DimTime}),
			Entry("hours", "h", validEntry{"h", 3600e9, false, types.DimTime}),
			// Pressure units
			Entry("psi", "psi", validEntry{"psi", 6894.76, true, types.DimPressure}),
			Entry("Pa", "Pa", validEntry{"Pa", 1.0, false, types.DimPressure}),
			Entry("bar", "bar", validEntry{"bar", 1e5, false, types.DimPressure}),
			// Frequency units
			Entry("Hz (uppercase)", "Hz", validEntry{"Hz", 1.0, false, types.DimFrequency}),
			Entry("hz (lowercase)", "hz", validEntry{"hz", 1.0, false, types.DimFrequency}),
			Entry("kHz", "kHz", validEntry{"kHz", 1e3, false, types.DimFrequency}),
			Entry("MHz", "MHz", validEntry{"MHz", 1e6, false, types.DimFrequency}),
			// Length units
			Entry("m (meters)", "m", validEntry{"m", 1.0, false, types.DimLength}),
			Entry("km", "km", validEntry{"km", 1e3, false, types.DimLength}),
			Entry("ft", "ft", validEntry{"ft", 0.3048, true, types.DimLength}),
			// Voltage units
			Entry("V", "V", validEntry{"V", 1.0, false, types.DimVoltage}),
			Entry("mV", "mV", validEntry{"mV", 1e-3, false, types.DimVoltage}),
		)

		DescribeTable("Invalid units",
			func(unitName string) {
				_, ok := units.Resolve(unitName)
				Expect(ok).To(BeFalse())
			},
			Entry("unknown unit", "foobar"),
			Entry("empty string", ""),
		)
	})

	Specify("Mutating a Unit Entry Should not Modify Value in Registry", func() {
		u := units.MustResolve("m")
		u.Scale = 12
		u2 := units.MustResolve("m")
		Expect(u2.Scale).To(Equal(1.0))
	})

	Describe("IsValid", func() {
		DescribeTable("Valid units",
			func(unitName string) {
				Expect(units.IsValid(unitName)).To(BeTrue())
			},
			Entry("psi", "psi"),
			Entry("ms", "ms"),
			Entry("Hz", "Hz"),
		)

		DescribeTable("Invalid units",
			func(unitName string) {
				Expect(units.IsValid(unitName)).To(BeFalse())
			},
			Entry("invalid", "invalid"),
			Entry("empty", ""),
		)
	})
})
