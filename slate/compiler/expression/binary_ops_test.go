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

var _ = Describe("Binary Operations", func() {
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

	Describe("Arithmetic Operations", func() {
		Context("Addition", func() {
			It("Should compile i32 addition", func() {
				bytecode, exprType := compileExpression("10i32 + 20i32")
				// i32.const 10, i32.const 20, i32.add
				Expect(bytecode).To(Equal(hexToBytes("41 0a 41 14 6a")))
				Expect(exprType).To(Equal("i32"))
			})

			It("Should compile i64 addition", func() {
				bytecode, exprType := compileExpression("100 + 200")
				// i64.const 100, i64.const 200, i64.add
				Expect(bytecode).To(Equal(hexToBytes("42 e4 00 42 c8 01 7c")))
				Expect(exprType).To(Equal("i64"))
			})

			It("Should compile f32 addition", func() {
				bytecode, exprType := compileExpression("1.5f32 + 2.5f32")
				// f32.const 1.5, f32.const 2.5, f32.add
				Expect(bytecode).To(Equal(hexToBytes("43 00 00 c0 3f 43 00 00 20 40 92")))
				Expect(exprType).To(Equal("f32"))
			})

			It("Should compile f64 addition", func() {
				bytecode, exprType := compileExpression("1.5 + 2.5")
				// f64.const 1.5, f64.const 2.5, f64.add
				Expect(bytecode).To(Equal(hexToBytes("44 00 00 00 00 00 00 f8 3f 44 00 00 00 00 00 00 04 40 a0")))
				Expect(exprType).To(Equal("f64"))
			})

			It("Should compile multiple additions (left-associative)", func() {
				bytecode, exprType := compileExpression("1i32 + 2i32 + 3i32")
				// i32.const 1, i32.const 2, i32.add, i32.const 3, i32.add
				Expect(bytecode).To(Equal(hexToBytes("41 01 41 02 6a 41 03 6a")))
				Expect(exprType).To(Equal("i32"))
			})
		})

		Context("Subtraction", func() {
			It("Should compile i32 subtraction", func() {
				bytecode, exprType := compileExpression("20i32 - 10i32")
				// i32.const 20, i32.const 10, i32.sub
				Expect(bytecode).To(Equal(hexToBytes("41 14 41 0a 6b")))
				Expect(exprType).To(Equal("i32"))
			})

			It("Should compile f64 subtraction", func() {
				bytecode, exprType := compileExpression("5.0 - 2.0")
				// f64.const 5.0, f64.const 2.0, f64.sub
				Expect(bytecode).To(Equal(hexToBytes("44 00 00 00 00 00 00 14 40 44 00 00 00 00 00 00 00 40 a1")))
				Expect(exprType).To(Equal("f64"))
			})
		})

		Context("Multiplication", func() {
			It("Should compile i32 multiplication", func() {
				bytecode, exprType := compileExpression("3i32 * 4i32")
				// i32.const 3, i32.const 4, i32.mul
				Expect(bytecode).To(Equal(hexToBytes("41 03 41 04 6c")))
				Expect(exprType).To(Equal("i32"))
			})

			It("Should compile f64 multiplication", func() {
				bytecode, exprType := compileExpression("2.5 * 4.0")
				// f64.const 2.5, f64.const 4.0, f64.mul
				Expect(bytecode).To(Equal(hexToBytes("44 00 00 00 00 00 00 04 40 44 00 00 00 00 00 00 10 40 a2")))
				Expect(exprType).To(Equal("f64"))
			})
		})

		Context("Division", func() {
			It("Should compile signed i32 division", func() {
				bytecode, exprType := compileExpression("20i32 / 4i32")
				// i32.const 20, i32.const 4, i32.div_s
				Expect(bytecode).To(Equal(hexToBytes("41 14 41 04 6d")))
				Expect(exprType).To(Equal("i32"))
			})

			It("Should compile unsigned u32 division", func() {
				bytecode, exprType := compileExpression("20u32 / 4u32")
				// i32.const 20, i32.const 4, i32.div_u
				Expect(bytecode).To(Equal(hexToBytes("41 14 41 04 6e")))
				Expect(exprType).To(Equal("u32"))
			})

			It("Should compile f64 division", func() {
				bytecode, exprType := compileExpression("10.0 / 2.0")
				// f64.const 10.0, f64.const 2.0, f64.div
				Expect(bytecode).To(Equal(hexToBytes("44 00 00 00 00 00 00 24 40 44 00 00 00 00 00 00 00 40 a3")))
				Expect(exprType).To(Equal("f64"))
			})
		})

		Context("Modulo", func() {
			It("Should compile signed i32 modulo", func() {
				bytecode, exprType := compileExpression("17i32 % 5i32")
				// i32.const 17, i32.const 5, i32.rem_s
				Expect(bytecode).To(Equal(hexToBytes("41 11 41 05 6f")))
				Expect(exprType).To(Equal("i32"))
			})

			It("Should compile unsigned u32 modulo", func() {
				bytecode, exprType := compileExpression("17u32 % 5u32")
				// i32.const 17, i32.const 5, i32.rem_u
				Expect(bytecode).To(Equal(hexToBytes("41 11 41 05 70")))
				Expect(exprType).To(Equal("u32"))
			})
		})

		Context("Operator Precedence", func() {
			It("Should respect multiplication over addition", func() {
				bytecode, exprType := compileExpression("2i32 + 3i32 * 4i32")
				// i32.const 2, i32.const 3, i32.const 4, i32.mul, i32.add
				Expect(bytecode).To(Equal(hexToBytes("41 02 41 03 41 04 6c 6a")))
				Expect(exprType).To(Equal("i32"))
			})

			It("Should respect parentheses", func() {
				bytecode, exprType := compileExpression("(2i32 + 3i32) * 4i32")
				// i32.const 2, i32.const 3, i32.add, i32.const 4, i32.mul
				Expect(bytecode).To(Equal(hexToBytes("41 02 41 03 6a 41 04 6c")))
				Expect(exprType).To(Equal("i32"))
			})
		})
	})

	Describe("Comparison Operations", func() {
		Context("Equality", func() {
			It("Should compile i32 equality", func() {
				bytecode, exprType := compileExpression("10i32 == 10i32")
				// i32.const 10, i32.const 10, i32.eq
				Expect(bytecode).To(Equal(hexToBytes("41 0a 41 0a 46")))
				Expect(exprType).To(Equal("u8")) // Returns boolean
			})

			It("Should compile f64 equality", func() {
				bytecode, exprType := compileExpression("3.14 == 3.14")
				// f64.const 3.14, f64.const 3.14, f64.eq
				expectedBytes := hexToBytes("44 1f 85 eb 51 b8 1e 09 40 44 1f 85 eb 51 b8 1e 09 40 61")
				Expect(bytecode).To(Equal(expectedBytes))
				Expect(exprType).To(Equal("u8"))
			})

			It("Should compile i32 inequality", func() {
				bytecode, exprType := compileExpression("10i32 != 20i32")
				// i32.const 10, i32.const 20, i32.ne
				Expect(bytecode).To(Equal(hexToBytes("41 0a 41 14 47")))
				Expect(exprType).To(Equal("u8"))
			})
		})

		Context("Relational", func() {
			It("Should compile signed i32 less than", func() {
				bytecode, exprType := compileExpression("5i32 < 10i32")
				// i32.const 5, i32.const 10, i32.lt_s
				Expect(bytecode).To(Equal(hexToBytes("41 05 41 0a 48")))
				Expect(exprType).To(Equal("u8"))
			})

			It("Should compile unsigned u32 less than", func() {
				bytecode, exprType := compileExpression("5u32 < 10u32")
				// i32.const 5, i32.const 10, i32.lt_u
				Expect(bytecode).To(Equal(hexToBytes("41 05 41 0a 49")))
				Expect(exprType).To(Equal("u8"))
			})

			It("Should compile f64 greater than", func() {
				bytecode, exprType := compileExpression("5.0 > 2.0")
				// f64.const 5.0, f64.const 2.0, f64.gt
				Expect(bytecode).To(Equal(hexToBytes("44 00 00 00 00 00 00 14 40 44 00 00 00 00 00 00 00 40 64")))
				Expect(exprType).To(Equal("u8"))
			})

			It("Should compile less than or equal", func() {
				bytecode, exprType := compileExpression("3i32 <= 3i32")
				// i32.const 3, i32.const 3, i32.le_s
				Expect(bytecode).To(Equal(hexToBytes("41 03 41 03 4c")))
				Expect(exprType).To(Equal("u8"))
			})

			It("Should compile greater than or equal", func() {
				bytecode, exprType := compileExpression("10i32 >= 5i32")
				// i32.const 10, i32.const 5, i32.ge_s
				Expect(bytecode).To(Equal(hexToBytes("41 0a 41 05 4e")))
				Expect(exprType).To(Equal("u8"))
			})
		})
	})

	Describe("Complex Expressions", func() {
		It("Should compile nested arithmetic", func() {
			bytecode, exprType := compileExpression("(10i32 + 20i32) * (30i32 - 10i32)")
			// (10 + 20) * (30 - 10)
			// i32.const 10, i32.const 20, i32.add, i32.const 30, i32.const 10, i32.sub, i32.mul
			Expect(bytecode).To(Equal(hexToBytes("41 0a 41 14 6a 41 1e 41 0a 6b 6c")))
			Expect(exprType).To(Equal("i32"))
		})

		It("Should compile comparison with arithmetic", func() {
			bytecode, exprType := compileExpression("2i32 + 3i32 > 4i32")
			// i32.const 2, i32.const 3, i32.add, i32.const 4, i32.gt_s
			Expect(bytecode).To(Equal(hexToBytes("41 02 41 03 6a 41 04 4a")))
			Expect(exprType).To(Equal("u8"))
		})
	})
})
