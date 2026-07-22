// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	v0 "github.com/synnaxlabs/arc/types/types/v0"
)

var (
	dimNone      = v0.Dimensions{}
	dimLength    = v0.Dimensions{Length: 1}
	dimTime      = v0.Dimensions{Time: 1}
	dimFrequency = v0.Dimensions{Time: -1}
	dimVelocity  = v0.Dimensions{Length: 1, Time: -1}
	dimPressure  = v0.Dimensions{Mass: 1, Length: -1, Time: -2}
)

var _ = Describe("Dimensions", func() {
	Describe("Mul", func() {
		It("Should add exponents (m * m = m^2)", func() {
			result := dimLength.Mul(dimLength)
			Expect(result.Length).To(Equal(int8(2)))
		})

		It("Should produce velocity (m * s^-1)", func() {
			result := dimLength.Mul(dimFrequency)
			Expect(result).To(Equal(dimVelocity))
		})

		It("Should handle dimensionless", func() {
			result := dimNone.Mul(dimLength)
			Expect(result).To(Equal(dimLength))
		})
	})

	Describe("Div", func() {
		It("Should subtract exponents (m / s = velocity)", func() {
			result := dimLength.Div(dimTime)
			Expect(result).To(Equal(dimVelocity))
		})

		It("Should cancel dimensions (m / m = dimensionless)", func() {
			result := dimLength.Div(dimLength)
			Expect(result.IsZero()).To(BeTrue())
		})

		It("Should produce frequency (1 / s)", func() {
			result := dimNone.Div(dimTime)
			Expect(result).To(Equal(dimFrequency))
		})
	})

	Describe("Scale", func() {
		It("Should multiply exponents (m^1 scaled by 2 = m^2)", func() {
			result := dimLength.Scale(2)
			Expect(result).To(Equal(v0.Dimensions{Length: 2}))
		})

		It("Should negate exponents when scaled by -1", func() {
			result := dimVelocity.Scale(-1)
			Expect(result).To(Equal(v0.Dimensions{Length: -1, Time: 1}))
		})

		It("Should zero all exponents when scaled by 0", func() {
			Expect(dimPressure.Scale(0)).To(Equal(dimNone))
		})
	})

	Describe("Equal", func() {
		It("Should return true for identical dimensions", func() {
			Expect(dimVelocity.Equal(v0.Dimensions{Length: 1, Time: -1})).To(BeTrue())
		})

		It("Should return false for differing dimensions", func() {
			Expect(dimLength.Equal(dimTime)).To(BeFalse())
		})
	})

	Describe("IsZero", func() {
		It("Should return true for dimensionless", func() {
			Expect(dimNone.IsZero()).To(BeTrue())
		})

		It("Should return false for dimensioned", func() {
			Expect(dimLength.IsZero()).To(BeFalse())
			Expect(dimPressure.IsZero()).To(BeFalse())
		})
	})

	Describe("String", func() {
		It("Should format velocity", func() {
			s := dimVelocity.String()
			Expect(s).To(ContainSubstring("length^1"))
			Expect(s).To(ContainSubstring("time^-1"))
		})

		It("Should return dimensionless for zero", func() {
			Expect(dimNone.String()).To(Equal("dimensionless"))
		})
	})
})

var _ = Describe("Unit", func() {
	Describe("Equal", func() {
		It("Should compare equal units", func() {
			u1 := v0.Unit{Dimensions: dimLength, Scale: 1000, Name: "km"}
			u2 := v0.Unit{Dimensions: dimLength, Scale: 1000, Name: "km"}
			Expect(u1.Equal(u2)).To(BeTrue())
		})

		It("Should detect different scales", func() {
			u1 := v0.Unit{Dimensions: dimLength, Scale: 1000, Name: "km"}
			u2 := v0.Unit{Dimensions: dimLength, Scale: 1, Name: "m"}
			Expect(u1.Equal(u2)).To(BeFalse())
		})

		It("Should detect different dimensions", func() {
			u1 := v0.Unit{Dimensions: dimLength, Scale: 1, Name: "m"}
			u2 := v0.Unit{Dimensions: dimTime, Scale: 1, Name: "s"}
			Expect(u1.Equal(u2)).To(BeFalse())
		})
	})
})
