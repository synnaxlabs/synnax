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
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Analysis", func() {
	Describe("ValidateBinaryOp", func() {
		Context("Addition and Subtraction", func() {
			It("Should allow adding same dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: units.MustResolve("psi")}
				right := types.Type{Kind: types.KindF64, Unit: units.MustResolve("Pa")}
				Expect(units.ValidateBinaryOp("+", left, right)).To(Succeed())
			})

			It("Should reject adding incompatible dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: units.MustResolve("psi")}
				right := types.Type{Kind: types.KindF64, Unit: units.MustResolve("s")}
				Expect(units.ValidateBinaryOp("+", left, right)).Error().To(MatchError(units.IncompatibleDimensionsError))
			})

			It("Should allow adding dimensionless to dimensioned", func() {
				left := types.Type{Kind: types.KindF64}
				right := types.Type{Kind: types.KindF64, Unit: units.MustResolve("psi")}
				Expect(units.ValidateBinaryOp("+", left, right)).To(Succeed())
			})

			It("Should allow subtracting same dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: units.MustResolve("psi")}
				right := types.Type{Kind: types.KindF64, Unit: units.MustResolve("Pa")}
				Expect(units.ValidateBinaryOp("-", left, right)).To(Succeed())
			})
		})

		Context("Multiplication and Division", func() {
			It("Should allow multiplying any dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: units.MustResolve("m")}
				right := types.Type{Kind: types.KindF64, Unit: units.MustResolve("m")}
				Expect(units.ValidateBinaryOp("*", left, right)).To(Succeed())
			})

			It("Should allow dividing any dimensions", func() {
				left := types.Type{Kind: types.KindF64, Unit: units.MustResolve("m")}
				right := types.Type{Kind: types.KindF64, Unit: units.MustResolve("s")}
				Expect(units.ValidateBinaryOp("/", left, right)).To(Succeed())
			})

			It("Should allow multiplying dimensionless values", func() {
				left := types.Type{Kind: types.KindF64}
				right := types.Type{Kind: types.KindF64}
				Expect(units.ValidateBinaryOp("*", left, right)).To(Succeed())
			})

			It("Should allow multiplying unit by dimensionless", func() {
				left := types.Type{Kind: types.KindF64, Unit: units.MustResolve("m")}
				right := types.Type{Kind: types.KindF64}
				Expect(units.ValidateBinaryOp("*", left, right)).To(Succeed())
			})
		})
	})

	Describe("ValidatePowerOp", func() {
		It("Should reject dimensioned exponent", func() {
			base := types.Type{Kind: types.KindF64, Unit: units.MustResolve("m")}
			exp := types.Type{Kind: types.KindF64, Unit: units.MustResolve("s")}
			err := units.ValidatePowerOp(base, exp, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("dimensionless"))
		})

		It("Should reject non-literal exponent with dimensioned base", func() {
			base := types.Type{Kind: types.KindF64, Unit: units.MustResolve("m")}
			exp := types.Type{Kind: types.KindF64}
			err := units.ValidatePowerOp(base, exp, false)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("literal integer exponent"))
		})

		It("Should allow literal exponent with dimensioned base", func() {
			base := types.Type{Kind: types.KindF64, Unit: units.MustResolve("m")}
			exp := types.Type{Kind: types.KindF64}
			Expect(units.ValidatePowerOp(base, exp, true)).To(Succeed())
		})

		It("Should allow dimensionless base and exponent", func() {
			base := types.Type{Kind: types.KindF64}
			exp := types.Type{Kind: types.KindF64}
			Expect(units.ValidatePowerOp(base, exp, false)).To(Succeed())
		})
	})

	Describe("ValidateBinaryOp - Comparisons", func() {
		It("Should allow comparing same dimensions", func() {
			left := types.Type{Kind: types.KindF64, Unit: units.MustResolve("psi")}
			right := types.Type{Kind: types.KindF64, Unit: units.MustResolve("Pa")}
			Expect(units.ValidateBinaryOp(">", left, right)).To(Succeed())
		})

		It("Should reject comparing incompatible dimensions", func() {
			left := types.Type{Kind: types.KindF64, Unit: units.MustResolve("psi")}
			right := types.Type{Kind: types.KindF64, Unit: units.MustResolve("s")}
			Expect(units.ValidateBinaryOp("<", left, right)).To(MatchError(units.IncompatibleDimensionsError))
		})

		It("Should allow comparing dimensionless values", func() {
			left := types.Type{Kind: types.KindF64}
			right := types.Type{Kind: types.KindF64}
			Expect(units.ValidateBinaryOp("==", left, right)).To(Succeed())
		})

		It("Should allow comparing unit to dimensionless", func() {
			left := types.Type{Kind: types.KindF64, Unit: units.MustResolve("psi")}
			right := types.Type{Kind: types.KindF64}
			Expect(units.ValidateBinaryOp(">=", left, right)).To(Succeed())
		})
	})

	Describe("ValidateBinaryOp - Modulo", func() {
		It("Should allow modulo with same dimensions", func() {
			left := types.Type{Kind: types.KindF64, Unit: units.MustResolve("m")}
			right := types.Type{Kind: types.KindF64, Unit: units.MustResolve("m")}
			Expect(units.ValidateBinaryOp("%", left, right)).To(Succeed())
		})

		It("Should reject modulo with incompatible dimensions", func() {
			left := types.Type{Kind: types.KindF64, Unit: units.MustResolve("m")}
			right := types.Type{Kind: types.KindF64, Unit: units.MustResolve("s")}
			Expect(units.ValidateBinaryOp("%", left, right)).To(MatchError(units.IncompatibleDimensionsError))
		})
	})

	Describe("ScaleFactor", func() {
		It("Should calculate km to m conversion", func() {
			km := MustBeOk(units.Resolve("km"))
			m := MustBeOk(units.Resolve("meter"))
			factor := MustSucceed(units.ScaleFactor(km, m))
			Expect(factor).To(Equal(1000.0)) // 1 km = 1000 m
		})

		It("Should calculate ms to s conversion", func() {
			ms := MustBeOk(units.Resolve("ms"))
			s := MustBeOk(units.Resolve("s"))
			factor := MustSucceed(units.ScaleFactor(ms, s))
			Expect(factor).To(Equal(1e-3)) // 1 ms = 0.001 s
		})

		It("Should calculate psi to Pa conversion", func() {
			psi := MustBeOk(units.Resolve("psi"))
			pa := MustBeOk(units.Resolve("Pa"))
			factor := MustSucceed(units.ScaleFactor(psi, pa))
			Expect(factor).To(BeNumerically("~", 6894.76, 0.01))
		})

		It("Should return 1.0 for same unit", func() {
			psi := MustBeOk(units.Resolve("psi"))
			factor := MustSucceed(units.ScaleFactor(psi, psi))
			Expect(factor).To(Equal(1.0))
		})

		It("Should return 1.0 for both nil", func() {
			factor := MustSucceed(units.ScaleFactor(nil, nil))
			Expect(factor).To(Equal(1.0))
		})

		It("Should error for incompatible dimensions", func() {
			psi := MustBeOk(units.Resolve("psi"))
			s := MustBeOk(units.Resolve("s"))
			Expect(units.ScaleFactor(psi, s)).Error().To(MatchError(units.IncompatibleDimensionsError))
		})

		It("Should error for nil to dimensioned", func() {
			psi := MustBeOk(units.Resolve("psi"))
			Expect(units.ScaleFactor(nil, psi)).Error().To(MatchError(ContainSubstring("cannot convert between dimensioned and dimensionless values")))
		})
	})
})
