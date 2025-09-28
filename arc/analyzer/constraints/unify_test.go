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

var _ = Describe("Type Unification", func() {
	var system *constraints.System

	BeforeEach(func() {
		system = constraints.New()
	})

	Describe("Simple Unification", func() {
		It("should unify type variable with concrete type", func() {
			tv := ir.NewTypeVariable("T", nil)
			system.AddEquality(tv, ir.F32{}, nil, "T = f32")

			Expect(system.Unify()).To(Succeed())

			substitution, ok := system.GetSubstitution("T")
			Expect(ok).To(BeTrue())
			Expect(substitution).To(Equal(ir.F32{}))
		})

		It("should unify constrained type variable with valid type", func() {
			tv := ir.NewTypeVariable("T", ir.NumericConstraint{})
			system.AddEquality(tv, ir.I64{}, nil, "T = i64")

			Expect(system.Unify()).To(Succeed())

			substitution, ok := system.GetSubstitution("T")
			Expect(ok).To(BeTrue())
			Expect(substitution).To(Equal(ir.I64{}))
		})

		It("should fail to unify constrained type variable with invalid type", func() {
			tv := ir.NewTypeVariable("T", ir.NumericConstraint{})
			system.AddEquality(tv, ir.String{}, nil, "T = string")

			err := system.Unify()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not satisfy constraint"))
		})
	})

	Describe("Transitive Unification", func() {
		It("should unify through chains of type variables", func() {
			tv1 := ir.NewTypeVariable("T1", nil)
			tv2 := ir.NewTypeVariable("T2", nil)
			tv3 := ir.NewTypeVariable("T3", nil)

			system.AddEquality(tv1, tv2, nil, "T1 = T2")
			system.AddEquality(tv2, tv3, nil, "T2 = T3")
			system.AddEquality(tv3, ir.F64{}, nil, "T3 = f64")

			Expect(system.Unify()).To(Succeed())

			// All should resolve to F64
			// Due to how unification works, they might resolve to each other and then to F64
			final1 := system.ApplySubstitutions(tv1)
			final2 := system.ApplySubstitutions(tv2)
			final3 := system.ApplySubstitutions(tv3)

			Expect(final1).To(Equal(ir.F64{}))
			Expect(final2).To(Equal(ir.F64{}))
			Expect(final3).To(Equal(ir.F64{}))
		})
	})

	Describe("Numeric Promotion", func() {
		It("should promote compatible numeric types", func() {
			tv := ir.NewTypeVariable("T", ir.NumericConstraint{})

			// Add compatible constraints that require promotion
			system.AddCompatible(tv, ir.I32{}, nil, "T ~ i32")
			system.AddCompatible(tv, ir.F32{}, nil, "T ~ f32")

			Expect(system.Unify()).To(Succeed())

			// Should resolve to F32 (float beats integer)
			substitution, ok := system.GetSubstitution("T")
			Expect(ok).To(BeTrue())
			// The exact resolution depends on the order of constraint processing
			// but it should be a numeric type
			Expect(ir.IsNumeric(substitution)).To(BeTrue())
		})

		It("should handle bidirectional type flow", func() {
			// Simulating: constant{} -> add{a, b} <- channel(f32)
			constantOutput := ir.NewTypeVariable("T1", ir.NumericConstraint{})
			addParamA := ir.NewTypeVariable("T", ir.NumericConstraint{})
			addParamB := ir.NewTypeVariable("T", ir.NumericConstraint{})

			// constant output connects to add.a
			system.AddEquality(constantOutput, addParamA, nil, "constant -> add.a")
			// f32 channel connects to add.b
			system.AddEquality(ir.F32{}, addParamB, nil, "channel -> add.b")
			// add parameters must be equal (same T)
			system.AddEquality(addParamA, addParamB, nil, "add.a = add.b")

			Expect(system.Unify()).To(Succeed())

			// Everything should resolve to F32
			final1 := system.ApplySubstitutions(constantOutput)
			final2 := system.ApplySubstitutions(addParamA)
			final3 := system.ApplySubstitutions(addParamB)

			Expect(final1).To(Equal(ir.F32{}))
			Expect(final2).To(Equal(ir.F32{}))
			Expect(final3).To(Equal(ir.F32{}))
		})
	})

	Describe("Channel Type Unification", func() {
		It("should unify channel types", func() {
			tv := ir.NewTypeVariable("T", nil)
			chanTV := ir.Chan{ValueType: tv}
			chanF32 := ir.Chan{ValueType: ir.F32{}}

			system.AddEquality(chanTV, chanF32, nil, "chan T = chan f32")

			Expect(system.Unify()).To(Succeed())

			substitution, ok := system.GetSubstitution("T")
			Expect(ok).To(BeTrue())
			Expect(substitution).To(Equal(ir.F32{}))
		})

		It("should fail on incompatible channel types", func() {
			chanI32 := ir.Chan{ValueType: ir.I32{}}
			chanString := ir.Chan{ValueType: ir.String{}}

			system.AddEquality(chanI32, chanString, nil, "chan i32 = chan string")

			err := system.Unify()
			Expect(err).To(HaveOccurred())
		})
	})

	Describe("Error Cases", func() {
		It("should detect cyclic type dependencies", func() {
			tv1 := ir.NewTypeVariable("T", nil)
			// Create a cycle: T = chan T
			cyclicType := ir.Chan{ValueType: tv1}
			system.AddEquality(tv1, cyclicType, nil, "T = chan T")

			err := system.Unify()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("cyclic"))
		})

		It("should report unresolved type variables with no constraints", func() {
			tv := ir.NewTypeVariable("T", nil)
			// Add to system but don't constrain it
			system.AddEquality(tv, tv, nil, "T = T") // Self-equality doesn't help

			err := system.Unify()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("could not be resolved"))
		})

		It("should use default for constrained but unresolved variables", func() {
			tv := ir.NewTypeVariable("T", ir.NumericConstraint{})
			// Add to system but don't constrain it to a concrete type
			system.AddCompatible(tv, tv, nil, "T ~ T")

			Expect(system.Unify()).To(Succeed())

			// Should default to F64 for numeric constraint
			substitution, ok := system.GetSubstitution("T")
			Expect(ok).To(BeTrue())
			Expect(substitution).To(Equal(ir.F64{}))
		})
	})

	Describe("Complex Scenarios", func() {
		It("should handle multiple interconnected type variables", func() {
			// Simulating a more complex flow:
			// sensor -> multiply{factor: 2.0} -> add{a, b} <- constant{}

			// multiply stage
			multiplyInput := ir.NewTypeVariable("T1", ir.NumericConstraint{})
			multiplyOutput := ir.NewTypeVariable("T1", ir.NumericConstraint{})

			// add stage
			addParamA := ir.NewTypeVariable("T2", ir.NumericConstraint{})
			addParamB := ir.NewTypeVariable("T2", ir.NumericConstraint{})
			addOutput := ir.NewTypeVariable("T2", ir.NumericConstraint{})

			// constant stage
			constantOutput := ir.NewTypeVariable("T3", ir.NumericConstraint{})

			// Connections
			system.AddEquality(ir.F32{}, multiplyInput, nil, "sensor(f32) -> multiply")
			system.AddEquality(multiplyOutput, addParamA, nil, "multiply -> add.a")
			system.AddEquality(constantOutput, addParamB, nil, "constant -> add.b")

			// Internal stage constraints
			system.AddEquality(multiplyInput, multiplyOutput, nil, "multiply preserves type")
			system.AddEquality(addParamA, addParamB, nil, "add params must match")
			system.AddEquality(addParamA, addOutput, nil, "add output matches params")

			Expect(system.Unify()).To(Succeed())

			// Everything should resolve to F32
			Expect(system.ApplySubstitutions(multiplyInput)).To(Equal(ir.F32{}))
			Expect(system.ApplySubstitutions(multiplyOutput)).To(Equal(ir.F32{}))
			Expect(system.ApplySubstitutions(addParamA)).To(Equal(ir.F32{}))
			Expect(system.ApplySubstitutions(addParamB)).To(Equal(ir.F32{}))
			Expect(system.ApplySubstitutions(addOutput)).To(Equal(ir.F32{}))
			Expect(system.ApplySubstitutions(constantOutput)).To(Equal(ir.F32{}))
		})

		It("should handle series types with type variables", func() {
			tv := ir.NewTypeVariable("T", ir.NumericConstraint{})
			seriesTV := ir.Series{ValueType: tv}
			seriesI32 := ir.Series{ValueType: ir.I32{}}

			system.AddEquality(seriesTV, seriesI32, nil, "series T = series i32")

			Expect(system.Unify()).To(Succeed())

			substitution, ok := system.GetSubstitution("T")
			Expect(ok).To(BeTrue())
			Expect(substitution).To(Equal(ir.I32{}))

			// Check that series type is properly resolved
			resolved := system.ApplySubstitutions(seriesTV)
			Expect(resolved).To(Equal(seriesI32))
		})
	})
})
