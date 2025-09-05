// Copyright 2025 Synnax Labs, Inc.
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
	"github.com/synnaxlabs/slate/compiler/expression"
	. "github.com/synnaxlabs/slate/compiler/testutil"
	. "github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Expression Compiler", func() {
	Describe("Identifier Compilation", func() {
		Context("Local Variables", func() {
			It("Should compile local variable references", func() {
				ctx := NewContext()
				ctx.Scope.Add("x", symbol.KindVariable, types.I32{}, nil)
				byteCode, exprType := compileWithCtx(ctx, "x")
				Expect(byteCode).To(Equal(WASM(OpLocalGet, 0)))
				Expect(exprType).To(Equal(types.I32{}))
			})

			It("Should compile expressions using multiple local variables", func() {
				ctx := NewContext()
				MustSucceed(ctx.Scope.Add("a", symbol.KindVariable, types.I32{}, nil))
				MustSucceed(ctx.Scope.Add("b", symbol.KindVariable, types.I32{}, nil))
				// Compile expression using both variables
				expr := MustSucceed(parser.ParseExpression("a + b"))
				exprType := MustSucceed(expression.Compile(ctx, expr))
				bytecode := ctx.Writer.Bytes()
				expected := WASM(
					OpLocalGet, 0, // Resolve 'a'
					OpLocalGet, 1, // Resolve 'b'
					OpI32Add, // Add them
				)
				Expect(bytecode).To(Equal(expected))
				Expect(exprType).To(Equal(types.I32{}))
			})

			It("Should compile complex expressions with local variables", func() {
				ctx := NewContext()
				// Add variables with different types
				MustSucceed(ctx.Scope.Add("x", symbol.KindVariable, types.F64{}, nil))
				MustSucceed(ctx.Scope.Add("y", symbol.KindVariable, types.F64{}, nil))
				MustSucceed(ctx.Scope.Add("z", symbol.KindVariable, types.F64{}, nil))
				bytecode, exprType := compileWithCtx(ctx, "(x + y) * z")
				expected := WASM(
					OpLocalGet, 0, // Resolve 'x'
					OpLocalGet, 1, // Resolve 'y'
					OpF64Add,      // x + y
					OpLocalGet, 2, // Resolve 'z'
					OpF64Mul, // (x + y) * z
				)
				Expect(bytecode).To(Equal(expected))
				Expect(exprType).To(Equal(types.F64{}))
			})

			It("Should compile comparisons using local variables", func() {
				ctx := NewContext()
				MustSucceed(ctx.Scope.Add("limit", symbol.KindVariable, types.I32{}, nil))
				MustSucceed(ctx.Scope.Add("value", symbol.KindVariable, types.I32{}, nil))
				bytecode, exprType := compileWithCtx(ctx, "value > limit")
				expected := WASM(
					OpLocalGet, 1, // Resolve 'value'
					OpLocalGet, 0, // Resolve 'limit'
					OpI32GtS, // value > limit
				)
				Expect(bytecode).To(Equal(expected))
				Expect(exprType).To(Equal(types.U8{})) // Comparisons return boolean
			})

			It("Should compile logical operations with local variables", func() {
				ctx := NewContext()
				MustSucceed(ctx.Scope.Add("enabled", symbol.KindVariable, types.U8{}, nil))
				MustSucceed(ctx.Scope.Add("ready", symbol.KindVariable, types.U8{}, nil))
				bytecode, exprType := compileWithCtx(ctx, "enabled && ready")
				expected := WASM(
					// Load 'enabled'
					OpLocalGet, 0,
					// Normalize to boolean (0 or 1)
					OpI32Const, int32(0),
					OpI32Ne,
					// Check if zero for short-circuit
					OpI32Eqz,
					OpIf, byte(I32), // If enabled is false (0)
					OpI32Const, int32(0), // Result is 0
					OpElse,
					// enabled was true, evaluate 'ready'
					OpLocalGet, 1,
					// Normalize to boolean
					OpI32Const, int32(0),
					OpI32Ne,
					OpEnd,
				)
				Expect(bytecode).To(Equal(expected))
				Expect(exprType).To(Equal(types.U8{}))
			})
		})

		Context("Function Parameters", func() {
			It("Should compile parameter references", func() {
				ctx := NewContext()
				MustSucceed(ctx.Scope.Add(
					"value",
					symbol.KindParam,
					types.F64{},
					nil,
				))
				bytecode, exprType := compileWithCtx(ctx, "value")
				expected := WASM(OpLocalGet, 0)
				Expect(bytecode).To(Equal(expected))
				Expect(exprType).To(Equal(types.F64{}))
			})
		})
	})
})
