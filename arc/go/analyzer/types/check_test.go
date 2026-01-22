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
)

var _ = Describe("Check", func() {
	var cs *constraints.System

	BeforeEach(func() {
		cs = constraints.New()
	})

	Describe("Invalid Types", func() {
		DescribeTable("should return nil without checking",
			func(t1, t2 types.Type) {
				ast := testutil.NewMockAST(1)
				err := atypes.Check(cs, t1, t2, ast, "test")
				Expect(err).ToNot(HaveOccurred())
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
			Expect(atypes.Check(cs, types.F64(), tv, ast, "another reason")).To(Succeed())
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
			Expect(atypes.Check(cs, tv, types.Chan(types.F32()), ast, "var with chan")).To(Succeed())
			Expect(cs.Constraints).To(HaveLen(1))
			Expect(cs.Constraints[0].Right.Kind).To(Equal(types.KindChan))
		})

		It("should add constraint for variable with series type", func() {
			tv := types.Variable("T", nil)
			ast := testutil.NewMockAST(1)
			Expect(atypes.Check(cs, types.Series(types.I64()), tv, ast, "series with var")).To(Succeed())
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
		DescribeTable("should return error for different types",
			func(t1, t2 types.Type, expectedT1, expectedT2 string) {
				ast := testutil.NewMockAST(1)
				err := atypes.Check(cs, t1, t2, ast, "test")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("type mismatch"))
				Expect(err.Error()).To(ContainSubstring(expectedT1))
				Expect(err.Error()).To(ContainSubstring(expectedT2))
			},
			Entry("f32 with f64", types.F32(), types.F64(), "f32", "f64"),
			Entry("i32 with i64", types.I32(), types.I64(), "i32", "i64"),
			Entry("u8 with i8", types.U8(), types.I8(), "u8", "i8"),
			Entry("string with i32", types.String(), types.I32(), "str", "i32"),
		)
	})

	Describe("Unit Mismatch", func() {
		It("should fail for same kind with different units", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Type{Kind: types.KindF32, Unit: &types.Unit{Name: "psi"}}
			t2 := types.Type{Kind: types.KindF32, Unit: &types.Unit{Name: "bar"}}
			err := atypes.Check(cs, t1, t2, ast, "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type mismatch"))
			Expect(err.Error()).To(ContainSubstring("psi"))
			Expect(err.Error()).To(ContainSubstring("bar"))
		})

		It("should fail for type with unit vs type without unit", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Type{Kind: types.KindF32, Unit: &types.Unit{Name: "psi"}}
			t2 := types.F32()
			err := atypes.Check(cs, t1, t2, ast, "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type mismatch"))
		})

		It("should succeed for same kind with same unit", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Type{Kind: types.KindF32, Unit: &types.Unit{Name: "psi", Scale: 1}}
			t2 := types.Type{Kind: types.KindF32, Unit: &types.Unit{Name: "psi", Scale: 1}}
			Expect(atypes.Check(cs, t1, t2, ast, "test")).To(Succeed())
		})
	})

	Describe("Structural Mismatch", func() {
		DescribeTable("should return error when structure differs",
			func(t1, t2 types.Type) {
				ast := testutil.NewMockAST(1)
				err := atypes.Check(cs, t1, t2, ast, "test")
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("type mismatch"))
			},
			Entry("chan with scalar", types.Chan(types.F32()), types.F32()),
			Entry("scalar with chan", types.F32(), types.Chan(types.F32())),
			Entry("series with scalar", types.Series(types.I64()), types.I64()),
			Entry("scalar with series", types.I64(), types.Series(types.I64())),
			Entry("chan with series", types.Chan(types.F32()), types.Series(types.F32())),
			Entry("series with chan", types.Series(types.I32()), types.Chan(types.I32())),
		)
	})

	Describe("Channel Types", func() {
		It("should succeed for matching channel types", func() {
			ast := testutil.NewMockAST(1)
			Expect(atypes.Check(cs, types.Chan(types.F32()), types.Chan(types.F32()), ast, "test")).To(Succeed())
		})

		It("should fail for channel element type mismatch", func() {
			ast := testutil.NewMockAST(1)
			err := atypes.Check(cs, types.Chan(types.F32()), types.Chan(types.F64()), ast, "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type mismatch"))
			Expect(err.Error()).To(ContainSubstring("f32"))
			Expect(err.Error()).To(ContainSubstring("f64"))
		})

		It("should fail for channel with different integer element types", func() {
			ast := testutil.NewMockAST(1)
			err := atypes.Check(cs, types.Chan(types.I32()), types.Chan(types.I64()), ast, "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type mismatch"))
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
			err := atypes.Check(cs, t1, t2, ast, "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type mismatch"))
		})

		It("should fail for nested channel depth mismatch", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Chan(types.Chan(types.I32()))
			t2 := types.Chan(types.I32())
			err := atypes.Check(cs, t1, t2, ast, "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type mismatch"))
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
			Expect(atypes.Check(cs, types.Series(types.I64()), types.Series(types.I64()), ast, "test")).To(Succeed())
		})

		It("should fail for series element type mismatch", func() {
			ast := testutil.NewMockAST(1)
			err := atypes.Check(cs, types.Series(types.F32()), types.Series(types.F64()), ast, "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type mismatch"))
			Expect(err.Error()).To(ContainSubstring("f32"))
			Expect(err.Error()).To(ContainSubstring("f64"))
		})

		It("should fail for series with different integer element types", func() {
			ast := testutil.NewMockAST(1)
			err := atypes.Check(cs, types.Series(types.I32()), types.Series(types.U32()), ast, "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type mismatch"))
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
			err := atypes.Check(cs, t1, t2, ast, "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type mismatch"))
		})

		It("should fail for nested series depth mismatch", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Series(types.Series(types.I32()))
			t2 := types.Series(types.I32())
			err := atypes.Check(cs, t1, t2, ast, "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type mismatch"))
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
		It("should succeed for channel containing series with matching element", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Chan(types.Series(types.F32()))
			t2 := types.Chan(types.Series(types.F32()))
			Expect(atypes.Check(cs, t1, t2, ast, "test")).To(Succeed())
		})

		It("should fail for channel containing series with mismatched element", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Chan(types.Series(types.F32()))
			t2 := types.Chan(types.Series(types.F64()))
			err := atypes.Check(cs, t1, t2, ast, "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type mismatch"))
		})

		It("should fail for channel vs series at same level", func() {
			ast := testutil.NewMockAST(1)
			t1 := types.Chan(types.Series(types.F32()))
			t2 := types.Chan(types.Chan(types.F32()))
			err := atypes.Check(cs, t1, t2, ast, "test")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("type mismatch"))
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
			Expect(cs.Constraints[0].Reason).To(Equal("deep (element types) (element types)"))
		})
	})

	Describe("Reason Propagation", func() {
		It("should preserve reason for direct type variable constraint", func() {
			ast := testutil.NewMockAST(1)
			tv := types.Variable("T", nil)
			Expect(atypes.Check(cs, tv, types.F32(), ast, "original reason")).To(Succeed())
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
			Expect(cs.Constraints[0].Reason).To(Equal("nested (element types) (element types)"))
		})
	})

	Describe("Constraint System Integration", func() {
		It("should not add constraints for matching concrete types", func() {
			ast := testutil.NewMockAST(1)
			Expect(atypes.Check(cs, types.F32(), types.F32(), ast, "test")).To(Succeed())
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
