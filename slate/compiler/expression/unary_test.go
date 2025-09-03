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

var _ = Describe("Unary Operations", func() {
	Describe("Unary Minus", func() {
		Context("Integer negation", func() {
			It("Should compile i32 negation", func() {
				bytecode, exprType := compileExpression("-10i32")
				// i32.const 10, i32.const -1, i32.mul
				Expect(bytecode).To(Equal(hexToBytes("41 0a 41 7f 6c")))
				Expect(exprType).To(Equal("i32"))
			})

			It("Should compile i64 negation", func() {
				bytecode, exprType := compileExpression("-100")
				// i64.const 100, i64.const -1, i64.mul
				Expect(bytecode).To(Equal(hexToBytes("42 e4 00 42 7f 7e")))
				Expect(exprType).To(Equal("i64"))
			})

			It("Should compile u32 negation", func() {
				bytecode, exprType := compileExpression("-5u32")
				// i32.const 5, i32.const -1, i32.mul
				Expect(bytecode).To(Equal(hexToBytes("41 05 41 7f 6c")))
				Expect(exprType).To(Equal("u32"))
			})
		})

		Context("Float negation", func() {
			It("Should compile f32 negation", func() {
				bytecode, exprType := compileExpression("-3.14f32")
				// f32.const 3.14, f32.neg
				Expect(bytecode).To(Equal(hexToBytes("43 c3 f5 48 40 8c")))
				Expect(exprType).To(Equal("f32"))
			})

			It("Should compile f64 negation", func() {
				bytecode, exprType := compileExpression("-2.5")
				// f64.const 2.5, f64.neg
				Expect(bytecode).To(Equal(hexToBytes("44 00 00 00 00 00 00 04 40 9a")))
				Expect(exprType).To(Equal("f64"))
			})
		})

		Context("Complex negation", func() {
			It("Should compile double negation", func() {
				bytecode, exprType := compileExpression("--42i32")
				// i32.const 42, i32.const -1, i32.mul, i32.const -1, i32.mul
				Expect(bytecode).To(Equal(hexToBytes("41 2a 41 7f 6c 41 7f 6c")))
				Expect(exprType).To(Equal("i32"))
			})

			It("Should compile negation in expression", func() {
				bytecode, exprType := compileExpression("10i32 + -5i32")
				// i32.const 10, i32.const 5, i32.const -1, i32.mul, i32.add
				Expect(bytecode).To(Equal(hexToBytes("41 0a 41 05 41 7f 6c 6a")))
				Expect(exprType).To(Equal("i32"))
			})

			It("Should compile negated parenthesized expression", func() {
				bytecode, exprType := compileExpression("-(10i32 + 5i32)")
				// i32.const 10, i32.const 5, i32.add, i32.const -1, i32.mul
				Expect(bytecode).To(Equal(hexToBytes("41 0a 41 05 6a 41 7f 6c")))
				Expect(exprType).To(Equal("i32"))
			})
		})
	})

	Describe("Logical NOT", func() {
		Context("Boolean negation", func() {
			It("Should compile NOT of comparison", func() {
				bytecode, exprType := compileExpression("!(5i32 > 3i32)")
				// i32.const 5, i32.const 3, i32.gt_s, i32.eqz
				Expect(bytecode).To(Equal(hexToBytes("41 05 41 03 4a 45")))
				Expect(exprType).To(Equal("u8"))
			})

			It("Should compile NOT of equality", func() {
				bytecode, exprType := compileExpression("!(10i32 == 10i32)")
				// i32.const 10, i32.const 10, i32.eq, i32.eqz
				Expect(bytecode).To(Equal(hexToBytes("41 0a 41 0a 46 45")))
				Expect(exprType).To(Equal("u8"))
			})

			It("Should compile double NOT", func() {
				bytecode, exprType := compileExpression("!!(5i32 < 10i32)")
				// i32.const 5, i32.const 10, i32.lt_s, i32.eqz, i32.eqz
				Expect(bytecode).To(Equal(hexToBytes("41 05 41 0a 48 45 45")))
				Expect(exprType).To(Equal("u8"))
			})
		})

		Context("NOT with other operations", func() {
			It("Should compile NOT with arithmetic comparison", func() {
				bytecode, exprType := compileExpression("!(2i32 + 3i32 > 4i32)")
				// i32.const 2, i32.const 3, i32.add, i32.const 4, i32.gt_s, i32.eqz
				Expect(bytecode).To(Equal(hexToBytes("41 02 41 03 6a 41 04 4a 45")))
				Expect(exprType).To(Equal("u8"))
			})
		})
	})

	Describe("Mixed Unary Operations", func() {
		It("Should compile negation and NOT in same expression", func() {
			bytecode, exprType := compileExpression("!(-5i32 < 0i32)")
			// i32.const 5, i32.const -1, i32.mul, i32.const 0, i32.lt_s, i32.eqz
			Expect(bytecode).To(Equal(hexToBytes("41 05 41 7f 6c 41 00 48 45")))
			Expect(exprType).To(Equal("u8"))
		})

		It("Should respect operator precedence with unary", func() {
			bytecode, exprType := compileExpression("-2i32 * 3i32")
			// i32.const 2, i32.const -1, i32.mul, i32.const 3, i32.mul
			Expect(bytecode).To(Equal(hexToBytes("41 02 41 7f 6c 41 03 6c")))
			Expect(exprType).To(Equal("i32"))
		})
	})
})
