// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package types_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/analyzer/constraints"
	"github.com/synnaxlabs/arc/analyzer/testutil"
	atypes "github.com/synnaxlabs/arc/analyzer/types"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Check", func() {
	var cs *constraints.System

	BeforeEach(func() {
		cs = constraints.New()
	})

	Describe("Untyped literals", func() {
		DescribeTable(
			"Should check an untyped value against an untyped target",
			func(target, value types.Type, fits bool) {
				t := types.Variable("target", &target)
				v := types.Variable("value", &value)
				if fits {
					Expect(atypes.Check(cs, t, v, testutil.NewMockAST(1), "test")).
						To(Succeed())
					return
				}
				Expect(atypes.Check(cs, t, v, testutil.NewMockAST(1), "test")).
					To(MatchError(ContainSubstring("got float")))
			},
			Entry(
				"float into integer",
				types.IntegerConstraint(),
				types.FloatConstraint(),
				false,
			),
			Entry(
				"integer into float",
				types.FloatConstraint(),
				types.IntegerConstraint(),
				true,
			),
			Entry(
				"integer into integer",
				types.IntegerConstraint(),
				types.IntegerConstraint(),
				true,
			),
		)
	})

	It("Should let an integer literal fix a generic numeric parameter", func() {
		numeric := types.NumericConstraint()
		param := types.Variable("T", &numeric)
		integer := types.IntegerConstraint()
		value := types.Variable("value", &integer)
		Expect(atypes.Check(cs, param, value, testutil.NewMockAST(1), "test")).
			To(Succeed())
		Expect(cs.Unify()).To(Succeed())
		Expect(cs.ApplySubstitutions(param)).To(Equal(types.I64()))
	})

	DescribeTable("Should widen a generic numeric parameter to a float in either order",
		func(integerFirst bool) {
			numeric := types.NumericConstraint()
			param := types.Variable("T", &numeric)
			integer := types.IntegerConstraint()
			intValue := types.Variable("int", &integer)
			float := types.FloatConstraint()
			floatValue := types.Variable("float", &float)
			first, second := intValue, floatValue
			if !integerFirst {
				first, second = floatValue, intValue
			}
			ast := testutil.NewMockAST(1)
			Expect(atypes.Check(cs, param, first, ast, "test")).To(Succeed())
			Expect(atypes.Check(cs, param, second, ast, "test")).To(Succeed())
			Expect(cs.Unify()).To(Succeed())
			Expect(cs.ApplySubstitutions(param)).To(Equal(types.F64()))
			Expect(cs.ApplySubstitutions(intValue)).To(Equal(types.F64()))
		},
		Entry("integer first", true),
		Entry("float first", false),
	)

	It("Should check an untyped value against the type its target is bound to", func() {
		integer := types.IntegerConstraint()
		target := types.Variable("target", &integer)
		Expect(cs.AddEquality(target, types.F32(), nil, "test")).To(Succeed())
		float := types.FloatConstraint()
		value := types.Variable("value", &float)
		Expect(atypes.Check(cs, target, value, testutil.NewMockAST(1), "test")).
			To(Succeed())
		Expect(cs.ApplySubstitutions(value)).To(Equal(types.F32()))
	})

	Describe("Invalid Types", func() {
		DescribeTable("should return nil without checking",
			func(t1, t2 types.Type) {
				ast := testutil.NewMockAST(1)
				Expect(atypes.Check(cs, t1, t2, ast, "test")).To(Succeed())
				Expect(cs.Constraints).To(BeEmpty())
			},
			Entry("invalid with concrete", types.Type{}, types.I32()),
			Entry("concrete with invalid", types.F32(), types.Type{}),
			Entry("invalid with invalid", types.Type{}, types.Type{}),
			Entry("invalid with type variable", types.Type{}, types.Variable("T", nil)),
			Entry("type variable with invalid", types.Variable("T", nil), types.Type{}),
			Entry("invalid with channel", types.Type{}, types.Chan(types.F32())),
			Entry("invalid with series", types.Type{}, types.Series(types.I64())),
		)
	})

	Describe("Type Variables", func() {
		It("should add equality constraint when first type is variable", func() {
			tv := types.Variable("T", nil)
			ast := testutil.NewMockAST(1)
			Expect(atypes.Check(cs, tv, types.I32(), ast, "test reason")).To(Succeed())
			Expect(cs.Constraints).To(HaveLen(1))
			Expect(cs.Constraints[0].Left).To(Equal(tv))
			Expect(cs.Constraints[0].Right).To(Equal(types.I32()))
			Expect(cs.Constraints[0].Reason).To(Equal("test reason"))
		})

		It("should add equality constraint when second type is variable", func() {
			tv := types.Variable("U", nil)
			ast := testutil.NewMockAST(1)
			Expect(
				atypes.Check(cs, types.F64(), tv, ast, "another reason"),
			).To(Succeed())
			Expect(cs.Constraints).To(HaveLen(1))
			Expect(cs.Constraints[0].Left).To(Equal(types.F64()))
			Expect(cs.Constraints[0].Right).To(Equal(tv))
			Expect(cs.Constraints[0].Reason).To(Equal("another reason"))
		})

		It("should add equality constraint when both types are variables", func() {
			tv1 := types.Variable("T", nil)
			tv2 := types.Variable("U", nil)
			ast := testutil.NewMockAST(1)
			Expect(atypes.Check(cs, tv1, tv2, ast, "both vars")).To(Succeed())
			Expect(cs.Constraints).To(HaveLen(1))
			Expect(cs.Constraints[0].Left).To(Equal(tv1))
			Expect(cs.Constraints[0].Right).To(Equal(tv2))
		})

		It("should add equality constraint when same variable on both sides", func() {
			tv := types.Variable("T", nil)
			ast := testutil.NewMockAST(1)
			Expect(atypes.Check(cs, tv, tv, ast, "same var")).To(Succeed())
			Expect(cs.Constraints).To(HaveLen(1))
		})

		It("should add constraint for variable with channel type", func() {
			tv := types.Variable("T", nil)
			ast := testutil.NewMockAST(1)
			Expect(
				atypes.Check(cs, tv, types.Chan(types.F32()), ast, "var with chan"),
			).To(Succeed())
			Expect(cs.Constraints).To(HaveLen(1))
			Expect(cs.Constraints[0].Right.Kind).To(Equal(types.KindChan))
		})

		It("should add constraint for variable with series type", func() {
			tv := types.Variable("T", nil)
			ast := testutil.NewMockAST(1)
			Expect(
				atypes.Check(cs, types.Series(types.I64()), tv, ast, "series with var"),
			).To(Succeed())
			Expect(cs.Constraints).To(HaveLen(1))
			Expect(cs.Constraints[0].Left.Kind).To(Equal(types.KindSeries))
		})
	})

	Describe("Matching Concrete Types", func() {
		DescribeTable("should succeed for equal types",
			func(t1, t2 types.Type) {
				ast := testutil.NewMockAST(1)
				Expect(atypes.Check(cs, t1, t2, ast, "test")).To(Succeed())
			},
			Entry("f32 with f32", types.F32(), types.F32()),
			Entry("f64 with f64", types.F64(), types.F64()),
			Entry("i8 with i8", types.I8(), types.I8()),
			Entry("i16 with i16", types.I16(), types.I16()),
			Entry("i32 with i32", types.I32(), types.I32()),
			Entry("i64 with i64", types.I64(), types.I64()),
			Entry("u8 with u8", types.U8(), types.U8()),
			Entry("u16 with u16", types.U16(), types.U16()),
			Entry("u32 with u32", types.U32(), types.U32()),
			Entry("u64 with u64", types.U64(), types.U64()),
			Entry("string with string", types.String(), types.String()),
		)
	})

	Describe("Mismatched Concrete Types", func() {
		DescribeTable(
			"should return error for different types",
			func(t1, t2 types.Type, expectedMsg string) {
				ast := testutil.NewMockAST(1)
				Expect(atypes.Check(cs, t1, t2, ast, "test")).
					Error().To(MatchError(ContainSubstring(expectedMsg)))
			},
			Entry(
				"f32 with f64",
				types.F32(),
				types.F64(),
				"type mismatch in test: expected f32, got f64",
			),
			Entry(
				"i32 with i64",
				types.I32(),
				types.I64(),
				"type mismatch in test: expected i32, got i64",
			),
			Entry(
				"u8 with i8",
				types.U8(),
				types.I8(),
				"type mismatch in test: expected u8, got i8",
			),
			Entry(
				"string with i32",
				types.String(),
				types.I32(),
				"type mismatch in test: expected str, got i32",
			),
		)
	})

	Describe("CheckConverted", func() {
		unit := func(name string, scale float64) *types.Unit {
			return &types.Unit{Dimensions: types.DimTime, Scale: scale, Name: name}
		}
		untyped := func(name string, u *types.Unit) types.Type {
			constraint := types.IntegerConstraint()
			t := types.Variable(name, &constraint)
			t.Unit = u
			return t
		}

		It("should succeed for the same kind with different units", func() {
			ast := testutil.NewMockAST(1)
			expected := types.Type{Kind: types.KindI64, Unit: unit("min", 6e10)}
			actual := types.Type{Kind: types.KindI64, Unit: unit("s", 1e9)}
			Expect(
				atypes.CheckConverted(cs, expected, actual, ast, "test"),
			).To(Succeed())
		})

		It("should fail for different kinds", func() {
			ast := testutil.NewMockAST(1)
			expected := types.Type{Kind: types.KindI64, Unit: unit("min", 6e10)}
			actual := types.Type{Kind: types.KindF64, Unit: unit("s", 1e9)}
			Expect(atypes.CheckConverted(cs, expected, actual, ast, "test")).Error().
				To(MatchError(ContainSubstring("type mismatch")))
		})

		It("should leave two untyped types apart", func() {
			ast := testutil.NewMockAST(1)
			expected, actual := untyped(
				"d",
				unit("min", 6e10),
			), untyped(
				"other",
				unit("s", 1e9),
			)
			Expect(
				atypes.CheckConverted(cs, expected, actual, ast, "test"),
			).To(Succeed())
			Expect(cs.ApplySubstitutions(expected)).To(Equal(expected))
			Expect(cs.ApplySubstitutions(actual)).To(Equal(actual))
		})

		It("should fail for two untyped types of different kinds", func() {
			ast := testutil.NewMockAST(1)
			constraint := types.FloatConstraint()
			expected := types.Variable("d", &constraint)
			expected.Unit = unit("ms", 1e6)
			actual := untyped("other", unit("s", 1e9))
			Expect(atypes.CheckConverted(cs, expected, actual, ast, "test")).Error().
				To(MatchError(ContainSubstring("type mismatch")))
			Expect(atypes.CheckConverted(cs, actual, expected, ast, "test")).Error().
				To(MatchError(ContainSubstring("type mismatch")))
		})

		It(
			"should give an untyped type the kind of a typed one and keep its unit",
			func() {
				ast := testutil.NewMockAST(1)
				expected := types.Type{Kind: types.KindI32, Unit: unit("min", 6e10)}
				actual := untyped("other", unit("s", 1e9))
				Expect(
					atypes.CheckConverted(cs, expected, actual, ast, "test"),
				).To(Succeed())
				resolved := cs.ApplySubstitutions(actual)
				Expect(resolved.Kind).To(Equal(types.KindI32))
				Expect(resolved.Unit.Name).To(Equal("s"))
			},
		)
	})

	Describe("Unit Compatibility", func() {
		It("should fail for same kind with different non-nil units", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Type{Kind: types.KindF32, Unit: &types.Unit{Name: "psi"}}
			t2 := types.Type{Kind: types.KindF32, Unit: &types.Unit{Name: "bar"}}
			Expect(atypes.Check(cs, t1, t2, ast, "test")).Error().
				To(MatchError(ContainSubstring("type mismatch")))
		})

		It("should succeed for type with unit vs type without unit", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Type{Kind: types.KindF32, Unit: &types.Unit{Name: "psi"}}
			t2 := types.F32()
			Expect(atypes.Check(cs, t1, t2, ast, "test")).To(Succeed())
		})

		It("should succeed for same kind with same unit", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Type{
				Kind: types.KindF32,
				Unit: &types.Unit{Name: "psi", Scale: 1},
			}
			t2 := types.Type{
				Kind: types.KindF32,
				Unit: &types.Unit{Name: "psi", Scale: 1},
			}
			Expect(atypes.Check(cs, t1, t2, ast, "test")).To(Succeed())
		})

		It("should succeed for i64 ns vs i64 (timestamp <-> int64)", func() {
			ast := testutil.NewMockAST(1)
			Expect(
				atypes.Check(cs, types.TimeStamp(), types.I64(), ast, "test"),
			).To(Succeed())
			Expect(
				atypes.Check(cs, types.I64(), types.TimeStamp(), ast, "test"),
			).To(Succeed())
		})

		It("should succeed for chan i64 ns <-> chan i64", func() {
			ast := testutil.NewMockAST(1)
			Expect(atypes.Check(
				cs,
				types.Chan(types.TimeStamp()),
				types.Chan(types.I64()),
				ast, "test",
			)).To(Succeed())
		})
	})

	Describe("Structural Mismatch", func() {
		DescribeTable("should return error when structure differs",
			func(t1, t2 types.Type, expectedMsg string) {
				ast := testutil.NewMockAST(1)
				Expect(atypes.Check(cs, t1, t2, ast, "test")).
					Error().To(MatchError(ContainSubstring(expectedMsg)))
			},
			Entry(
				"chan with scalar",
				types.Chan(types.F32()),
				types.F32(),
				"type mismatch in test: expected chan f32, got f32",
			),
			Entry(
				"scalar with chan",
				types.F32(),
				types.Chan(types.F32()),
				"type mismatch in test: expected f32, got chan f32",
			),
			Entry(
				"series with scalar",
				types.Series(types.I64()),
				types.I64(),
				"type mismatch in test: expected series i64, got i64",
			),
			Entry(
				"scalar with series",
				types.I64(),
				types.Series(types.I64()),
				"type mismatch in test: expected i64, got series i64",
			),
			Entry(
				"chan with series",
				types.Chan(types.F32()),
				types.Series(types.F32()),
				"type mismatch in test: expected chan f32, got series f32",
			),
			Entry(
				"series with chan",
				types.Series(types.I32()),
				types.Chan(types.I32()),
				"type mismatch in test: expected series i32, got chan i32",
			),
		)
	})

	Describe("Channel Types", func() {
		It("should succeed for matching channel types", func() {
			ast := testutil.NewMockAST(1)
			Expect(
				atypes.Check(
					cs,
					types.Chan(types.F32()),
					types.Chan(types.F32()),
					ast,
					"test",
				),
			).To(Succeed())
		})

		It("should fail for channel element type mismatch", func() {
			ast := testutil.NewMockAST(1)
			Expect(
				atypes.Check(cs, types.Chan(types.F32()), types.Chan(types.F64()), ast, "test"),
			).
				Error().
				To(MatchError(ContainSubstring("type mismatch in test (element types): expected f32, got f64")))
		})

		It("should fail for channel with different integer element types", func() {
			ast := testutil.NewMockAST(1)
			Expect(
				atypes.Check(cs, types.Chan(types.I32()), types.Chan(types.I64()), ast, "test"),
			).
				Error().
				To(MatchError(ContainSubstring("type mismatch in test (element types): expected i32, got i64")))
		})

		It("should succeed for nested matching channels", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Chan(types.Chan(types.I32()))
			t2 := types.Chan(types.Chan(types.I32()))
			Expect(atypes.Check(cs, t1, t2, ast, "test")).To(Succeed())
		})

		It("should fail for nested channel element type mismatch", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Chan(types.Chan(types.I32()))
			t2 := types.Chan(types.Chan(types.I64()))
			Expect(atypes.Check(cs, t1, t2, ast, "test")).
				Error().
				To(MatchError(ContainSubstring("type mismatch in test (element types) (element types): expected i32, got i64")))
		})

		It("should fail for nested channel depth mismatch", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Chan(types.Chan(types.I32()))
			t2 := types.Chan(types.I32())
			Expect(atypes.Check(cs, t1, t2, ast, "test")).
				Error().
				To(MatchError(ContainSubstring("type mismatch in test (element types): expected chan i32, got i32")))
		})

		It("should add constraint for channel with variable element", func() {
			ast := testutil.NewMockAST(1)
			tv := types.Variable("T", nil)
			t1 := types.Chan(tv)
			t2 := types.Chan(types.F32())
			Expect(atypes.Check(cs, t1, t2, ast, "chan elem")).To(Succeed())
			Expect(cs.Constraints).To(HaveLen(1))
			Expect(cs.Constraints[0].Left).To(Equal(tv))
			Expect(cs.Constraints[0].Right).To(Equal(types.F32()))
			Expect(cs.Constraints[0].Reason).To(Equal("chan elem (element types)"))
		})
	})

	Describe("Series Types", func() {
		It("should succeed for matching series types", func() {
			ast := testutil.NewMockAST(1)
			Expect(
				atypes.Check(
					cs,
					types.Series(types.I64()),
					types.Series(types.I64()),
					ast,
					"test",
				),
			).To(Succeed())
		})

		It("should fail for series element type mismatch", func() {
			ast := testutil.NewMockAST(1)
			Expect(
				atypes.Check(cs, types.Series(types.F32()), types.Series(types.F64()), ast, "test"),
			).
				Error().
				To(MatchError(ContainSubstring("type mismatch in test (element types): expected f32, got f64")))
		})

		It("should fail for series with different integer element types", func() {
			ast := testutil.NewMockAST(1)
			Expect(
				atypes.Check(cs, types.Series(types.I32()), types.Series(types.U32()), ast, "test"),
			).
				Error().
				To(MatchError(ContainSubstring("type mismatch in test (element types): expected i32, got u32")))
		})

		It("should succeed for nested matching series", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Series(types.Series(types.F64()))
			t2 := types.Series(types.Series(types.F64()))
			Expect(atypes.Check(cs, t1, t2, ast, "test")).To(Succeed())
		})

		It("should fail for nested series element type mismatch", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Series(types.Series(types.F32()))
			t2 := types.Series(types.Series(types.F64()))
			Expect(atypes.Check(cs, t1, t2, ast, "test")).
				Error().
				To(MatchError(ContainSubstring("type mismatch in test (element types) (element types): expected f32, got f64")))
		})

		It("should fail for nested series depth mismatch", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Series(types.Series(types.I32()))
			t2 := types.Series(types.I32())
			Expect(atypes.Check(cs, t1, t2, ast, "test")).
				Error().
				To(MatchError(ContainSubstring("type mismatch in test (element types): expected series i32, got i32")))
		})

		It("should add constraint for series with variable element", func() {
			ast := testutil.NewMockAST(1)
			tv := types.Variable("T", nil)
			t1 := types.Series(tv)
			t2 := types.Series(types.I64())
			Expect(atypes.Check(cs, t1, t2, ast, "series elem")).To(Succeed())
			Expect(cs.Constraints).To(HaveLen(1))
			Expect(cs.Constraints[0].Left).To(Equal(tv))
			Expect(cs.Constraints[0].Right).To(Equal(types.I64()))
			Expect(cs.Constraints[0].Reason).To(Equal("series elem (element types)"))
		})
	})

	Describe("Mixed Channel and Series", func() {
		It(
			"should succeed for channel containing series with matching element",
			func() {
				ast := testutil.NewMockAST(1)
				t1 := types.Chan(types.Series(types.F32()))
				t2 := types.Chan(types.Series(types.F32()))
				Expect(atypes.Check(cs, t1, t2, ast, "test")).To(Succeed())
			},
		)

		It("should fail for channel containing series with mismatched element", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Chan(types.Series(types.F32()))
			t2 := types.Chan(types.Series(types.F64()))
			Expect(atypes.Check(cs, t1, t2, ast, "test")).
				Error().
				To(MatchError(ContainSubstring("type mismatch in test (element types) (element types): expected f32, got f64")))
		})

		It("should fail for channel vs series at same level", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Chan(types.Series(types.F32()))
			t2 := types.Chan(types.Chan(types.F32()))
			Expect(atypes.Check(cs, t1, t2, ast, "test")).
				Error().
				To(MatchError(ContainSubstring("type mismatch in test (element types): expected series f32, got chan f32")))
		})

		It("should add constraint for deeply nested variable", func() {
			ast := testutil.NewMockAST(1)
			tv := types.Variable("T", nil)
			t1 := types.Chan(types.Series(tv))
			t2 := types.Chan(types.Series(types.I32()))
			Expect(atypes.Check(cs, t1, t2, ast, "deep")).To(Succeed())
			Expect(cs.Constraints).To(HaveLen(1))
			Expect(cs.Constraints[0].Left).To(Equal(tv))
			Expect(cs.Constraints[0].Right).To(Equal(types.I32()))
			Expect(
				cs.Constraints[0].Reason,
			).To(Equal("deep (element types) (element types)"))
		})
	})

	Describe("Reason Propagation", func() {
		It("should preserve reason for direct type variable constraint", func() {
			ast := testutil.NewMockAST(1)
			tv := types.Variable("T", nil)
			Expect(
				atypes.Check(cs, tv, types.F32(), ast, "original reason"),
			).To(Succeed())
			Expect(cs.Constraints[0].Reason).To(Equal("original reason"))
		})

		It("should append element types suffix for channel unwrap", func() {
			ast := testutil.NewMockAST(1)
			tv := types.Variable("T", nil)
			t1 := types.Chan(tv)
			t2 := types.Chan(types.I32())
			Expect(atypes.Check(cs, t1, t2, ast, "base reason")).To(Succeed())
			Expect(cs.Constraints[0].Reason).To(Equal("base reason (element types)"))
		})

		It("should append element types suffix for series unwrap", func() {
			ast := testutil.NewMockAST(1)
			tv := types.Variable("T", nil)
			t1 := types.Series(types.F64())
			t2 := types.Series(tv)
			Expect(atypes.Check(cs, t1, t2, ast, "series reason")).To(Succeed())
			Expect(cs.Constraints[0].Reason).To(Equal("series reason (element types)"))
		})

		It("should append multiple suffixes for nested unwrap", func() {
			ast := testutil.NewMockAST(1)
			tv := types.Variable("T", nil)
			t1 := types.Series(types.Series(tv))
			t2 := types.Series(types.Series(types.U64()))
			Expect(atypes.Check(cs, t1, t2, ast, "nested")).To(Succeed())
			Expect(
				cs.Constraints[0].Reason,
			).To(Equal("nested (element types) (element types)"))
		})
	})

	Describe("Constraint System Integration", func() {
		It("should not add constraints for matching concrete types", func() {
			ast := testutil.NewMockAST(1)
			Expect(
				atypes.Check(cs, types.F32(), types.F32(), ast, "test"),
			).To(Succeed())
			Expect(cs.Constraints).To(BeEmpty())
		})

		It("should not add constraints for matching channel types", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Chan(types.I64())
			t2 := types.Chan(types.I64())
			Expect(atypes.Check(cs, t1, t2, ast, "test")).To(Succeed())
			Expect(cs.Constraints).To(BeEmpty())
		})

		It("should not add constraints for matching series types", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Series(types.F32())
			t2 := types.Series(types.F32())
			Expect(atypes.Check(cs, t1, t2, ast, "test")).To(Succeed())
			Expect(cs.Constraints).To(BeEmpty())
		})

		It("should accumulate multiple constraints from multiple checks", func() {
			ast := testutil.NewMockAST(1)
			tv1 := types.Variable("T", nil)
			tv2 := types.Variable("U", nil)
			Expect(atypes.Check(cs, tv1, types.I32(), ast, "first")).To(Succeed())
			Expect(atypes.Check(cs, tv2, types.F64(), ast, "second")).To(Succeed())
			Expect(cs.Constraints).To(HaveLen(2))
			Expect(cs.Constraints[0].Reason).To(Equal("first"))
			Expect(cs.Constraints[1].Reason).To(Equal("second"))
		})
	})
})

var _ = Describe("CastOf", func() {
	It("Should give the type to cast to for a numeric mismatch", func() {
		err := atypes.Check(constraints.New(), types.F64(), types.I64(), nil, "")
		Expect(MustBeOk(atypes.CastOf(err))).To(Equal("f64"))
	})

	DescribeTable("Should give no type for a channel or a series",
		func(expected, actual types.Type) {
			err := atypes.Check(constraints.New(), expected, actual, nil, "")
			Expect(err).To(MatchError(ContainSubstring("type mismatch")))
			_, ok := atypes.CastOf(err)
			Expect(ok).To(BeFalse())
		},
		Entry("a series argument", types.F32(), types.Series(types.I32())),
		Entry("a series parameter", types.Series(types.F32()), types.F32()),
		Entry("channel elements", types.Chan(types.F32()), types.Chan(types.I32())),
	)

	It("Should give no type for a mismatch that a cast does not fix", func() {
		err := atypes.Check(constraints.New(), types.F64(), types.String(), nil, "")
		Expect(err).To(MatchError(ContainSubstring("type mismatch")))
		_, ok := atypes.CastOf(err)
		Expect(ok).To(BeFalse())
	})
})
