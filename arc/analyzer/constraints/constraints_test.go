// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package constraints_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/ir"
)

var _ = Describe("Constraint System", func() {
	var system *constraints.System

	BeforeEach(func() {
		system = constraints.New()
	})

	Describe("Basic Constraint Collection", func() {
		It("should track type variables", func() {
			tv1 := ir.NewTypeVariable("T", nil)
			tv2 := ir.NewTypeVariable("U", ir.NumericConstraint{})

			system.AddEquality(tv1, ir.F32{}, nil, "test")
			system.AddEquality(tv2, ir.I32{}, nil, "test")

			Expect(system.TypeVariables()).To(HaveLen(2))
			Expect(system.Constraints()).To(HaveLen(2))
		})

		It("should track constraints between type variables", func() {
			tv1 := ir.NewTypeVariable("T", nil)
			tv2 := ir.NewTypeVariable("U", nil)

			system.AddEquality(tv1, tv2, nil, "T = U")

			Expect(system.TypeVariables()).To(HaveLen(2))
			Expect(system.Constraints()).To(HaveLen(1))
		})

		It("should handle nested type variables in channels", func() {
			tv := ir.NewTypeVariable("T", ir.NumericConstraint{})
			chanType := ir.Chan{ValueType: tv}

			system.AddEquality(chanType, ir.Chan{ValueType: ir.F32{}}, nil, "channel types")

			Expect(system.TypeVariables()).To(HaveLen(1))
			Expect(system.TypeVariables()["T"]).NotTo(BeNil())
		})
	})

	Describe("ApplySubstitutions", func() {
		It("should apply simple substitutions", func() {
			tv := ir.NewTypeVariable("T", nil)
			system.SetSubstitution("T", ir.F32{})

			result := system.ApplySubstitutions(tv)
			Expect(result).To(Equal(ir.F32{}))
		})

		It("should apply substitutions recursively", func() {
			tv1 := ir.NewTypeVariable("T", nil)
			tv2 := ir.NewTypeVariable("U", nil)

			system.SetSubstitution("T", tv2)
			system.SetSubstitution("U", ir.I64{})

			result := system.ApplySubstitutions(tv1)
			Expect(result).To(Equal(ir.I64{}))
		})

		It("should apply substitutions in compound types", func() {
			tv := ir.NewTypeVariable("T", nil)
			chanType := ir.Chan{ValueType: tv}

			system.SetSubstitution("T", ir.F64{})

			result := system.ApplySubstitutions(chanType)
			Expect(result).To(Equal(ir.Chan{ValueType: ir.F64{}}))
		})

		It("should apply substitutions in series types", func() {
			tv := ir.NewTypeVariable("T", ir.NumericConstraint{})
			seriesType := ir.Series{ValueType: tv}

			system.SetSubstitution("T", ir.I32{})

			result := system.ApplySubstitutions(seriesType)
			Expect(result).To(Equal(ir.Series{ValueType: ir.I32{}}))
		})
	})
})
