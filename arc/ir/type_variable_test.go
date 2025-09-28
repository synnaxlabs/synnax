// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package ir_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/ir"
)

var _ = Describe("Type Variables", func() {
	Describe("TypeVariable", func() {
		It("should create unconstrained type variables", func() {
			tv := ir.NewTypeVariable("T", nil)
			Expect(tv.Name).To(Equal("T"))
			Expect(tv.Constraint).To(BeNil())
			Expect(tv.String()).To(Equal("T"))
		})

		It("should create constrained type variables", func() {
			tv := ir.NewTypeVariable("T", ir.NumericConstraint{})
			Expect(tv.Name).To(Equal("T"))
			Expect(tv.Constraint).To(Equal(ir.NumericConstraint{}))
			Expect(tv.String()).To(Equal("T:numeric"))
		})

		It("should identify type variables", func() {
			tv := ir.NewTypeVariable("T", nil)
			Expect(ir.IsTypeVariable(tv)).To(BeTrue())
			Expect(ir.IsTypeVariable(ir.F32{})).To(BeFalse())
		})
	})

	Describe("NumericConstraint", func() {
		It("should have a string representation", func() {
			constraint := ir.NumericConstraint{}
			Expect(constraint.String()).To(Equal("numeric"))
		})
	})

	Describe("IsNumeric with TypeVariables", func() {
		It("should recognize numeric-constrained type variables as numeric", func() {
			tv := ir.NewTypeVariable("T", ir.NumericConstraint{})
			Expect(ir.IsNumeric(tv)).To(BeTrue())
		})

		It("should not recognize unconstrained type variables as numeric", func() {
			tv := ir.NewTypeVariable("T", nil)
			Expect(ir.IsNumeric(tv)).To(BeFalse())
		})

		It("should recognize type variables constrained to numeric types", func() {
			tv := ir.NewTypeVariable("T", ir.F32{})
			Expect(ir.IsNumeric(tv)).To(BeTrue())
		})
	})
})
