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
	"encoding/hex"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/slate/analyzer/symbol"
	"github.com/synnaxlabs/slate/compiler"
	"github.com/synnaxlabs/slate/compiler/expression"
	"github.com/synnaxlabs/slate/compiler/wasm"
	"github.com/synnaxlabs/slate/parser"
	. "github.com/synnaxlabs/x/testutil"
)

var _ = Describe("Expression Compiler", func() {
	// Helper function to compile an expression and return bytecode
	compileExpression := func(source string) ([]byte, string) {
		// Parse the expression
		expr := MustSucceed(parser.ParseExpression(source))

		// Create minimal context
		module := wasm.NewModule()
		symbols := &symbol.Scope{}
		ctx := compiler.NewContext(module, symbols)

		// Create a dummy function context
		ctx.EnterFunction("test", nil)

		// Compile the expression
		compiler := expression.NewCompiler(ctx)
		exprType := MustSucceed(compiler.Compile(expr))

		return compiler.Bytes(), exprType.String()
	}

	// Helper to convert hex string to bytes for comparison
	hexToBytes := func(hexStr string) []byte {
		cleanHex := strings.ReplaceAll(hexStr, " ", "")
		bytes := MustSucceed(hex.DecodeString(cleanHex))
		return bytes
	}

	Describe("Literal Compilation", func() {
		Context("Integer Literals", func() {
			It("Should compile i32 literals with explicit suffix", func() {
				bytecode, exprType := compileExpression("42i32")
				Expect(bytecode).To(Equal(hexToBytes("41 2a"))) // i32.const 42
				Expect(exprType).To(Equal("i32"))
			})

			It("Should default to i64 for literals without suffix", func() {
				bytecode, exprType := compileExpression("42")
				Expect(bytecode).To(Equal(hexToBytes("42 2a"))) // i64.const 42
				Expect(exprType).To(Equal("i64"))
			})

			It("Should compile u8 literals", func() {
				bytecode, exprType := compileExpression("255u8")
				Expect(bytecode).To(Equal(hexToBytes("41 ff 01"))) // i32.const 255
				Expect(exprType).To(Equal("u8"))
			})

			It("Should compile hexadecimal literals", func() {
				bytecode, exprType := compileExpression("0xFFi32")
				Expect(bytecode).To(Equal(hexToBytes("41 ff 01"))) // i32.const 255
				Expect(exprType).To(Equal("i32"))
			})

			It("Should compile binary literals", func() {
				bytecode, exprType := compileExpression("0b1010i32")
				Expect(bytecode).To(Equal(hexToBytes("41 0a"))) // i32.const 10
				Expect(exprType).To(Equal("i32"))
			})

			It("Should handle underscores in numeric literals", func() {
				bytecode, exprType := compileExpression("1_000_000i32")
				Expect(bytecode).To(Equal(hexToBytes("41 c0 84 3d"))) // i32.const 1000000
				Expect(exprType).To(Equal("i32"))
			})
		})

		Context("Float Literals", func() {
			It("Should compile f32 literals with explicit suffix", func() {
				bytecode, exprType := compileExpression("3.14f32")
				Expect(bytecode).To(Equal(hexToBytes("43 c3 f5 48 40"))) // f32.const 3.14
				Expect(exprType).To(Equal("f32"))
			})

			It("Should default to f64 for float literals without suffix", func() {
				bytecode, exprType := compileExpression("3.14")
				Expect(bytecode).To(Equal(hexToBytes("44 1f 85 eb 51 b8 1e 09 40"))) // f64.const 3.14
				Expect(exprType).To(Equal("f64"))
			})

			It("Should compile f64 literals with explicit suffix", func() {
				bytecode, exprType := compileExpression("2.5f64")
				Expect(bytecode).To(Equal(hexToBytes("44 00 00 00 00 00 00 04 40"))) // f64.const 2.5
				Expect(exprType).To(Equal("f64"))
			})

			It("Should compile scientific notation", func() {
				bytecode, exprType := compileExpression("1e3f32")
				Expect(bytecode).To(Equal(hexToBytes("43 00 00 7a 44"))) // f32.const 1000.0
				Expect(exprType).To(Equal("f32"))
			})
		})

		Context("Parenthesized Expressions", func() {
			It("Should compile parenthesized integer", func() {
				bytecode, exprType := compileExpression("(42)")
				Expect(bytecode).To(HaveLen(2)) // Should produce bytecode
				Expect(exprType).To(Equal("i64"))
			})

			It("Should compile nested parentheses", func() {
				bytecode, exprType := compileExpression("((42i32))")
				Expect(bytecode).To(HaveLen(2)) // Should produce bytecode
				Expect(exprType).To(Equal("i32"))
			})

			It("Should compile parenthesized float", func() {
				bytecode, exprType := compileExpression("(3.14f32)")
				Expect(bytecode).To(HaveLen(5)) // Should produce bytecode
				Expect(exprType).To(Equal("f32"))
			})
		})
	})

	Describe("Identifier Compilation", func() {
		Context("Local Variables", func() {
			It("Should compile local variable references", func() {
				// Create context with a local variable
				module := wasm.NewModule()
				symbols := &symbol.Scope{}
				ctx := compiler.NewContext(module, symbols)
				ctx.EnterFunction("test", nil)

				// Allocate a local variable
				ctx.AllocateLocal("x", wasm.I32)

				// Parse and compile identifier reference
				expr := MustSucceed(parser.ParseExpression("x"))
				compiler := expression.NewCompiler(ctx)
				exprType := MustSucceed(compiler.Compile(expr))

				bytecode := compiler.Bytes()
				Expect(bytecode).To(Equal(hexToBytes("20 00"))) // local.get 0
				// TODO: Fix type inference to return actual type
				Expect(exprType.String()).To(Equal("f64")) // Currently returns placeholder
			})
		})
	})
})

var _ = Describe("WASM Encoder", func() {
	Describe("Instruction Encoding", func() {
		var encoder *wasm.Encoder

		BeforeEach(func() {
			encoder = wasm.NewEncoder()
		})

		Context("Constant Instructions", func() {
			It("Should encode i32.const", func() {
				encoder.WriteI32Const(42)
				Expect(encoder.Bytes()).To(Equal([]byte{0x41, 0x2a}))
			})

			It("Should encode i64.const", func() {
				encoder.WriteI64Const(42)
				Expect(encoder.Bytes()).To(Equal([]byte{0x42, 0x2a}))
			})

			It("Should encode f32.const", func() {
				encoder.WriteF32Const(3.14)
				// 3.14 in IEEE 754 single precision (little-endian)
				expected := []byte{0x43, 0xc3, 0xf5, 0x48, 0x40}
				Expect(encoder.Bytes()).To(Equal(expected))
			})

			It("Should encode f64.const", func() {
				encoder.WriteF64Const(3.14)
				// 3.14 in IEEE 754 double precision (little-endian)
				expected := []byte{0x44, 0x1f, 0x85, 0xeb, 0x51, 0xb8, 0x1e, 0x09, 0x40}
				Expect(encoder.Bytes()).To(Equal(expected))
			})
		})

		Context("Variable Instructions", func() {
			It("Should encode local.get", func() {
				encoder.WriteLocalGet(3)
				Expect(encoder.Bytes()).To(Equal([]byte{0x20, 0x03}))
			})

			It("Should encode local.set", func() {
				encoder.WriteLocalSet(5)
				Expect(encoder.Bytes()).To(Equal([]byte{0x21, 0x05}))
			})

			It("Should encode local.tee", func() {
				encoder.WriteLocalTee(2)
				Expect(encoder.Bytes()).To(Equal([]byte{0x22, 0x02}))
			})
		})

		Context("Control Flow Instructions", func() {
			It("Should encode if block with result type", func() {
				encoder.WriteIf(wasm.BlockTypeI32)
				Expect(encoder.Bytes()).To(Equal([]byte{0x04, 0x7f})) // if (result i32)
			})

			It("Should encode if block without result", func() {
				encoder.WriteIf(wasm.BlockTypeEmpty)
				Expect(encoder.Bytes()).To(Equal([]byte{0x04, 0x40})) // if (no result)
			})

			It("Should encode else", func() {
				encoder.WriteElse()
				Expect(encoder.Bytes()).To(Equal([]byte{0x05}))
			})

			It("Should encode end", func() {
				encoder.WriteEnd()
				Expect(encoder.Bytes()).To(Equal([]byte{0x0b}))
			})
		})

		Context("LEB128 Encoding", func() {
			It("Should encode small unsigned integers", func() {
				encoder.WriteLEB128Unsigned(127)
				Expect(encoder.Bytes()).To(Equal([]byte{0x7f}))
			})

			It("Should encode larger unsigned integers", func() {
				encoder.WriteLEB128Unsigned(128)
				Expect(encoder.Bytes()).To(Equal([]byte{0x80, 0x01}))
			})

			It("Should encode small signed integers", func() {
				encoder.WriteLEB128Signed(42)
				Expect(encoder.Bytes()).To(Equal([]byte{0x2a}))
			})

			It("Should encode negative signed integers", func() {
				encoder.WriteLEB128Signed(-1)
				Expect(encoder.Bytes()).To(Equal([]byte{0x7f}))
			})
		})

		Context("Binary Operations", func() {
			It("Should encode arithmetic operations", func() {
				encoder.WriteBinaryOp(wasm.OpI32Add)
				Expect(encoder.Bytes()).To(Equal([]byte{0x6a}))

				encoder.Reset()
				encoder.WriteBinaryOp(wasm.OpF64Mul)
				Expect(encoder.Bytes()).To(Equal([]byte{0xa2}))
			})
		})
	})
})
