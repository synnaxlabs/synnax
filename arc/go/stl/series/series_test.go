// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package series_test

import (
	"fmt"
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Series", func() {
	var (
		rt *testutil.Runtime
		ss *series.ProgramState
	)

	BeforeEach(func(ctx SpecContext) {
		rt = testutil.NewRuntime(ctx)
		ss = series.NewProgramState()
		MustSucceed(series.NewHost(ctx, rt.Underlying(), ss))
		rt.Passthrough(ctx, "series")
	})

	AfterEach(func(ctx SpecContext) {
		Expect(rt.Close(ctx)).To(Succeed())
	})

	call := func(ctx SpecContext, fn string, args ...uint64) []uint64 {
		return rt.Call(ctx, "series", fn, args...)
	}

	callU32 := func(ctx SpecContext, fn string, args ...uint64) uint32 {
		return testutil.AsU32(call(ctx, fn, args...)[0])
	}

	describeI32Type := func(suffix string, dt telem.DataType, a, b uint32) {
		Describe(suffix, func() {
			It(
				"Should create, set, index, and perform element ops",
				func(ctx SpecContext) {
					h := callU32(ctx, "create_empty_"+suffix, testutil.U32(3))
					Expect(h).ToNot(BeZero())
					callU32(
						ctx,
						"set_element_"+suffix,
						testutil.U32(h),
						testutil.U32(0),
						testutil.U32(a),
					)
					callU32(
						ctx,
						"set_element_"+suffix,
						testutil.U32(h),
						testutil.U32(1),
						testutil.U32(b),
					)
					callU32(
						ctx,
						"set_element_"+suffix,
						testutil.U32(h),
						testutil.U32(2),
						testutil.U32(a+b),
					)
					Expect(
						callU32(ctx, "index_"+suffix, testutil.U32(h), testutil.U32(0)),
					).To(Equal(a))
					Expect(
						callU32(ctx, "index_"+suffix, testutil.U32(h), testutil.U32(1)),
					).To(Equal(b))
					Expect(
						callU32(ctx, "index_"+suffix, testutil.U32(h), testutil.U32(2)),
					).To(Equal(a + b))

					rh := callU32(
						ctx,
						"element_add_"+suffix,
						testutil.U32(h),
						testutil.U32(1),
					)
					Expect(rh).ToNot(BeZero())
					ser := MustBeOk(ss.Get(rh))
					Expect(ser.DataType).To(Equal(dt))

					_ = callU32(
						ctx,
						"element_sub_"+suffix,
						testutil.U32(h),
						testutil.U32(1),
					)
					_ = callU32(
						ctx,
						"element_mul_"+suffix,
						testutil.U32(h),
						testutil.U32(2),
					)
					_ = callU32(
						ctx,
						"element_div_"+suffix,
						testutil.U32(h),
						testutil.U32(2),
					)
					_ = callU32(
						ctx,
						"element_mod_"+suffix,
						testutil.U32(h),
						testutil.U32(2),
					)

					_ = callU32(
						ctx,
						"element_rsub_"+suffix,
						testutil.U32(10),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_rdiv_"+suffix,
						testutil.U32(10),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_rmod_"+suffix,
						testutil.U32(10),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_radd_"+suffix,
						testutil.U32(1),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_rmul_"+suffix,
						testutil.U32(2),
						testutil.U32(h),
					)
				},
			)

			It("Should perform series arithmetic", func(ctx SpecContext) {
				h1 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h1),
					testutil.U32(0),
					testutil.U32(a),
				)
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h1),
					testutil.U32(1),
					testutil.U32(b),
				)
				h2 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h2),
					testutil.U32(0),
					testutil.U32(b),
				)
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h2),
					testutil.U32(1),
					testutil.U32(a),
				)

				for _, name := range []string{
					"series_add_", "series_sub_", "series_mul_",
					"series_div_", "series_mod_",
				} {
					rh := callU32(ctx, name+suffix, testutil.U32(h1), testutil.U32(h2))
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should perform series and scalar comparisons", func(ctx SpecContext) {
				h1 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h1),
					testutil.U32(0),
					testutil.U32(a),
				)
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h1),
					testutil.U32(1),
					testutil.U32(b),
				)
				h2 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h2),
					testutil.U32(0),
					testutil.U32(b),
				)
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h2),
					testutil.U32(1),
					testutil.U32(a),
				)

				for _, name := range []string{
					"compare_gt_", "compare_lt_", "compare_ge_",
					"compare_le_", "compare_eq_", "compare_ne_",
				} {
					rh := callU32(ctx, name+suffix, testutil.U32(h1), testutil.U32(h2))
					Expect(rh).ToNot(BeZero())
					cmpSer := MustBeOk(ss.Get(rh))
					Expect(cmpSer.DataType).To(Equal(telem.BooleanT))
				}

				for _, name := range []string{
					"compare_gt_scalar_", "compare_lt_scalar_",
					"compare_ge_scalar_", "compare_le_scalar_",
					"compare_eq_scalar_", "compare_ne_scalar_",
				} {
					rh := callU32(ctx, name+suffix, testutil.U32(h1), testutil.U32(a))
					Expect(rh).ToNot(BeZero())
				}

				_ = callU32(ctx, "index_"+suffix, testutil.U32(h1), testutil.U32(0))
			})

			It("Should return 0 for invalid handle", func(ctx SpecContext) {
				Expect(
					callU32(ctx, "index_"+suffix, testutil.U32(9999), testutil.U32(0)),
				).To(Equal(uint32(0)))
				Expect(
					callU32(
						ctx,
						"element_add_"+suffix,
						testutil.U32(9999),
						testutil.U32(1),
					),
				).To(Equal(uint32(0)))
			})
		})
	}

	describeI64Type := func(suffix string, dt telem.DataType, a, b uint64) {
		Describe(suffix, func() {
			It(
				"Should create, set, index, and perform element ops",
				func(ctx SpecContext) {
					h := callU32(ctx, "create_empty_"+suffix, testutil.U32(3))
					Expect(h).ToNot(BeZero())
					callU32(
						ctx,
						"set_element_"+suffix,
						testutil.U32(h),
						testutil.U32(0),
						testutil.U64(a),
					)
					callU32(
						ctx,
						"set_element_"+suffix,
						testutil.U32(h),
						testutil.U32(1),
						testutil.U64(b),
					)
					callU32(
						ctx,
						"set_element_"+suffix,
						testutil.U32(h),
						testutil.U32(2),
						testutil.U64(a+b),
					)
					Expect(
						testutil.AsU64(
							call(ctx, "index_"+suffix, testutil.U32(h), testutil.U32(0))[0],
						),
					).To(Equal(a))
					Expect(
						testutil.AsU64(
							call(ctx, "index_"+suffix, testutil.U32(h), testutil.U32(1))[0],
						),
					).To(Equal(b))

					rh := callU32(
						ctx,
						"element_add_"+suffix,
						testutil.U32(h),
						testutil.U64(1),
					)
					Expect(rh).ToNot(BeZero())
					ser := MustBeOk(ss.Get(rh))
					Expect(ser.DataType).To(Equal(dt))

					_ = callU32(
						ctx,
						"element_sub_"+suffix,
						testutil.U32(h),
						testutil.U64(1),
					)
					_ = callU32(
						ctx,
						"element_mul_"+suffix,
						testutil.U32(h),
						testutil.U64(2),
					)
					_ = callU32(
						ctx,
						"element_div_"+suffix,
						testutil.U32(h),
						testutil.U64(2),
					)
					_ = callU32(
						ctx,
						"element_mod_"+suffix,
						testutil.U32(h),
						testutil.U64(2),
					)

					_ = callU32(
						ctx,
						"element_rsub_"+suffix,
						testutil.U64(10),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_rdiv_"+suffix,
						testutil.U64(10),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_rmod_"+suffix,
						testutil.U64(10),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_radd_"+suffix,
						testutil.U64(1),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_rmul_"+suffix,
						testutil.U64(2),
						testutil.U32(h),
					)
				},
			)

			It("Should perform series arithmetic", func(ctx SpecContext) {
				h1 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h1),
					testutil.U32(0),
					testutil.U64(a),
				)
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h1),
					testutil.U32(1),
					testutil.U64(b),
				)
				h2 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h2),
					testutil.U32(0),
					testutil.U64(b),
				)
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h2),
					testutil.U32(1),
					testutil.U64(a),
				)

				for _, name := range []string{
					"series_add_", "series_sub_", "series_mul_",
					"series_div_", "series_mod_",
				} {
					rh := callU32(ctx, name+suffix, testutil.U32(h1), testutil.U32(h2))
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should perform series and scalar comparisons", func(ctx SpecContext) {
				h1 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h1),
					testutil.U32(0),
					testutil.U64(a),
				)
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h1),
					testutil.U32(1),
					testutil.U64(b),
				)
				h2 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h2),
					testutil.U32(0),
					testutil.U64(b),
				)
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h2),
					testutil.U32(1),
					testutil.U64(a),
				)

				for _, name := range []string{
					"compare_gt_", "compare_lt_", "compare_ge_",
					"compare_le_", "compare_eq_", "compare_ne_",
				} {
					rh := callU32(ctx, name+suffix, testutil.U32(h1), testutil.U32(h2))
					Expect(rh).ToNot(BeZero())
					cmpSer := MustBeOk(ss.Get(rh))
					Expect(cmpSer.DataType).To(Equal(telem.BooleanT))
				}

				for _, name := range []string{
					"compare_gt_scalar_", "compare_lt_scalar_",
					"compare_ge_scalar_", "compare_le_scalar_",
					"compare_eq_scalar_", "compare_ne_scalar_",
				} {
					rh := callU32(ctx, name+suffix, testutil.U32(h1), testutil.U64(a))
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should return 0 for invalid handle", func(ctx SpecContext) {
				Expect(
					testutil.AsU64(
						call(ctx, "index_"+suffix, testutil.U32(9999), testutil.U32(0))[0],
					),
				).To(Equal(uint64(0)))
			})
		})
	}

	describeF32 := func() {
		Describe("f32", func() {
			It(
				"Should create, set, index, and perform element ops",
				func(ctx SpecContext) {
					h := callU32(ctx, "create_empty_f32", testutil.U32(3))
					Expect(h).ToNot(BeZero())
					callU32(
						ctx,
						"set_element_f32",
						testutil.U32(h),
						testutil.U32(0),
						testutil.F32(1.5),
					)
					callU32(
						ctx,
						"set_element_f32",
						testutil.U32(h),
						testutil.U32(1),
						testutil.F32(2.5),
					)
					Expect(
						testutil.AsF32(
							call(ctx, "index_f32", testutil.U32(h), testutil.U32(0))[0],
						),
					).To(Equal(float32(1.5)))
					Expect(
						testutil.AsF32(
							call(ctx, "index_f32", testutil.U32(h), testutil.U32(1))[0],
						),
					).To(Equal(float32(2.5)))

					rh := callU32(
						ctx,
						"element_add_f32",
						testutil.U32(h),
						testutil.F32(1.0),
					)
					Expect(rh).ToNot(BeZero())
					ser := MustBeOk(ss.Get(rh))
					Expect(ser.DataType).To(Equal(telem.Float32T))

					_ = callU32(
						ctx,
						"element_sub_f32",
						testutil.U32(h),
						testutil.F32(1.0),
					)
					_ = callU32(
						ctx,
						"element_mul_f32",
						testutil.U32(h),
						testutil.F32(2.0),
					)
					_ = callU32(
						ctx,
						"element_div_f32",
						testutil.U32(h),
						testutil.F32(2.0),
					)
					_ = callU32(
						ctx,
						"element_mod_f32",
						testutil.U32(h),
						testutil.F32(2.0),
					)

					_ = callU32(
						ctx,
						"element_rsub_f32",
						testutil.F32(10.0),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_rdiv_f32",
						testutil.F32(10.0),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_rmod_f32",
						testutil.F32(10.0),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_radd_f32",
						testutil.F32(1.0),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_rmul_f32",
						testutil.F32(2.0),
						testutil.U32(h),
					)
				},
			)

			It("Should perform series arithmetic", func(ctx SpecContext) {
				h1 := callU32(ctx, "create_empty_f32", testutil.U32(2))
				callU32(
					ctx,
					"set_element_f32",
					testutil.U32(h1),
					testutil.U32(0),
					testutil.F32(3.0),
				)
				callU32(
					ctx,
					"set_element_f32",
					testutil.U32(h1),
					testutil.U32(1),
					testutil.F32(4.0),
				)
				h2 := callU32(ctx, "create_empty_f32", testutil.U32(2))
				callU32(
					ctx,
					"set_element_f32",
					testutil.U32(h2),
					testutil.U32(0),
					testutil.F32(1.0),
				)
				callU32(
					ctx,
					"set_element_f32",
					testutil.U32(h2),
					testutil.U32(1),
					testutil.F32(2.0),
				)

				for _, name := range []string{
					"series_add_", "series_sub_", "series_mul_",
					"series_div_", "series_mod_",
				} {
					rh := callU32(ctx, name+"f32", testutil.U32(h1), testutil.U32(h2))
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should perform series and scalar comparisons", func(ctx SpecContext) {
				h1 := callU32(ctx, "create_empty_f32", testutil.U32(2))
				callU32(
					ctx,
					"set_element_f32",
					testutil.U32(h1),
					testutil.U32(0),
					testutil.F32(1.0),
				)
				callU32(
					ctx,
					"set_element_f32",
					testutil.U32(h1),
					testutil.U32(1),
					testutil.F32(5.0),
				)
				h2 := callU32(ctx, "create_empty_f32", testutil.U32(2))
				callU32(
					ctx,
					"set_element_f32",
					testutil.U32(h2),
					testutil.U32(0),
					testutil.F32(5.0),
				)
				callU32(
					ctx,
					"set_element_f32",
					testutil.U32(h2),
					testutil.U32(1),
					testutil.F32(1.0),
				)

				for _, name := range []string{
					"compare_gt_", "compare_lt_", "compare_ge_",
					"compare_le_", "compare_eq_", "compare_ne_",
				} {
					rh := callU32(ctx, name+"f32", testutil.U32(h1), testutil.U32(h2))
					Expect(rh).ToNot(BeZero())
					cmpSer := MustBeOk(ss.Get(rh))
					Expect(cmpSer.DataType).To(Equal(telem.BooleanT))
				}

				for _, name := range []string{
					"compare_gt_scalar_", "compare_lt_scalar_",
					"compare_ge_scalar_", "compare_le_scalar_",
					"compare_eq_scalar_", "compare_ne_scalar_",
				} {
					rh := callU32(ctx, name+"f32", testutil.U32(h1), testutil.F32(3.0))
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should return 0 for invalid handle", func(ctx SpecContext) {
				Expect(
					testutil.AsF32(
						call(ctx, "index_f32", testutil.U32(9999), testutil.U32(0))[0],
					),
				).To(Equal(float32(0)))
			})
		})
	}

	describeF64 := func() {
		Describe("f64", func() {
			It(
				"Should create, set, index, and perform element ops",
				func(ctx SpecContext) {
					h := callU32(ctx, "create_empty_f64", testutil.U32(3))
					Expect(h).ToNot(BeZero())
					callU32(
						ctx,
						"set_element_f64",
						testutil.U32(h),
						testutil.U32(0),
						testutil.F64(1.5),
					)
					callU32(
						ctx,
						"set_element_f64",
						testutil.U32(h),
						testutil.U32(1),
						testutil.F64(2.5),
					)
					Expect(
						testutil.AsF64(
							call(ctx, "index_f64", testutil.U32(h), testutil.U32(0))[0],
						),
					).To(Equal(1.5))
					Expect(
						testutil.AsF64(
							call(ctx, "index_f64", testutil.U32(h), testutil.U32(1))[0],
						),
					).To(Equal(2.5))

					rh := callU32(
						ctx,
						"element_add_f64",
						testutil.U32(h),
						testutil.F64(1.0),
					)
					Expect(rh).ToNot(BeZero())
					ser := MustBeOk(ss.Get(rh))
					Expect(ser.DataType).To(Equal(telem.Float64T))

					_ = callU32(
						ctx,
						"element_sub_f64",
						testutil.U32(h),
						testutil.F64(1.0),
					)
					_ = callU32(
						ctx,
						"element_mul_f64",
						testutil.U32(h),
						testutil.F64(2.0),
					)
					_ = callU32(
						ctx,
						"element_div_f64",
						testutil.U32(h),
						testutil.F64(2.0),
					)
					_ = callU32(
						ctx,
						"element_mod_f64",
						testutil.U32(h),
						testutil.F64(2.0),
					)

					_ = callU32(
						ctx,
						"element_rsub_f64",
						testutil.F64(10.0),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_rdiv_f64",
						testutil.F64(10.0),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_rmod_f64",
						testutil.F64(10.0),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_radd_f64",
						testutil.F64(1.0),
						testutil.U32(h),
					)
					_ = callU32(
						ctx,
						"element_rmul_f64",
						testutil.F64(2.0),
						testutil.U32(h),
					)
				},
			)

			It("Should perform series arithmetic", func(ctx SpecContext) {
				h1 := callU32(ctx, "create_empty_f64", testutil.U32(2))
				callU32(
					ctx,
					"set_element_f64",
					testutil.U32(h1),
					testutil.U32(0),
					testutil.F64(3.0),
				)
				callU32(
					ctx,
					"set_element_f64",
					testutil.U32(h1),
					testutil.U32(1),
					testutil.F64(4.0),
				)
				h2 := callU32(ctx, "create_empty_f64", testutil.U32(2))
				callU32(
					ctx,
					"set_element_f64",
					testutil.U32(h2),
					testutil.U32(0),
					testutil.F64(1.0),
				)
				callU32(
					ctx,
					"set_element_f64",
					testutil.U32(h2),
					testutil.U32(1),
					testutil.F64(2.0),
				)

				for _, name := range []string{
					"series_add_", "series_sub_", "series_mul_",
					"series_div_", "series_mod_",
				} {
					rh := callU32(ctx, name+"f64", testutil.U32(h1), testutil.U32(h2))
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should perform series and scalar comparisons", func(ctx SpecContext) {
				h1 := callU32(ctx, "create_empty_f64", testutil.U32(2))
				callU32(
					ctx,
					"set_element_f64",
					testutil.U32(h1),
					testutil.U32(0),
					testutil.F64(1.0),
				)
				callU32(
					ctx,
					"set_element_f64",
					testutil.U32(h1),
					testutil.U32(1),
					testutil.F64(5.0),
				)
				h2 := callU32(ctx, "create_empty_f64", testutil.U32(2))
				callU32(
					ctx,
					"set_element_f64",
					testutil.U32(h2),
					testutil.U32(0),
					testutil.F64(5.0),
				)
				callU32(
					ctx,
					"set_element_f64",
					testutil.U32(h2),
					testutil.U32(1),
					testutil.F64(1.0),
				)

				for _, name := range []string{
					"compare_gt_", "compare_lt_", "compare_ge_",
					"compare_le_", "compare_eq_", "compare_ne_",
				} {
					rh := callU32(ctx, name+"f64", testutil.U32(h1), testutil.U32(h2))
					Expect(rh).ToNot(BeZero())
					cmpSer := MustBeOk(ss.Get(rh))
					Expect(cmpSer.DataType).To(Equal(telem.BooleanT))
				}

				for _, name := range []string{
					"compare_gt_scalar_", "compare_lt_scalar_",
					"compare_ge_scalar_", "compare_le_scalar_",
					"compare_eq_scalar_", "compare_ne_scalar_",
				} {
					rh := callU32(ctx, name+"f64", testutil.U32(h1), testutil.F64(3.0))
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should return 0 for invalid handle", func(ctx SpecContext) {
				Expect(
					testutil.AsF64(
						call(ctx, "index_f64", testutil.U32(9999), testutil.U32(0))[0],
					),
				).To(Equal(float64(0)))
			})
		})
	}

	describeI32Type("u8", telem.Uint8T, 3, 5)
	describeI32Type("u16", telem.Uint16T, 100, 200)
	describeI32Type("u32", telem.Uint32T, 1000, 2000)
	describeI32Type("i8", telem.Int8T, 3, 5)
	describeI32Type("i16", telem.Int16T, 100, 200)
	describeI32Type("i32", telem.Int32T, 1000, 2000)
	describeI64Type("u64", telem.Uint64T, 1000, 2000)
	describeI64Type("i64", telem.Int64T, 100, 200)
	describeF32()
	describeF64()

	Describe("negate", func() {
		It("Should negate signed i32 types", func(ctx SpecContext) {
			for _, suffix := range []string{"i8", "i16", "i32"} {
				h := callU32(ctx, "create_empty_"+suffix, testutil.U32(1))
				callU32(
					ctx,
					"set_element_"+suffix,
					testutil.U32(h),
					testutil.U32(0),
					testutil.U32(5),
				)
				rh := callU32(ctx, "negate_"+suffix, testutil.U32(h))
				Expect(rh).ToNot(BeZero())
			}
		})

		It("Should negate signed i64 types", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_i64", testutil.U32(1))
			callU32(
				ctx,
				"set_element_i64",
				testutil.U32(h),
				testutil.U32(0),
				testutil.U64(5),
			)
			rh := callU32(ctx, "negate_i64", testutil.U32(h))
			Expect(rh).ToNot(BeZero())
		})

		It("Should negate f32", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_f32", testutil.U32(1))
			callU32(
				ctx,
				"set_element_f32",
				testutil.U32(h),
				testutil.U32(0),
				testutil.F32(3.14),
			)
			rh := callU32(ctx, "negate_f32", testutil.U32(h))
			Expect(rh).ToNot(BeZero())
			ser := MustBeOk(ss.Get(rh))
			Expect(ser.ValueAt[float32](0)).To(BeNumerically("~", -3.14, 0.001))
		})

		It("Should negate f64", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_f64", testutil.U32(1))
			callU32(
				ctx,
				"set_element_f64",
				testutil.U32(h),
				testutil.U32(0),
				testutil.F64(3.14),
			)
			rh := callU32(ctx, "negate_f64", testutil.U32(h))
			Expect(rh).ToNot(BeZero())
			ser := MustBeOk(ss.Get(rh))
			Expect(ser.ValueAt[float64](0)).To(BeNumerically("~", -3.14, 0.001))
		})
	})

	Describe("index_bool", func() {
		It("Should read bool elements by index", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_bool", testutil.U32(2))
			callU32(
				ctx,
				"set_element_bool",
				testutil.U32(h),
				testutil.U32(0),
				testutil.U32(1),
			)
			callU32(
				ctx,
				"set_element_bool",
				testutil.U32(h),
				testutil.U32(1),
				testutil.U32(0),
			)
			Expect(
				callU32(ctx, "index_bool", testutil.U32(h), testutil.U32(0)),
			).To(Equal(uint32(1)))
			Expect(
				callU32(ctx, "index_bool", testutil.U32(h), testutil.U32(1)),
			).To(Equal(uint32(0)))
		})

		It("Should return 0 for an invalid handle", func(ctx SpecContext) {
			Expect(
				callU32(ctx, "index_bool", testutil.U32(9999), testutil.U32(0)),
			).To(BeZero())
		})
	})

	Describe("NewSymbols", func() {
		It("Should register the logical host funcs", func() {
			mod := series.NewSymbols()[0]
			names := make([]string, 0, len(mod.Children()))
			for _, child := range mod.Children() {
				names = append(names, child.Name)
			}
			Expect(names).To(ContainElements(
				"and", "or", "and_scalar", "or_scalar", "not",
			))
		})
	})

	Describe("not", func() {
		It("Should logically negate a bool series", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_bool", testutil.U32(2))
			callU32(
				ctx,
				"set_element_bool",
				testutil.U32(h),
				testutil.U32(0),
				testutil.U32(0),
			)
			callU32(
				ctx,
				"set_element_bool",
				testutil.U32(h),
				testutil.U32(1),
				testutil.U32(1),
			)
			rh := callU32(ctx, "not", testutil.U32(h))
			Expect(rh).ToNot(BeZero())
			ser := MustBeOk(ss.Get(rh))
			Expect(ser.ValueAt[bool](0)).To(BeTrue())
			Expect(ser.ValueAt[bool](1)).To(BeFalse())
		})
	})

	Describe("logical and/or", func() {
		mkBool := func(ctx SpecContext, vals ...uint32) uint32 {
			h := callU32(ctx, "create_empty_bool", testutil.U32(uint32(len(vals))))
			for i, v := range vals {
				callU32(
					ctx,
					"set_element_bool",
					testutil.U32(h),
					testutil.U32(uint32(i)),
					testutil.U32(v),
				)
			}
			return h
		}

		DescribeTable("series-series",
			func(ctx SpecContext, fn string, lhs, rhs, expected []uint32) {
				out := callU32(
					ctx,
					fn,
					testutil.U32(mkBool(ctx, lhs...)),
					testutil.U32(mkBool(ctx, rhs...)),
				)
				ser := MustBeOk(ss.Get(out))
				for i, e := range expected {
					Expect(ser.ValueAt[bool](i)).To(Equal(e != 0), "index %d", i)
				}
			},
			Entry("and truth table", "and",
				[]uint32{0, 0, 1, 1}, []uint32{0, 1, 0, 1}, []uint32{0, 0, 0, 1}),
			Entry("or truth table", "or",
				[]uint32{0, 0, 1, 1}, []uint32{0, 1, 0, 1}, []uint32{0, 1, 1, 1}),
		)

		DescribeTable(
			"series-scalar",
			func(ctx SpecContext, fn string, series []uint32, scalar uint32, expected []uint32) {
				out := callU32(
					ctx,
					fn,
					testutil.U32(mkBool(ctx, series...)),
					testutil.U32(scalar),
				)
				ser := MustBeOk(ss.Get(out))
				for i, e := range expected {
					Expect(ser.ValueAt[bool](i)).To(Equal(e != 0), "index %d", i)
				}
			},
			Entry("and true is identity", "and_scalar",
				[]uint32{0, 1}, uint32(1), []uint32{0, 1}),
			Entry("and false zeroes", "and_scalar",
				[]uint32{0, 1}, uint32(0), []uint32{0, 0}),
			Entry("or true fills", "or_scalar",
				[]uint32{0, 1}, uint32(1), []uint32{1, 1}),
			Entry("or false is identity", "or_scalar",
				[]uint32{0, 1}, uint32(0), []uint32{0, 1}),
		)

		DescribeTable("invalid handles return zero",
			func(ctx SpecContext, fn string) {
				Expect(
					callU32(ctx, fn, testutil.U32(9999), testutil.U32(9998)),
				).To(BeZero())
			},
			Entry("and", "and"),
			Entry("or", "or"),
			Entry("and_scalar", "and_scalar"),
			Entry("or_scalar", "or_scalar"),
		)
	})

	Describe("element index", func() {
		DescribeTable(
			"a negative index counts from the end",
			func(ctx SpecContext, suffix string, first, middle, last uint64) {
				h := testutil.U32(callU32(ctx, "create_empty_"+suffix, testutil.U32(3)))
				call(ctx, "set_element_"+suffix, h, testutil.I32(-3), first)
				call(ctx, "set_element_"+suffix, h, testutil.I32(-2), middle)
				call(ctx, "set_element_"+suffix, h, testutil.I32(-1), last)
				for index, expected := range []uint64{first, middle, last} {
					Expect(
						call(ctx, "index_"+suffix, h, testutil.I32(int32(index)))[0],
					).To(Equal(expected))
					Expect(
						call(ctx, "index_"+suffix, h, testutil.I32(int32(index-3)))[0],
					).To(Equal(expected))
				}
			},
			Entry("i32", "i32", testutil.U32(7), testutil.U32(8), testutil.U32(9)),
			Entry("i64", "i64", testutil.U64(7), testutil.U64(8), testutil.U64(9)),
			Entry(
				"f64",
				"f64",
				testutil.F64(1.5),
				testutil.F64(2.5),
				testutil.F64(3.5),
			),
			Entry("bool", "bool", testutil.U32(1), testutil.U32(0), testutil.U32(1)),
		)

		DescribeTable("an out-of-range index panics",
			func(ctx SpecContext, suffix string, index int32) {
				h := testutil.U32(callU32(ctx, "create_empty_"+suffix, testutil.U32(3)))
				message := fmt.Sprintf(
					"index %d out of bounds for series of size 3",
					index,
				)
				Expect(func() {
					call(ctx, "index_"+suffix, h, testutil.I32(index))
				}).To(PanicWith(ContainSubstring(message)))
				Expect(func() {
					call(ctx, "set_element_"+suffix, h, testutil.I32(index), 1)
				}).To(PanicWith(ContainSubstring(message)))
			},
			Entry("i32 past the end", "i32", int32(3)),
			Entry("i32 before the start", "i32", int32(-4)),
			Entry("i64 past the end", "i64", int32(3)),
			Entry("i64 before the start", "i64", int32(-4)),
			Entry("f64 past the end", "f64", int32(3)),
			Entry("f64 before the start", "f64", int32(-4)),
			Entry("bool past the end", "bool", int32(3)),
			Entry("bool before the start", "bool", int32(-4)),
		)
	})

	Describe("series-series length mismatch", func() {
		DescribeTable("panics",
			func(ctx SpecContext, create, fn string) {
				h1 := callU32(ctx, create, testutil.U32(2))
				h2 := callU32(ctx, create, testutil.U32(3))
				Expect(func() {
					call(ctx, fn, testutil.U32(h1), testutil.U32(h2))
				}).To(PanicWith(ContainSubstring("length mismatch")))
			},
			Entry("arithmetic", "create_empty_i32", "series_add_i32"),
			Entry("comparison", "create_empty_i32", "compare_gt_i32"),
			Entry("logical and", "create_empty_bool", "and"),
			Entry("logical or", "create_empty_bool", "or"),
		)
	})

	Describe("create_empty_str", func() {
		It("Should create an empty string series", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_str", testutil.U32(0))
			Expect(h).ToNot(BeZero())
			ser := MustBeOk(ss.Get(h))
			Expect(ser.DataType).To(Equal(telem.StringT))
			Expect(ser.Len()).To(BeZero())
		})

		It("Should ignore the requested length", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_str", testutil.U32(3))
			res := call(ctx, "len", testutil.U32(h))
			Expect(testutil.AsU64(res[0])).To(BeZero())
		})

		It("Should return a distinct handle on each call", func(ctx SpecContext) {
			h1 := callU32(ctx, "create_empty_str", testutil.U32(0))
			h2 := callU32(ctx, "create_empty_str", testutil.U32(0))
			Expect(h1).ToNot(Equal(h2))
		})

		It("Should return 0 when slicing the empty series", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_str", testutil.U32(0))
			Expect(
				callU32(
					ctx,
					"slice",
					testutil.U32(h),
					testutil.U32(0),
					testutil.U32(1),
				),
			).To(BeZero())
		})
	})

	Describe("len", func() {
		It("Should return the series length", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_i32", testutil.U32(5))
			res := call(ctx, "len", testutil.U32(h))
			Expect(testutil.AsU64(res[0])).To(Equal(uint64(5)))
		})

		It("Should return 0 for invalid handle", func(ctx SpecContext) {
			res := call(ctx, "len", testutil.U32(9999))
			Expect(testutil.AsU64(res[0])).To(Equal(uint64(0)))
		})
	})

	Describe("index_from", func() {
		DescribeTable("should return an index that fits in an i32",
			func(ctx SpecContext, fn string, index uint64, expected int32) {
				Expect(
					int32(testutil.AsU32(call(ctx, fn, index)[0])),
				).To(Equal(expected))
			},
			Entry("i64 zero", "index_from_i64", testutil.I64(0), int32(0)),
			Entry("i64 negative", "index_from_i64", testutil.I64(-1), int32(-1)),
			Entry(
				"i64 maximum",
				"index_from_i64",
				testutil.I64(math.MaxInt32),
				int32(math.MaxInt32),
			),
			Entry(
				"i64 minimum",
				"index_from_i64",
				testutil.I64(math.MinInt32),
				int32(math.MinInt32),
			),
			Entry(
				"u64 maximum",
				"index_from_u64",
				testutil.U64(math.MaxInt32),
				int32(math.MaxInt32),
			),
		)

		DescribeTable("should panic for an index that does not fit in an i32",
			func(ctx SpecContext, fn string, index uint64, text string) {
				Expect(func() { call(ctx, fn, index) }).To(PanicWith(ContainSubstring(
					"index " + text + " out of bounds for an i32 series index",
				)))
			},
			Entry(
				"i64 above the range",
				"index_from_i64",
				testutil.I64(math.MaxInt32+1),
				"2147483648",
			),
			Entry(
				"i64 below the range",
				"index_from_i64",
				testutil.I64(math.MinInt32-1),
				"-2147483649",
			),
			Entry(
				"u64 above the range",
				"index_from_u64",
				testutil.U64(math.MaxInt32+1),
				"2147483648",
			),
			Entry(
				"u64 maximum",
				"index_from_u64",
				testutil.U64(math.MaxUint64),
				"18446744073709551615",
			),
		)
	})

	Describe("slice", func() {
		It("Should count the data of a slice toward MaxBytes", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_f64", testutil.U32(1<<17))
			Expect(func() {
				for range series.MaxBytes >> 20 {
					callU32(
						ctx,
						"slice",
						testutil.U32(h),
						testutil.U32(0),
						testutil.U32(1<<17),
					)
				}
			}).To(PanicWith(ContainSubstring(series.ErrLimit.Error())))
		})

		It("Should keep the parent on a write to a slice", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_i32", testutil.U32(4))
			rh := callU32(
				ctx,
				"slice",
				testutil.U32(h),
				testutil.U32(1),
				testutil.U32(3),
			)
			callU32(
				ctx,
				"set_element_i32",
				testutil.U32(rh),
				testutil.U32(0),
				testutil.U32(9),
			)
			parent := MustBeOk(ss.Get(h))
			Expect(parent.ValueAt[int32](1)).To(BeZero())
		})

		It("Should keep a slice on a write to its parent", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_i32", testutil.U32(4))
			rh := callU32(
				ctx,
				"slice",
				testutil.U32(h),
				testutil.U32(1),
				testutil.U32(3),
			)
			callU32(
				ctx,
				"set_element_i32",
				testutil.U32(h),
				testutil.U32(2),
				testutil.U32(7),
			)
			sliced := MustBeOk(ss.Get(rh))
			Expect(sliced.ValueAt[int32](1)).To(BeZero())
		})

		It("Should extract a subrange", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_i32", testutil.U32(4))
			callU32(
				ctx,
				"set_element_i32",
				testutil.U32(h),
				testutil.U32(0),
				testutil.U32(10),
			)
			callU32(
				ctx,
				"set_element_i32",
				testutil.U32(h),
				testutil.U32(1),
				testutil.U32(20),
			)
			callU32(
				ctx,
				"set_element_i32",
				testutil.U32(h),
				testutil.U32(2),
				testutil.U32(30),
			)
			callU32(
				ctx,
				"set_element_i32",
				testutil.U32(h),
				testutil.U32(3),
				testutil.U32(40),
			)

			rh := callU32(
				ctx,
				"slice",
				testutil.U32(h),
				testutil.U32(1),
				testutil.U32(3),
			)
			Expect(rh).ToNot(BeZero())
			ser := MustBeOk(ss.Get(rh))
			Expect(ser.Len()).To(Equal(int64(2)))
			Expect(ser.ValueAt[int32](0)).To(Equal(int32(20)))
			Expect(ser.ValueAt[int32](1)).To(Equal(int32(30)))
		})

		It("Should return 0 for invalid handle", func(ctx SpecContext) {
			Expect(
				callU32(
					ctx,
					"slice",
					testutil.U32(9999),
					testutil.U32(0),
					testutil.U32(1),
				),
			).To(Equal(uint32(0)))
		})

		It("Should return 0 for empty range", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_i32", testutil.U32(4))
			Expect(
				callU32(
					ctx,
					"slice",
					testutil.U32(h),
					testutil.U32(2),
					testutil.U32(2),
				),
			).To(Equal(uint32(0)))
		})

		DescribeTable("Should count a negative bound from the end and stop at the ends",
			func(ctx SpecContext, start, end int, expected []int32) {
				h := callU32(ctx, "create_empty_i32", testutil.U32(4))
				for i := range uint32(4) {
					callU32(
						ctx,
						"set_element_i32",
						testutil.U32(h),
						testutil.U32(i),
						testutil.U32((i+1)*10),
					)
				}
				rh := callU32(
					ctx,
					"slice",
					testutil.U32(h),
					testutil.U32(uint32(int32(start))),
					testutil.U32(uint32(int32(end))),
				)
				if len(expected) == 0 {
					Expect(rh).To(BeZero())
					return
				}
				ser := MustBeOk(ss.Get(rh))
				Expect(ser.Len()).To(Equal(int64(len(expected))))
				for i, v := range expected {
					Expect(ser.ValueAt[int32](i)).To(Equal(v))
				}
			},
			Entry("an open end", 2, math.MaxInt32, []int32{30, 40}),
			Entry("an end past the length", 2, 10, []int32{30, 40}),
			Entry("a negative end", 0, -1, []int32{10, 20, 30}),
			Entry("a negative start", -2, math.MaxInt32, []int32{30, 40}),
			Entry("two negative bounds", -3, -1, []int32{20, 30}),
			Entry("a negative start before the series", -10, 2, []int32{10, 20}),
			Entry("a negative end before the series", 0, -10, nil),
			Entry("a start past the length", 4, math.MaxInt32, nil),
		)
	})
})
