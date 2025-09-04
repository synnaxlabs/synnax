// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package statement_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/compiler/core"
	"github.com/synnaxlabs/slate/compiler/statement"
	. "github.com/synnaxlabs/slate/compiler/testutil"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/symbol"
	"github.com/synnaxlabs/slate/types"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Statement Compiler", func() {
	var (
		ctx        *core.Context
		returnType types.Type
	)

	JustBeforeEach(func() {
		ctx = NewContextWithFunctionType(types.Function{Return: returnType})
	})

	Describe("Variable Declarations", func() {
		It("should compile local variable declaration", func() {
			MustSucceed(ctx.Scope.AddSymbol("x", symbol.KindVariable, types.I32{}, nil))
			stmt := MustSucceed(parser.ParseStatement("x := 42"))
			Expect(statement.Compile(ctx, stmt)).To(Succeed())
			expected := WASM(
				wasm.OpI64Const, int64(42),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, 0,
			)
			Expect(ctx.Writer.Bytes()).To(Equal(expected))
		})

		It("should compile variable assignment", func() {
			// Add symbol and allocate local
			MustSucceed(ctx.Scope.AddSymbol("x", symbol.KindVariable, types.I32{}, nil))
			// Parse assignment: x = 100
			stmt := MustSucceed(parser.ParseStatement("x = 100"))
			Expect(statement.Compile(ctx, stmt)).To(Succeed())

			// Expected bytecode:
			// i64.const 100
			// i32.wrap_i64
			// local.set 0
			expected := WASM(
				wasm.OpI64Const, int64(100),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, 0,
			)
			Expect(ctx.Writer.Bytes()).To(Equal(expected))
		})
	})

	Describe("Control Flow", func() {
		It("should compile if statement", func() {
			// Add symbol for condition variable
			MustSucceed(ctx.Scope.AddSymbol("flag", symbol.KindVariable, types.U8{}, nil))
			MustSucceed(ctx.Scope.AddSymbol("x", symbol.KindVariable, types.I32{}, nil))
			// Parse if statement
			stmt := MustSucceed(parser.ParseStatement("if flag { x = 1 }"))
			Expect(statement.Compile(ctx, stmt)).To(Succeed())
			// Expected bytecode:
			// local.get 0 (flag)
			// if BlockTypeEmpty
			//   i64.const 1
			//   i32.wrap_i64
			//   local.set 1 (x)
			// end
			expected := WASM(
				wasm.OpLocalGet, 0,
				wasm.OpIf, wasm.BlockTypeEmpty,
				wasm.OpI64Const, int64(1),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, 1,
				wasm.OpEnd,
			)
			Expect(ctx.Writer.Bytes()).To(Equal(expected))
		})

		It("should compile if-else statement", func() {
			// Add scope
			MustSucceed(ctx.Scope.AddSymbol("cond", symbol.KindVariable, types.U8{}, nil))
			MustSucceed(ctx.Scope.AddSymbol("result", symbol.KindVariable, types.I32{}, nil))
			// Parse if-else statement
			code := `if cond {
				result = 1
			} else {
				result = 0
			}`
			stmt := MustSucceed(parser.ParseStatement(code))
			Expect(statement.Compile(ctx, stmt)).To(Succeed())
			// Check bytecode contains if, else, and end
			bytecode := ctx.Writer.Bytes()
			Expect(bytecode).To(ContainElement(byte(wasm.OpIf)))
			Expect(bytecode).To(ContainElement(byte(wasm.OpElse)))
			Expect(bytecode).To(ContainElement(byte(wasm.OpEnd)))
		})

		Describe("Return", func() {
			BeforeEach(func() {
				returnType = types.I32{}
			})
			It("should compile return statement with value", func() {
				stmt := MustSucceed(parser.ParseStatement("return 42"))
				Expect(statement.Compile(ctx, stmt)).To(Succeed())
				expected := WASM(
					wasm.OpI64Const, int64(42),
					wasm.OpI32WrapI64,
					wasm.OpReturn,
				)
				Expect(ctx.Writer.Bytes()).To(Equal(expected))
			})
		})

		It("should compile void return statement", func() {
			// Parse return statement
			stmt := MustSucceed(parser.ParseStatement("return"))
			Expect(statement.Compile(ctx, stmt)).To(Succeed())
			// Expected bytecode:
			// return
			expected := WASM(wasm.OpReturn)
			Expect(ctx.Writer.Bytes()).To(Equal(expected))
		})
	})

	Describe("Block Compilation", func() {
		It("should compile multiple statements in a block", func() {
			// Add scope
			MustSucceed(ctx.Scope.AddSymbol("x", symbol.KindVariable, types.I32{}, nil))
			MustSucceed(ctx.Scope.AddSymbol("y", symbol.KindVariable, types.I32{}, nil))

			// Parse a block with multiple statements
			code := `{
				x := 10
				y := 20
				x = x + y
			}`
			block := MustSucceed(parser.ParseBlock(code))
			// Compile the block
			Expect(statement.CompileBlock(ctx, block)).To(Succeed())
			// Expected bytecode:
			// x := 10
			// y := 20
			// x = x + y
			expected := WASM(
				wasm.OpI64Const, int64(10),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, 0, // x
				wasm.OpI64Const, int64(20),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, 1, // y
				wasm.OpLocalGet, 0, // x
				wasm.OpLocalGet, 1, // y
				wasm.OpI32Add,
				wasm.OpLocalSet, 0, // x
			)
			Expect(ctx.Writer.Bytes()).To(Equal(expected))
		})

		It("should compile if-else with exact bytecode", func() {
			// Add scope
			MustSucceed(ctx.Scope.AddSymbol("cond", symbol.KindVariable, types.U8{}, nil))
			MustSucceed(ctx.Scope.AddSymbol("result", symbol.KindVariable, types.I32{}, nil))
			// Parse if-else statement
			code := `if cond {
				result = 100
			} else {
				result = 200
			}`
			stmt := MustSucceed(parser.ParseStatement(code))

			Expect(statement.Compile(ctx, stmt)).To(Succeed())

			// Expected bytecode with WASM utility
			expected := WASM(
				wasm.OpLocalGet, 0, // cond
				wasm.OpIf, wasm.BlockTypeEmpty,
				wasm.OpI64Const, int64(100),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, 1, // result = 100
				wasm.OpElse,
				wasm.OpI64Const, int64(200),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, 1, // result = 200
				wasm.OpEnd,
			)
			Expect(ctx.Writer.Bytes()).To(Equal(expected))
		})
	})
})
