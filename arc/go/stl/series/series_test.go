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
		MustSucceed(series.NewModule(ctx, ss, rt.Underlying()))
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
			It("Should create, set, index, and perform element ops", func(ctx SpecContext) {
				h := callU32(ctx, "create_empty_"+suffix, testutil.U32(3))
				Expect(h).ToNot(BeZero())
				callU32(ctx, "set_element_"+suffix, testutil.U32(h), testutil.U32(0), testutil.U32(a))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h), testutil.U32(1), testutil.U32(b))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h), testutil.U32(2), testutil.U32(a+b))
				Expect(callU32(ctx, "index_"+suffix, testutil.U32(h), testutil.U32(0))).To(Equal(a))
				Expect(callU32(ctx, "index_"+suffix, testutil.U32(h), testutil.U32(1))).To(Equal(b))
				Expect(callU32(ctx, "index_"+suffix, testutil.U32(h), testutil.U32(2))).To(Equal(a + b))

				rh := callU32(ctx, "element_add_"+suffix, testutil.U32(h), testutil.U32(1))
				Expect(rh).ToNot(BeZero())
				ser := MustBeOk(ss.Get(rh))
				Expect(ser.DataType).To(Equal(dt))

				_ = callU32(ctx, "element_sub_"+suffix, testutil.U32(h), testutil.U32(1))
				_ = callU32(ctx, "element_mul_"+suffix, testutil.U32(h), testutil.U32(2))
				_ = callU32(ctx, "element_div_"+suffix, testutil.U32(h), testutil.U32(2))
				_ = callU32(ctx, "element_mod_"+suffix, testutil.U32(h), testutil.U32(2))

				_ = callU32(ctx, "element_rsub_"+suffix, testutil.U32(10), testutil.U32(h))
				_ = callU32(ctx, "element_rdiv_"+suffix, testutil.U32(10), testutil.U32(h))
				_ = callU32(ctx, "element_rmod_"+suffix, testutil.U32(10), testutil.U32(h))
				_ = callU32(ctx, "element_radd_"+suffix, testutil.U32(1), testutil.U32(h))
				_ = callU32(ctx, "element_rmul_"+suffix, testutil.U32(2), testutil.U32(h))
			})

			It("Should perform series arithmetic", func(ctx SpecContext) {
				h1 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h1), testutil.U32(0), testutil.U32(a))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h1), testutil.U32(1), testutil.U32(b))
				h2 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h2), testutil.U32(0), testutil.U32(b))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h2), testutil.U32(1), testutil.U32(a))

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
				callU32(ctx, "set_element_"+suffix, testutil.U32(h1), testutil.U32(0), testutil.U32(a))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h1), testutil.U32(1), testutil.U32(b))
				h2 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h2), testutil.U32(0), testutil.U32(b))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h2), testutil.U32(1), testutil.U32(a))

				for _, name := range []string{
					"compare_gt_", "compare_lt_", "compare_ge_",
					"compare_le_", "compare_eq_", "compare_ne_",
				} {
					rh := callU32(ctx, name+suffix, testutil.U32(h1), testutil.U32(h2))
					Expect(rh).ToNot(BeZero())
					cmpSer := MustBeOk(ss.Get(rh))
					Expect(cmpSer.DataType).To(Equal(telem.Uint8T))
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
				Expect(callU32(ctx, "index_"+suffix, testutil.U32(9999), testutil.U32(0))).To(Equal(uint32(0)))
				Expect(callU32(ctx, "element_add_"+suffix, testutil.U32(9999), testutil.U32(1))).To(Equal(uint32(0)))
			})
		})
	}

	describeI64Type := func(suffix string, dt telem.DataType, a, b uint64) {
		Describe(suffix, func() {
			It("Should create, set, index, and perform element ops", func(ctx SpecContext) {
				h := callU32(ctx, "create_empty_"+suffix, testutil.U32(3))
				Expect(h).ToNot(BeZero())
				callU32(ctx, "set_element_"+suffix, testutil.U32(h), testutil.U32(0), testutil.U64(a))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h), testutil.U32(1), testutil.U64(b))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h), testutil.U32(2), testutil.U64(a+b))
				Expect(testutil.AsU64(call(ctx, "index_"+suffix, testutil.U32(h), testutil.U32(0))[0])).To(Equal(a))
				Expect(testutil.AsU64(call(ctx, "index_"+suffix, testutil.U32(h), testutil.U32(1))[0])).To(Equal(b))

				rh := callU32(ctx, "element_add_"+suffix, testutil.U32(h), testutil.U64(1))
				Expect(rh).ToNot(BeZero())
				ser := MustBeOk(ss.Get(rh))
				Expect(ser.DataType).To(Equal(dt))

				_ = callU32(ctx, "element_sub_"+suffix, testutil.U32(h), testutil.U64(1))
				_ = callU32(ctx, "element_mul_"+suffix, testutil.U32(h), testutil.U64(2))
				_ = callU32(ctx, "element_div_"+suffix, testutil.U32(h), testutil.U64(2))
				_ = callU32(ctx, "element_mod_"+suffix, testutil.U32(h), testutil.U64(2))

				_ = callU32(ctx, "element_rsub_"+suffix, testutil.U64(10), testutil.U32(h))
				_ = callU32(ctx, "element_rdiv_"+suffix, testutil.U64(10), testutil.U32(h))
				_ = callU32(ctx, "element_rmod_"+suffix, testutil.U64(10), testutil.U32(h))
				_ = callU32(ctx, "element_radd_"+suffix, testutil.U64(1), testutil.U32(h))
				_ = callU32(ctx, "element_rmul_"+suffix, testutil.U64(2), testutil.U32(h))
			})

			It("Should perform series arithmetic", func(ctx SpecContext) {
				h1 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h1), testutil.U32(0), testutil.U64(a))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h1), testutil.U32(1), testutil.U64(b))
				h2 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h2), testutil.U32(0), testutil.U64(b))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h2), testutil.U32(1), testutil.U64(a))

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
				callU32(ctx, "set_element_"+suffix, testutil.U32(h1), testutil.U32(0), testutil.U64(a))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h1), testutil.U32(1), testutil.U64(b))
				h2 := callU32(ctx, "create_empty_"+suffix, testutil.U32(2))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h2), testutil.U32(0), testutil.U64(b))
				callU32(ctx, "set_element_"+suffix, testutil.U32(h2), testutil.U32(1), testutil.U64(a))

				for _, name := range []string{
					"compare_gt_", "compare_lt_", "compare_ge_",
					"compare_le_", "compare_eq_", "compare_ne_",
				} {
					rh := callU32(ctx, name+suffix, testutil.U32(h1), testutil.U32(h2))
					Expect(rh).ToNot(BeZero())
					cmpSer := MustBeOk(ss.Get(rh))
					Expect(cmpSer.DataType).To(Equal(telem.Uint8T))
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
				Expect(testutil.AsU64(call(ctx, "index_"+suffix, testutil.U32(9999), testutil.U32(0))[0])).To(Equal(uint64(0)))
			})
		})
	}

	describeF32 := func() {
		Describe("f32", func() {
			It("Should create, set, index, and perform element ops", func(ctx SpecContext) {
				h := callU32(ctx, "create_empty_f32", testutil.U32(3))
				Expect(h).ToNot(BeZero())
				callU32(ctx, "set_element_f32", testutil.U32(h), testutil.U32(0), testutil.F32(1.5))
				callU32(ctx, "set_element_f32", testutil.U32(h), testutil.U32(1), testutil.F32(2.5))
				Expect(testutil.AsF32(call(ctx, "index_f32", testutil.U32(h), testutil.U32(0))[0])).To(Equal(float32(1.5)))
				Expect(testutil.AsF32(call(ctx, "index_f32", testutil.U32(h), testutil.U32(1))[0])).To(Equal(float32(2.5)))

				rh := callU32(ctx, "element_add_f32", testutil.U32(h), testutil.F32(1.0))
				Expect(rh).ToNot(BeZero())
				ser := MustBeOk(ss.Get(rh))
				Expect(ser.DataType).To(Equal(telem.Float32T))

				_ = callU32(ctx, "element_sub_f32", testutil.U32(h), testutil.F32(1.0))
				_ = callU32(ctx, "element_mul_f32", testutil.U32(h), testutil.F32(2.0))
				_ = callU32(ctx, "element_div_f32", testutil.U32(h), testutil.F32(2.0))
				_ = callU32(ctx, "element_mod_f32", testutil.U32(h), testutil.F32(2.0))

				_ = callU32(ctx, "element_rsub_f32", testutil.F32(10.0), testutil.U32(h))
				_ = callU32(ctx, "element_rdiv_f32", testutil.F32(10.0), testutil.U32(h))
				_ = callU32(ctx, "element_rmod_f32", testutil.F32(10.0), testutil.U32(h))
				_ = callU32(ctx, "element_radd_f32", testutil.F32(1.0), testutil.U32(h))
				_ = callU32(ctx, "element_rmul_f32", testutil.F32(2.0), testutil.U32(h))
			})

			It("Should perform series arithmetic", func(ctx SpecContext) {
				h1 := callU32(ctx, "create_empty_f32", testutil.U32(2))
				callU32(ctx, "set_element_f32", testutil.U32(h1), testutil.U32(0), testutil.F32(3.0))
				callU32(ctx, "set_element_f32", testutil.U32(h1), testutil.U32(1), testutil.F32(4.0))
				h2 := callU32(ctx, "create_empty_f32", testutil.U32(2))
				callU32(ctx, "set_element_f32", testutil.U32(h2), testutil.U32(0), testutil.F32(1.0))
				callU32(ctx, "set_element_f32", testutil.U32(h2), testutil.U32(1), testutil.F32(2.0))

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
				callU32(ctx, "set_element_f32", testutil.U32(h1), testutil.U32(0), testutil.F32(1.0))
				callU32(ctx, "set_element_f32", testutil.U32(h1), testutil.U32(1), testutil.F32(5.0))
				h2 := callU32(ctx, "create_empty_f32", testutil.U32(2))
				callU32(ctx, "set_element_f32", testutil.U32(h2), testutil.U32(0), testutil.F32(5.0))
				callU32(ctx, "set_element_f32", testutil.U32(h2), testutil.U32(1), testutil.F32(1.0))

				for _, name := range []string{
					"compare_gt_", "compare_lt_", "compare_ge_",
					"compare_le_", "compare_eq_", "compare_ne_",
				} {
					rh := callU32(ctx, name+"f32", testutil.U32(h1), testutil.U32(h2))
					Expect(rh).ToNot(BeZero())
					cmpSer := MustBeOk(ss.Get(rh))
					Expect(cmpSer.DataType).To(Equal(telem.Uint8T))
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
				Expect(testutil.AsF32(call(ctx, "index_f32", testutil.U32(9999), testutil.U32(0))[0])).To(Equal(float32(0)))
			})
		})
	}

	describeF64 := func() {
		Describe("f64", func() {
			It("Should create, set, index, and perform element ops", func(ctx SpecContext) {
				h := callU32(ctx, "create_empty_f64", testutil.U32(3))
				Expect(h).ToNot(BeZero())
				callU32(ctx, "set_element_f64", testutil.U32(h), testutil.U32(0), testutil.F64(1.5))
				callU32(ctx, "set_element_f64", testutil.U32(h), testutil.U32(1), testutil.F64(2.5))
				Expect(testutil.AsF64(call(ctx, "index_f64", testutil.U32(h), testutil.U32(0))[0])).To(Equal(1.5))
				Expect(testutil.AsF64(call(ctx, "index_f64", testutil.U32(h), testutil.U32(1))[0])).To(Equal(2.5))

				rh := callU32(ctx, "element_add_f64", testutil.U32(h), testutil.F64(1.0))
				Expect(rh).ToNot(BeZero())
				ser := MustBeOk(ss.Get(rh))
				Expect(ser.DataType).To(Equal(telem.Float64T))

				_ = callU32(ctx, "element_sub_f64", testutil.U32(h), testutil.F64(1.0))
				_ = callU32(ctx, "element_mul_f64", testutil.U32(h), testutil.F64(2.0))
				_ = callU32(ctx, "element_div_f64", testutil.U32(h), testutil.F64(2.0))
				_ = callU32(ctx, "element_mod_f64", testutil.U32(h), testutil.F64(2.0))

				_ = callU32(ctx, "element_rsub_f64", testutil.F64(10.0), testutil.U32(h))
				_ = callU32(ctx, "element_rdiv_f64", testutil.F64(10.0), testutil.U32(h))
				_ = callU32(ctx, "element_rmod_f64", testutil.F64(10.0), testutil.U32(h))
				_ = callU32(ctx, "element_radd_f64", testutil.F64(1.0), testutil.U32(h))
				_ = callU32(ctx, "element_rmul_f64", testutil.F64(2.0), testutil.U32(h))
			})

			It("Should perform series arithmetic", func(ctx SpecContext) {
				h1 := callU32(ctx, "create_empty_f64", testutil.U32(2))
				callU32(ctx, "set_element_f64", testutil.U32(h1), testutil.U32(0), testutil.F64(3.0))
				callU32(ctx, "set_element_f64", testutil.U32(h1), testutil.U32(1), testutil.F64(4.0))
				h2 := callU32(ctx, "create_empty_f64", testutil.U32(2))
				callU32(ctx, "set_element_f64", testutil.U32(h2), testutil.U32(0), testutil.F64(1.0))
				callU32(ctx, "set_element_f64", testutil.U32(h2), testutil.U32(1), testutil.F64(2.0))

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
				callU32(ctx, "set_element_f64", testutil.U32(h1), testutil.U32(0), testutil.F64(1.0))
				callU32(ctx, "set_element_f64", testutil.U32(h1), testutil.U32(1), testutil.F64(5.0))
				h2 := callU32(ctx, "create_empty_f64", testutil.U32(2))
				callU32(ctx, "set_element_f64", testutil.U32(h2), testutil.U32(0), testutil.F64(5.0))
				callU32(ctx, "set_element_f64", testutil.U32(h2), testutil.U32(1), testutil.F64(1.0))

				for _, name := range []string{
					"compare_gt_", "compare_lt_", "compare_ge_",
					"compare_le_", "compare_eq_", "compare_ne_",
				} {
					rh := callU32(ctx, name+"f64", testutil.U32(h1), testutil.U32(h2))
					Expect(rh).ToNot(BeZero())
					cmpSer := MustBeOk(ss.Get(rh))
					Expect(cmpSer.DataType).To(Equal(telem.Uint8T))
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
				Expect(testutil.AsF64(call(ctx, "index_f64", testutil.U32(9999), testutil.U32(0))[0])).To(Equal(float64(0)))
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
				callU32(ctx, "set_element_"+suffix, testutil.U32(h), testutil.U32(0), testutil.U32(5))
				rh := callU32(ctx, "negate_"+suffix, testutil.U32(h))
				Expect(rh).ToNot(BeZero())
			}
		})

		It("Should negate signed i64 types", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_i64", testutil.U32(1))
			callU32(ctx, "set_element_i64", testutil.U32(h), testutil.U32(0), testutil.U64(5))
			rh := callU32(ctx, "negate_i64", testutil.U32(h))
			Expect(rh).ToNot(BeZero())
		})

		It("Should negate f32", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_f32", testutil.U32(1))
			callU32(ctx, "set_element_f32", testutil.U32(h), testutil.U32(0), testutil.F32(3.14))
			rh := callU32(ctx, "negate_f32", testutil.U32(h))
			Expect(rh).ToNot(BeZero())
			ser := MustBeOk(ss.Get(rh))
			Expect(telem.ValueAt[float32](ser, 0)).To(BeNumerically("~", -3.14, 0.001))
		})

		It("Should negate f64", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_f64", testutil.U32(1))
			callU32(ctx, "set_element_f64", testutil.U32(h), testutil.U32(0), testutil.F64(3.14))
			rh := callU32(ctx, "negate_f64", testutil.U32(h))
			Expect(rh).ToNot(BeZero())
			ser := MustBeOk(ss.Get(rh))
			Expect(telem.ValueAt[float64](ser, 0)).To(BeNumerically("~", -3.14, 0.001))
		})
	})

	Describe("not_u8", func() {
		It("Should bitwise-NOT a u8 series", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_u8", testutil.U32(2))
			callU32(ctx, "set_element_u8", testutil.U32(h), testutil.U32(0), testutil.U32(0))
			callU32(ctx, "set_element_u8", testutil.U32(h), testutil.U32(1), testutil.U32(255))
			rh := callU32(ctx, "not_u8", testutil.U32(h))
			Expect(rh).ToNot(BeZero())
			ser := MustBeOk(ss.Get(rh))
			Expect(telem.ValueAt[uint8](ser, 0)).To(Equal(uint8(255)))
			Expect(telem.ValueAt[uint8](ser, 1)).To(Equal(uint8(0)))
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

	Describe("slice", func() {
		It("Should extract a subrange", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_i32", testutil.U32(4))
			callU32(ctx, "set_element_i32", testutil.U32(h), testutil.U32(0), testutil.U32(10))
			callU32(ctx, "set_element_i32", testutil.U32(h), testutil.U32(1), testutil.U32(20))
			callU32(ctx, "set_element_i32", testutil.U32(h), testutil.U32(2), testutil.U32(30))
			callU32(ctx, "set_element_i32", testutil.U32(h), testutil.U32(3), testutil.U32(40))

			rh := callU32(ctx, "slice", testutil.U32(h), testutil.U32(1), testutil.U32(3))
			Expect(rh).ToNot(BeZero())
			ser := MustBeOk(ss.Get(rh))
			Expect(ser.Len()).To(Equal(int64(2)))
			Expect(telem.ValueAt[int32](ser, 0)).To(Equal(int32(20)))
			Expect(telem.ValueAt[int32](ser, 1)).To(Equal(int32(30)))
		})

		It("Should return 0 for invalid handle", func(ctx SpecContext) {
			Expect(callU32(ctx, "slice", testutil.U32(9999), testutil.U32(0), testutil.U32(1))).To(Equal(uint32(0)))
		})

		It("Should return 0 for empty range", func(ctx SpecContext) {
			h := callU32(ctx, "create_empty_i32", testutil.U32(4))
			Expect(callU32(ctx, "slice", testutil.U32(h), testutil.U32(2), testutil.U32(2))).To(Equal(uint32(0)))
		})
	})
})
