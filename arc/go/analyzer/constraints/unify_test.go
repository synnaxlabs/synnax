// Copyright 2026 Synnax Labs, Inc.
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
)

var _ = Describe("Type Unification", func() {
	Describe("Simple Unification", func() {
		It("should unify type variable with concrete type", func() {
			var (
				system = constraints.New()
				tv     = types.Variable("T", nil)
			)
			system.AddEquality(tv, types.F32(), nil, "T = f32")
			Expect(system.Unify()).To(Succeed())
			Expect(system.Substitutions["T"]).To(Equal(types.F32()))
		})

		It("should unify constrained type variable with valid type", func() {
			var (
				system     = constraints.New()
				constraint = types.NumericConstraint()
				tv         = types.Variable("T", &constraint)
			)
			system.AddEquality(tv, types.I64(), nil, "T = i64")
			Expect(system.Unify()).To(Succeed())
			Expect(system.Substitutions["T"]).To(Equal(types.I64()))
		})

		It("should fail to unify constrained type variable with invalid type", func() {
			var (
				system     = constraints.New()
				constraint = types.NumericConstraint()
				tv         = types.Variable("T", &constraint)
			)
			Expect(system.AddEquality(tv, types.String(), nil, "T = string")).To(MatchError(ContainSubstring("is not compatible with")))
		})
	})

	Describe("Transitive Unification", func() {
		It("should unify through chains of type variables", func() {
			var (
				system = constraints.New()
				tv1    = types.Variable("T1", nil)
				tv2    = types.Variable("T2", nil)
				tv3    = types.Variable("T3", nil)
			)
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
		testPromotion := func(constraint, value, expected types.Type) {
			system := constraints.New()
			tv := types.Variable("T", &constraint)
			Expect(system.AddCompatible(tv, value, nil, "promotion test")).To(Succeed())
			Expect(system.Unify()).To(Succeed())
			Expect(system.Substitutions["T"]).To(Equal(expected))
		}

		DescribeTable("Rule 1: Float Promotion - when either type is float",
			testPromotion,
			Entry("f32 ~ i32 → f32", types.F32(), types.I32(), types.F32()),
			Entry("i32 ~ f32 → f32", types.I32(), types.F32(), types.F32()),
			Entry("f32 ~ i64 → f64", types.F32(), types.I64(), types.F64()),
			Entry("f64 ~ i32 → f64", types.F64(), types.I32(), types.F64()),
			Entry("f64 ~ i64 → f64", types.F64(), types.I64(), types.F64()),
			Entry("f32 ~ u32 → f32", types.F32(), types.U32(), types.F32()),
		)

		DescribeTable("Rule 2: 64-bit Integer Promotion - when both are integers and either is 64-bit",
			testPromotion,
			Entry("u64 ~ u32 → u64", types.U64(), types.U32(), types.U64()),
			Entry("u64 ~ u64 → u64", types.U64(), types.U64(), types.U64()),
			Entry("i64 ~ u64 → f64", types.I64(), types.U64(), types.F64()),
			Entry("u64 ~ i64 → f64", types.U64(), types.I64(), types.F64()),
			Entry("i64 ~ u32 → f64", types.I64(), types.U32(), types.F64()),
			Entry("u64 ~ i32 → f64", types.U64(), types.I32(), types.F64()),
			Entry("i64 ~ i32 → f64", types.I64(), types.I32(), types.F64()),
			Entry("i32 ~ i64 → f64", types.I32(), types.I64(), types.F64()),
		)

		DescribeTable("Rule 3: 32-bit and Smaller Integer Promotion - when both are integers ≤32-bit",
			testPromotion,
			Entry("i32 ~ u32 → i32", types.I32(), types.U32(), types.I32()),
			Entry("u32 ~ i32 → i32", types.U32(), types.I32(), types.I32()),
			Entry("i32 ~ i16 → i32", types.I32(), types.I16(), types.I32()),
			Entry("u32 ~ u16 → u32", types.U32(), types.U16(), types.U32()),
			Entry("i16 ~ u16 → i32", types.I16(), types.U16(), types.I32()),
			Entry("i16 ~ i8 → i32", types.I16(), types.I8(), types.I32()),
			Entry("u16 ~ u8 → u32", types.U16(), types.U8(), types.U32()),
			Entry("i8 ~ u8 → i32", types.I8(), types.U8(), types.I32()),
			Entry("u32 ~ u32 → u32", types.U32(), types.U32(), types.U32()),
			Entry("i32 ~ i32 → i32", types.I32(), types.I32(), types.I32()),
		)

		It("should handle bidirectional type flow", func() {
			var (
				system         = constraints.New()
				constraint     = types.NumericConstraint()
				constantOutput = types.Variable("T1", &constraint)
				addParamA      = types.Variable("T", &constraint)
				addParamB      = types.Variable("T", &constraint)
			)
			system.AddEquality(constantOutput, addParamA, nil, "constant -> add.a")
			system.AddEquality(types.F32(), addParamB, nil, "channel -> add.b")
			system.AddEquality(addParamA, addParamB, nil, "add.a = add.b")
			Expect(system.Unify()).To(Succeed())
			Expect(system.ApplySubstitutions(constantOutput)).To(Equal(types.F32()))
			Expect(system.ApplySubstitutions(addParamA)).To(Equal(types.F32()))
			Expect(system.ApplySubstitutions(addParamB)).To(Equal(types.F32()))
		})

		It("should propagate type from channel through binary operator", func() {
			var (
				system         = constraints.New()
				constraint     = types.NumericConstraint()
				onOutput       = types.Chan(types.F32())
				geLeft         = types.Variable("ge_T", &constraint)
				geRight        = types.Variable("ge_T", &constraint)
				constantOutput = types.Variable("constant_T", &constraint)
			)
			system.AddEquality(onOutput, types.Chan(geLeft), nil, "on -> ge.left")
			system.AddEquality(constantOutput, geRight, nil, "constant -> ge.right")
			Expect(system.Unify()).To(Succeed())
			Expect(system.ApplySubstitutions(constantOutput)).To(Equal(types.F32()))
		})

		It("should link type variables with same name", func() {
			var (
				system     = constraints.New()
				constraint = types.NumericConstraint()
				tv1        = types.Variable("T", &constraint)
				tv2        = types.Variable("T", &constraint)
			)
			system.AddEquality(tv1, types.F32(), nil, "tv1 = f32")
			Expect(system.Unify()).To(Succeed())
			Expect(system.ApplySubstitutions(tv2)).To(Equal(types.F32()))
		})

		It("should allow compatible numeric types without type variables", func() {
			system := constraints.New()
			system.AddCompatible(types.I32(), types.F32(), nil, "i32 ~ f32")
			Expect(system.Unify()).To(Succeed())
		})
	})

	Describe("Channel Type Unification", func() {
		It("should unify channel types", func() {
			var (
				system  = constraints.New()
				tv      = types.Variable("T", nil)
				chanTV  = types.Chan(tv)
				chanF32 = types.Chan(types.F32())
			)
			system.AddEquality(chanTV, chanF32, nil, "chan T = chan f32")
			Expect(system.Unify()).To(Succeed())
			Expect(system.Substitutions["T"]).To(Equal(types.F32()))
		})

		It("should fail on incompatible channel types", func() {
			var (
				system     = constraints.New()
				chanI32    = types.Chan(types.I32())
				chanString = types.Chan(types.String())
			)
			Expect(system.AddEquality(chanI32, chanString, nil, "chan i32 = chan string")).To(MatchError(ContainSubstring("is not compatible with")))
		})
	})

	Describe("Error Cases", func() {
		DescribeTable("should detect cyclic type dependencies",
			func(makeCyclic func(types.Type) types.Type) {
				var (
					system     = constraints.New()
					tv         = types.Variable("T", nil)
					cyclicType = makeCyclic(tv)
				)
				Expect(system.AddEquality(tv, cyclicType, nil, "T = cyclic T")).To(MatchError(ContainSubstring("is not compatible with")))
			},
			Entry("chan T", func(tv types.Type) types.Type { return types.Chan(tv) }),
			Entry("series T", func(tv types.Type) types.Type { return types.Series(tv) }),
		)

		It("should report unresolved type variables with no constraints", func() {
			var (
				system = constraints.New()
				tv     = types.Variable("T", nil)
			)
			system.AddEquality(tv, tv, nil, "T = T")
			Expect(system.Unify()).To(MatchError(ContainSubstring("unresolved type variable")))
		})

		It("should use default for constrained but unresolved variables", func() {
			var (
				system     = constraints.New()
				constraint = types.NumericConstraint()
				tv         = types.Variable("T", &constraint)
			)
			system.AddCompatible(tv, tv, nil, "T ~ T")
			Expect(system.Unify()).To(Succeed())
			Expect(system.Substitutions["T"]).To(Equal(types.F64()))
		})

		DescribeTable("should fail when unifying incompatible compound types",
			func(compoundType, otherType types.Type) {
				system := constraints.New()
				Expect(system.AddEquality(compoundType, otherType, nil, "incompatible")).To(MatchError(ContainSubstring("is not compatible with")))
			},
			Entry("chan f32 = i32", types.Chan(types.F32()), types.I32()),
			Entry("series f32 = string", types.Series(types.F32()), types.String()),
			Entry("chan f32 = series f32", types.Chan(types.F32()), types.Series(types.F32())),
		)

		It("should fail when constraint doesn't match and not compatible", func() {
			var (
				system        = constraints.New()
				f32Constraint = types.F32()
				tv            = types.Variable("T", &f32Constraint)
			)
			Expect(system.AddEquality(tv, types.I32(), nil, "T = i32")).To(MatchError(ContainSubstring("is not compatible with")))
		})
	})

	Describe("Type Variable Constraint Preferences", func() {
		It("should prefer constrained type variable over unconstrained", func() {
			var (
				system     = constraints.New()
				constraint = types.NumericConstraint()
				tv1        = types.Variable("T1", &constraint)
				tv2        = types.Variable("T2", nil)
			)
			system.AddEquality(tv1, tv2, nil, "T1 = T2")
			Expect(system.Unify()).To(Succeed())
			sub2 := system.Substitutions["T2"]
			Expect(sub2.Kind).To(Equal(types.KindVariable))
			Expect(sub2.Name).To(Equal("T1"))
		})

		It("should prefer constrained over unconstrained in reverse order", func() {
			var (
				system     = constraints.New()
				constraint = types.NumericConstraint()
				tv1        = types.Variable("T1", nil)
				tv2        = types.Variable("T2", &constraint)
			)
			system.AddEquality(tv1, tv2, nil, "T1 = T2")
			Expect(system.Unify()).To(Succeed())
			sub1 := system.Substitutions["T1"]
			Expect(sub1.Kind).To(Equal(types.KindVariable))
			Expect(sub1.Name).To(Equal("T2"))
		})

		It("should handle two unconstrained variables with different names", func() {
			var (
				system = constraints.New()
				tv1    = types.Variable("A", nil)
				tv2    = types.Variable("B", nil)
			)
			system.AddEquality(tv1, tv2, nil, "A = B")
			system.AddEquality(tv1, types.F32(), nil, "A = f32")
			Expect(system.Unify()).To(Succeed())
			Expect(system.ApplySubstitutions(tv1)).To(Equal(types.F32()))
			Expect(system.ApplySubstitutions(tv2)).To(Equal(types.F32()))
		})
	})

	Describe("Default Type Selection", func() {
		testDefault := func(constraint, expected types.Type) {
			var (
				system = constraints.New()
				tv     = types.Variable("T", &constraint)
			)
			system.AddEquality(tv, tv, nil, "T = T")
			Expect(system.Unify()).To(Succeed())
			Expect(system.Substitutions["T"]).To(Equal(expected))
		}

		DescribeTable("should default constraint to expected type",
			testDefault,
			Entry("numeric → f64", types.NumericConstraint(), types.F64()),
			Entry("integer → i64", types.IntegerConstraint(), types.I64()),
			Entry("float → f64", types.FloatConstraint(), types.F64()),
			Entry("f32 → f32", types.F32(), types.F32()),
		)
	})

	Describe("Order Independence (Fixpoint Iteration)", func() {
		It("should produce same result regardless of constraint order", func() {
			addConstraints := func(s *constraints.System, order int) types.Type {
				var (
					a = types.Variable("A", nil)
					b = types.Variable("B", nil)
					c = types.Variable("C", nil)
				)
				switch order {
				case 1: // forward
					Expect(s.AddEquality(a, b, nil, "A = B")).To(Succeed())
					Expect(s.AddEquality(b, c, nil, "B = C")).To(Succeed())
					Expect(s.AddEquality(c, types.F32(), nil, "C = f32")).To(Succeed())
				case 2: // reverse
					Expect(s.AddEquality(c, types.F32(), nil, "C = f32")).To(Succeed())
					Expect(s.AddEquality(b, c, nil, "B = C")).To(Succeed())
					Expect(s.AddEquality(a, b, nil, "A = B")).To(Succeed())
				case 3: // middle-out
					Expect(s.AddEquality(b, c, nil, "B = C")).To(Succeed())
					Expect(s.AddEquality(c, types.F32(), nil, "C = f32")).To(Succeed())
					Expect(s.AddEquality(a, b, nil, "A = B")).To(Succeed())
				}
				return a
			}

			for _, order := range []int{1, 2, 3} {
				system := constraints.New()
				a := addConstraints(system, order)
				Expect(system.Unify()).To(Succeed())
				Expect(system.ApplySubstitutions(a)).To(Equal(types.F32()))
			}
		})

		It("should handle circular constraints without concrete type", func() {
			var (
				system     = constraints.New()
				constraint = types.NumericConstraint()
				tv1        = types.Variable("A", &constraint)
				tv2        = types.Variable("B", &constraint)
				tv3        = types.Variable("C", &constraint)
			)
			system.AddEquality(tv1, tv2, nil, "A = B")
			system.AddEquality(tv2, tv3, nil, "B = C")
			system.AddEquality(tv3, tv1, nil, "C = A")
			Expect(system.Unify()).To(Succeed())
			Expect(system.ApplySubstitutions(tv1)).To(Equal(types.F64()))
		})

		It("should handle complex graph with multiple constraint paths", func() {
			var (
				system = constraints.New()
				tv1    = types.Variable("T1", nil)
				tv2    = types.Variable("T2", nil)
				tv3    = types.Variable("T3", nil)
				tv4    = types.Variable("T4", nil)
			)
			system.AddEquality(tv1, tv2, nil, "T1 = T2")
			system.AddEquality(tv2, tv3, nil, "T2 = T3")
			system.AddEquality(tv1, types.F32(), nil, "T1 = f32")
			system.AddEquality(tv3, tv4, nil, "T3 = T4")
			Expect(system.Unify()).To(Succeed())
			Expect(system.ApplySubstitutions(tv4)).To(Equal(types.F32()))
		})

		It("should detect constraint ordering bugs with compatible constraints", func() {
			constraint := types.NumericConstraint()
			for _, firstType := range []types.Type{types.I32(), types.F32()} {
				var (
					system     = constraints.New()
					tv         = types.Variable("T", &constraint)
					secondType = types.F32()
				)
				if firstType.Kind == types.KindF32 {
					secondType = types.I32()
				}
				system.AddCompatible(tv, firstType, nil, "T ~ first")
				system.AddCompatible(tv, secondType, nil, "T ~ second")
				Expect(system.Unify()).To(Succeed())
				Expect(system.Substitutions["T"]).To(Equal(types.F32()))
			}
		})
	})

	Describe("Float Literal Constraints", func() {
		DescribeTable("should allow float literal to be assigned to float type",
			func(targetType types.Type) {
				var (
					system          = constraints.New()
					floatConstraint = types.FloatConstraint()
					lit             = types.Variable("lit", &floatConstraint)
				)
				system.AddEquality(targetType, lit, nil, "assignment")
				Expect(system.Unify()).To(Succeed())
				Expect(system.ApplySubstitutions(lit)).To(Equal(targetType))
			},
			Entry("f32", types.F32()),
			Entry("f64", types.F64()),
		)

		DescribeTable("should reject float literal assigned to incompatible type",
			func(targetType types.Type) {
				var (
					system          = constraints.New()
					floatConstraint = types.FloatConstraint()
					tv              = types.Variable("T", &floatConstraint)
				)
				Expect(system.AddEquality(tv, targetType, nil, "T = target")).To(MatchError(ContainSubstring("is not compatible with")))
			},
			Entry("i32", types.I32()),
			Entry("string", types.String()),
		)

		It("should allow float literal in compatible context with any numeric", func() {
			var (
				system          = constraints.New()
				floatConstraint = types.FloatConstraint()
				tv              = types.Variable("T", &floatConstraint)
			)
			system.AddCompatible(tv, types.I32(), nil, "T ~ i32")
			Expect(system.Unify()).To(Succeed())
		})

		It("should reject float literal in compatible context with non-numeric", func() {
			var (
				system          = constraints.New()
				floatConstraint = types.FloatConstraint()
				tv              = types.Variable("T", &floatConstraint)
			)
			Expect(system.AddCompatible(tv, types.String(), nil, "T ~ string")).To(MatchError(ContainSubstring("is not compatible with")))
		})
	})

	Describe("Integer Literal Constraints", func() {
		It("should reject integer literal assigned to non-numeric type", func() {
			var (
				system        = constraints.New()
				intConstraint = types.IntegerConstraint()
				tv            = types.Variable("T", &intConstraint)
			)
			Expect(system.AddEquality(tv, types.String(), nil, "T = string")).To(MatchError(ContainSubstring("is not compatible with")))
		})

		DescribeTable("should allow integer literal to be assigned to integer type",
			func(targetType types.Type) {
				var (
					system        = constraints.New()
					intConstraint = types.IntegerConstraint()
					lit           = types.Variable("lit", &intConstraint)
				)
				system.AddEquality(targetType, lit, nil, "assignment")
				Expect(system.Unify()).To(Succeed())
				Expect(system.ApplySubstitutions(lit)).To(Equal(targetType))
			},
			Entry("i8", types.I8()),
			Entry("i16", types.I16()),
			Entry("i32", types.I32()),
			Entry("i64", types.I64()),
			Entry("u8", types.U8()),
			Entry("u16", types.U16()),
			Entry("u32", types.U32()),
			Entry("u64", types.U64()),
		)

		DescribeTable("should allow integer literal to be coerced to float type",
			func(targetType types.Type) {
				var (
					system        = constraints.New()
					intConstraint = types.IntegerConstraint()
					lit           = types.Variable("lit", &intConstraint)
				)
				system.AddEquality(targetType, lit, nil, "assignment")
				Expect(system.Unify()).To(Succeed())
				Expect(system.ApplySubstitutions(lit)).To(Equal(targetType))
			},
			Entry("f32", types.F32()),
			Entry("f64", types.F64()),
		)

		It("should not prematurely resolve integer literals in compatible context", func() {
			var (
				system        = constraints.New()
				intConstraint = types.IntegerConstraint()
				lit1          = types.Variable("lit_1", &intConstraint)
				lit2          = types.Variable("lit_2", &intConstraint)
			)
			system.AddCompatible(lit1, lit2, nil, "lit1 ~ lit2")
			system.AddEquality(types.I64(), lit1, nil, "assignment to i64")
			Expect(system.Unify()).To(Succeed())
			Expect(system.ApplySubstitutions(lit1)).To(Equal(types.I64()))
			Expect(system.ApplySubstitutions(lit2)).To(Equal(types.I64()))
		})

		It("should unify integer literal with i64 in nested expression (regression)", func() {
			var (
				system        = constraints.New()
				intConstraint = types.IntegerConstraint()
				lit1          = types.Variable("lit_1", &intConstraint)
				lit2          = types.Variable("lit_2", &intConstraint)
				lit3          = types.Variable("lit_3", &intConstraint)
				lit4          = types.Variable("lit_4", &intConstraint)
				lit5          = types.Variable("lit_5", &intConstraint)
				lit6          = types.Variable("lit_6", &intConstraint)
				add           = types.Variable("add_result", &intConstraint)
				mul           = types.Variable("mul_result", &intConstraint)
				sub           = types.Variable("sub_result", &intConstraint)
				div           = types.Variable("div_result", &intConstraint)
				mod           = types.Variable("mod_result", &intConstraint)
			)
			// Binary operations: ((((1 + 2) * 3) - 4) / 5) % 6
			system.AddCompatible(lit1, lit2, nil, "1 + 2")
			system.AddCompatible(lit1, add, nil, "result of 1 + 2")
			system.AddCompatible(add, lit3, nil, "add * 3")
			system.AddCompatible(add, mul, nil, "result of add * 3")
			system.AddCompatible(mul, lit4, nil, "mul - 4")
			system.AddCompatible(mul, sub, nil, "result of mul - 4")
			system.AddCompatible(sub, lit5, nil, "sub / 5")
			system.AddCompatible(sub, div, nil, "result of sub / 5")
			system.AddCompatible(div, lit6, nil, "div % 6")
			system.AddCompatible(div, mod, nil, "result of div % 6")
			system.AddEquality(types.I64(), mod, nil, "x i64 := expression")

			Expect(system.Unify()).To(Succeed())
			for _, lit := range []types.Type{lit1, lit2, lit3, lit4, lit5, lit6, mod} {
				Expect(system.ApplySubstitutions(lit)).To(Equal(types.I64()))
			}
		})
	})

	Describe("UnificationError", func() {
		It("Should implement error interface with correct message", func() {
			err := &constraints.UnificationError{
				Message: "type mismatch: i32 is not compatible with f64",
			}
			var e error = err
			Expect(e.Error()).To(Equal("type mismatch: i32 is not compatible with f64"))
		})

		It("Should preserve constraint context", func() {
			constraint := constraints.Constraint{
				Left:   types.I32(),
				Right:  types.F64(),
				Reason: "assignment",
			}
			err := &constraints.UnificationError{
				Constraint: &constraint,
				Left:       types.I32(),
				Right:      types.F64(),
				Message:    "type mismatch in assignment: i32 is not compatible with f64",
			}
			Expect(err.Constraint).ToNot(BeNil())
			Expect(err.Constraint.Reason).To(Equal("assignment"))
			Expect(err.Left).To(Equal(types.I32()))
			Expect(err.Right).To(Equal(types.F64()))
		})

		It("Should be returned by UnifyConstraint on type mismatch", func() {
			system := constraints.New()
			constraint := constraints.Constraint{
				Left:   types.I32(),
				Right:  types.String(),
				Reason: "test",
			}
			err := system.UnifyConstraint(constraint)
			Expect(err).To(HaveOccurred())

			ue, ok := err.(*constraints.UnificationError)
			Expect(ok).To(BeTrue(), "expected UnificationError")
			Expect(ue.Message).To(ContainSubstring("is not compatible with"))
		})

		It("Should include conversion hint for numeric type mismatches", func() {
			system := constraints.New()
			constraint := constraints.Constraint{
				Left:   types.I64(),
				Right:  types.F64(),
				Reason: "test",
			}
			err := system.UnifyConstraint(constraint)
			Expect(err).To(HaveOccurred())

			ue, ok := err.(*constraints.UnificationError)
			Expect(ok).To(BeTrue())
			Expect(ue.Hint).To(ContainSubstring("use"))
			Expect(ue.Hint).To(ContainSubstring("to convert"))
		})
	})

	Describe("Complex Scenarios", func() {
		It("should handle multiple interconnected type variables", func() {
			var (
				system         = constraints.New()
				constraint     = types.NumericConstraint()
				multiplyInput  = types.Variable("T1", &constraint)
				multiplyOutput = types.Variable("T1", &constraint)
				addParamA      = types.Variable("T2", &constraint)
				addParamB      = types.Variable("T2", &constraint)
				addOutput      = types.Variable("T2", &constraint)
				constantOutput = types.Variable("T3", &constraint)
			)
			system.AddEquality(types.F32(), multiplyInput, nil, "sensor(f32) -> multiply")
			system.AddEquality(multiplyOutput, addParamA, nil, "multiply -> add.a")
			system.AddEquality(constantOutput, addParamB, nil, "constant -> add.b")
			system.AddEquality(multiplyInput, multiplyOutput, nil, "multiply preserves type")
			system.AddEquality(addParamA, addParamB, nil, "add params must match")
			system.AddEquality(addParamA, addOutput, nil, "add output matches params")

			Expect(system.Unify()).To(Succeed())
			for _, tv := range []types.Type{multiplyInput, multiplyOutput, addParamA, addParamB, addOutput, constantOutput} {
				Expect(system.ApplySubstitutions(tv)).To(Equal(types.F32()))
			}
		})

		It("should handle series types with type variables", func() {
			var (
				system     = constraints.New()
				constraint = types.NumericConstraint()
				tv         = types.Variable("T", &constraint)
				seriesTV   = types.Series(tv)
				seriesI32  = types.Series(types.I32())
			)
			system.AddEquality(seriesTV, seriesI32, nil, "series T = series i32")
			Expect(system.Unify()).To(Succeed())
			Expect(system.Substitutions["T"]).To(Equal(types.I32()))
			Expect(system.ApplySubstitutions(seriesTV)).To(Equal(seriesI32))
		})

		It("should handle type variable unification with self through chain", func() {
			var (
				system = constraints.New()
				tv1    = types.Variable("T1", nil)
				tv2    = types.Variable("T2", nil)
			)
			system.AddEquality(tv1, tv2, nil, "T1 = T2")
			system.AddEquality(tv2, tv1, nil, "T2 = T1")
			system.AddEquality(tv1, types.F32(), nil, "T1 = f32")
			Expect(system.Unify()).To(Succeed())
			Expect(system.ApplySubstitutions(tv1)).To(Equal(types.F32()))
			Expect(system.ApplySubstitutions(tv2)).To(Equal(types.F32()))
		})

		It("should handle complex promotion chain requiring iteration", func() {
			var (
				system     = constraints.New()
				constraint = types.NumericConstraint()
				t1         = types.Variable("T1", &constraint)
				t2         = types.Variable("T2", &constraint)
				t3         = types.Variable("T3", &constraint)
			)
			system.AddCompatible(t1, types.I32(), nil, "T1 ~ i32")
			system.AddCompatible(t2, types.I64(), nil, "T2 ~ i64")
			system.AddCompatible(t3, types.F32(), nil, "T3 ~ f32")
			system.AddCompatible(t1, t2, nil, "T1 ~ T2")
			system.AddCompatible(t2, t3, nil, "T2 ~ T3")
			system.AddCompatible(t3, t1, nil, "T3 ~ T1")

			Expect(system.Unify()).To(Succeed())
			for _, tv := range []types.Type{t1, t2, t3} {
				Expect(system.ApplySubstitutions(tv).IsNumeric()).To(BeTrue())
			}
		})

		It("should iterate when constraint order causes cascading updates", func() {
			var (
				system     = constraints.New()
				constraint = types.NumericConstraint()
				a          = types.Variable("A", &constraint)
				b          = types.Variable("B", &constraint)
				c          = types.Variable("C", &constraint)
				d          = types.Variable("D", &constraint)
			)
			system.AddCompatible(a, types.I32(), nil, "A ~ i32")
			system.AddCompatible(b, types.I64(), nil, "B ~ i64")
			system.AddCompatible(c, a, nil, "C ~ A")
			system.AddCompatible(d, b, nil, "D ~ B")
			system.AddCompatible(c, d, nil, "C ~ D")
			system.AddCompatible(a, b, nil, "A ~ B")

			Expect(system.Unify()).To(Succeed())
			for _, tv := range []types.Type{a, b, c, d} {
				Expect(system.ApplySubstitutions(tv).IsNumeric()).To(BeTrue())
			}
		})
	})
})
