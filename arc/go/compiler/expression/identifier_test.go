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
	"github.com/synnaxlabs/arc/ir"
	"github.com/synnaxlabs/arc/parser"
	"github.com/synnaxlabs/arc/symbol"
	"github.com/synnaxlabs/arc/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Identifier Compilation", func() {
	Context("Local Variables", func() {
		It("Should compile local variable references", func() {
			ctx := NewContext(bCtx)
			scope := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scope).ToNot(BeNil())
			byteCode, exprType := compileWithCtx(ctx, "x")
			Expect(byteCode).To(MatchOpcodes(OpLocalGet, 0))
			Expect(exprType).To(Equal(types.I32()))
		})

		It("Should compile expressions using multiple local variables", func() {
			ctx := NewContext(bCtx)
			scopeA := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "a", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scopeA).ToNot(BeNil())
			scopeB := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "b", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scopeB).ToNot(BeNil())
			// Compile expression using both variables
			expr := MustSucceed(parser.ParseExpression("a + b"))
			exprType := MustSucceed(expression.Compile(context.Child(ctx, expr)))
			bytecode := ctx.Writer.Bytes()
			Expect(bytecode).To(MatchOpcodes(
				OpLocalGet, 0, // Resolve 'a'
				OpLocalGet, 1, // Resolve 'b'
				OpI32Add, // Add them
			))
			Expect(exprType).To(Equal(types.I32()))
		})

		It("Should compile complex expressions with local variables", func() {
			ctx := NewContext(bCtx)
			// Add variables with different types
			scopeX := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "x", Kind: symbol.KindVariable, Type: types.F64()}))
			Expect(scopeX).ToNot(BeNil())
			scopeY := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "y", Kind: symbol.KindVariable, Type: types.F64()}))
			Expect(scopeY).ToNot(BeNil())
			scopeZ := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "z", Kind: symbol.KindVariable, Type: types.F64()}))
			Expect(scopeZ).ToNot(BeNil())
			bytecode, exprType := compileWithCtx(ctx, "(x + y) * z")
			Expect(bytecode).To(MatchOpcodes(
				OpLocalGet, 0, // Resolve 'x'
				OpLocalGet, 1, // Resolve 'y'
				OpF64Add,      // x + y
				OpLocalGet, 2, // Resolve 'z'
				OpF64Mul, // (x + y) * z
			))
			Expect(exprType).To(Equal(types.F64()))
		})

		It("Should compile comparisons using local variables", func() {
			ctx := NewContext(bCtx)
			scopeLimit := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "limit", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scopeLimit).ToNot(BeNil())
			scopeValue := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "value", Kind: symbol.KindVariable, Type: types.I32()}))
			Expect(scopeValue).ToNot(BeNil())
			bytecode, exprType := compileWithCtx(ctx, "value > limit")
			Expect(bytecode).To(MatchOpcodes(
				OpLocalGet, 1, // Resolve 'value'
				OpLocalGet, 0, // Resolve 'limit'
				OpI32GtS, // value > limit
			))
			Expect(exprType).To(Equal(types.U8())) // Comparisons return boolean
		})

		It("Should compile logical operations with local variables", func() {
			ctx := NewContext(bCtx)
			scopeEnabled := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "enabled", Kind: symbol.KindVariable, Type: types.U8()}))
			Expect(scopeEnabled).ToNot(BeNil())
			scopeReady := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "ready", Kind: symbol.KindVariable, Type: types.U8()}))
			Expect(scopeReady).ToNot(BeNil())
			bytecode, exprType := compileWithCtx(ctx, "enabled and ready")
			Expect(bytecode).To(MatchOpcodes(
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
			))
			Expect(exprType).To(Equal(types.U8()))
		})
	})

	Context("Channel Reads", func() {
		It("Should compile a channel read", func() {
			ctx := NewContext(bCtx)
			scope := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "x", Kind: symbol.KindChannel, Type: types.Chan(types.I32())}))
			Expect(scope).ToNot(BeNil())
			byteCode, exprType := compileWithCtx(ctx, "x")
			i := ctx.Imports.ChannelRead["i32"]
			Expect(exprType).To(Equal(types.I32()))
			Expect(byteCode).To(MatchOpcodes(
				OpI32Const,
				int32(0),
				OpCall,
				uint64(i),
			))
		})

		It("Should correctly compile a channel read inside of an addition expression", func() {
			ctx := NewContext(bCtx)
			scope := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "x", Kind: symbol.KindChannel, Type: types.Chan(types.I32())}))
			Expect(scope).ToNot(BeNil())
			byteCode, exprType := compileWithCtx(ctx, "x + 1")
			i := ctx.Imports.ChannelRead["i32"]
			Expect(exprType).To(Equal(types.I32()))
			Expect(byteCode).To(MatchOpcodes(
				OpI32Const,
				int32(0),
				OpCall,
				uint64(i),
				OpI32Const,
				int32(1),
				OpI32Add,
			))
		})
	})

	Context("Function Parameters", func() {
		It("Should compile parameter references", func() {
			ctx := NewContext(bCtx)
			scope := MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "value",
				Kind: symbol.KindInput,
				Type: types.F64(),
			}))
			Expect(scope).ToNot(BeNil())
			bytecode, exprType := compileWithCtx(ctx, "value")
			Expect(bytecode).To(MatchOpcodes(OpLocalGet, 0))
			Expect(exprType).To(Equal(types.F64()))
		})
	})

	Context("User-Defined Function Calls", func() {
		It("Should compile a simple function call with no arguments", func() {
			ctx := NewContext(bCtx)
			ctx.FunctionIndices = map[string]uint32{"getVal": 5}

			funcType := types.Function(types.FunctionProperties{
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
			})
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "getVal",
				Kind: symbol.KindFunction,
				Type: funcType,
			}))

			byteCode, exprType := compileWithCtx(ctx, "getVal()")
			Expect(byteCode).To(MatchOpcodes(OpCall, uint32(5)))
			Expect(exprType).To(Equal(types.I64()))
		})

		It("Should compile a function call with arguments", func() {
			ctx := NewContext(bCtx)
			ctx.FunctionIndices = map[string]uint32{"add": 3}

			funcType := types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "a", Type: types.I64()}, {Name: "b", Type: types.I64()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
			})
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name: "add",
				Kind: symbol.KindFunction,
				Type: funcType,
			}))

			byteCode, exprType := compileWithCtx(ctx, "add(10, 32)")
			Expect(byteCode).To(MatchOpcodes(
				OpI64Const, int64(10),
				OpI64Const, int64(32),
				OpCall, uint32(3),
			))
			Expect(exprType).To(Equal(types.I64()))
		})

		It("Should compile nested function calls", func() {
			ctx := NewContext(bCtx)
			ctx.FunctionIndices = map[string]uint32{"inner": 2, "outer": 3}

			innerType := types.Function(types.FunctionProperties{
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
			})
			outerType := types.Function(types.FunctionProperties{
				Inputs:  types.Params{{Name: "x", Type: types.I64()}},
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
			})
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "inner", Kind: symbol.KindFunction, Type: innerType}))
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{Name: "outer", Kind: symbol.KindFunction, Type: outerType}))

			byteCode, exprType := compileWithCtx(ctx, "outer(inner())")
			Expect(byteCode).To(MatchOpcodes(
				OpCall, uint32(2), // inner() called first
				OpCall, uint32(3), // outer() called with result
			))
			Expect(exprType).To(Equal(types.I64()))
		})

		It("Should compile function call in binary expression", func() {
			ctx := NewContext(bCtx)
			ctx.FunctionIndices = map[string]uint32{"getValue": 7}

			funcType := types.Function(types.FunctionProperties{
				Outputs: types.Params{{Name: ir.DefaultOutputParam, Type: types.I64()}},
			})
			MustSucceed(ctx.Scope.Add(ctx, symbol.Symbol{
				Name:    "getValue",
				Kind:    symbol.KindFunction,
				Type:    funcType,
			}))

			byteCode, exprType := compileWithCtx(ctx, "getValue() + 5")
			Expect(byteCode).To(MatchOpcodes(
				OpCall, uint32(7),
				OpI64Const, int64(5),
				OpI64Add,
			))
			Expect(exprType).To(Equal(types.I64()))
		})
	})
})
