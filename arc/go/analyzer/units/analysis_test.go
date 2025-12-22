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

var _ = Describe("Analysis", func() {
	var (
		psiUnit    *types.Unit
		paUnit     *types.Unit
		secondUnit *types.Unit
		meterUnit  *types.Unit
	)

	BeforeEach(func() {
		psi, _ := units.Lookup("psi")
		psiUnit = &psi
		pa, _ := units.Lookup("Pa")
		paUnit = &pa
		sec, _ := units.Lookup("s")
		secondUnit = &sec
		m, _ := units.Lookup("meter")
		meterUnit = &m
	})

	Describe("CheckBinaryOp", func() {
		Context("Addition and Subtraction", func() {
			It("Should allow adding same dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: psiUnit}
				right := types.Type{Kind: types.KindF64, Unit: paUnit}

				result, err := units.CheckBinaryOp("+", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Kind).To(Equal(types.KindF64))
				Expect(result.Unit).ToNot(BeNil())
				Expect(result.Unit.Dimensions).To(Equal(types.DimPressure))
			})

			It("Should reject adding incompatible dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: psiUnit}
				right := types.Type{Kind: types.KindF64, Unit: secondUnit}

				_, err := units.CheckBinaryOp("+", left, right, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("incompatible dimensions"))
			})

			It("Should allow adding dimensionless to dimensioned", func() {
				left := types.Type{Kind: types.KindF64} // dimensionless
				right := types.Type{Kind: types.KindF64, Unit: psiUnit}

				result, err := units.CheckBinaryOp("+", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Unit).To(Equal(psiUnit))
			})

			It("Should allow subtracting same dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: psiUnit}
				right := types.Type{Kind: types.KindF64, Unit: paUnit}

				result, err := units.CheckBinaryOp("-", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Unit.Dimensions).To(Equal(types.DimPressure))
			})
		})

		Context("Multiplication", func() {
			It("Should multiply dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: meterUnit}
				right := types.Type{Kind: types.KindF64, Unit: meterUnit}

				result, err := units.CheckBinaryOp("*", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				// m * m = m^2
				Expect(result.Unit.Dimensions.Length).To(Equal(int8(2)))
			})

			It("Should handle velocity calculation (length / time)", func() {
				left := types.Type{Kind: types.KindF64, Unit: meterUnit}
				right := types.Type{Kind: types.KindF64, Unit: secondUnit}

				result, err := units.CheckBinaryOp("/", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				// m / s = velocity
				Expect(result.Unit.Dimensions).To(Equal(types.DimVelocity))
			})

			It("Should return dimensionless when dimensions cancel", func() {
				left := types.Type{Kind: types.KindF64, Unit: meterUnit}
				right := types.Type{Kind: types.KindF64, Unit: meterUnit}

				result, err := units.CheckBinaryOp("/", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				// m / m = dimensionless
				Expect(result.Unit).To(BeNil())
			})

			It("Should allow multiplying dimensionless values", func() {
				left := types.Type{Kind: types.KindF64}
				right := types.Type{Kind: types.KindF64}

				result, err := units.CheckBinaryOp("*", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Unit).To(BeNil())
			})

			It("Should allow multiplying unit by dimensionless", func() {
				left := types.Type{Kind: types.KindF64, Unit: meterUnit}
				right := types.Type{Kind: types.KindF64} // dimensionless

				result, err := units.CheckBinaryOp("*", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Unit.Dimensions).To(Equal(types.DimLength))
			})
		})

		Context("Division", func() {
			It("Should divide dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: meterUnit}
				right := types.Type{Kind: types.KindF64, Unit: secondUnit}

				result, err := units.CheckBinaryOp("/", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				// m / s = m*s^-1
				Expect(result.Unit.Dimensions.Length).To(Equal(int8(1)))
				Expect(result.Unit.Dimensions.Time).To(Equal(int8(-1)))
			})

			It("Should handle frequency (1/time)", func() {
				left := types.Type{Kind: types.KindF64} // dimensionless (1)
				right := types.Type{Kind: types.KindF64, Unit: secondUnit}

				result, err := units.CheckBinaryOp("/", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				// 1 / s = Hz
				Expect(result.Unit.Dimensions).To(Equal(types.DimFrequency))
			})
		})

		Context("Power", func() {
			It("Should reject dimensioned exponent", func() {
				left := types.Type{Kind: types.KindF64, Unit: meterUnit}
				right := types.Type{Kind: types.KindF64, Unit: secondUnit}

				_, err := units.CheckBinaryOp("^", left, right, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("dimensionless"))
			})

			It("Should reject non-literal exponent with dimensioned base", func() {
				left := types.Type{Kind: types.KindF64, Unit: meterUnit}
				right := types.Type{Kind: types.KindF64} // dimensionless but not literal

				_, err := units.CheckBinaryOp("^", left, right, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("literal integer exponent"))
			})

			It("Should scale dimensions with literal exponent", func() {
				left := types.Type{Kind: types.KindF64, Unit: meterUnit}
				right := types.Type{Kind: types.KindF64}
				exp := 2

				result, err := units.CheckBinaryOp("^", left, right, &exp)
				Expect(err).ToNot(HaveOccurred())
				// m^2 = length^2
				Expect(result.Unit.Dimensions.Length).To(Equal(int8(2)))
			})

			It("Should handle negative exponents", func() {
				left := types.Type{Kind: types.KindF64, Unit: secondUnit}
				right := types.Type{Kind: types.KindF64}
				exp := -2

				result, err := units.CheckBinaryOp("^", left, right, &exp)
				Expect(err).ToNot(HaveOccurred())
				// s^-2 = time^-2
				Expect(result.Unit.Dimensions.Time).To(Equal(int8(-2)))
			})

			It("Should return dimensionless for exponent 0", func() {
				left := types.Type{Kind: types.KindF64, Unit: meterUnit}
				right := types.Type{Kind: types.KindF64}
				exp := 0

				result, err := units.CheckBinaryOp("^", left, right, &exp)
				Expect(err).ToNot(HaveOccurred())
				// m^0 = dimensionless
				Expect(result.Unit).To(BeNil())
			})

			It("Should allow dimensionless base and exponent", func() {
				left := types.Type{Kind: types.KindF64}
				right := types.Type{Kind: types.KindF64}

				result, err := units.CheckBinaryOp("^", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Unit).To(BeNil())
			})
		})

		Context("Comparisons", func() {
			It("Should allow comparing same dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: psiUnit}
				right := types.Type{Kind: types.KindF64, Unit: paUnit}

				result, err := units.CheckBinaryOp(">", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Kind).To(Equal(types.KindU8)) // boolean
				Expect(result.Unit).To(BeNil())
			})

			It("Should reject comparing incompatible dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: psiUnit}
				right := types.Type{Kind: types.KindF64, Unit: secondUnit}

				_, err := units.CheckBinaryOp("<", left, right, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("incompatible dimensions"))
			})

			It("Should allow comparing dimensionless values", func() {
				left := types.Type{Kind: types.KindF64}
				right := types.Type{Kind: types.KindF64}

				result, err := units.CheckBinaryOp("==", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Kind).To(Equal(types.KindU8))
			})

			It("Should allow comparing unit to dimensionless", func() {
				left := types.Type{Kind: types.KindF64, Unit: psiUnit}
				right := types.Type{Kind: types.KindF64}

				result, err := units.CheckBinaryOp(">=", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				Expect(result.Kind).To(Equal(types.KindU8))
			})
		})

		Context("Modulo", func() {
			It("Should follow additive rules - keep dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: meterUnit}
				right := types.Type{Kind: types.KindF64, Unit: meterUnit}

				result, err := units.CheckBinaryOp("%", left, right, nil)
				Expect(err).ToNot(HaveOccurred())
				// m % m = m (remainder of length is still length)
				Expect(result.Unit).ToNot(BeNil())
				Expect(result.Unit.Dimensions).To(Equal(types.DimLength))
			})

			It("Should reject incompatible dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: meterUnit}
				right := types.Type{Kind: types.KindF64, Unit: secondUnit}

				_, err := units.CheckBinaryOp("%", left, right, nil)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("incompatible dimensions"))
			})
		})
	})

	Describe("PromoteNumeric", func() {
		It("Should promote to wider type", func() {
			result := units.PromoteNumeric(
				types.Type{Kind: types.KindF32},
				types.Type{Kind: types.KindF64},
			)
			Expect(result.Kind).To(Equal(types.KindF64))
		})

		It("Should promote i32 and f32 to f32", func() {
			result := units.PromoteNumeric(
				types.Type{Kind: types.KindI32},
				types.Type{Kind: types.KindF32},
			)
			Expect(result.Kind).To(Equal(types.KindF32))
		})

		It("Should keep same type", func() {
			result := units.PromoteNumeric(
				types.Type{Kind: types.KindI64},
				types.Type{Kind: types.KindI64},
			)
			Expect(result.Kind).To(Equal(types.KindI64))
		})
	})

	Describe("SameDimensions", func() {
		It("Should return true for same dimensions", func() {
			Expect(units.SameDimensions(psiUnit, paUnit)).To(BeTrue())
		})

		It("Should return false for different dimensions", func() {
			Expect(units.SameDimensions(psiUnit, secondUnit)).To(BeFalse())
		})

		It("Should return true for both nil", func() {
			Expect(units.SameDimensions(nil, nil)).To(BeTrue())
		})

		It("Should return false for one nil", func() {
			Expect(units.SameDimensions(psiUnit, nil)).To(BeFalse())
			Expect(units.SameDimensions(nil, psiUnit)).To(BeFalse())
		})
	})

	Describe("ScaleConversionFactor", func() {
		It("Should calculate km to m conversion", func() {
			km, _ := units.Lookup("km")
			m, _ := units.Lookup("meter")
			factor, err := units.ScaleConversionFactor(&km, &m)
			Expect(err).ToNot(HaveOccurred())
			Expect(factor).To(Equal(1000.0)) // 1 km = 1000 m
		})

		It("Should calculate ms to s conversion", func() {
			ms, _ := units.Lookup("ms")
			s, _ := units.Lookup("s")
			factor, err := units.ScaleConversionFactor(&ms, &s)
			Expect(err).ToNot(HaveOccurred())
			Expect(factor).To(Equal(1e-3)) // 1 ms = 0.001 s
		})

		It("Should calculate psi to Pa conversion", func() {
			psi, _ := units.Lookup("psi")
			pa, _ := units.Lookup("Pa")
			factor, err := units.ScaleConversionFactor(&psi, &pa)
			Expect(err).ToNot(HaveOccurred())
			Expect(factor).To(BeNumerically("~", 6894.76, 0.01))
		})

		It("Should return 1.0 for same unit", func() {
			psi, _ := units.Lookup("psi")
			factor, err := units.ScaleConversionFactor(&psi, &psi)
			Expect(err).ToNot(HaveOccurred())
			Expect(factor).To(Equal(1.0))
		})

		It("Should return 1.0 for both nil", func() {
			factor, err := units.ScaleConversionFactor(nil, nil)
			Expect(err).ToNot(HaveOccurred())
			Expect(factor).To(Equal(1.0))
		})

		It("Should error for incompatible dimensions", func() {
			psi, _ := units.Lookup("psi")
			s, _ := units.Lookup("s")
			_, err := units.ScaleConversionFactor(&psi, &s)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("incompatible dimensions"))
		})

		It("Should error for nil to dimensioned", func() {
			psi, _ := units.Lookup("psi")
			_, err := units.ScaleConversionFactor(nil, &psi)
			Expect(err).To(HaveOccurred())
		})
	})
})
