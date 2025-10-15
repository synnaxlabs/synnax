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
	"github.com/synnaxlabs/arc/compiler/context"
	"github.com/synnaxlabs/arc/compiler/expression"
	. "github.com/synnaxlabs/arc/compiler/testutil"
	. "github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Identifier Compilation", func() {
	Context("Local Variables", func() {
		It("Should compile local variable references", func() {
			ctx := NewContext(bCtx)
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			byteCode, exprType := compileWithCtx(ctx, "x")
			Expect(byteCode).To(Equal(WASM(OpLocalGet, 0)))
			Expect(exprType).To(Equal(types.I32()))
		})

		It("Should compile expressions using multiple local variables", func() {
			ctx := NewContext(bCtx)
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "a", Kind: symbol.KindVariable, Type: types.I32()}))
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "b", Kind: symbol.KindVariable, Type: types.I32()}))
			// Compile expression using both variables
			expr := MustSucceed(parser.ParseExpression("a + b"))
			exprType := MustSucceed(expression.Compile(context.Child(ctx, expr)))
			bytecode := ctx.Writer.Bytes()
			expected := WASM(
				OpLocalGet, 0, // Resolve 'a'
				OpLocalGet, 1, // Resolve 'b'
				OpI32Add, // Add them
			)
			Expect(bytecode).To(Equal(expected))
			Expect(exprType).To(Equal(types.I32()))
		})

		It("Should compile complex expressions with local variables", func() {
			ctx := NewContext(bCtx)
			// Add variables with different types
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.F64()}))
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "y", Kind: symbol.KindVariable, Type: types.F64()}))
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "z", Kind: symbol.KindVariable, Type: types.F64()}))
			bytecode, exprType := compileWithCtx(ctx, "(x + y) * z")
			expected := WASM(
				OpLocalGet, 0, // Resolve 'x'
				OpLocalGet, 1, // Resolve 'y'
				OpF64Add,      // x + y
				OpLocalGet, 2, // Resolve 'z'
				OpF64Mul, // (x + y) * z
			)
			Expect(bytecode).To(Equal(expected))
			Expect(exprType).To(Equal(types.F64()))
		})

		It("Should compile comparisons using local variables", func() {
			ctx := NewContext(bCtx)
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "limit", Kind: symbol.KindVariable, Type: types.I32()}))
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "value", Kind: symbol.KindVariable, Type: types.I32()}))
			bytecode, exprType := compileWithCtx(ctx, "value > limit")
			expected := WASM(
				OpLocalGet, 1, // Resolve 'value'
				OpLocalGet, 0, // Resolve 'limit'
				OpI32GtS, // value > limit
			)
			Expect(bytecode).To(Equal(expected))
			Expect(exprType).To(Equal(types.U8())) // Comparisons return boolean
		})

		It("Should compile logical operations with local variables", func() {
			ctx := NewContext(bCtx)
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "enabled", Kind: symbol.KindVariable, Type: types.U8()}))
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "ready", Kind: symbol.KindVariable, Type: types.U8()}))
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
			Expect(exprType).To(Equal(types.U8()))
		})
	})

	Context("Channel Reads", func() {
		It("Should compile a channel read", func() {
			ctx := NewContext(bCtx)
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "x", Kind: symbol.KindChannel, Type: types.Chan(types.I32())}))
			byteCode, exprType := compileWithCtx(ctx, "x")
			i := ctx.Imports.ChannelRead["i32"]
			Expect(exprType).To(Equal(types.I32()))
			Expect(byteCode).To(Equal(WASM(
				OpI32Const,
				int32(0),
				OpCall,
				uint64(i),
			)))
		})

		It("Should correctly compile a channel read inside of an addition expression", func() {
			ctx := NewContext(bCtx)
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "x", Kind: symbol.KindChannel, Type: types.Chan(types.I32())}))
			byteCode, exprType := compileWithCtx(ctx, "x + 1")
			i := ctx.Imports.ChannelRead["i32"]
			Expect(exprType).To(Equal(types.I32()))
			Expect(byteCode).To(Equal(WASM(
				OpI32Const,
				int32(0),
				OpCall,
				uint64(i),
				OpI32Const,
				int32(1),
				OpI32Add,
			)))
		})
	})

	Context("Function Parameters", func() {
		It("Should compile parameter references", func() {
			ctx := NewContext(bCtx)
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "value",
				Kind: symbol.KindInput,
				Type: types.F64(),
			}))
			bytecode, exprType := compileWithCtx(ctx, "value")
			expected := WASM(OpLocalGet, 0)
			Expect(bytecode).To(Equal(expected))
			Expect(exprType).To(Equal(types.F64()))
		})
	})
})
