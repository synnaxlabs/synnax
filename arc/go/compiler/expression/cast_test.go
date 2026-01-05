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
	ccontext "github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	. "github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/types"
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
	)

	It("Should propagate literal parsing errors", func() {
		// Test that non-exact float-to-int conversions are rejected
		expr, diag := parser.ParseExpression("i32(3.14)")
		Expect(diag).To(BeNil())
		ctx := NewContext(bCtx)
		_, err := expression.Compile(ccontext.Child(ctx, expr))
		Expect(err).To(MatchError(ContainSubstring("cannot convert non-integer float")))
	})

	It("Should propagate overflow errors from literals", func() {
		// Test that overflow validation is enforced
		expr, diag := parser.ParseExpression("i8(128)")
		Expect(diag).To(BeNil())
		ctx := NewContext(bCtx)
		_, err := expression.Compile(ccontext.Child(ctx, expr))
		Expect(err).To(MatchError(ContainSubstring("out of range for i8")))
	})
})
