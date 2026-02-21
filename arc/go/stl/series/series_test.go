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
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/runtime/state"
	"github.com/synnaxlabs/arc/stl/series"
	"github.com/synnaxlabs/arc/stl/testutil"
	"github.com/synnaxlabs/x/telem"
	. "github.com/synnaxlabs/x/testutil"
)

type i32CreateEmpty = func(context.Context, uint32) uint32
type i32SetElement = func(context.Context, uint32, uint32, uint32) uint32
type i32Index = func(context.Context, uint32, uint32) uint32
type i32ElementOp = func(context.Context, uint32, uint32) uint32
type i32ReverseOp = func(context.Context, uint32, uint32) uint32
type seriesOp = func(context.Context, uint32, uint32) uint32
type negateOp = func(context.Context, uint32) uint32

type i64SetElement = func(context.Context, uint32, uint32, uint64) uint32
type i64Index = func(context.Context, uint32, uint32) uint64
type i64ElementOp = func(context.Context, uint32, uint64) uint32
type i64ReverseOp = func(context.Context, uint64, uint32) uint32
type i64ScalarCmp = func(context.Context, uint32, uint64) uint32

type f32SetElement = func(context.Context, uint32, uint32, float32) uint32
type f32Index = func(context.Context, uint32, uint32) float32
type f32ElementOp = func(context.Context, uint32, float32) uint32
type f32ReverseOp = func(context.Context, float32, uint32) uint32
type f32ScalarCmp = func(context.Context, uint32, float32) uint32

type f64SetElement = func(context.Context, uint32, uint32, float64) uint32
type f64Index = func(context.Context, uint32, uint32) float64
type f64ElementOp = func(context.Context, uint32, float64) uint32
type f64ReverseOp = func(context.Context, float64, uint32) uint32
type f64ScalarCmp = func(context.Context, uint32, float64) uint32

var ctx = context.Background()

var _ = Describe("Series", func() {
	var (
		rt *testutil.MockHostRuntime
		ss *state.SeriesHandleStore
	)

	BeforeEach(func() {
		rt = testutil.NewMockHostRuntime()
		ss = state.NewSeriesHandleStore()
		mod := series.NewModule(ss)
		Expect(mod.BindTo(ctx, rt)).To(Succeed())
	})

	describeI32Type := func(suffix string, dt telem.DataType, a, b uint32) {
		Describe(suffix, func() {
			It("Should create, set, index, and perform element ops", func() {
				create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_"+suffix)
				set := testutil.Get[i32SetElement](rt, "series", "set_element_"+suffix)
				index := testutil.Get[i32Index](rt, "series", "index_"+suffix)

				h := create(ctx, 3)
				Expect(h).ToNot(BeZero())
				set(ctx, h, 0, a)
				set(ctx, h, 1, b)
				set(ctx, h, 2, a+b)
				Expect(index(ctx, h, 0)).To(Equal(a))
				Expect(index(ctx, h, 1)).To(Equal(b))
				Expect(index(ctx, h, 2)).To(Equal(a + b))

				add := testutil.Get[i32ElementOp](rt, "series", "element_add_"+suffix)
				sub := testutil.Get[i32ElementOp](rt, "series", "element_sub_"+suffix)
				mul := testutil.Get[i32ElementOp](rt, "series", "element_mul_"+suffix)
				div := testutil.Get[i32ElementOp](rt, "series", "element_div_"+suffix)
				mod := testutil.Get[i32ElementOp](rt, "series", "element_mod_"+suffix)

				rh := add(ctx, h, 1)
				Expect(rh).ToNot(BeZero())
				ser := MustBeOk(ss.Get(rh))
				Expect(ser.DataType).To(Equal(dt))

				_ = sub(ctx, h, 1)
				_ = mul(ctx, h, 2)
				_ = div(ctx, h, 2)
				_ = mod(ctx, h, 2)

				rsub := testutil.Get[i32ReverseOp](rt, "series", "element_rsub_"+suffix)
				rdiv := testutil.Get[i32ReverseOp](rt, "series", "element_rdiv_"+suffix)
				rmod := testutil.Get[i32ReverseOp](rt, "series", "element_rmod_"+suffix)
				radd := testutil.Get[i32ReverseOp](rt, "series", "element_radd_"+suffix)
				rmul := testutil.Get[i32ReverseOp](rt, "series", "element_rmul_"+suffix)

				_ = rsub(ctx, 10, h)
				_ = rdiv(ctx, 10, h)
				_ = rmod(ctx, 10, h)
				_ = radd(ctx, 1, h)
				_ = rmul(ctx, 2, h)
			})

			It("Should perform series arithmetic", func() {
				create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_"+suffix)
				set := testutil.Get[i32SetElement](rt, "series", "set_element_"+suffix)

				h1 := create(ctx, 2)
				set(ctx, h1, 0, a)
				set(ctx, h1, 1, b)
				h2 := create(ctx, 2)
				set(ctx, h2, 0, b)
				set(ctx, h2, 1, a)

				for _, name := range []string{
					"series_add_", "series_sub_", "series_mul_",
					"series_div_", "series_mod_",
				} {
					fn := testutil.Get[seriesOp](rt, "series", name+suffix)
					rh := fn(ctx, h1, h2)
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should perform series and scalar comparisons", func() {
				create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_"+suffix)
				set := testutil.Get[i32SetElement](rt, "series", "set_element_"+suffix)
				index := testutil.Get[i32Index](rt, "series", "index_"+suffix)

				h1 := create(ctx, 2)
				set(ctx, h1, 0, a)
				set(ctx, h1, 1, b)
				h2 := create(ctx, 2)
				set(ctx, h2, 0, b)
				set(ctx, h2, 1, a)

				for _, name := range []string{
					"compare_gt_", "compare_lt_", "compare_ge_",
					"compare_le_", "compare_eq_", "compare_ne_",
				} {
					fn := testutil.Get[seriesOp](rt, "series", name+suffix)
					rh := fn(ctx, h1, h2)
					Expect(rh).ToNot(BeZero())
					cmpSer := MustBeOk(ss.Get(rh))
					Expect(cmpSer.DataType).To(Equal(telem.Uint8T))
				}

				for _, name := range []string{
					"compare_gt_scalar_", "compare_lt_scalar_",
					"compare_ge_scalar_", "compare_le_scalar_",
					"compare_eq_scalar_", "compare_ne_scalar_",
				} {
					fn := testutil.Get[i32ElementOp](rt, "series", name+suffix)
					rh := fn(ctx, h1, a)
					Expect(rh).ToNot(BeZero())
				}

				_ = index(ctx, h1, 0)
			})

			It("Should return 0 for invalid handle", func() {
				index := testutil.Get[i32Index](rt, "series", "index_"+suffix)
				Expect(index(ctx, 9999, 0)).To(Equal(uint32(0)))

				add := testutil.Get[i32ElementOp](rt, "series", "element_add_"+suffix)
				Expect(add(ctx, 9999, 1)).To(Equal(uint32(0)))
			})
		})
	}

	describeI64Type := func(suffix string, dt telem.DataType, a, b uint64) {
		Describe(suffix, func() {
			It("Should create, set, index, and perform element ops", func() {
				create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_"+suffix)
				set := testutil.Get[i64SetElement](rt, "series", "set_element_"+suffix)
				index := testutil.Get[i64Index](rt, "series", "index_"+suffix)

				h := create(ctx, 3)
				Expect(h).ToNot(BeZero())
				set(ctx, h, 0, a)
				set(ctx, h, 1, b)
				set(ctx, h, 2, a+b)
				Expect(index(ctx, h, 0)).To(Equal(a))
				Expect(index(ctx, h, 1)).To(Equal(b))

				add := testutil.Get[i64ElementOp](rt, "series", "element_add_"+suffix)
				rh := add(ctx, h, 1)
				Expect(rh).ToNot(BeZero())
				ser := MustBeOk(ss.Get(rh))
				Expect(ser.DataType).To(Equal(dt))

				sub := testutil.Get[i64ElementOp](rt, "series", "element_sub_"+suffix)
				_ = sub(ctx, h, 1)
				mul := testutil.Get[i64ElementOp](rt, "series", "element_mul_"+suffix)
				_ = mul(ctx, h, 2)
				div := testutil.Get[i64ElementOp](rt, "series", "element_div_"+suffix)
				_ = div(ctx, h, 2)
				mod := testutil.Get[i64ElementOp](rt, "series", "element_mod_"+suffix)
				_ = mod(ctx, h, 2)

				rsub := testutil.Get[i64ReverseOp](rt, "series", "element_rsub_"+suffix)
				_ = rsub(ctx, 10, h)
				rdiv := testutil.Get[i64ReverseOp](rt, "series", "element_rdiv_"+suffix)
				_ = rdiv(ctx, 10, h)
				rmod := testutil.Get[i64ReverseOp](rt, "series", "element_rmod_"+suffix)
				_ = rmod(ctx, 10, h)
				radd := testutil.Get[i64ReverseOp](rt, "series", "element_radd_"+suffix)
				_ = radd(ctx, 1, h)
				rmul := testutil.Get[i64ReverseOp](rt, "series", "element_rmul_"+suffix)
				_ = rmul(ctx, 2, h)
			})

			It("Should perform series arithmetic", func() {
				create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_"+suffix)
				set := testutil.Get[i64SetElement](rt, "series", "set_element_"+suffix)

				h1 := create(ctx, 2)
				set(ctx, h1, 0, a)
				set(ctx, h1, 1, b)
				h2 := create(ctx, 2)
				set(ctx, h2, 0, b)
				set(ctx, h2, 1, a)

				for _, name := range []string{
					"series_add_", "series_sub_", "series_mul_",
					"series_div_", "series_mod_",
				} {
					fn := testutil.Get[seriesOp](rt, "series", name+suffix)
					rh := fn(ctx, h1, h2)
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should perform series and scalar comparisons", func() {
				create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_"+suffix)
				set := testutil.Get[i64SetElement](rt, "series", "set_element_"+suffix)

				h1 := create(ctx, 2)
				set(ctx, h1, 0, a)
				set(ctx, h1, 1, b)
				h2 := create(ctx, 2)
				set(ctx, h2, 0, b)
				set(ctx, h2, 1, a)

				for _, name := range []string{
					"compare_gt_", "compare_lt_", "compare_ge_",
					"compare_le_", "compare_eq_", "compare_ne_",
				} {
					fn := testutil.Get[seriesOp](rt, "series", name+suffix)
					rh := fn(ctx, h1, h2)
					Expect(rh).ToNot(BeZero())
					cmpSer := MustBeOk(ss.Get(rh))
					Expect(cmpSer.DataType).To(Equal(telem.Uint8T))
				}

				for _, name := range []string{
					"compare_gt_scalar_", "compare_lt_scalar_",
					"compare_ge_scalar_", "compare_le_scalar_",
					"compare_eq_scalar_", "compare_ne_scalar_",
				} {
					fn := testutil.Get[i64ScalarCmp](rt, "series", name+suffix)
					rh := fn(ctx, h1, a)
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should return 0 for invalid handle", func() {
				index := testutil.Get[i64Index](rt, "series", "index_"+suffix)
				Expect(index(ctx, 9999, 0)).To(Equal(uint64(0)))
			})
		})
	}

	describeF32 := func() {
		Describe("f32", func() {
			It("Should create, set, index, and perform element ops", func() {
				create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_f32")
				set := testutil.Get[f32SetElement](rt, "series", "set_element_f32")
				index := testutil.Get[f32Index](rt, "series", "index_f32")

				h := create(ctx, 3)
				Expect(h).ToNot(BeZero())
				set(ctx, h, 0, 1.5)
				set(ctx, h, 1, 2.5)
				Expect(index(ctx, h, 0)).To(Equal(float32(1.5)))
				Expect(index(ctx, h, 1)).To(Equal(float32(2.5)))

				add := testutil.Get[f32ElementOp](rt, "series", "element_add_f32")
				rh := add(ctx, h, 1.0)
				Expect(rh).ToNot(BeZero())
				ser := MustBeOk(ss.Get(rh))
				Expect(ser.DataType).To(Equal(telem.Float32T))

				sub := testutil.Get[f32ElementOp](rt, "series", "element_sub_f32")
				_ = sub(ctx, h, 1.0)
				mul := testutil.Get[f32ElementOp](rt, "series", "element_mul_f32")
				_ = mul(ctx, h, 2.0)
				div := testutil.Get[f32ElementOp](rt, "series", "element_div_f32")
				_ = div(ctx, h, 2.0)
				mod := testutil.Get[f32ElementOp](rt, "series", "element_mod_f32")
				_ = mod(ctx, h, 2.0)

				rsub := testutil.Get[f32ReverseOp](rt, "series", "element_rsub_f32")
				_ = rsub(ctx, 10.0, h)
				rdiv := testutil.Get[f32ReverseOp](rt, "series", "element_rdiv_f32")
				_ = rdiv(ctx, 10.0, h)
				rmod := testutil.Get[f32ReverseOp](rt, "series", "element_rmod_f32")
				_ = rmod(ctx, 10.0, h)
				radd := testutil.Get[f32ReverseOp](rt, "series", "element_radd_f32")
				_ = radd(ctx, 1.0, h)
				rmul := testutil.Get[f32ReverseOp](rt, "series", "element_rmul_f32")
				_ = rmul(ctx, 2.0, h)
			})

			It("Should perform series arithmetic", func() {
				create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_f32")
				set := testutil.Get[f32SetElement](rt, "series", "set_element_f32")

				h1 := create(ctx, 2)
				set(ctx, h1, 0, 3.0)
				set(ctx, h1, 1, 4.0)
				h2 := create(ctx, 2)
				set(ctx, h2, 0, 1.0)
				set(ctx, h2, 1, 2.0)

				for _, name := range []string{
					"series_add_", "series_sub_", "series_mul_",
					"series_div_", "series_mod_",
				} {
					fn := testutil.Get[seriesOp](rt, "series", name+"f32")
					rh := fn(ctx, h1, h2)
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should perform series and scalar comparisons", func() {
				create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_f32")
				set := testutil.Get[f32SetElement](rt, "series", "set_element_f32")

				h1 := create(ctx, 2)
				set(ctx, h1, 0, 1.0)
				set(ctx, h1, 1, 5.0)
				h2 := create(ctx, 2)
				set(ctx, h2, 0, 5.0)
				set(ctx, h2, 1, 1.0)

				for _, name := range []string{
					"compare_gt_", "compare_lt_", "compare_ge_",
					"compare_le_", "compare_eq_", "compare_ne_",
				} {
					fn := testutil.Get[seriesOp](rt, "series", name+"f32")
					rh := fn(ctx, h1, h2)
					Expect(rh).ToNot(BeZero())
					cmpSer := MustBeOk(ss.Get(rh))
					Expect(cmpSer.DataType).To(Equal(telem.Uint8T))
				}

				for _, name := range []string{
					"compare_gt_scalar_", "compare_lt_scalar_",
					"compare_ge_scalar_", "compare_le_scalar_",
					"compare_eq_scalar_", "compare_ne_scalar_",
				} {
					fn := testutil.Get[f32ScalarCmp](rt, "series", name+"f32")
					rh := fn(ctx, h1, float32(3.0))
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should return 0 for invalid handle", func() {
				index := testutil.Get[f32Index](rt, "series", "index_f32")
				Expect(index(ctx, 9999, 0)).To(Equal(float32(0)))
			})
		})
	}

	describeF64 := func() {
		Describe("f64", func() {
			It("Should create, set, index, and perform element ops", func() {
				create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_f64")
				set := testutil.Get[f64SetElement](rt, "series", "set_element_f64")
				index := testutil.Get[f64Index](rt, "series", "index_f64")

				h := create(ctx, 3)
				Expect(h).ToNot(BeZero())
				set(ctx, h, 0, 1.5)
				set(ctx, h, 1, 2.5)
				Expect(index(ctx, h, 0)).To(Equal(1.5))
				Expect(index(ctx, h, 1)).To(Equal(2.5))

				add := testutil.Get[f64ElementOp](rt, "series", "element_add_f64")
				rh := add(ctx, h, 1.0)
				Expect(rh).ToNot(BeZero())
				ser := MustBeOk(ss.Get(rh))
				Expect(ser.DataType).To(Equal(telem.Float64T))

				sub := testutil.Get[f64ElementOp](rt, "series", "element_sub_f64")
				_ = sub(ctx, h, 1.0)
				mul := testutil.Get[f64ElementOp](rt, "series", "element_mul_f64")
				_ = mul(ctx, h, 2.0)
				div := testutil.Get[f64ElementOp](rt, "series", "element_div_f64")
				_ = div(ctx, h, 2.0)
				mod := testutil.Get[f64ElementOp](rt, "series", "element_mod_f64")
				_ = mod(ctx, h, 2.0)

				rsub := testutil.Get[f64ReverseOp](rt, "series", "element_rsub_f64")
				_ = rsub(ctx, 10.0, h)
				rdiv := testutil.Get[f64ReverseOp](rt, "series", "element_rdiv_f64")
				_ = rdiv(ctx, 10.0, h)
				rmod := testutil.Get[f64ReverseOp](rt, "series", "element_rmod_f64")
				_ = rmod(ctx, 10.0, h)
				radd := testutil.Get[f64ReverseOp](rt, "series", "element_radd_f64")
				_ = radd(ctx, 1.0, h)
				rmul := testutil.Get[f64ReverseOp](rt, "series", "element_rmul_f64")
				_ = rmul(ctx, 2.0, h)
			})

			It("Should perform series arithmetic", func() {
				create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_f64")
				set := testutil.Get[f64SetElement](rt, "series", "set_element_f64")

				h1 := create(ctx, 2)
				set(ctx, h1, 0, 3.0)
				set(ctx, h1, 1, 4.0)
				h2 := create(ctx, 2)
				set(ctx, h2, 0, 1.0)
				set(ctx, h2, 1, 2.0)

				for _, name := range []string{
					"series_add_", "series_sub_", "series_mul_",
					"series_div_", "series_mod_",
				} {
					fn := testutil.Get[seriesOp](rt, "series", name+"f64")
					rh := fn(ctx, h1, h2)
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should perform series and scalar comparisons", func() {
				create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_f64")
				set := testutil.Get[f64SetElement](rt, "series", "set_element_f64")

				h1 := create(ctx, 2)
				set(ctx, h1, 0, 1.0)
				set(ctx, h1, 1, 5.0)
				h2 := create(ctx, 2)
				set(ctx, h2, 0, 5.0)
				set(ctx, h2, 1, 1.0)

				for _, name := range []string{
					"compare_gt_", "compare_lt_", "compare_ge_",
					"compare_le_", "compare_eq_", "compare_ne_",
				} {
					fn := testutil.Get[seriesOp](rt, "series", name+"f64")
					rh := fn(ctx, h1, h2)
					Expect(rh).ToNot(BeZero())
					cmpSer := MustBeOk(ss.Get(rh))
					Expect(cmpSer.DataType).To(Equal(telem.Uint8T))
				}

				for _, name := range []string{
					"compare_gt_scalar_", "compare_lt_scalar_",
					"compare_ge_scalar_", "compare_le_scalar_",
					"compare_eq_scalar_", "compare_ne_scalar_",
				} {
					fn := testutil.Get[f64ScalarCmp](rt, "series", name+"f64")
					rh := fn(ctx, h1, 3.0)
					Expect(rh).ToNot(BeZero())
				}
			})

			It("Should return 0 for invalid handle", func() {
				index := testutil.Get[f64Index](rt, "series", "index_f64")
				Expect(index(ctx, 9999, 0)).To(Equal(float64(0)))
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
		It("Should negate signed i32 types", func() {
			for _, suffix := range []string{"i8", "i16", "i32"} {
				create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_"+suffix)
				set := testutil.Get[i32SetElement](rt, "series", "set_element_"+suffix)
				neg := testutil.Get[negateOp](rt, "series", "negate_"+suffix)

				h := create(ctx, 1)
				set(ctx, h, 0, 5)
				rh := neg(ctx, h)
				Expect(rh).ToNot(BeZero())
			}
		})

		It("Should negate signed i64 types", func() {
			create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_i64")
			set := testutil.Get[i64SetElement](rt, "series", "set_element_i64")
			neg := testutil.Get[negateOp](rt, "series", "negate_i64")

			h := create(ctx, 1)
			set(ctx, h, 0, 5)
			rh := neg(ctx, h)
			Expect(rh).ToNot(BeZero())
		})

		It("Should negate f32", func() {
			create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_f32")
			set := testutil.Get[f32SetElement](rt, "series", "set_element_f32")
			neg := testutil.Get[negateOp](rt, "series", "negate_f32")

			h := create(ctx, 1)
			set(ctx, h, 0, float32(3.14))
			rh := neg(ctx, h)
			Expect(rh).ToNot(BeZero())
			ser := MustBeOk(ss.Get(rh))
			Expect(telem.ValueAt[float32](ser, 0)).To(BeNumerically("~", -3.14, 0.001))
		})

		It("Should negate f64", func() {
			create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_f64")
			set := testutil.Get[f64SetElement](rt, "series", "set_element_f64")
			neg := testutil.Get[negateOp](rt, "series", "negate_f64")

			h := create(ctx, 1)
			set(ctx, h, 0, 3.14)
			rh := neg(ctx, h)
			Expect(rh).ToNot(BeZero())
			ser := MustBeOk(ss.Get(rh))
			Expect(telem.ValueAt[float64](ser, 0)).To(BeNumerically("~", -3.14, 0.001))
		})
	})

	Describe("not_u8", func() {
		It("Should bitwise-NOT a u8 series", func() {
			create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_u8")
			set := testutil.Get[i32SetElement](rt, "series", "set_element_u8")
			notU8 := testutil.Get[negateOp](rt, "series", "not_u8")

			h := create(ctx, 2)
			set(ctx, h, 0, 0)
			set(ctx, h, 1, 255)
			rh := notU8(ctx, h)
			Expect(rh).ToNot(BeZero())
			ser := MustBeOk(ss.Get(rh))
			Expect(telem.ValueAt[uint8](ser, 0)).To(Equal(uint8(255)))
			Expect(telem.ValueAt[uint8](ser, 1)).To(Equal(uint8(0)))
		})
	})

	Describe("len", func() {
		It("Should return the series length", func() {
			lenFn := testutil.Get[func(context.Context, uint32) uint64](rt, "series", "len")
			create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_i32")

			h := create(ctx, 5)
			Expect(lenFn(ctx, h)).To(Equal(uint64(5)))
		})

		It("Should return 0 for invalid handle", func() {
			lenFn := testutil.Get[func(context.Context, uint32) uint64](rt, "series", "len")
			Expect(lenFn(ctx, 9999)).To(Equal(uint64(0)))
		})
	})

	Describe("slice", func() {
		It("Should extract a subrange", func() {
			sliceFn := testutil.Get[func(context.Context, uint32, uint32, uint32) uint32](rt, "series", "slice")
			create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_i32")
			set := testutil.Get[i32SetElement](rt, "series", "set_element_i32")

			h := create(ctx, 4)
			set(ctx, h, 0, 10)
			set(ctx, h, 1, 20)
			set(ctx, h, 2, 30)
			set(ctx, h, 3, 40)

			rh := sliceFn(ctx, h, 1, 3)
			Expect(rh).ToNot(BeZero())
			ser := MustBeOk(ss.Get(rh))
			Expect(ser.Len()).To(Equal(int64(2)))
			Expect(telem.ValueAt[int32](ser, 0)).To(Equal(int32(20)))
			Expect(telem.ValueAt[int32](ser, 1)).To(Equal(int32(30)))
		})

		It("Should return 0 for invalid handle", func() {
			sliceFn := testutil.Get[func(context.Context, uint32, uint32, uint32) uint32](rt, "series", "slice")
			Expect(sliceFn(ctx, 9999, 0, 1)).To(Equal(uint32(0)))
		})

		It("Should return 0 for empty range", func() {
			sliceFn := testutil.Get[func(context.Context, uint32, uint32, uint32) uint32](rt, "series", "slice")
			create := testutil.Get[i32CreateEmpty](rt, "series", "create_empty_i32")

			h := create(ctx, 4)
			Expect(sliceFn(ctx, h, 2, 2)).To(Equal(uint32(0)))
		})
	})
})
