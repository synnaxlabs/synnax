// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package op_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/telem/op"
)

func TestOp(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Op Suite")
}

var _ = Describe("Vectorized Operations", func() {
	Describe("Comparison Operations", func() {
		Context("Equal Length Series", func() {
			It("should perform GreaterThanF32", func() {
				a := telem.NewSeriesV[float32](1, 5, 3, 8, 2)
				b := telem.NewSeriesV[float32](2, 4, 3, 7, 3)
				output := telem.Series{DataType: telem.Uint8T}

				op.GreaterThanF32(a, b, &output)

				expected := []uint8{0, 1, 0, 1, 0}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should perform LessThanF64", func() {
				a := telem.NewSeriesV[float64](1.5, 2.5, 3.5)
				b := telem.NewSeriesV[float64](2.0, 2.0, 4.0)
				output := telem.Series{DataType: telem.Uint8T}

				op.LessThanF64(a, b, &output)

				expected := []uint8{1, 0, 1}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should perform EqualI32", func() {
				a := telem.NewSeriesV[int32](10, 20, 30, 40)
				b := telem.NewSeriesV[int32](10, 25, 30, 35)
				output := telem.Series{DataType: telem.Uint8T}

				op.EqualI32(a, b, &output)

				expected := []uint8{1, 0, 1, 0}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})
		})

		Context("Different Length Series - Last Value Repetition", func() {
			It("should repeat last value of shorter 'a' series", func() {
				a := telem.NewSeriesV[float64](1.0, 2.0, 3.0)
				b := telem.NewSeriesV[float64](2.0, 1.0, 1.0, 1.0, 1.0)
				output := telem.Series{DataType: telem.Uint8T}

				op.GreaterThanF64(a, b, &output)

				// a values: [1.0, 2.0, 3.0, 3.0, 3.0] (3.0 repeats)
				// b values: [2.0, 1.0, 1.0, 1.0, 1.0]
				// result:   [0,   1,   1,   1,   1]
				expected := []uint8{0, 1, 1, 1, 1}
				Expect(output.Len()).To(Equal(int64(5)))
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should repeat last value of shorter 'b' series", func() {
				a := telem.NewSeriesV[int32](10, 20, 30, 40, 50)
				b := telem.NewSeriesV[int32](15, 25)
				output := telem.Series{DataType: telem.Uint8T}

				op.LessThanI32(a, b, &output)

				// a values: [10, 20, 30, 40, 50]
				// b values: [15, 25, 25, 25, 25] (25 repeats)
				// result:   [1,  1,  0,  0,  0]
				expected := []uint8{1, 1, 0, 0, 0}
				Expect(output.Len()).To(Equal(int64(5)))
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should handle single element series", func() {
				a := telem.NewSeriesV[float32](5.0)
				b := telem.NewSeriesV[float32](3.0, 4.0, 6.0, 7.0)
				output := telem.Series{DataType: telem.Uint8T}

				op.GreaterThanOrEqualF32(a, b, &output)

				// a values: [5.0, 5.0, 5.0, 5.0] (5.0 repeats)
				// b values: [3.0, 4.0, 6.0, 7.0]
				// result:   [1,   1,   0,   0]
				expected := []uint8{1, 1, 0, 0}
				Expect(output.Len()).To(Equal(int64(4)))
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})
		})
	})

	Describe("Arithmetic Operations", func() {
		Context("Equal Length Series", func() {
			It("should perform AddF32", func() {
				a := telem.NewSeriesV[float32](1.0, 2.0, 3.0, 4.0)
				b := telem.NewSeriesV[float32](0.5, 1.5, 2.5, 3.5)
				output := telem.Series{DataType: telem.Float32T}

				op.AddF32(a, b, &output)

				expected := []float32{1.5, 3.5, 5.5, 7.5}
				Expect(telem.UnmarshalSlice[float32](output.Data, telem.Float32T)).To(Equal(expected))
			})

			It("should perform SubtractF64", func() {
				a := telem.NewSeriesV[float64](10.0, 20.0, 30.0)
				b := telem.NewSeriesV[float64](3.0, 5.0, 7.0)
				output := telem.Series{DataType: telem.Float64T}

				op.SubtractF64(a, b, &output)

				expected := []float64{7.0, 15.0, 23.0}
				Expect(telem.UnmarshalSlice[float64](output.Data, telem.Float64T)).To(Equal(expected))
			})

			It("should perform MultiplyI32", func() {
				a := telem.NewSeriesV[int32](2, 3, 4, 5)
				b := telem.NewSeriesV[int32](3, 4, 5, 6)
				output := telem.Series{DataType: telem.Int32T}

				op.MultiplyI32(a, b, &output)

				expected := []int32{6, 12, 20, 30}
				Expect(telem.UnmarshalSlice[int32](output.Data, telem.Int32T)).To(Equal(expected))
			})

			It("should perform DivideI64", func() {
				a := telem.NewSeriesV[int64](20, 30, 40, 50)
				b := telem.NewSeriesV[int64](2, 3, 4, 5)
				output := telem.Series{DataType: telem.Int64T}

				op.DivideI64(a, b, &output)

				expected := []int64{10, 10, 10, 10}
				Expect(telem.UnmarshalSlice[int64](output.Data, telem.Int64T)).To(Equal(expected))
			})
		})

		Context("Different Length Series - Last Value Repetition", func() {
			It("should repeat last value of shorter 'a' series for addition", func() {
				a := telem.NewSeriesV[float64](1.0, 2.0)
				b := telem.NewSeriesV[float64](10.0, 20.0, 30.0, 40.0, 50.0)
				output := telem.Series{DataType: telem.Float64T}

				op.AddF64(a, b, &output)

				// a values: [1.0, 2.0, 2.0, 2.0, 2.0] (2.0 repeats)
				// b values: [10.0, 20.0, 30.0, 40.0, 50.0]
				// result:   [11.0, 22.0, 32.0, 42.0, 52.0]
				expected := []float64{11.0, 22.0, 32.0, 42.0, 52.0}
				Expect(output.Len()).To(Equal(int64(5)))
				Expect(telem.UnmarshalSlice[float64](output.Data, telem.Float64T)).To(Equal(expected))
			})

			It("should repeat last value of shorter 'b' series for subtraction", func() {
				a := telem.NewSeriesV[int64](100, 200, 300, 400)
				b := telem.NewSeriesV[int64](10, 20)
				output := telem.Series{DataType: telem.Int64T}

				op.SubtractI64(a, b, &output)

				// a values: [100, 200, 300, 400]
				// b values: [10,  20,  20,  20] (20 repeats)
				// result:   [90,  180, 280, 380]
				expected := []int64{90, 180, 280, 380}
				Expect(output.Len()).To(Equal(int64(4)))
				Expect(telem.UnmarshalSlice[int64](output.Data, telem.Int64T)).To(Equal(expected))
			})

			It("should repeat last value for multiplication", func() {
				a := telem.NewSeriesV[float32](2.0, 3.0, 4.0)
				b := telem.NewSeriesV[float32](5.0)
				output := telem.Series{DataType: telem.Float32T}

				op.MultiplyF32(a, b, &output)

				// a values: [2.0, 3.0, 4.0]
				// b values: [5.0, 5.0, 5.0] (5.0 repeats)
				// result:   [10.0, 15.0, 20.0]
				expected := []float32{10.0, 15.0, 20.0}
				Expect(output.Len()).To(Equal(int64(3)))
				Expect(telem.UnmarshalSlice[float32](output.Data, telem.Float32T)).To(Equal(expected))
			})

			It("should repeat last value for division", func() {
				a := telem.NewSeriesV[int32](100, 200, 300, 400, 500)
				b := telem.NewSeriesV[int32](10, 5)
				output := telem.Series{DataType: telem.Int32T}

				op.DivideI32(a, b, &output)

				// a values: [100, 200, 300, 400, 500]
				// b values: [10,  5,   5,   5,   5] (5 repeats)
				// result:   [10,  40,  60,  80,  100]
				expected := []int32{10, 40, 60, 80, 100}
				Expect(output.Len()).To(Equal(int64(5)))
				Expect(telem.UnmarshalSlice[int32](output.Data, telem.Int32T)).To(Equal(expected))
			})
		})

		Context("Edge Cases", func() {
			It("should handle both single element series", func() {
				a := telem.NewSeriesV[uint64](42)
				b := telem.NewSeriesV[uint64](8)
				output := telem.Series{DataType: telem.Uint64T}

				op.AddU64(a, b, &output)

				expected := []uint64{50}
				Expect(output.Len()).To(Equal(int64(1)))
				Expect(telem.UnmarshalSlice[uint64](output.Data, telem.Uint64T)).To(Equal(expected))
			})

			It("should resize output correctly when initially empty", func() {
				a := telem.NewSeriesV[uint16](1, 2, 3)
				b := telem.NewSeriesV[uint16](4, 5, 6)
				output := telem.Series{DataType: telem.Uint16T}

				op.MultiplyU16(a, b, &output)

				expected := []uint16{4, 10, 18}
				Expect(output.Len()).To(Equal(int64(3)))
				Expect(telem.UnmarshalSlice[uint16](output.Data, telem.Uint16T)).To(Equal(expected))
			})
		})
	})

	Describe("Logical Operations", func() {
		Context("AND Operation", func() {
			It("should perform bitwise AND on equal length series", func() {
				a := telem.NewSeriesV[uint8](1, 1, 0, 0)
				b := telem.NewSeriesV[uint8](1, 0, 1, 0)
				output := telem.Series{DataType: telem.Uint8T}

				op.AndU8(a, b, &output)

				// Truth table: 1&1=1, 1&0=0, 0&1=0, 0&0=0
				expected := []uint8{1, 0, 0, 0}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should handle different length series with last value repetition", func() {
				a := telem.NewSeriesV[uint8](1, 0)
				b := telem.NewSeriesV[uint8](1, 1, 1, 1, 1)
				output := telem.Series{DataType: telem.Uint8T}

				op.AndU8(a, b, &output)

				// a values: [1, 0, 0, 0, 0] (0 repeats)
				// b values: [1, 1, 1, 1, 1]
				// result:   [1, 0, 0, 0, 0]
				expected := []uint8{1, 0, 0, 0, 0}
				Expect(output.Len()).To(Equal(int64(5)))
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should work with all bits set", func() {
				a := telem.NewSeriesV[uint8](0xFF, 0xF0, 0x0F)
				b := telem.NewSeriesV[uint8](0xFF, 0x0F, 0xF0)
				output := telem.Series{DataType: telem.Uint8T}

				op.AndU8(a, b, &output)

				expected := []uint8{0xFF, 0x00, 0x00}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})
		})

		Context("OR Operation", func() {
			It("should perform bitwise OR on equal length series", func() {
				a := telem.NewSeriesV[uint8](1, 1, 0, 0)
				b := telem.NewSeriesV[uint8](1, 0, 1, 0)
				output := telem.Series{DataType: telem.Uint8T}

				op.OrU8(a, b, &output)

				// Truth table: 1|1=1, 1|0=1, 0|1=1, 0|0=0
				expected := []uint8{1, 1, 1, 0}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should handle different length series with last value repetition", func() {
				a := telem.NewSeriesV[uint8](1, 0)
				b := telem.NewSeriesV[uint8](0, 0, 0, 0, 0)
				output := telem.Series{DataType: telem.Uint8T}

				op.OrU8(a, b, &output)

				// a values: [1, 0, 0, 0, 0] (0 repeats)
				// b values: [0, 0, 0, 0, 0]
				// result:   [1, 0, 0, 0, 0]
				expected := []uint8{1, 0, 0, 0, 0}
				Expect(output.Len()).To(Equal(int64(5)))
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should work with all bits set", func() {
				a := telem.NewSeriesV[uint8](0xFF, 0xF0, 0x0F)
				b := telem.NewSeriesV[uint8](0x00, 0x0F, 0xF0)
				output := telem.Series{DataType: telem.Uint8T}

				op.OrU8(a, b, &output)

				expected := []uint8{0xFF, 0xFF, 0xFF}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})
		})

		Context("XOR Operation", func() {
			It("should perform bitwise XOR on equal length series", func() {
				a := telem.NewSeriesV[uint8](1, 1, 0, 0)
				b := telem.NewSeriesV[uint8](1, 0, 1, 0)
				output := telem.Series{DataType: telem.Uint8T}

				op.XorU8(a, b, &output)

				// Truth table: 1^1=0, 1^0=1, 0^1=1, 0^0=0
				expected := []uint8{0, 1, 1, 0}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should handle different length series with last value repetition", func() {
				a := telem.NewSeriesV[uint8](1, 1)
				b := telem.NewSeriesV[uint8](0, 1, 0, 1, 0)
				output := telem.Series{DataType: telem.Uint8T}

				op.XorU8(a, b, &output)

				// a values: [1, 1, 1, 1, 1] (1 repeats)
				// b values: [0, 1, 0, 1, 0]
				// result:   [1, 0, 1, 0, 1]
				expected := []uint8{1, 0, 1, 0, 1}
				Expect(output.Len()).To(Equal(int64(5)))
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should work with all bits set", func() {
				a := telem.NewSeriesV[uint8](0xFF, 0xF0, 0x0F, 0xAA)
				b := telem.NewSeriesV[uint8](0x00, 0x0F, 0xF0, 0xAA)
				output := telem.Series{DataType: telem.Uint8T}

				op.XorU8(a, b, &output)

				expected := []uint8{0xFF, 0xFF, 0xFF, 0x00}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})
		})

		Context("NOT Operation", func() {
			It("should perform bitwise NOT", func() {
				input := telem.NewSeriesV[uint8](1, 0, 1, 0)
				output := telem.Series{DataType: telem.Uint8T}

				op.Not(input, &output)

				// NOT inverts all bits: ^1 = 0xFE, ^0 = 0xFF
				expected := []uint8{0xFE, 0xFF, 0xFE, 0xFF}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should handle empty series", func() {
				input := telem.Series{DataType: telem.Uint8T}
				output := telem.Series{DataType: telem.Uint8T}

				op.Not(input, &output)

				Expect(output.Len()).To(Equal(int64(0)))
			})

			It("should handle single element", func() {
				input := telem.NewSeriesV[uint8](0xAA)
				output := telem.Series{DataType: telem.Uint8T}

				op.Not(input, &output)

				expected := []uint8{0x55}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should work with all bits combinations", func() {
				input := telem.NewSeriesV[uint8](0xFF, 0x00, 0xF0, 0x0F, 0xAA, 0x55)
				output := telem.Series{DataType: telem.Uint8T}

				op.Not(input, &output)

				expected := []uint8{0x00, 0xFF, 0x0F, 0xF0, 0x55, 0xAA}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})
		})

		Context("Combined Logical Operations", func() {
			It("should allow combining AND and OR operations", func() {
				a := telem.NewSeriesV[uint8](1, 1, 0, 0)
				b := telem.NewSeriesV[uint8](1, 0, 1, 0)
				c := telem.NewSeriesV[uint8](1, 1, 1, 1)

				// (a AND b) OR c
				andResult := telem.Series{DataType: telem.Uint8T}
				op.AndU8(a, b, &andResult)

				orResult := telem.Series{DataType: telem.Uint8T}
				op.OrU8(andResult, c, &orResult)

				expected := []uint8{1, 1, 1, 1}
				Expect(telem.UnmarshalSlice[uint8](orResult.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should allow NOT of AND result", func() {
				a := telem.NewSeriesV[uint8](1, 1, 0, 0)
				b := telem.NewSeriesV[uint8](1, 0, 1, 0)

				andResult := telem.Series{DataType: telem.Uint8T}
				op.AndU8(a, b, &andResult)

				notResult := telem.Series{DataType: telem.Uint8T}
				op.Not(andResult, &notResult)

				// AND: [1, 0, 0, 0]
				// NOT: [0xFE, 0xFF, 0xFF, 0xFF]
				expected := []uint8{0xFE, 0xFF, 0xFF, 0xFF}
				Expect(telem.UnmarshalSlice[uint8](notResult.Data, telem.Uint8T)).To(Equal(expected))
			})
		})
	})

})
