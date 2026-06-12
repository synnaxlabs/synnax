// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package expression_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	acontext "github.com/synnaxlabs/arc/analyzer/context"
	aexpression "github.com/synnaxlabs/arc/analyzer/expression"
	ccontext "github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	. "github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/parser"
	. "github.com/synnaxlabs/arc/symbol/testutil"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Type Cast Compilation", func() {
	DescribeTable("should compile type casts correctly",
		expectExpression,

		// Integer to Integer
		Entry(
			"i64 to i32",
			"i32(42)",
			types.I32(),
			OpI32Const,
			int32(42),
		),

		Entry(
			"i32 expression to i64",
			"i64(i32(42))",
			types.I64(),
			OpI32Const,
			int32(42),
			OpI64ExtendI32S,
		),

		// Integer to Float
		Entry(
			"i64 to f32",
			"f32(42)",
			types.F32(),
			OpF32Const,
			float32(42),
		),

		Entry(
			"i64 to f64",
			"f64(42)",
			types.F64(),
			OpF64Const,
			float64(42),
		),

		// Float to Integer (only exact conversions allowed for literals)
		Entry(
			"f64 to i32 (exact)",
			"i32(3.0)",
			types.I32(),
			OpI32Const,
			int32(3),
		),

		Entry(
			"f64 to i64 (exact)",
			"i64(3.0)",
			types.I64(),
			OpI64Const,
			int64(3),
		),

		// Float to Float
		Entry(
			"f64 to f32",
			"f32(3.14)",
			types.F32(),
			OpF32Const,
			float32(3.14),
		),

		Entry(
			"f32 expression to f64",
			"f64(f32(3.14))",
			types.F64(),
			OpF32Const,
			float32(3.14),
			OpF64PromoteF32,
		),

		// Unsigned Types
		Entry(
			"i64 to u32",
			"u32(42)",
			types.U32(),
			OpI32Const,
			int32(42),
		),

		Entry(
			"u32 to f32",
			"f32(u32(42))",
			types.F32(),
			OpI32Const,
			int32(42),
			OpF32ConvertI32U,
		),

		// Numeric to String — dispatches via Resolver.EmitNumericToString. The Call
		// index here is uint32(0) because each entry compiles in a fresh context, so
		// the relevant string.from_* host fn is the first call registered.
		Entry(
			"i8 to str",
			"str(i8(42))",
			types.String(),
			OpI32Const, int32(42),
			OpCall, uint32(0),
		),
		Entry(
			"i8 to str (negative)",
			"str(i8(-127))",
			types.String(),
			OpI32Const, int32(127),
			OpI32Const, int32(-1),
			OpI32Mul,
			OpCall, uint32(0),
		),
		Entry(
			"i16 to str",
			"str(i16(42))",
			types.String(),
			OpI32Const, int32(42),
			OpCall, uint32(0),
		),
		Entry(
			"i16 to str (negative)",
			"str(i16(-32767))",
			types.String(),
			OpI32Const, int32(32767),
			OpI32Const, int32(-1),
			OpI32Mul,
			OpCall, uint32(0),
		),
		Entry(
			"i32 to str",
			"str(i32(42))",
			types.String(),
			OpI32Const, int32(42),
			OpCall, uint32(0),
		),
		Entry(
			"i32 to str (negative)",
			"str(i32(-2147483647))",
			types.String(),
			OpI32Const, int32(2147483647),
			OpI32Const, int32(-1),
			OpI32Mul,
			OpCall, uint32(0),
		),
		Entry(
			"i64 to str",
			"str(i64(42))",
			types.String(),
			OpI64Const, int64(42),
			OpCall, uint32(0),
		),
		Entry(
			"i64 to str (negative)",
			"str(i64(-9223372036854775807))",
			types.String(),
			OpI64Const, int64(9223372036854775807),
			OpI64Const, int64(-1),
			OpI64Mul,
			OpCall, uint32(0),
		),
		Entry(
			"u8 to str",
			"str(u8(255))",
			types.String(),
			OpI32Const, int32(255),
			OpCall, uint32(0),
		),
		Entry(
			"u16 to str",
			"str(u16(42))",
			types.String(),
			OpI32Const, int32(42),
			OpCall, uint32(0),
		),
		Entry(
			"u32 to str",
			"str(u32(42))",
			types.String(),
			OpI32Const, int32(42),
			OpCall, uint32(0),
		),
		Entry(
			"u64 to str",
			"str(u64(42))",
			types.String(),
			OpI64Const, int64(42),
			OpCall, uint32(0),
		),
		Entry(
			"f32 to str",
			"str(f32(3.14))",
			types.String(),
			OpF32Const, float32(3.14),
			OpCall, uint32(0),
		),
		Entry(
			"f32 to str (negative)",
			"str(f32(-3.14))",
			types.String(),
			OpF32Const, float32(3.14),
			OpF32Neg,
			OpCall, uint32(0),
		),
		Entry(
			"f32 to str (integer-valued, trailing zero)",
			"str(f32(1.0))",
			types.String(),
			OpF32Const, float32(1.0),
			OpCall, uint32(0),
		),
		Entry(
			"f32 to str (single trailing zero)",
			"str(f32(3.10))",
			types.String(),
			OpF32Const, float32(3.10),
			OpCall, uint32(0),
		),
		Entry(
			"f32 to str (multiple trailing zeros)",
			"str(f32(100.000))",
			types.String(),
			OpF32Const, float32(100.000),
			OpCall, uint32(0),
		),
		Entry(
			"f64 to str",
			"str(f64(3.14))",
			types.String(),
			OpF64Const, float64(3.14),
			OpCall, uint32(0),
		),
		Entry(
			"f64 to str (negative)",
			"str(f64(-3.14))",
			types.String(),
			OpF64Const, float64(3.14),
			OpF64Neg,
			OpCall, uint32(0),
		),
		Entry(
			"f64 to str (integer-valued, trailing zero)",
			"str(f64(1.0))",
			types.String(),
			OpF64Const, float64(1.0),
			OpCall, uint32(0),
		),
		Entry(
			"f64 to str (single trailing zero)",
			"str(f64(3.10))",
			types.String(),
			OpF64Const, float64(3.10),
			OpCall, uint32(0),
		),
		Entry(
			"f64 to str (multiple trailing zeros)",
			"str(f64(100.000))",
			types.String(),
			OpF64Const, float64(100.000),
			OpCall, uint32(0),
		),
		Entry(
			"integer literal to str (natural i64)",
			"str(42)",
			types.String(),
			OpI64Const, int64(42),
			OpCall, uint32(0),
		),
		Entry(
			"integer literal to str (negative)",
			"str(-42)",
			types.String(),
			OpI64Const, int64(42),
			OpI64Const, int64(-1),
			OpI64Mul,
			OpCall, uint32(0),
		),
		Entry(
			"float literal to str (natural f64)",
			"str(3.14)",
			types.String(),
			OpF64Const, float64(3.14),
			OpCall, uint32(0),
		),
		Entry(
			"float literal to str (negative)",
			"str(-3.14)",
			types.String(),
			OpF64Const, float64(3.14),
			OpF64Neg,
			OpCall, uint32(0),
		),
		Entry(
			"float literal to str (negative zero)",
			"str(-0.0)",
			types.String(),
			OpF64Const, float64(0),
			OpF64Neg,
			OpCall, uint32(0),
		),
		Entry(
			"float literal to str (negative zero with trailing zeros)",
			"str(-0.0000)",
			types.String(),
			OpF64Const, float64(0),
			OpF64Neg,
			OpCall, uint32(0),
		),
		Entry(
			"float literal to str (trailing zero)",
			"str(1.0)",
			types.String(),
			OpF64Const, float64(1.0),
			OpCall, uint32(0),
		),
		Entry(
			"float literal to str (multiple trailing zeros)",
			"str(100.000)",
			types.String(),
			OpF64Const, float64(100.000),
			OpCall, uint32(0),
		),
		Entry(
			"str to str (no-op)",
			`str("hello")`,
			types.String(),
			OpI32Const, int32(0),
			OpI32Const, int32(5),
			OpCall, uint32(0),
		),
	)

	DescribeTable(
		"should reject str to numeric casts (analyzer-gated)",
		func(bCtx SpecContext, source string) {
			expr := MustSucceed(parser.ParseExpression(source))
			analyzerCtx := acontext.NewRoot(bCtx, expr, NewRoot(nil))
			aexpression.Analyze(analyzerCtx)
			Expect(analyzerCtx.Diagnostics.Ok()).To(BeFalse())
			Expect(analyzerCtx.Diagnostics.String()).To(ContainSubstring("cannot cast"))
		},
		Entry("str to i8", `i8("hello")`),
		Entry("str to i16", `i16("hello")`),
		Entry("str to i32", `i32("hello")`),
		Entry("str to i64", `i64("hello")`),
		Entry("str to u8", `u8("hello")`),
		Entry("str to u16", `u16("hello")`),
		Entry("str to u32", `u32("hello")`),
		Entry("str to u64", `u64("hello")`),
		Entry("str to f32", `f32("hello")`),
		Entry("str to f64", `f64("hello")`),
	)

	It("Should propagate literal parsing errors", func(bCtx SpecContext) {
		// Test that non-exact float-to-int conversions are rejected
		expr := MustSucceed(parser.ParseExpression("i32(3.14)"))
		ctx := NewContext(bCtx)
		_, err := expression.Compile(ccontext.Child(ctx, expr))
		Expect(err).To(MatchError(ContainSubstring("cannot convert non-integer float")))
	})

	It("Should propagate overflow errors from literals", func(bCtx SpecContext) {
		// Test that overflow validation is enforced
		expr := MustSucceed(parser.ParseExpression("i8(128)"))
		ctx := NewContext(bCtx)
		_, err := expression.Compile(ccontext.Child(ctx, expr))
		Expect(err).To(MatchError(ContainSubstring("out of range for i8")))
	})
})
