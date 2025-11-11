// Copyright 2025 Synnax Labs, Inc.
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

var _ = Describe("Type Conversions", func() {
	Describe("ConvertType", func() {
		It("Should convert i8 to i32", func() {
			arcType := types.I8()
			wasmType := wasm.ConvertType(arcType)
			Expect(wasmType).To(Equal(wasm.I32))
		})

		It("Should convert u8 to i32", func() {
			arcType := types.U8()
			wasmType := wasm.ConvertType(arcType)
			Expect(wasmType).To(Equal(wasm.I32))
		})

		It("Should convert i16 to i32", func() {
			arcType := types.I16()
			wasmType := wasm.ConvertType(arcType)
			Expect(wasmType).To(Equal(wasm.I32))
		})

		It("Should convert u16 to i32", func() {
			arcType := types.U16()
			wasmType := wasm.ConvertType(arcType)
			Expect(wasmType).To(Equal(wasm.I32))
		})

		It("Should convert i32 to i32", func() {
			arcType := types.I32()
			wasmType := wasm.ConvertType(arcType)
			Expect(wasmType).To(Equal(wasm.I32))
		})

		It("Should convert u32 to i32", func() {
			arcType := types.U32()
			wasmType := wasm.ConvertType(arcType)
			Expect(wasmType).To(Equal(wasm.I32))
		})

		It("Should convert i64 to i64", func() {
			arcType := types.I64()
			wasmType := wasm.ConvertType(arcType)
			Expect(wasmType).To(Equal(wasm.I64))
		})

		It("Should convert u64 to i64", func() {
			arcType := types.U64()
			wasmType := wasm.ConvertType(arcType)
			Expect(wasmType).To(Equal(wasm.I64))
		})

		It("Should convert f32 to f32", func() {
			arcType := types.F32()
			wasmType := wasm.ConvertType(arcType)
			Expect(wasmType).To(Equal(wasm.F32))
		})

		It("Should convert f64 to f64", func() {
			arcType := types.F64()
			wasmType := wasm.ConvertType(arcType)
			Expect(wasmType).To(Equal(wasm.F64))
		})

		It("Should convert timestamp to i64", func() {
			arcType := types.TimeStamp()
			wasmType := wasm.ConvertType(arcType)
			Expect(wasmType).To(Equal(wasm.I64))
		})

		It("Should convert timespan to i64", func() {
			arcType := types.TimeSpan()
			wasmType := wasm.ConvertType(arcType)
			Expect(wasmType).To(Equal(wasm.I64))
		})
	})
})

var _ = Describe("Binary Opcodes", func() {
	Describe("Integer Addition", func() {
		It("Should map + operator to i32.add for i32", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("+", types.I32())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32Add)}))
		})

		It("Should map + operator to i64.add for i64", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("+", types.I64())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI64Add)}))
		})
	})

	Describe("Float Addition", func() {
		It("Should map + operator to f32.add for f32", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("+", types.F32())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF32Add)}))
		})

		It("Should map + operator to f64.add for f64", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("+", types.F64())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF64Add)}))
		})
	})

	Describe("Subtraction", func() {
		It("Should map - operator to i32.sub for i32", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("-", types.I32())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32Sub)}))
		})

		It("Should map - operator to i64.sub for i64", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("-", types.I64())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI64Sub)}))
		})

		It("Should map - operator to f32.sub for f32", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("-", types.F32())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF32Sub)}))
		})

		It("Should map - operator to f64.sub for f64", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("-", types.F64())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF64Sub)}))
		})
	})

	Describe("Multiplication", func() {
		It("Should map * operator to i32.mul for i32", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("*", types.I32())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32Mul)}))
		})

		It("Should map * operator to i64.mul for i64", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("*", types.I64())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI64Mul)}))
		})

		It("Should map * operator to f32.mul for f32", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("*", types.F32())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF32Mul)}))
		})

		It("Should map * operator to f64.mul for f64", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("*", types.F64())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF64Mul)}))
		})
	})

	Describe("Division", func() {
		It("Should map / operator to i32.div_s for i32 (signed)", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("/", types.I32())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32DivS)}))
		})

		It("Should map / operator to i32.div_u for u32 (unsigned)", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("/", types.U32())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32DivU)}))
		})

		It("Should map / operator to i64.div_s for i64 (signed)", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("/", types.I64())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI64DivS)}))
		})

		It("Should map / operator to i64.div_u for u64 (unsigned)", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("/", types.U64())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI64DivU)}))
		})

		It("Should map / operator to f32.div for f32", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("/", types.F32())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF32Div)}))
		})

		It("Should map / operator to f64.div for f64", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("/", types.F64())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF64Div)}))
		})
	})

	Describe("Modulo", func() {
		It("Should map % operator to i32.rem_s for i32 (signed)", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("%", types.I32())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32RemS)}))
		})

		It("Should map % operator to i32.rem_u for u32 (unsigned)", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("%", types.U32())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32RemU)}))
		})

		It("Should map % operator to i64.rem_s for i64 (signed)", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("%", types.I64())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI64RemS)}))
		})

		It("Should map % operator to i64.rem_u for u64 (unsigned)", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("%", types.U64())
			Expect(err).ToNot(HaveOccurred())
			Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI64RemU)}))
		})

		It("Should error on float modulo", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("%", types.F32())
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("float modulo not yet implemented")))
		})
	})

	Describe("Comparison Operators", func() {
		Context("Equality", func() {
			It("Should map == operator to i32.eq for i32", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("==", types.I32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32Eq)}))
			})

			It("Should map == operator to i64.eq for i64", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("==", types.I64())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI64Eq)}))
			})

			It("Should map == operator to f32.eq for f32", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("==", types.F32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF32Eq)}))
			})

			It("Should map == operator to f64.eq for f64", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("==", types.F64())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF64Eq)}))
			})
		})

		Context("Inequality", func() {
			It("Should map != operator to i32.ne for i32", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("!=", types.I32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32Ne)}))
			})

			It("Should map != operator to i64.ne for i64", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("!=", types.I64())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI64Ne)}))
			})

			It("Should map != operator to f32.ne for f32", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("!=", types.F32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF32Ne)}))
			})

			It("Should map != operator to f64.ne for f64", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("!=", types.F64())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF64Ne)}))
			})
		})

		Context("Less Than", func() {
			It("Should map < operator to i32.lt_s for i32 (signed)", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("<", types.I32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32LtS)}))
			})

			It("Should map < operator to i32.lt_u for u32 (unsigned)", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("<", types.U32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32LtU)}))
			})

			It("Should map < operator to i64.lt_s for i64 (signed)", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("<", types.I64())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI64LtS)}))
			})

			It("Should map < operator to i64.lt_u for u64 (unsigned)", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("<", types.U64())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI64LtU)}))
			})

			It("Should map < operator to f32.lt for f32", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("<", types.F32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF32Lt)}))
			})

			It("Should map < operator to f64.lt for f64", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("<", types.F64())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF64Lt)}))
			})
		})

		Context("Greater Than", func() {
			It("Should map > operator to i32.gt_s for i32 (signed)", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred(">", types.I32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32GtS)}))
			})

			It("Should map > operator to i32.gt_u for u32 (unsigned)", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred(">", types.U32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32GtU)}))
			})

			It("Should map > operator to f32.gt for f32", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred(">", types.F32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF32Gt)}))
			})

			It("Should map > operator to f64.gt for f64", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred(">", types.F64())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF64Gt)}))
			})
		})

		Context("Less Than or Equal", func() {
			It("Should map <= operator to i32.le_s for i32 (signed)", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("<=", types.I32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32LeS)}))
			})

			It("Should map <= operator to i32.le_u for u32 (unsigned)", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("<=", types.U32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32LeU)}))
			})

			It("Should map <= operator to f32.le for f32", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("<=", types.F32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF32Le)}))
			})

			It("Should map <= operator to f64.le for f64", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred("<=", types.F64())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF64Le)}))
			})
		})

		Context("Greater Than or Equal", func() {
			It("Should map >= operator to i32.ge_s for i32 (signed)", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred(">=", types.I32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32GeS)}))
			})

			It("Should map >= operator to i32.ge_u for u32 (unsigned)", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred(">=", types.U32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpI32GeU)}))
			})

			It("Should map >= operator to f32.ge for f32", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred(">=", types.F32())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF32Ge)}))
			})

			It("Should map >= operator to f64.ge for f64", func() {
				writer := wasm.NewWriter()
				err := writer.WriteBinaryOpInferred(">=", types.F64())
				Expect(err).ToNot(HaveOccurred())
				Expect(writer.Bytes()).To(Equal([]byte{byte(wasm.OpF64Ge)}))
			})
		})
	})

	Describe("Error Cases", func() {
		It("Should error on unknown operator", func() {
			writer := wasm.NewWriter()
			err := writer.WriteBinaryOpInferred("&", types.I32())
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ContainSubstring("unknown operator")))
		})
	})
})
