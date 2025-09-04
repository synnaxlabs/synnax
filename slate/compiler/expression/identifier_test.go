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
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/compiler/expression"
	. "github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Expression Compiler", func() {
	Describe("Identifier Compilation", func() {
		Context("Local Variables", func() {
			It("Should compile local variable references", func() {
				module := NewModule()
				symbols := &symbol.Scope{}
				MustSucceed(symbols.AddSymbol(
					"x",
					symbol.KindVariable,
					types.I32{},
					nil,
				))
				ctx := compiler.NewContext(module, symbols)
				ctx.EnterFunction("test", nil)
				ctx.AllocateLocal("x", I32)
				expr := MustSucceed(parser.ParseExpression("x"))
				compiler := expression.NewCompiler(ctx)
				exprType := MustSucceed(compiler.Compile(expr))
				bytecode := compiler.Bytes()
				expected := WASM(
					OpLocalGet, uint32(0),
				)
				Expect(bytecode).To(Equal(expected))
				Expect(exprType).To(Equal(types.I32{}))
			})
			
			It("Should compile expressions using multiple local variables", func() {
				module := NewModule()
				symbols := &symbol.Scope{}
				
				// Add multiple variables to symbol table
				MustSucceed(symbols.AddSymbol("a", symbol.KindVariable, types.I32{}, nil))
				MustSucceed(symbols.AddSymbol("b", symbol.KindVariable, types.I32{}, nil))
				
				ctx := compiler.NewContext(module, symbols)
				ctx.EnterFunction("test", nil)
				
				// Allocate locals for both variables
				ctx.AllocateLocal("a", I32)
				ctx.AllocateLocal("b", I32)
				
				// Compile expression using both variables
				expr := MustSucceed(parser.ParseExpression("a + b"))
				compiler := expression.NewCompiler(ctx)
				exprType := MustSucceed(compiler.Compile(expr))
				
				bytecode := compiler.Bytes()
				expected := WASM(
					OpLocalGet, uint32(0),  // Get 'a'
					OpLocalGet, uint32(1),  // Get 'b'
					OpI32Add,               // Add them
				)
				Expect(bytecode).To(Equal(expected))
				Expect(exprType).To(Equal(types.I32{}))
			})
			
			It("Should compile complex expressions with local variables", func() {
				module := NewModule()
				symbols := &symbol.Scope{}
				
				// Add variables with different types
				MustSucceed(symbols.AddSymbol("x", symbol.KindVariable, types.F64{}, nil))
				MustSucceed(symbols.AddSymbol("y", symbol.KindVariable, types.F64{}, nil))
				MustSucceed(symbols.AddSymbol("z", symbol.KindVariable, types.F64{}, nil))
				
				ctx := compiler.NewContext(module, symbols)
				ctx.EnterFunction("test", nil)
				
				// Allocate locals
				ctx.AllocateLocal("x", F64)
				ctx.AllocateLocal("y", F64)
				ctx.AllocateLocal("z", F64)
				
				// Compile complex expression: (x + y) * z
				expr := MustSucceed(parser.ParseExpression("(x + y) * z"))
				compiler := expression.NewCompiler(ctx)
				exprType := MustSucceed(compiler.Compile(expr))
				
				bytecode := compiler.Bytes()
				expected := WASM(
					OpLocalGet, uint32(0),  // Get 'x'
					OpLocalGet, uint32(1),  // Get 'y'
					OpF64Add,               // x + y
					OpLocalGet, uint32(2),  // Get 'z'
					OpF64Mul,               // (x + y) * z
				)
				Expect(bytecode).To(Equal(expected))
				Expect(exprType).To(Equal(types.F64{}))
			})
			
			It("Should compile comparisons using local variables", func() {
				module := NewModule()
				symbols := &symbol.Scope{}
				
				MustSucceed(symbols.AddSymbol("limit", symbol.KindVariable, types.I32{}, nil))
				MustSucceed(symbols.AddSymbol("value", symbol.KindVariable, types.I32{}, nil))
				
				ctx := compiler.NewContext(module, symbols)
				ctx.EnterFunction("test", nil)
				
				ctx.AllocateLocal("limit", I32)
				ctx.AllocateLocal("value", I32)
				
				// Compile comparison: value > limit
				expr := MustSucceed(parser.ParseExpression("value > limit"))
				compiler := expression.NewCompiler(ctx)
				exprType := MustSucceed(compiler.Compile(expr))
				
				bytecode := compiler.Bytes()
				expected := WASM(
					OpLocalGet, uint32(1),  // Get 'value'
					OpLocalGet, uint32(0),  // Get 'limit'
					OpI32GtS,               // value > limit
				)
				Expect(bytecode).To(Equal(expected))
				Expect(exprType).To(Equal(types.U8{}))  // Comparisons return boolean
			})
			
			It("Should compile logical operations with local variables", func() {
				module := NewModule()
				symbols := &symbol.Scope{}
				
				MustSucceed(symbols.AddSymbol("enabled", symbol.KindVariable, types.U8{}, nil))
				MustSucceed(symbols.AddSymbol("ready", symbol.KindVariable, types.U8{}, nil))
				
				ctx := compiler.NewContext(module, symbols)
				ctx.EnterFunction("test", nil)
				
				ctx.AllocateLocal("enabled", I32)  // U8 uses I32 in WASM
				ctx.AllocateLocal("ready", I32)
				
				// Compile: enabled && ready
				expr := MustSucceed(parser.ParseExpression("enabled && ready"))
				compiler := expression.NewCompiler(ctx)
				exprType := MustSucceed(compiler.Compile(expr))
				
				// Exact bytecode for: enabled && ready with short-circuit evaluation
				bytecode := compiler.Bytes()
				expected := WASM(
					// Load 'enabled'
					OpLocalGet, uint32(0),
					// Normalize to boolean (0 or 1)
					OpI32Const, int32(0),
					OpI32Ne,
					// Check if zero for short-circuit
					OpI32Eqz,
					OpIf, byte(I32),  // If enabled is false (0)
					OpI32Const, int32(0),  // Result is 0
					OpElse,
					// enabled was true, evaluate 'ready'
					OpLocalGet, uint32(1),
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
				module := NewModule()
				symbols := &symbol.Scope{}
				MustSucceed(symbols.AddSymbol(
					"value",
					symbol.KindParam,
					types.F64{},
					nil,
				))
				ctx := compiler.NewContext(module, symbols)
				ctx.EnterFunction("test", nil)
				ctx.AllocateLocal("value", F64)
				expr := MustSucceed(parser.ParseExpression("value"))
				compiler := expression.NewCompiler(ctx)
				exprType := MustSucceed(compiler.Compile(expr))
				bytecode := compiler.Bytes()
				expected := WASM(
					OpLocalGet, uint32(0),
				)
				Expect(bytecode).To(Equal(expected))
				Expect(exprType).To(Equal(types.F64{}))
			})
		})
	})
})
