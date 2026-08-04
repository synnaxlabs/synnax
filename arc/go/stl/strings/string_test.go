// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package strings_test

import (
	"math"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/stl/strings"
	"github.com/synnaxlabs/arc/stl/testutil"
	. "github.com/synnaxlabs/x/testutil"
	"github.com/tetratelabs/wazero/experimental/wazerotest"
)

var _ = Describe("Strings", func() {
	var (
		rt  *testutil.Runtime
		ss  *strings.ProgramState
		mem *wazerotest.Memory
	)

	call := func(ctx SpecContext, fn string, args ...uint64) []uint64 {
		return rt.Call(ctx, "strings", fn, args...)
	}
	callU32 := func(ctx SpecContext, fn string, args ...uint64) uint32 {
		return testutil.AsU32(call(ctx, fn, args...)[0])
	}
	callU64 := func(ctx SpecContext, fn string, args ...uint64) uint64 {
		return call(ctx, fn, args...)[0]
	}

	BeforeEach(func(ctx SpecContext) {
		rt = testutil.NewRuntime(ctx)
		ss = strings.NewProgramState()
		mem = wazerotest.NewMemory(1)
		MustSucceed(strings.NewHost(ctx, rt.Underlying(), ss, mem))
		rt.Passthrough(ctx, "strings")
	})

	AfterEach(func(ctx SpecContext) {
		Expect(rt.Close(ctx)).To(Succeed())
	})

	Describe("from_literal", func() {
		It("Should create a handle from WASM memory", func(ctx SpecContext) {
			mem.Write(0, []byte("hello"))
			h := callU32(ctx, "from_literal", testutil.U32(0), testutil.U32(5))
			Expect(h).ToNot(BeZero())
			Expect(MustBeOk(ss.Get(h))).To(Equal("hello"))
		})

		It("Should return 0 when memory read fails", func(ctx SpecContext) {
			Expect(
				callU32(ctx, "from_literal", testutil.U32(100000), testutil.U32(5)),
			).To(Equal(uint32(0)))
		})

		It(
			"Should return handle 0 for empty string with length 0",
			func(ctx SpecContext) {
				Expect(
					callU32(ctx, "from_literal", testutil.U32(0), testutil.U32(0)),
				).To(Equal(uint32(0)))
			},
		)
	})

	Describe("from_literal with nil memory", func() {
		It("Should return 0 when memory is nil", func(ctx SpecContext) {
			rt2 := testutil.NewRuntime(ctx)
			DeferCleanup(rt2.Close)
			ss2 := strings.NewProgramState()
			MustSucceed(strings.NewHost(ctx, rt2.Underlying(), ss2, nil))
			rt2.Passthrough(ctx, "strings")
			res := rt2.Call(
				ctx,
				"strings",
				"from_literal",
				testutil.U32(0),
				testutil.U32(5),
			)
			Expect(testutil.AsU32(res[0])).To(Equal(uint32(0)))
		})
	})

	Describe("concat", func() {
		It("Should concatenate two strings", func(ctx SpecContext) {
			h1 := ss.Create("hello ")
			h2 := ss.Create("world")
			rh := callU32(ctx, "concat", testutil.U32(h1), testutil.U32(h2))
			Expect(rh).ToNot(BeZero())
			Expect(MustBeOk(ss.Get(rh))).To(Equal("hello world"))
		})

		It("Should return 0 for invalid handles", func(ctx SpecContext) {
			Expect(
				callU32(ctx, "concat", testutil.U32(9999), testutil.U32(9998)),
			).To(Equal(uint32(0)))
		})
	})

	Describe("equal", func() {
		It("Should return 1 for equal strings", func(ctx SpecContext) {
			h1 := ss.Create("same")
			h2 := ss.Create("same")
			Expect(
				callU32(ctx, "equal", testutil.U32(h1), testutil.U32(h2)),
			).To(Equal(uint32(1)))
		})

		It("Should return 0 for different strings", func(ctx SpecContext) {
			h1 := ss.Create("foo")
			h2 := ss.Create("bar")
			Expect(
				callU32(ctx, "equal", testutil.U32(h1), testutil.U32(h2)),
			).To(Equal(uint32(0)))
		})

		It("Should return 0 for invalid handles", func(ctx SpecContext) {
			Expect(
				callU32(ctx, "equal", testutil.U32(9999), testutil.U32(9998)),
			).To(Equal(uint32(0)))
		})
	})

	Describe("len", func() {
		It("Should return byte length", func(ctx SpecContext) {
			h := ss.Create("hello")
			Expect(callU64(ctx, "len", testutil.U32(h))).To(Equal(uint64(5)))
		})

		It("Should return 0 for invalid handle", func(ctx SpecContext) {
			Expect(callU64(ctx, "len", testutil.U32(9999))).To(Equal(uint64(0)))
		})
	})

	Describe("from_i32", func() {
		DescribeTable("Should format signed 32-bit integers in base-10",
			func(ctx SpecContext, value int32, expected string) {
				h := callU32(ctx, "from_i32", testutil.I32(value))
				Expect(MustBeOk(ss.Get(h))).To(Equal(expected))
			},
			Entry("zero", int32(0), "0"),
			Entry("positive", int32(42), "42"),
			Entry("negative", int32(-42), "-42"),
			Entry("i8 min as i32", int32(-128), "-128"),
			Entry("i8 max as i32", int32(127), "127"),
			Entry("i16 min as i32", int32(-32768), "-32768"),
			Entry("i16 max as i32", int32(32767), "32767"),
			Entry("i32 min", int32(-2147483648), "-2147483648"),
			Entry("i32 max", int32(2147483647), "2147483647"),
		)
	})

	Describe("from_u32", func() {
		DescribeTable("Should format unsigned 32-bit integers in base-10",
			func(ctx SpecContext, value uint32, expected string) {
				h := callU32(ctx, "from_u32", testutil.U32(value))
				Expect(MustBeOk(ss.Get(h))).To(Equal(expected))
			},
			Entry("zero", uint32(0), "0"),
			Entry("u8 max", uint32(255), "255"),
			Entry("u16 max", uint32(65535), "65535"),
			Entry("u32 max", uint32(4294967295), "4294967295"),
		)
	})

	Describe("from_i64", func() {
		DescribeTable("Should format signed 64-bit integers in base-10",
			func(ctx SpecContext, value int64, expected string) {
				h := callU32(ctx, "from_i64", testutil.I64(value))
				Expect(MustBeOk(ss.Get(h))).To(Equal(expected))
			},
			Entry("zero", int64(0), "0"),
			Entry("positive", int64(9223372036854775807), "9223372036854775807"),
			Entry("negative", int64(-9223372036854775808), "-9223372036854775808"),
		)
	})

	Describe("from_u64", func() {
		DescribeTable("Should format unsigned 64-bit integers in base-10",
			func(ctx SpecContext, value uint64, expected string) {
				h := callU32(ctx, "from_u64", testutil.U64(value))
				Expect(MustBeOk(ss.Get(h))).To(Equal(expected))
			},
			Entry("zero", uint64(0), "0"),
			Entry("max", uint64(18446744073709551615), "18446744073709551615"),
		)
	})

	Describe("from_f32", func() {
		DescribeTable(
			"Should format f32 values with shortest round-trippable representation",
			func(ctx SpecContext, value float32, expected string) {
				h := callU32(ctx, "from_f32", testutil.F32(value))
				Expect(MustBeOk(ss.Get(h))).To(Equal(expected))
			},
			Entry("simple decimal", float32(3.14), "3.14"),
			Entry("zero", float32(0.0), "0"),
			Entry("negative zero", float32(math.Copysign(0, -1)), "-0"),
			Entry("negative", float32(-2.5), "-2.5"),
			Entry("1.0 (integer-valued)", float32(1.0), "1"),
			Entry("10.0 (integer-valued)", float32(10.0), "10"),
			Entry("100.00 (multiple trailing zeros)", float32(100.00), "100"),
			Entry("3.10 (single trailing zero)", float32(3.10), "3.1"),
			Entry("1.50 (single trailing zero)", float32(1.50), "1.5"),
			Entry("0.10 (leading + trailing zero)", float32(0.10), "0.1"),
			Entry("-3.10 (negative trailing zero)", float32(-3.10), "-3.1"),
			Entry("-1.0 (negative integer-valued)", float32(-1.0), "-1"),
			Entry("5.000 (three trailing zeros)", float32(5.000), "5"),
		)

		It("Should format NaN", func(ctx SpecContext) {
			h := callU32(ctx, "from_f32", testutil.F32(float32(math.NaN())))
			Expect(MustBeOk(ss.Get(h))).To(Equal("NaN"))
		})

		It("Should format positive infinity", func(ctx SpecContext) {
			h := callU32(ctx, "from_f32", testutil.F32(float32(math.Inf(1))))
			Expect(MustBeOk(ss.Get(h))).To(Equal("+Inf"))
		})

		It("Should format negative infinity", func(ctx SpecContext) {
			h := callU32(ctx, "from_f32", testutil.F32(float32(math.Inf(-1))))
			Expect(MustBeOk(ss.Get(h))).To(Equal("-Inf"))
		})
	})

	Describe("from_f64", func() {
		DescribeTable(
			"Should format f64 values with shortest round-trippable representation",
			func(ctx SpecContext, value float64, expected string) {
				h := callU32(ctx, "from_f64", testutil.F64(value))
				Expect(MustBeOk(ss.Get(h))).To(Equal(expected))
			},
			Entry("simple decimal", float64(3.14159), "3.14159"),
			Entry("zero", float64(0.0), "0"),
			Entry("negative zero", math.Copysign(0, -1), "-0"),
			Entry("negative", float64(-2.5), "-2.5"),
			Entry("high precision", float64(0.1234567890123456), "0.1234567890123456"),
			Entry("1.0 (integer-valued)", float64(1.0), "1"),
			Entry("10.0 (integer-valued)", float64(10.0), "10"),
			Entry("100.00 (multiple trailing zeros)", float64(100.00), "100"),
			Entry("3.10 (single trailing zero)", float64(3.10), "3.1"),
			Entry("1.50 (single trailing zero)", float64(1.50), "1.5"),
			Entry("0.10 (leading + trailing zero)", float64(0.10), "0.1"),
			Entry("-3.10 (negative trailing zero)", float64(-3.10), "-3.1"),
			Entry("-1.0 (negative integer-valued)", float64(-1.0), "-1"),
			Entry("5.000 (three trailing zeros)", float64(5.000), "5"),
			Entry("1e10 integer-valued", float64(10000000000.0), "1e+10"),
		)

		It("Should format NaN", func(ctx SpecContext) {
			h := callU32(ctx, "from_f64", testutil.F64(math.NaN()))
			Expect(MustBeOk(ss.Get(h))).To(Equal("NaN"))
		})

		It("Should format positive infinity", func(ctx SpecContext) {
			h := callU32(ctx, "from_f64", testutil.F64(math.Inf(1)))
			Expect(MustBeOk(ss.Get(h))).To(Equal("+Inf"))
		})

		It("Should format negative infinity", func(ctx SpecContext) {
			h := callU32(ctx, "from_f64", testutil.F64(math.Inf(-1)))
			Expect(MustBeOk(ss.Get(h))).To(Equal("-Inf"))
		})
	})

	Describe("format_* spec read failures", func() {
		writeSpec := func(spec string) (uint32, uint32) {
			mem.Write(0, []byte(spec))
			return 0, uint32(len(spec))
		}

		It(
			"Should format an i32 against a spec read from memory",
			func(ctx SpecContext) {
				ptr, length := writeSpec("05d")
				h := callU32(
					ctx,
					"format_i32",
					testutil.I32(7),
					testutil.U32(ptr),
					testutil.U32(length),
				)
				Expect(MustBeOk(ss.Get(h))).To(Equal("00007"))
			},
		)

		It(
			"Should format an f64 against a spec read from memory",
			func(ctx SpecContext) {
				ptr, length := writeSpec(".2f")
				h := callU32(
					ctx,
					"format_f64",
					testutil.F64(3.14159),
					testutil.U32(ptr),
					testutil.U32(length),
				)
				Expect(MustBeOk(ss.Get(h))).To(Equal("3.14"))
			},
		)

		DescribeTable("Should return handle 0 when the spec read is out-of-bounds",
			func(ctx SpecContext, fn string, value uint64) {
				Expect(
					callU32(ctx, fn, value, testutil.U32(1<<30), testutil.U32(4)),
				).To(Equal(uint32(0)))
			},
			Entry("format_i32 with OOB spec", "format_i32", testutil.I32(42)),
			Entry("format_u32 with OOB spec", "format_u32", testutil.U32(42)),
			Entry("format_i64 with OOB spec", "format_i64", testutil.I64(42)),
			Entry("format_u64 with OOB spec", "format_u64", testutil.U64(42)),
			Entry("format_f32 with OOB spec", "format_f32", testutil.F32(3.14)),
			Entry("format_f64 with OOB spec", "format_f64", testutil.F64(3.14)),
		)
	})

	Describe("format_str", func() {
		writeSpec := func(spec string) (uint32, uint32) {
			mem.Write(0, []byte(spec))
			return 0, uint32(len(spec))
		}

		It(
			"Should format a string handle against a spec read from memory",
			func(ctx SpecContext) {
				h := ss.Create("hi")
				ptr, length := writeSpec("5s")
				rh := callU32(
					ctx,
					"format_str",
					testutil.U32(h),
					testutil.U32(ptr),
					testutil.U32(length),
				)
				Expect(MustBeOk(ss.Get(rh))).To(Equal("   hi"))
			},
		)

		It(
			"Should format an unknown handle as the empty-string spec result",
			func(ctx SpecContext) {
				ptr, length := writeSpec("q")
				rh := callU32(
					ctx,
					"format_str",
					testutil.U32(9999),
					testutil.U32(ptr),
					testutil.U32(length),
				)
				Expect(rh).To(BeZero())
			},
		)

		It(
			"Should return handle 0 when the spec read is out-of-bounds",
			func(ctx SpecContext) {
				h := ss.Create("hi")
				rh := callU32(
					ctx,
					"format_str",
					testutil.U32(h),
					testutil.U32(1<<30),
					testutil.U32(4),
				)
				Expect(rh).To(BeZero())
			},
		)
	})

	Describe("Host.SetMemory", func() {
		It("Should swap the backing memory used by format_*", func(ctx SpecContext) {
			rt2 := testutil.NewRuntime(ctx)
			DeferCleanup(rt2.Close)
			ss2 := strings.NewProgramState()
			h := MustSucceed(strings.NewHost(ctx, rt2.Underlying(), ss2, nil))
			rt2.Passthrough(ctx, "strings")
			mem2 := wazerotest.NewMemory(1)
			h.SetMemory(mem2)
			mem2.Write(0, []byte("05d"))
			res := rt2.Call(
				ctx,
				"strings",
				"format_i32",
				testutil.I32(7),
				testutil.U32(0),
				testutil.U32(3),
			)
			Expect(MustBeOk(ss2.Get(testutil.AsU32(res[0])))).To(Equal("00007"))
		})
	})

	Describe("cross-function handle reuse", func() {
		It(
			"Should use from_literal result in concat and verify with equal",
			func(ctx SpecContext) {
				mem.Write(0, []byte("helloworld"))
				h1 := callU32(ctx, "from_literal", testutil.U32(0), testutil.U32(5))
				h2 := callU32(ctx, "from_literal", testutil.U32(5), testutil.U32(5))
				result := callU32(ctx, "concat", testutil.U32(h1), testutil.U32(h2))
				expected := ss.Create("helloworld")
				Expect(
					callU32(ctx, "equal", testutil.U32(result), testutil.U32(expected)),
				).To(Equal(uint32(1)))
			},
		)
	})
})
