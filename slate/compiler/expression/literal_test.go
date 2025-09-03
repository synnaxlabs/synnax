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
)

var _ = Describe("Literal Compilation", func() {
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
