// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package wasm_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/arc/compiler/wasm"
	"github.com/synnaxlabs/arc/types"
)

var _ = Describe("WASM Writer", func() {
	Describe("Instruction Encoding", func() {
		var encoder *wasm.Writer

		BeforeEach(func() {
			encoder = wasm.NewWriter()
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

		Context("Narrow Integers", func() {
			DescribeTable("WriteNarrow should wrap to the width of the type",
				func(t types.Type, expected []byte) {
					encoder.WriteNarrow(t)
					Expect(encoder.Bytes()).To(Equal(expected))
				},
				Entry("u8", types.U8(), []byte{0x41, 0xff, 0x01, 0x71}),
				Entry("u16", types.U16(), []byte{0x41, 0xff, 0xff, 0x03, 0x71}),
				Entry("i8", types.I8(), []byte{0x41, 0x18, 0x74, 0x41, 0x18, 0x75}),
				Entry("i16", types.I16(), []byte{0x41, 0x10, 0x74, 0x41, 0x10, 0x75}),
			)

			DescribeTable("WriteNarrow should write nothing for a full-width type",
				func(t types.Type) {
					encoder.WriteNarrow(t)
					Expect(encoder.Bytes()).To(BeEmpty())
				},
				Entry("i32", types.I32()),
				Entry("u32", types.U32()),
				Entry("i64", types.I64()),
				Entry("u64", types.U64()),
				Entry("f32", types.F32()),
				Entry("f64", types.F64()),
				Entry("bool", types.Bool()),
			)

			DescribeTable(
				"WriteArithmetic should wrap a narrow result",
				func(op string, t types.Type, expected []byte) {
					Expect(encoder.WriteArithmetic(op, t)).To(Succeed())
					Expect(encoder.Bytes()).To(Equal(expected))
				},
				Entry("u8 add", "+", types.U8(), []byte{0x6a, 0x41, 0xff, 0x01, 0x71}),
				Entry(
					"u8 subtract",
					"-",
					types.U8(),
					[]byte{0x6b, 0x41, 0xff, 0x01, 0x71},
				),
				Entry(
					"u8 multiply",
					"*",
					types.U8(),
					[]byte{0x6c, 0x41, 0xff, 0x01, 0x71},
				),
				Entry(
					"u8 divide",
					"/",
					types.U8(),
					[]byte{0x6e, 0x41, 0xff, 0x01, 0x71},
				),
				Entry(
					"u8 modulo",
					"%",
					types.U8(),
					[]byte{0x70, 0x41, 0xff, 0x01, 0x71},
				),
				Entry(
					"i8 add",
					"+",
					types.I8(),
					[]byte{0x6a, 0x41, 0x18, 0x74, 0x41, 0x18, 0x75},
				),
				Entry(
					"i16 divide",
					"/",
					types.I16(),
					[]byte{0x6d, 0x41, 0x10, 0x74, 0x41, 0x10, 0x75},
				),
				Entry("i32 add", "+", types.I32(), []byte{0x6a}),
				Entry("i64 add", "+", types.I64(), []byte{0x7c}),
			)

			It(
				"WriteArithmetic should return an error for an unknown operator",
				func() {
					Expect(encoder.WriteArithmetic("?", types.U8())).
						To(MatchError(ContainSubstring("unknown operator: ?")))
					Expect(encoder.Bytes()).To(BeEmpty())
				},
			)

			DescribeTable("WriteBinaryOpInferred should not wrap",
				func(op string, expected []byte) {
					Expect(encoder.WriteBinaryOpInferred(op, types.U8())).To(Succeed())
					Expect(encoder.Bytes()).To(Equal(expected))
				},
				Entry("add", "+", []byte{0x6a}),
				Entry("comparison", "<", []byte{0x49}),
			)
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
				encoder.WriteBinaryOp(wasm.OpF64Mul)
				Expect(encoder.Bytes()).To(Equal([]byte{0x6a, 0xa2}))
			})
		})

		Context("Unary Operations", func() {
			It("Should encode i32.eqz", func() {
				encoder.WriteI32Eqz()
				Expect(encoder.Bytes()).To(Equal([]byte{0x45}))
			})
		})

		Context("Call Placeholders", func() {
			It("Should write call opcode and 5-byte LEB128 operand", func() {
				offset := encoder.WriteCallPlaceholder(42)
				Expect(offset).To(Equal(1))
				bytes := encoder.Bytes()
				Expect(bytes[0]).To(Equal(byte(0x10)))
				Expect(bytes).To(HaveLen(6))
			})

			It("Should patch the operand at the correct offset", func() {
				encoder.WriteI32Const(0)
				offset := encoder.WriteCallPlaceholder(0)
				encoder.WriteEnd()

				encoder.PatchCall(offset, 99)

				bytes := encoder.Bytes()
				Expect(bytes[0]).To(Equal(byte(0x41)))
				Expect(bytes[offset-1]).To(Equal(byte(0x10)))
				lastByte := bytes[offset+4]
				Expect(lastByte & 0x80).To(Equal(byte(0x00)))
			})

			It("Should round-trip placeholder and patch correctly", func() {
				offset := encoder.WriteCallPlaceholder(0)
				encoder.PatchCall(offset, 7)

				patched := encoder.Bytes()
				Expect(patched[0]).To(Equal(byte(0x10)))
				Expect(patched[1]).To(Equal(byte(0x07 | 0x80)))
				Expect(patched[2]).To(Equal(byte(0x00 | 0x80)))
				Expect(patched[3]).To(Equal(byte(0x00 | 0x80)))
				Expect(patched[4]).To(Equal(byte(0x00 | 0x80)))
				Expect(patched[5]).To(Equal(byte(0x00)))
			})

			It("Should encode fixed-5 LEB128 for zero", func() {
				encoder.WriteLEB128Fixed5(0)
				Expect(encoder.Bytes()).To(Equal([]byte{0x80, 0x80, 0x80, 0x80, 0x00}))
			})

			It("Should encode fixed-5 LEB128 for a larger value", func() {
				encoder.WriteLEB128Fixed5(128)
				Expect(encoder.Bytes()).To(Equal([]byte{0x80, 0x81, 0x80, 0x80, 0x00}))
			})
		})
	})
})
