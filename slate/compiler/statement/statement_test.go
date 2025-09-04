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
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/compiler/statement"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	"github.com/synnaxlabs/slate/types"
	. "github.com/synnaxlabs/x/testutil"
)

func TestStatement(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Statement Compiler Suite")
}

// WASM builds WASM bytecode from a variadic slice of opcodes and operands
func WASM(instructions ...any) []byte {
	encoder := wasm.NewEncoder()

	for i := 0; i < len(instructions); i++ {
		switch instr := instructions[i].(type) {
		case wasm.Opcode:
			switch instr {
			case wasm.OpI32Const:
				encoder.WriteI32Const(instructions[i+1].(int32))
				i++ // Skip the operand
			case wasm.OpI64Const:
				encoder.WriteI64Const(instructions[i+1].(int64))
				i++ // Skip the operand
			case wasm.OpF32Const:
				encoder.WriteF32Const(instructions[i+1].(float32))
				i++ // Skip the operand
			case wasm.OpF64Const:
				encoder.WriteF64Const(instructions[i+1].(float64))
				i++ // Skip the operand
			case wasm.OpLocalGet:
				encoder.WriteLocalGet(instructions[i+1].(uint32))
				i++ // Skip the operand
			case wasm.OpLocalSet:
				encoder.WriteLocalSet(instructions[i+1].(uint32))
				i++ // Skip the operand
			case wasm.OpIf:
				// Check if there's a block type following
				if i+1 < len(instructions) {
					if bt, ok := instructions[i+1].(wasm.BlockType); ok {
						encoder.WriteIf(bt)
						i++ // Skip the block type
					} else {
						// Default to empty block
						encoder.WriteIf(wasm.BlockTypeEmpty)
					}
				} else {
					encoder.WriteIf(wasm.BlockTypeEmpty)
				}
			default:
				encoder.WriteOpcode(instr)
			}
		case int:
			// Handle plain integers as i32 constants
			encoder.WriteI32Const(int32(instr))
		case string:
			// Could be used for comments, skip
		default:
			panic("Unexpected instruction type in WASM builder")
		}
	}

	return encoder.Bytes()
}

var _ = Describe("Statement Compiler", func() {
	var (
		ctx      *compiler.Context
		stmtComp *statement.Compiler
		symbols  *symbol.Scope
		module   *wasm.Module
	)

	BeforeEach(func() {
		module = wasm.NewModule()
		symbols = &symbol.Scope{}
		ctx = compiler.NewContext(module, symbols)
		ctx.EnterFunction("test", nil)
		stmtComp = statement.NewCompiler(ctx)
	})

	Describe("Variable Declarations", func() {
		It("should compile local variable declaration", func() {
			// Add symbol for the variable
			MustSucceed(symbols.AddSymbol("x", symbol.KindVariable, types.I32{}, nil))
			// Parse statement: x := 42
			stmt := MustSucceed(parser.ParseStatement("x := 42"))
			Expect(stmtComp.CompileStatement(stmt)).To(Succeed())
			
			// Expected bytecode:
			// i64.const 42
			// i32.wrap_i64
			// local.set 0
			expected := WASM(
				wasm.OpI64Const, int64(42),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, uint32(0),
			)
			Expect(stmtComp.Bytes()).To(Equal(expected))
		})

		It("should compile variable assignment", func() {
			// Add symbol and allocate local
			MustSucceed(symbols.AddSymbol("x", symbol.KindVariable, types.I32{}, nil))
			ctx.AllocateLocal("x", wasm.I32)

			// Parse assignment: x = 100
			stmt := MustSucceed(parser.ParseStatement("x = 100"))

			Expect(stmtComp.CompileStatement(stmt)).To(Succeed())

			// Expected bytecode:
			// i64.const 100
			// i32.wrap_i64
			// local.set 0
			expected := WASM(
				wasm.OpI64Const, int64(100),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, uint32(0),
			)
			Expect(stmtComp.Bytes()).To(Equal(expected))
		})
	})

	Describe("Control Flow", func() {
		It("should compile if statement", func() {
			// Add symbol for condition variable
			MustSucceed(symbols.AddSymbol("flag", symbol.KindVariable, types.U8{}, nil))
			MustSucceed(symbols.AddSymbol("x", symbol.KindVariable, types.I32{}, nil))
			ctx.AllocateLocal("flag", wasm.I32)
			ctx.AllocateLocal("x", wasm.I32)

			// Parse if statement
			stmt := MustSucceed(parser.ParseStatement("if flag { x = 1 }"))

			Expect(stmtComp.CompileStatement(stmt)).To(Succeed())

			// Expected bytecode:
			// local.get 0 (flag)
			// if BlockTypeEmpty
			//   i64.const 1
			//   i32.wrap_i64
			//   local.set 1 (x)
			// end
			expected := WASM(
				wasm.OpLocalGet, uint32(0),
				wasm.OpIf, wasm.BlockTypeEmpty,
				wasm.OpI64Const, int64(1),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, uint32(1),
				wasm.OpEnd,
			)
			Expect(stmtComp.Bytes()).To(Equal(expected))
		})

		It("should compile if-else statement", func() {
			// Add symbols
			MustSucceed(symbols.AddSymbol("cond", symbol.KindVariable, types.U8{}, nil))
			MustSucceed(symbols.AddSymbol("result", symbol.KindVariable, types.I32{}, nil))
			ctx.AllocateLocal("cond", wasm.I32)
			ctx.AllocateLocal("result", wasm.I32)

			// Parse if-else statement
			code := `if cond {
				result = 1
			} else {
				result = 0
			}`
			stmt := MustSucceed(parser.ParseStatement(code))

			Expect(stmtComp.CompileStatement(stmt)).To(Succeed())

			// Check bytecode contains if, else, and end
			bytecode := stmtComp.Bytes()
			Expect(bytecode).To(ContainElement(byte(wasm.OpIf)))
			Expect(bytecode).To(ContainElement(byte(wasm.OpElse)))
			Expect(bytecode).To(ContainElement(byte(wasm.OpEnd)))
		})

		It("should compile return statement with value", func() {
			// Set function return type
			ctx.Current.ReturnType = types.I32{}

			// Parse return statement
			stmt := MustSucceed(parser.ParseStatement("return 42"))

			Expect(stmtComp.CompileStatement(stmt)).To(Succeed())

			// Expected bytecode:
			// i64.const 42
			// i32.wrap_i64
			// return
			expected := WASM(
				wasm.OpI64Const, int64(42),
				wasm.OpI32WrapI64,
				wasm.OpReturn,
			)
			Expect(stmtComp.Bytes()).To(Equal(expected))
		})

		It("should compile void return statement", func() {
			// Parse return statement
			stmt := MustSucceed(parser.ParseStatement("return"))

			Expect(stmtComp.CompileStatement(stmt)).To(Succeed())

			// Expected bytecode:
			// return
			expected := WASM(wasm.OpReturn)
			Expect(stmtComp.Bytes()).To(Equal(expected))
		})
	})

	Describe("Block Compilation", func() {
		It("should compile multiple statements in a block", func() {
			// Add symbols
			MustSucceed(symbols.AddSymbol("x", symbol.KindVariable, types.I32{}, nil))
			MustSucceed(symbols.AddSymbol("y", symbol.KindVariable, types.I32{}, nil))

			// Parse a block with multiple statements
			code := `{
				x := 10
				y := 20
				x = x + y
			}`
			block := MustSucceed(parser.ParseBlock(code))

			// Compile the block
			Expect(stmtComp.CompileBlock(block)).To(Succeed())

			// Expected bytecode:
			// x := 10
			// y := 20
			// x = x + y
			expected := WASM(
				wasm.OpI64Const, int64(10),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, uint32(0), // x
				wasm.OpI64Const, int64(20),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, uint32(1), // y
				wasm.OpLocalGet, uint32(0), // x
				wasm.OpLocalGet, uint32(1), // y
				wasm.OpI32Add,
				wasm.OpLocalSet, uint32(0), // x
			)
			Expect(stmtComp.Bytes()).To(Equal(expected))
		})
		
		It("should compile if-else with exact bytecode", func() {
			// Add symbols
			MustSucceed(symbols.AddSymbol("cond", symbol.KindVariable, types.U8{}, nil))
			MustSucceed(symbols.AddSymbol("result", symbol.KindVariable, types.I32{}, nil))
			ctx.AllocateLocal("cond", wasm.I32)
			ctx.AllocateLocal("result", wasm.I32)
			
			// Parse if-else statement
			code := `if cond {
				result = 100
			} else {
				result = 200
			}`
			stmt := MustSucceed(parser.ParseStatement(code))
			
			Expect(stmtComp.CompileStatement(stmt)).To(Succeed())
			
			// Expected bytecode with WASM utility
			expected := WASM(
				wasm.OpLocalGet, uint32(0), // cond
				wasm.OpIf, wasm.BlockTypeEmpty,
				wasm.OpI64Const, int64(100),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, uint32(1), // result = 100
				wasm.OpElse,
				wasm.OpI64Const, int64(200),
				wasm.OpI32WrapI64,
				wasm.OpLocalSet, uint32(1), // result = 200
				wasm.OpEnd,
			)
			Expect(stmtComp.Bytes()).To(Equal(expected))
		})
	})
})
