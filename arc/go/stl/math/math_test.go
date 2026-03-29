// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package math_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	stlmath "github.com/synnaxlabs/arc/stl/math"
	"github.com/synnaxlabs/arc/stl/testutil"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Math", func() {
	var (
		rt  *testutil.Runtime
	)

	BeforeEach(func(ctx SpecContext) {
		rt = testutil.NewRuntime(ctx)
		MustSucceed(stlmath.NewModule(ctx, rt.Underlying()))
		rt.Passthrough(ctx, "math")
	})

	AfterEach(func(ctx SpecContext) {
		Expect(rt.Close(ctx)).To(Succeed())
	})

	Describe("pow", func() {
		It("Should compute i32 power", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_i32", testutil.U32(3), testutil.U32(2))
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(9)))
			res = rt.Call(ctx, "math", "pow_i32", testutil.U32(2), testutil.U32(10))
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(1024)))
		})

		It("Should compute i32 power with negative base", func(ctx SpecContext) {
			var negThree int32 = -3
			res := rt.Call(ctx, "math", "pow_i32", testutil.I32(negThree), testutil.U32(2))
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(9)))
			var negTwo int32 = -2
			var expected int32 = -8
			res = rt.Call(ctx, "math", "pow_i32", testutil.I32(negTwo), testutil.U32(3))
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(uint32(expected))))
		})

		It("Should compute u64 power", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_u64", testutil.U64(2), testutil.U64(10))
			Expect(testutil.AsU64(res[0])).To(Equal(uint64(1024)))
		})

		It("Should compute f32 power", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_f32", testutil.F32(2.0), testutil.F32(3.0))
			Expect(testutil.AsF32(res[0])).To(BeNumerically("~", 8.0, 0.001))
		})

		It("Should compute f64 power", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_f64", testutil.F64(2.0), testutil.F64(0.5))
			Expect(testutil.AsF64(res[0])).To(BeNumerically("~", 1.41421356, 0.0001))
		})

		It("Should truncate negative integer exponents to zero", func(ctx SpecContext) {
			negOne := int32(-1)
			res := rt.Call(ctx, "math", "pow_i32", testutil.U32(2), testutil.I32(negOne))
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(0)))
		})

		It("Should compute f64 negative exponents", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_f64", testutil.F64(2.0), testutil.F64(-1.0))
			Expect(testutil.AsF64(res[0])).To(BeNumerically("~", 0.5, 0.0001))
		})

		It("Should compute f32 negative exponents", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_f32", testutil.F32(4.0), testutil.F32(-0.5))
			Expect(testutil.AsF32(res[0])).To(BeNumerically("~", 0.5, 0.001))
		})

		It("Should compute f64 with negative base", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_f64", testutil.F64(-3.0), testutil.F64(2.0))
			Expect(testutil.AsF64(res[0])).To(BeNumerically("~", 9.0, 0.0001))
			res = rt.Call(ctx, "math", "pow_f64", testutil.F64(-2.0), testutil.F64(3.0))
			Expect(testutil.AsF64(res[0])).To(BeNumerically("~", -8.0, 0.0001))
			res = rt.Call(ctx, "math", "pow_f64", testutil.F64(-2.0), testutil.F64(-1.0))
			Expect(testutil.AsF64(res[0])).To(BeNumerically("~", -0.5, 0.0001))
		})

		It("Should compute f32 with negative base", func(ctx SpecContext) {
			res := rt.Call(ctx, "math", "pow_f32", testutil.F32(-3.0), testutil.F32(2.0))
			Expect(testutil.AsF32(res[0])).To(BeNumerically("~", 9.0, 0.001))
			res = rt.Call(ctx, "math", "pow_f32", testutil.F32(-2.0), testutil.F32(3.0))
			Expect(testutil.AsF32(res[0])).To(BeNumerically("~", -8.0, 0.001))
		})
	})
})
