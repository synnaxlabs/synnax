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
	Describe("Lookup", func() {
		Context("Time units", func() {
			It("Should find nanoseconds", func() {
				u, ok := units.Lookup("ns")
				Expect(ok).To(BeTrue())
				Expect(u.Name).To(Equal("ns"))
				Expect(u.Scale).To(Equal(1.0)) // ns is the base unit for time
				Expect(u.Dimensions).To(Equal(types.DimTime))
			})

			It("Should find milliseconds", func() {
				u, ok := units.Lookup("ms")
				Expect(ok).To(BeTrue())
				Expect(u.Name).To(Equal("ms"))
				Expect(u.Scale).To(Equal(1e6)) // 1ms = 1 million ns
				Expect(u.Dimensions).To(Equal(types.DimTime))
			})

			It("Should find seconds", func() {
				u, ok := units.Lookup("s")
				Expect(ok).To(BeTrue())
				Expect(u.Name).To(Equal("s"))
				Expect(u.Scale).To(Equal(1e9)) // 1s = 1 billion ns
				Expect(u.Dimensions).To(Equal(types.DimTime))
			})

			It("Should find hours", func() {
				u, ok := units.Lookup("h")
				Expect(ok).To(BeTrue())
				Expect(u.Name).To(Equal("h"))
				Expect(u.Scale).To(Equal(3600e9)) // 1h = 3600 billion ns
				Expect(u.Dimensions).To(Equal(types.DimTime))
			})
		})

		Context("Pressure units", func() {
			It("Should find psi", func() {
				u, ok := units.Lookup("psi")
				Expect(ok).To(BeTrue())
				Expect(u.Name).To(Equal("psi"))
				Expect(u.Scale).To(BeNumerically("~", 6894.76, 0.01))
				Expect(u.Dimensions).To(Equal(types.DimPressure))
			})

			It("Should find Pa", func() {
				u, ok := units.Lookup("Pa")
				Expect(ok).To(BeTrue())
				Expect(u.Name).To(Equal("Pa"))
				Expect(u.Scale).To(Equal(1.0))
				Expect(u.Dimensions).To(Equal(types.DimPressure))
			})

			It("Should find bar", func() {
				u, ok := units.Lookup("bar")
				Expect(ok).To(BeTrue())
				Expect(u.Name).To(Equal("bar"))
				Expect(u.Scale).To(Equal(1e5))
				Expect(u.Dimensions).To(Equal(types.DimPressure))
			})
		})

		Context("Frequency units", func() {
			It("Should find Hz (uppercase)", func() {
				u, ok := units.Lookup("Hz")
				Expect(ok).To(BeTrue())
				Expect(u.Dimensions).To(Equal(types.DimFrequency))
			})

			It("Should find hz (lowercase)", func() {
				u, ok := units.Lookup("hz")
				Expect(ok).To(BeTrue())
				Expect(u.Dimensions).To(Equal(types.DimFrequency))
			})

			It("Should find kHz", func() {
				u, ok := units.Lookup("kHz")
				Expect(ok).To(BeTrue())
				Expect(u.Scale).To(Equal(1e3))
			})

			It("Should find MHz", func() {
				u, ok := units.Lookup("MHz")
				Expect(ok).To(BeTrue())
				Expect(u.Scale).To(Equal(1e6))
			})
		})

		Context("Length units", func() {
			It("Should find km", func() {
				u, ok := units.Lookup("km")
				Expect(ok).To(BeTrue())
				Expect(u.Scale).To(Equal(1e3))
				Expect(u.Dimensions).To(Equal(types.DimLength))
			})

			It("Should find ft", func() {
				u, ok := units.Lookup("ft")
				Expect(ok).To(BeTrue())
				Expect(u.Scale).To(BeNumerically("~", 0.3048, 0.0001))
				Expect(u.Dimensions).To(Equal(types.DimLength))
			})
		})

		Context("Voltage units", func() {
			It("Should find V", func() {
				u, ok := units.Lookup("V")
				Expect(ok).To(BeTrue())
				Expect(u.Scale).To(Equal(1.0))
				Expect(u.Dimensions).To(Equal(types.DimVoltage))
			})

			It("Should find mV", func() {
				u, ok := units.Lookup("mV")
				Expect(ok).To(BeTrue())
				Expect(u.Scale).To(Equal(1e-3))
			})
		})

		Context("Invalid units", func() {
			It("Should not find unknown unit", func() {
				_, ok := units.Lookup("foobar")
				Expect(ok).To(BeFalse())
			})

			It("Should not find empty string", func() {
				_, ok := units.Lookup("")
				Expect(ok).To(BeFalse())
			})
		})
	})

	Describe("IsValidUnit", func() {
		It("Should return true for valid units", func() {
			Expect(units.IsValidUnit("psi")).To(BeTrue())
			Expect(units.IsValidUnit("ms")).To(BeTrue())
			Expect(units.IsValidUnit("Hz")).To(BeTrue())
		})

		It("Should return false for invalid units", func() {
			Expect(units.IsValidUnit("invalid")).To(BeFalse())
			Expect(units.IsValidUnit("")).To(BeFalse())
		})
	})
})
