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
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Type Unification", func() {
	var system *constraints.System
	BeforeEach(func() { system = constraints.New() })

	Describe("Simple Unification", func() {
		It("should unify type variable with concrete type", func() {
			tv := types.NewTypeVariable("T", nil)
			system.AddEquality(tv, types.F32(), nil, "T = f32")
			Expect(system.Unify()).To(Succeed())
			substitution := MustBeOk(system.GetSubstitution("T"))
			Expect(substitution).To(Equal(types.F32()))
		})

		It("should unify constrained type variable with valid type", func() {
			constraint := types.NumericConstraint()
			tv := types.NewTypeVariable("T", &constraint)
			system.AddEquality(tv, types.I64(), nil, "T = i64")
			Expect(system.Unify()).To(Succeed())
			substitution := MustBeOk(system.GetSubstitution("T"))
			Expect(substitution).To(Equal(types.I64()))
		})

		It("should fail to unify constrained type variable with invalid type", func() {
			constraint := types.NumericConstraint()
			tv := types.NewTypeVariable("T", &constraint)
			system.AddEquality(tv, types.String(), nil, "T = string")
			Expect(system.Unify()).To(MatchError(ContainSubstring("does not satisfy constraint")))
		})
	})

	Describe("Transitive Unification", func() {
		It("should unify through chains of type variables", func() {
			tv1 := types.NewTypeVariable("T1", nil)
			tv2 := types.NewTypeVariable("T2", nil)
			tv3 := types.NewTypeVariable("T3", nil)
			system.AddEquality(tv1, tv2, nil, "T1 = T2")
			system.AddEquality(tv2, tv3, nil, "T2 = T3")
			system.AddEquality(tv3, types.F64(), nil, "T3 = f64")
			Expect(system.Unify()).To(Succeed())
			Expect(system.ApplySubstitutions(tv1)).To(Equal(types.F64()))
			Expect(system.ApplySubstitutions(tv2)).To(Equal(types.F64()))
			Expect(system.ApplySubstitutions(tv3)).To(Equal(types.F64()))
		})
	})

	Describe("Numeric Promotion", func() {
		It("should promote compatible numeric types", func() {
			constraint := types.NumericConstraint()
			tv := types.NewTypeVariable("T", &constraint)
			system.AddCompatible(tv, types.I32(), nil, "T ~ i32")
			system.AddCompatible(tv, types.F32(), nil, "T ~ f32")
			Expect(system.Unify()).To(Succeed())
			substitution := MustBeOk(system.GetSubstitution("T"))
			Expect(substitution.IsNumeric()).To(BeTrue())
		})

		It("should handle bidirectional type flow", func() {
			constraint := types.NumericConstraint()
			// Simulating: constant{} -> add{a, b} <- channel(f32)
			constantOutput := types.NewTypeVariable("T1", &constraint)
			addParamA := types.NewTypeVariable("T", &constraint)
			addParamB := types.NewTypeVariable("T", &constraint)
			system.AddEquality(constantOutput, addParamA, nil, "constant -> add.a")
			system.AddEquality(types.F32(), addParamB, nil, "channel -> add.b")
			system.AddEquality(addParamA, addParamB, nil, "add.a = add.b")
			Expect(system.Unify()).To(Succeed())
			Expect(system.ApplySubstitutions(constantOutput)).To(Equal(types.F32()))
			Expect(system.ApplySubstitutions(addParamA)).To(Equal(types.F32()))
			Expect(system.ApplySubstitutions(addParamB)).To(Equal(types.F32()))
		})

		Describe("Channel Type Unification", func() {
			It("should unify channel types", func() {
				tv := types.NewTypeVariable("T", nil)
				chanTV := types.Chan(tv)
				chanF32 := types.Chan(types.F32())
				system.AddEquality(chanTV, chanF32, nil, "chan T = chan f32")
				Expect(system.Unify()).To(Succeed())
				substitution := MustBeOk(system.GetSubstitution("T"))
				Expect(substitution).To(Equal(types.F32()))
			})

			It("should fail on incompatible channel types", func() {
				chanI32 := types.Chan(types.I32())
				chanString := types.Chan(types.String())
				system.AddEquality(chanI32, chanString, nil, "chan i32 = chan string")
				Expect(system.Unify()).To(MatchError(ContainSubstring("failed to unify")))
			})
		})

		Describe("Error Cases", func() {
			It("should detect cyclic type dependencies", func() {
				tv1 := types.NewTypeVariable("T", nil)
				cyclicType := types.Chan(tv1)
				system.AddEquality(tv1, cyclicType, nil, "T = chan T")
				Expect(system.Unify()).To(MatchError(ContainSubstring("cyclic")))
			})

			It("should report unresolved type variables with no constraints", func() {
				tv := types.NewTypeVariable("T", nil)
				system.AddEquality(tv, tv, nil, "T = T") // Self-equality doesn't help
				Expect(system.Unify()).To(MatchError(ContainSubstring("could not be resolved")))
			})

			It("should use default for constrained but unresolved variables", func() {
				constraint := types.NumericConstraint()
				tv := types.NewTypeVariable("T", &constraint)
				system.AddCompatible(tv, tv, nil, "T ~ T")
				Expect(system.Unify()).To(Succeed())
				substitution := MustBeOk(system.GetSubstitution("T"))
				Expect(substitution).To(Equal(types.F64()))
			})
		})

		Describe("Complex Scenarios", func() {
			It("should handle multiple interconnected type variables", func() {
				// sensor -> multiply{factor: 2.0} -> add{a, b} <- constant{}
				var (
					constraint     = types.NumericConstraint()
					multiplyInput  = types.NewTypeVariable("T1", &constraint)
					multiplyOutput = types.NewTypeVariable("T1", &constraint)
					addParamA      = types.NewTypeVariable("T2", &constraint)
					addParamB      = types.NewTypeVariable("T2", &constraint)
					addOutput      = types.NewTypeVariable("T2", &constraint)
					constantOutput = types.NewTypeVariable("T3", &constraint)
				)
				system.AddEquality(
					types.F32(),
					multiplyInput,
					nil,
					"sensor(f32) -> multiply",
				)
				system.AddEquality(
					multiplyOutput,
					addParamA,
					nil,
					"multiply -> add.a",
				)
				system.AddEquality(
					constantOutput,
					addParamB,
					nil,
					"constant -> add.b",
				)
				system.AddEquality(
					multiplyInput,
					multiplyOutput,
					nil,
					"multiply preserves type",
				)
				system.AddEquality(
					addParamA,
					addParamB,
					nil,
					"add params must match",
				)
				system.AddEquality(
					addParamA,
					addOutput,
					nil,
					"add output matches params",
				)
				Expect(system.Unify()).To(Succeed())
				Expect(system.ApplySubstitutions(multiplyInput)).To(Equal(types.F32()))
				Expect(system.ApplySubstitutions(multiplyOutput)).To(Equal(types.F32()))
				Expect(system.ApplySubstitutions(addParamA)).To(Equal(types.F32()))
				Expect(system.ApplySubstitutions(addParamB)).To(Equal(types.F32()))
				Expect(system.ApplySubstitutions(addOutput)).To(Equal(types.F32()))
				Expect(system.ApplySubstitutions(constantOutput)).To(Equal(types.F32()))
			})

			It("should handle series types with type variables", func() {
				constraint := types.NumericConstraint()
				tv := types.NewTypeVariable("T", &constraint)
				seriesTV := types.Series(tv)
				seriesI32 := types.Series(types.I32())
				system.AddEquality(seriesTV, seriesI32, nil, "series T = series i32")
				Expect(system.Unify()).To(Succeed())
				Expect(MustBeOk(system.GetSubstitution("T"))).To(Equal(types.I32()))
				Expect(system.ApplySubstitutions(seriesTV)).To(Equal(seriesI32))
			})
		})
	})
})
