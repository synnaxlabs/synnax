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
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Format String Compilation", func() {
	Describe("Single Placeholder, No Spec", func() {
		DescribeTable("compiles numeric literal placeholder via from_<type> conversion",
			expectExpression,

			Entry(
				"integer literal placeholder",
				`f"{42}"`,
				types.String(),
				OpI64Const, int64(42),
				OpCall, uint32(0),
			),
			Entry(
				"float literal placeholder",
				`f"{3.14}"`,
				types.String(),
				OpF64Const, float64(3.14),
				OpCall, uint32(0),
			),
			Entry(
				"explicit i32 cast placeholder",
				`f"{i32(7)}"`,
				types.String(),
				OpI32Const, int32(7),
				OpCall, uint32(0),
			),
			Entry(
				"explicit u8 cast placeholder",
				`f"{u8(255)}"`,
				types.String(),
				OpI32Const, int32(255),
				OpCall, uint32(0),
			),
			Entry(
				"explicit f32 cast placeholder",
				`f"{f32(2.5)}"`,
				types.String(),
				OpF32Const, float32(2.5),
				OpCall, uint32(0),
			),
		)
	})

	Describe("Single Placeholder, With Format Spec", func() {
		DescribeTable("compiles numeric placeholder with spec via emitSpecBytes + format_<type>",
			expectExpression,

			Entry(
				"i32 with :05d",
				`f"{i32(7):05d}"`,
				types.String(),
				OpI32Const, int32(7),
				OpI32Const, int32(0),
				OpI32Const, int32(3),
				OpCall, uint32(0),
			),
			Entry(
				"f64 with :.2f",
				`f"{f64(3.14):.2f}"`,
				types.String(),
				OpF64Const, float64(3.14),
				OpI32Const, int32(0),
				OpI32Const, int32(3),
				OpCall, uint32(0),
			),
			Entry(
				"u8 with :x",
				`f"{u8(255):x}"`,
				types.String(),
				OpI32Const, int32(255),
				OpI32Const, int32(0),
				OpI32Const, int32(1),
				OpCall, uint32(0),
			),
		)
	})

	Describe("String Variable Placeholder", func() {
		It("compiles string variable placeholder with no spec as identity", func(bCtx SpecContext) {
			bytecode, exprType := compileWithAnalyzer(bCtx, `f"{name}"`, []symbol.Symbol{
				scalarSymbol("name", types.String(), 0),
			})
			Expect(exprType).To(Equal(types.String()))
			Expect(bytecode).ToNot(BeEmpty())
		})

		It("compiles string variable placeholder with spec via format_str", func(bCtx SpecContext) {
			bytecode, exprType := compileWithAnalyzer(bCtx, `f"{name:s}"`, []symbol.Symbol{
				scalarSymbol("name", types.String(), 0),
			})
			Expect(exprType).To(Equal(types.String()))
			Expect(bytecode).ToNot(BeEmpty())
		})
	})

	Describe("Mixed Literal and Placeholder Segments", func() {
		It("compiles literal + placeholder with concat", func(bCtx SpecContext) {
			bytecode, exprType := compileExpression(bCtx, `f"x={42}"`)
			Expect(exprType).To(Equal(types.String()))
			Expect(bytecode).ToNot(BeEmpty())
		})

		It("compiles placeholder + literal with concat", func(bCtx SpecContext) {
			bytecode, exprType := compileExpression(bCtx, `f"{42} done"`)
			Expect(exprType).To(Equal(types.String()))
			Expect(bytecode).ToNot(BeEmpty())
		})

		It("compiles two placeholders separated by literal with two concat ops", func(bCtx SpecContext) {
			bytecode, exprType := compileExpression(bCtx, `f"{1} and {2}"`)
			Expect(exprType).To(Equal(types.String()))
			Expect(bytecode).ToNot(BeEmpty())
		})

		It("compiles three placeholders with mixed specs", func(bCtx SpecContext) {
			bytecode, exprType := compileExpression(bCtx, `f"{1}, {i32(2):05d}, {f64(3.14):.2f}"`)
			Expect(exprType).To(Equal(types.String()))
			Expect(bytecode).ToNot(BeEmpty())
		})

		It("compiles adjacent placeholders with no separator", func(bCtx SpecContext) {
			bytecode, exprType := compileExpression(bCtx, `f"{1}{2}"`)
			Expect(exprType).To(Equal(types.String()))
			Expect(bytecode).ToNot(BeEmpty())
		})

		It("compiles a multi-line format string with literal newlines around a placeholder", func(bCtx SpecContext) {
			bytecode, exprType := compileExpression(bCtx, "f`line1\n{42}\nline3`")
			Expect(exprType).To(Equal(types.String()))
			Expect(bytecode).ToNot(BeEmpty())
		})

		It("compiles a multi-line format string with multiple placeholders across lines", func(bCtx SpecContext) {
			bytecode, exprType := compileExpression(bCtx, "f`a={1}\nb={2}`")
			Expect(exprType).To(Equal(types.String()))
			Expect(bytecode).ToNot(BeEmpty())
		})

		It("compiles an rf-prefixed format string with placeholder and backslash literal", func(bCtx SpecContext) {
			bytecode, exprType := compileExpression(bCtx, `rf"path\to\{42}"`)
			Expect(exprType).To(Equal(types.String()))
			Expect(bytecode).ToNot(BeEmpty())
		})

		It("compiles an rf-prefixed multi-line format string with placeholder across newlines", func(bCtx SpecContext) {
			bytecode, exprType := compileExpression(bCtx, "rf`path\\to\n{42}`")
			Expect(exprType).To(Equal(types.String()))
			Expect(bytecode).ToNot(BeEmpty())
		})
	})

	Describe("Malformed Format String Body", func() {
		It("propagates literal.FmtStrParse errors from compileRawStringLiteral", func(bCtx SpecContext) {
			// `{` parses as a raw-string token, but the body is malformed:
			// literal.FmtStrParse rejects an unmatched '{', exercising the second
			// error branch in compileRawStringLiteral.
			expr := MustSucceed(parser.ParseExpression(`f"{"`))
			ctx := NewContext(bCtx)
			_, err := expression.Compile(ccontext.Child(ctx, expr))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("unmatched"))
		})
	})
})
