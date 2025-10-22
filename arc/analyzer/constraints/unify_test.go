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

			It("should detect cycles in series types", func() {
				tv := types.NewTypeVariable("T", nil)
				cyclicType := types.Series(tv)
				system.AddEquality(tv, cyclicType, nil, "T = series T")
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

			It("should fail when unifying channel with non-channel type", func() {
				chanF32 := types.Chan(types.F32())
				system.AddEquality(chanF32, types.I32(), nil, "chan f32 = i32")
				Expect(system.Unify()).To(MatchError(ContainSubstring("cannot unify channel")))
			})

			It("should fail when unifying series with non-series type", func() {
				seriesF32 := types.Series(types.F32())
				system.AddEquality(seriesF32, types.String(), nil, "series f32 = string")
				Expect(system.Unify()).To(MatchError(ContainSubstring("cannot unify channel")))
			})

			It("should fail when constraint doesn't match and not compatible", func() {
				f32Constraint := types.F32()
				tv := types.NewTypeVariable("T", &f32Constraint)
				system.AddEquality(tv, types.I32(), nil, "T = i32")
				Expect(system.Unify()).To(MatchError(ContainSubstring("does not match constraint")))
			})
		})

		It("should propagate type from channel through binary operator", func() {
			constraint := types.NumericConstraint()
			onOutput := types.Chan(types.F32())
			geLeft := types.NewTypeVariable("ge_T", &constraint)
			geRight := types.NewTypeVariable("ge_T", &constraint)
			constantOutput := types.NewTypeVariable("constant_T", &constraint)
			system.AddEquality(onOutput, types.Chan(geLeft), nil, "on -> ge.left")
			system.AddEquality(constantOutput, geRight, nil, "constant -> ge.right")
			Expect(system.Unify()).To(Succeed())
			Expect(system.ApplySubstitutions(constantOutput)).To(Equal(types.F32()))
		})

		It("should link type variables with same name", func() {
			constraint := types.NumericConstraint()
			tv1 := types.NewTypeVariable("T", &constraint)
			tv2 := types.NewTypeVariable("T", &constraint)
			system.AddEquality(tv1, types.F32(), nil, "tv1 = f32")
			Expect(system.Unify()).To(Succeed())
			Expect(system.ApplySubstitutions(tv2)).To(Equal(types.F32()))
		})

		Describe("Type Variable Constraint Preferences", func() {
			It("should prefer constrained type variable over unconstrained", func() {
				constraint := types.NumericConstraint()
				tv1 := types.NewTypeVariable("T1", &constraint)
				tv2 := types.NewTypeVariable("T2", nil)
				system.AddEquality(tv1, tv2, nil, "T1 = T2")
				Expect(system.Unify()).To(Succeed())
				sub2 := MustBeOk(system.GetSubstitution("T2"))
				Expect(sub2.Kind).To(Equal(types.KindTypeVariable))
				Expect(sub2.Name).To(Equal("T1"))
			})

			It("should prefer constrained over unconstrained in reverse order", func() {
				constraint := types.NumericConstraint()
				tv1 := types.NewTypeVariable("T1", nil)
				tv2 := types.NewTypeVariable("T2", &constraint)
				system.AddEquality(tv1, tv2, nil, "T1 = T2")
				Expect(system.Unify()).To(Succeed())
				sub1 := MustBeOk(system.GetSubstitution("T1"))
				Expect(sub1.Kind).To(Equal(types.KindTypeVariable))
				Expect(sub1.Name).To(Equal("T2"))
			})

			It("should handle two unconstrained variables with different names", func() {
				tv1 := types.NewTypeVariable("A", nil)
				tv2 := types.NewTypeVariable("B", nil)
				system.AddEquality(tv1, tv2, nil, "A = B")
				system.AddEquality(tv1, types.F32(), nil, "A = f32")
				Expect(system.Unify()).To(Succeed())
				Expect(system.ApplySubstitutions(tv1)).To(Equal(types.F32()))
				Expect(system.ApplySubstitutions(tv2)).To(Equal(types.F32()))
			})
		})

		Describe("Numeric Promotion with Compatible Constraints", func() {
			It("should promote i32 constraint with f32 value under compatible", func() {
				i32Constraint := types.I32()
				tv := types.NewTypeVariable("T", &i32Constraint)
				system.AddCompatible(tv, types.F32(), nil, "T ~ f32")
				Expect(system.Unify()).To(Succeed())
				substitution := MustBeOk(system.GetSubstitution("T"))
				Expect(substitution).To(Equal(types.F32()))
			})

			It("should promote i32 constraint with f64 value under compatible", func() {
				i32Constraint := types.I32()
				tv := types.NewTypeVariable("T", &i32Constraint)
				system.AddCompatible(tv, types.F64(), nil, "T ~ f64")
				Expect(system.Unify()).To(Succeed())
				substitution := MustBeOk(system.GetSubstitution("T"))
				Expect(substitution).To(Equal(types.F64()))
			})

			It("should promote i32 constraint with i64 value to f64 under compatible", func() {
				i32Constraint := types.I32()
				tv := types.NewTypeVariable("T", &i32Constraint)
				system.AddCompatible(tv, types.I64(), nil, "T ~ i64")
				Expect(system.Unify()).To(Succeed())
				substitution := MustBeOk(system.GetSubstitution("T"))
				// When mixing 32-bit and 64-bit signed integers, promotes to F64
				Expect(substitution).To(Equal(types.F64()))
			})

			It("should promote u32 constraint with u64 value under compatible", func() {
				u32Constraint := types.U32()
				tv := types.NewTypeVariable("T", &u32Constraint)
				system.AddCompatible(tv, types.U64(), nil, "T ~ u64")
				Expect(system.Unify()).To(Succeed())
				substitution := MustBeOk(system.GetSubstitution("T"))
				Expect(substitution).To(Equal(types.U64()))
			})

			It("should promote i32 constraint with u32 value to i32 under compatible", func() {
				i32Constraint := types.I32()
				tv := types.NewTypeVariable("T", &i32Constraint)
				system.AddCompatible(tv, types.U32(), nil, "T ~ u32")
				Expect(system.Unify()).To(Succeed())
				substitution := MustBeOk(system.GetSubstitution("T"))
				Expect(substitution).To(Equal(types.I32()))
			})

			It("should promote f32 constraint with i64 value to f64 under compatible", func() {
				f32Constraint := types.F32()
				tv := types.NewTypeVariable("T", &f32Constraint)
				system.AddCompatible(tv, types.I64(), nil, "T ~ i64")
				Expect(system.Unify()).To(Succeed())
				substitution := MustBeOk(system.GetSubstitution("T"))
				Expect(substitution).To(Equal(types.F64()))
			})

			It("should allow compatible numeric types without type variables", func() {
				system.AddCompatible(types.I32(), types.F32(), nil, "i32 ~ f32")
				Expect(system.Unify()).To(Succeed())
			})
		})

		Describe("Default Type Selection", func() {
			It("should default numeric constraint to f64", func() {
				constraint := types.NumericConstraint()
				tv := types.NewTypeVariable("T", &constraint)
				system.AddEquality(tv, tv, nil, "T = T")
				Expect(system.Unify()).To(Succeed())
				substitution := MustBeOk(system.GetSubstitution("T"))
				Expect(substitution).To(Equal(types.F64()))
			})

			It("should use concrete constraint type as default", func() {
				f32Constraint := types.F32()
				tv := types.NewTypeVariable("T", &f32Constraint)
				// Unify with another constrained variable with same constraint
				tv2 := types.NewTypeVariable("T", &f32Constraint)
				system.AddEquality(tv, tv2, nil, "T = T")
				Expect(system.Unify()).To(Succeed())
				substitution := MustBeOk(system.GetSubstitution("T"))
				Expect(substitution).To(Equal(types.F32()))
			})
		})

		Describe("Order Independence (Fixpoint Iteration)", func() {
			It("should produce same result regardless of constraint order", func() {
				// Test case: A = B, B = C, C = f32
				// Order 1: forward (A→B→C→f32)
				system1 := constraints.New()
				tv1a := types.NewTypeVariable("A", nil)
				tv1b := types.NewTypeVariable("B", nil)
				tv1c := types.NewTypeVariable("C", nil)
				system1.AddEquality(tv1a, tv1b, nil, "A = B")
				system1.AddEquality(tv1b, tv1c, nil, "B = C")
				system1.AddEquality(tv1c, types.F32(), nil, "C = f32")
				Expect(system1.Unify()).To(Succeed())

				// Order 2: reverse (f32→C→B→A)
				system2 := constraints.New()
				tv2a := types.NewTypeVariable("A", nil)
				tv2b := types.NewTypeVariable("B", nil)
				tv2c := types.NewTypeVariable("C", nil)
				system2.AddEquality(tv2c, types.F32(), nil, "C = f32")
				system2.AddEquality(tv2b, tv2c, nil, "B = C")
				system2.AddEquality(tv2a, tv2b, nil, "A = B")
				Expect(system2.Unify()).To(Succeed())

				// Order 3: middle-out (B→C→f32, then A→B)
				system3 := constraints.New()
				tv3a := types.NewTypeVariable("A", nil)
				tv3b := types.NewTypeVariable("B", nil)
				tv3c := types.NewTypeVariable("C", nil)
				system3.AddEquality(tv3b, tv3c, nil, "B = C")
				system3.AddEquality(tv3c, types.F32(), nil, "C = f32")
				system3.AddEquality(tv3a, tv3b, nil, "A = B")
				Expect(system3.Unify()).To(Succeed())

				// All should resolve to f32
				Expect(system1.ApplySubstitutions(tv1a)).To(Equal(types.F32()))
				Expect(system2.ApplySubstitutions(tv2a)).To(Equal(types.F32()))
				Expect(system3.ApplySubstitutions(tv3a)).To(Equal(types.F32()))
			})

			It("should handle circular constraints without concrete type", func() {
				// A = B, B = C, C = A (no concrete type)
				constraint := types.NumericConstraint()
				tv1 := types.NewTypeVariable("A", &constraint)
				tv2 := types.NewTypeVariable("B", &constraint)
				tv3 := types.NewTypeVariable("C", &constraint)
				system.AddEquality(tv1, tv2, nil, "A = B")
				system.AddEquality(tv2, tv3, nil, "B = C")
				system.AddEquality(tv3, tv1, nil, "C = A")
				Expect(system.Unify()).To(Succeed())
				// Should default to f64 since all have numeric constraint
				Expect(system.ApplySubstitutions(tv1)).To(Equal(types.F64()))
			})

			It("should handle complex graph with multiple constraint paths", func() {
				// Graph: T1 = T2, T2 = T3, T1 = f32, T3 = T4
				// Multiple paths to same conclusion
				tv1 := types.NewTypeVariable("T1", nil)
				tv2 := types.NewTypeVariable("T2", nil)
				tv3 := types.NewTypeVariable("T3", nil)
				tv4 := types.NewTypeVariable("T4", nil)
				system.AddEquality(tv1, tv2, nil, "T1 = T2")
				system.AddEquality(tv2, tv3, nil, "T2 = T3")
				system.AddEquality(tv1, types.F32(), nil, "T1 = f32")
				system.AddEquality(tv3, tv4, nil, "T3 = T4")
				Expect(system.Unify()).To(Succeed())
				Expect(system.ApplySubstitutions(tv4)).To(Equal(types.F32()))
			})

			It("should detect constraint ordering bugs with compatible constraints", func() {
				// Order shouldn't matter: T ~ i32, T ~ f32 should work
				constraint := types.NumericConstraint()

				// Order 1: i32 first
				system1 := constraints.New()
				tv1 := types.NewTypeVariable("T", &constraint)
				system1.AddCompatible(tv1, types.I32(), nil, "T ~ i32")
				system1.AddCompatible(tv1, types.F32(), nil, "T ~ f32")
				Expect(system1.Unify()).To(Succeed())

				// Order 2: f32 first
				system2 := constraints.New()
				tv2 := types.NewTypeVariable("T", &constraint)
				system2.AddCompatible(tv2, types.F32(), nil, "T ~ f32")
				system2.AddCompatible(tv2, types.I32(), nil, "T ~ i32")
				Expect(system2.Unify()).To(Succeed())

				// Both should resolve to f32 (float promotion)
				result1 := MustBeOk(system1.GetSubstitution("T"))
				result2 := MustBeOk(system2.GetSubstitution("T"))
				Expect(result1).To(Equal(result2), "Results should be identical regardless of order")
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
