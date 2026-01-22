// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package op_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/synnaxlabs/x/telem"
	"github.com/synnaxlabs/x/telem/op"
)

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

		Context("Modulo Operations", func() {
			It("should perform ModuloI32 on equal length series", func() {
				a := telem.NewSeriesV[int32](10, 15, 23, 7)
				b := telem.NewSeriesV[int32](3, 4, 5, 3)
				output := telem.Series{DataType: telem.Int32T}

				op.ModuloI32(a, b, &output)

				// [10%3, 15%4, 23%5, 7%3] = [1, 3, 3, 1]
				expected := []int32{1, 3, 3, 1}
				Expect(output.Len()).To(Equal(int64(4)))
				Expect(telem.UnmarshalSlice[int32](output.Data, telem.Int32T)).To(Equal(expected))
			})

			It("should perform ModuloU64 on equal length series", func() {
				a := telem.NewSeriesV[uint64](100, 250, 17)
				b := telem.NewSeriesV[uint64](7, 7, 7)
				output := telem.Series{DataType: telem.Uint64T}

				op.ModuloU64(a, b, &output)

				// [100%7, 250%7, 17%7] = [2, 5, 3]
				expected := []uint64{2, 5, 3}
				Expect(output.Len()).To(Equal(int64(3)))
				Expect(telem.UnmarshalSlice[uint64](output.Data, telem.Uint64T)).To(Equal(expected))
			})

			It("should repeat last value for modulo with different lengths", func() {
				a := telem.NewSeriesV[int64](10, 20, 30, 40)
				b := telem.NewSeriesV[int64](3, 7)
				output := telem.Series{DataType: telem.Int64T}

				op.ModuloI64(a, b, &output)

				// a values: [10, 20, 30, 40]
				// b values: [3,  7,  7,  7] (7 repeats)
				// result:   [1,  6,  2,  5]
				expected := []int64{1, 6, 2, 5}
				Expect(output.Len()).To(Equal(int64(4)))
				Expect(telem.UnmarshalSlice[int64](output.Data, telem.Int64T)).To(Equal(expected))
			})

			It("should perform ModuloF64 using math.Mod", func() {
				a := telem.NewSeriesV[float64](10.5, 15.3, 7.8)
				b := telem.NewSeriesV[float64](3.0, 4.0, 2.5)
				output := telem.Series{DataType: telem.Float64T}

				op.ModuloF64(a, b, &output)

				// math.Mod(10.5, 3.0) = 1.5
				// math.Mod(15.3, 4.0) = 3.3
				// math.Mod(7.8, 2.5) = 0.3
				result := telem.UnmarshalSlice[float64](output.Data, telem.Float64T)
				Expect(output.Len()).To(Equal(int64(3)))
				Expect(result[0]).To(BeNumerically("~", 1.5, 0.001))
				Expect(result[1]).To(BeNumerically("~", 3.3, 0.001))
				Expect(result[2]).To(BeNumerically("~", 0.3, 0.001))
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

		Context("NOT Operation", func() {
			It("should perform bitwise NOT", func() {
				input := telem.NewSeriesV[uint8](1, 0, 1, 0)
				output := telem.Series{DataType: telem.Uint8T}

				op.NotU8(input, &output)

				// NOT inverts all bits: ^1 = 0xFE, ^0 = 0xFF
				expected := []uint8{0xFE, 0xFF, 0xFE, 0xFF}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should handle empty series", func() {
				input := telem.Series{DataType: telem.Uint8T}
				output := telem.Series{DataType: telem.Uint8T}

				op.NotU8(input, &output)

				Expect(output.Len()).To(Equal(int64(0)))
			})

			It("should handle single element", func() {
				input := telem.NewSeriesV[uint8](0xAA)
				output := telem.Series{DataType: telem.Uint8T}

				op.NotU8(input, &output)

				expected := []uint8{0x55}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should work with all bits combinations", func() {
				input := telem.NewSeriesV[uint8](0xFF, 0x00, 0xF0, 0x0F, 0xAA, 0x55)
				output := telem.Series{DataType: telem.Uint8T}

				op.NotU8(input, &output)

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
				op.NotU8(andResult, &notResult)

				// AND: [1, 0, 0, 0]
				// NOT: [0xFE, 0xFF, 0xFF, 0xFF]
				expected := []uint8{0xFE, 0xFF, 0xFF, 0xFF}
				Expect(telem.UnmarshalSlice[uint8](notResult.Data, telem.Uint8T)).To(Equal(expected))
			})
		})
	})

	Describe("Scalar Operations", func() {
		Describe("Scalar Arithmetic", func() {
			It("should add scalar to all elements for F64", func() {
				series := telem.NewSeriesV[float64](1.0, 2.0, 3.0)
				output := telem.Series{DataType: telem.Float64T}

				op.AddScalarF64(series, 10.0, &output)

				expected := []float64{11.0, 12.0, 13.0}
				Expect(telem.UnmarshalSlice[float64](output.Data, telem.Float64T)).To(Equal(expected))
			})

			It("should subtract scalar from all elements for I32", func() {
				series := telem.NewSeriesV[int32](10, 20, 30)
				output := telem.Series{DataType: telem.Int32T}

				op.SubtractScalarI32(series, 5, &output)

				expected := []int32{5, 15, 25}
				Expect(telem.UnmarshalSlice[int32](output.Data, telem.Int32T)).To(Equal(expected))
			})

			It("should multiply all elements by scalar for U8", func() {
				series := telem.NewSeriesV[uint8](2, 4, 6)
				output := telem.Series{DataType: telem.Uint8T}

				op.MultiplyScalarU8(series, 3, &output)

				expected := []uint8{6, 12, 18}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should divide all elements by scalar for F32", func() {
				series := telem.NewSeriesV[float32](10.0, 20.0, 30.0)
				output := telem.Series{DataType: telem.Float32T}

				op.DivideScalarF32(series, 2.0, &output)

				expected := []float32{5.0, 10.0, 15.0}
				Expect(telem.UnmarshalSlice[float32](output.Data, telem.Float32T)).To(Equal(expected))
			})

			It("should handle I64 scalar operations", func() {
				series := telem.NewSeriesV[int64](100, 200, 300)
				output := telem.Series{DataType: telem.Int64T}

				op.AddScalarI64(series, 50, &output)

				expected := []int64{150, 250, 350}
				Expect(telem.UnmarshalSlice[int64](output.Data, telem.Int64T)).To(Equal(expected))
			})

			It("should handle U64 scalar operations", func() {
				series := telem.NewSeriesV[uint64](1000, 2000, 3000)
				output := telem.Series{DataType: telem.Uint64T}

				op.MultiplyScalarU64(series, 2, &output)

				expected := []uint64{2000, 4000, 6000}
				Expect(telem.UnmarshalSlice[uint64](output.Data, telem.Uint64T)).To(Equal(expected))
			})

			It("should handle empty series", func() {
				series := telem.Series{DataType: telem.Float64T}
				output := telem.Series{DataType: telem.Float64T}

				op.AddScalarF64(series, 10.0, &output)

				Expect(output.Len()).To(Equal(int64(0)))
			})

			It("should handle single element series", func() {
				series := telem.NewSeriesV[int32](42)
				output := telem.Series{DataType: telem.Int32T}

				op.MultiplyScalarI32(series, 2, &output)

				expected := []int32{84}
				Expect(telem.UnmarshalSlice[int32](output.Data, telem.Int32T)).To(Equal(expected))
			})

			It("should perform modulo scalar for I32", func() {
				series := telem.NewSeriesV[int32](10, 15, 23, 7)
				output := telem.Series{DataType: telem.Int32T}

				op.ModuloScalarI32(series, 3, &output)

				// [10%3, 15%3, 23%3, 7%3] = [1, 0, 2, 1]
				expected := []int32{1, 0, 2, 1}
				Expect(telem.UnmarshalSlice[int32](output.Data, telem.Int32T)).To(Equal(expected))
			})

			It("should perform modulo scalar for U64", func() {
				series := telem.NewSeriesV[uint64](100, 250, 17)
				output := telem.Series{DataType: telem.Uint64T}

				op.ModuloScalarU64(series, 7, &output)

				// [100%7, 250%7, 17%7] = [2, 5, 3]
				expected := []uint64{2, 5, 3}
				Expect(telem.UnmarshalSlice[uint64](output.Data, telem.Uint64T)).To(Equal(expected))
			})

			It("should perform modulo scalar for F64 using math.Mod", func() {
				series := telem.NewSeriesV[float64](10.5, 15.3, 7.8)
				output := telem.Series{DataType: telem.Float64T}

				op.ModuloScalarF64(series, 3.0, &output)

				// math.Mod(10.5, 3.0) = 1.5
				// math.Mod(15.3, 3.0) = 0.3
				// math.Mod(7.8, 3.0) = 1.8
				result := telem.UnmarshalSlice[float64](output.Data, telem.Float64T)
				Expect(output.Len()).To(Equal(int64(3)))
				Expect(result[0]).To(BeNumerically("~", 1.5, 0.001))
				Expect(result[1]).To(BeNumerically("~", 0.3, 0.001))
				Expect(result[2]).To(BeNumerically("~", 1.8, 0.001))
			})
		})

		Describe("Scalar Comparison", func() {
			It("should compare greater than scalar for F64", func() {
				series := telem.NewSeriesV[float64](1.0, 5.0, 3.0)
				output := telem.Series{DataType: telem.Uint8T}

				op.GreaterThanScalarF64(series, 2.0, &output)

				expected := []uint8{0, 1, 1}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should compare greater than or equal scalar for I32", func() {
				series := telem.NewSeriesV[int32](1, 2, 3, 4)
				output := telem.Series{DataType: telem.Uint8T}

				op.GreaterThanOrEqualScalarI32(series, 3, &output)

				expected := []uint8{0, 0, 1, 1}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should compare less than scalar for F32", func() {
				series := telem.NewSeriesV[float32](1.0, 5.0, 3.0)
				output := telem.Series{DataType: telem.Uint8T}

				op.LessThanScalarF32(series, 3.0, &output)

				expected := []uint8{1, 0, 0}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should compare less than or equal scalar for I64", func() {
				series := telem.NewSeriesV[int64](10, 20, 30, 40)
				output := telem.Series{DataType: telem.Uint8T}

				op.LessThanOrEqualScalarI64(series, 25, &output)

				expected := []uint8{1, 1, 0, 0}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should compare equal to scalar for I32", func() {
				series := telem.NewSeriesV[int32](1, 2, 2, 3)
				output := telem.Series{DataType: telem.Uint8T}

				op.EqualScalarI32(series, 2, &output)

				expected := []uint8{0, 1, 1, 0}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should compare not equal to scalar for U8", func() {
				series := telem.NewSeriesV[uint8](1, 2, 3, 2, 1)
				output := telem.Series{DataType: telem.Uint8T}

				op.NotEqualScalarU8(series, 2, &output)

				expected := []uint8{1, 0, 1, 0, 1}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should handle empty series comparison", func() {
				series := telem.Series{DataType: telem.Float64T}
				output := telem.Series{DataType: telem.Uint8T}

				op.GreaterThanScalarF64(series, 10.0, &output)

				Expect(output.Len()).To(Equal(int64(0)))
			})

			It("should handle all true comparison", func() {
				series := telem.NewSeriesV[int32](10, 20, 30, 40)
				output := telem.Series{DataType: telem.Uint8T}

				op.GreaterThanScalarI32(series, 0, &output)

				expected := []uint8{1, 1, 1, 1}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should handle all false comparison", func() {
				series := telem.NewSeriesV[int32](10, 20, 30, 40)
				output := telem.Series{DataType: telem.Uint8T}

				op.GreaterThanScalarI32(series, 100, &output)

				expected := []uint8{0, 0, 0, 0}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})
		})

		Describe("Reverse Scalar Arithmetic", func() {
			It("should perform scalar - series (reverse subtract) for F64", func() {
				series := telem.NewSeriesV[float64](1.0, 2.0, 3.0)
				output := telem.Series{DataType: telem.Float64T}

				op.ReverseSubtractScalarF64(series, 10.0, &output)

				// 10.0 - [1.0, 2.0, 3.0] = [9.0, 8.0, 7.0]
				expected := []float64{9.0, 8.0, 7.0}
				Expect(telem.UnmarshalSlice[float64](output.Data, telem.Float64T)).To(Equal(expected))
			})

			It("should perform scalar / series (reverse divide) for F64", func() {
				series := telem.NewSeriesV[float64](2.0, 4.0, 5.0)
				output := telem.Series{DataType: telem.Float64T}

				op.ReverseDivideScalarF64(series, 20.0, &output)

				// 20.0 / [2.0, 4.0, 5.0] = [10.0, 5.0, 4.0]
				expected := []float64{10.0, 5.0, 4.0}
				Expect(telem.UnmarshalSlice[float64](output.Data, telem.Float64T)).To(Equal(expected))
			})

			It("should perform reverse subtract for I32", func() {
				series := telem.NewSeriesV[int32](5, 10, 15)
				output := telem.Series{DataType: telem.Int32T}

				op.ReverseSubtractScalarI32(series, 20, &output)

				// 20 - [5, 10, 15] = [15, 10, 5]
				expected := []int32{15, 10, 5}
				Expect(telem.UnmarshalSlice[int32](output.Data, telem.Int32T)).To(Equal(expected))
			})

			It("should perform reverse divide for I64", func() {
				series := telem.NewSeriesV[int64](2, 5, 10)
				output := telem.Series{DataType: telem.Int64T}

				op.ReverseDivideScalarI64(series, 100, &output)

				// 100 / [2, 5, 10] = [50, 20, 10]
				expected := []int64{50, 20, 10}
				Expect(telem.UnmarshalSlice[int64](output.Data, telem.Int64T)).To(Equal(expected))
			})

			It("should perform reverse subtract for F32", func() {
				series := telem.NewSeriesV[float32](1.5, 2.5, 3.5)
				output := telem.Series{DataType: telem.Float32T}

				op.ReverseSubtractScalarF32(series, 5.0, &output)

				// 5.0 - [1.5, 2.5, 3.5] = [3.5, 2.5, 1.5]
				expected := []float32{3.5, 2.5, 1.5}
				Expect(telem.UnmarshalSlice[float32](output.Data, telem.Float32T)).To(Equal(expected))
			})

			It("should perform reverse divide for U8", func() {
				series := telem.NewSeriesV[uint8](2, 4, 5)
				output := telem.Series{DataType: telem.Uint8T}

				op.ReverseDivideScalarU8(series, 20, &output)

				// 20 / [2, 4, 5] = [10, 5, 4]
				expected := []uint8{10, 5, 4}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should handle empty series for reverse subtract", func() {
				series := telem.Series{DataType: telem.Float64T}
				output := telem.Series{DataType: telem.Float64T}

				op.ReverseSubtractScalarF64(series, 10.0, &output)

				Expect(output.Len()).To(Equal(int64(0)))
			})

			It("should handle single element for reverse divide", func() {
				series := telem.NewSeriesV[int32](5)
				output := telem.Series{DataType: telem.Int32T}

				op.ReverseDivideScalarI32(series, 100, &output)

				// 100 / [5] = [20]
				expected := []int32{20}
				Expect(telem.UnmarshalSlice[int32](output.Data, telem.Int32T)).To(Equal(expected))
			})
		})

		Describe("Reverse Modulo Scalar", func() {
			It("should perform scalar % series (reverse modulo) for F64", func() {
				series := telem.NewSeriesV[float64](3.0, 4.0, 2.5)
				output := telem.Series{DataType: telem.Float64T}

				op.ReverseModuloScalarF64(series, 10.5, &output)

				result := telem.UnmarshalSlice[float64](output.Data, telem.Float64T)
				Expect(output.Len()).To(Equal(int64(3)))
				Expect(result[0]).To(BeNumerically("~", 1.5, 0.001))
				Expect(result[1]).To(BeNumerically("~", 2.5, 0.001))
				Expect(result[2]).To(BeNumerically("~", 0.5, 0.001))
			})

			It("should perform scalar % series (reverse modulo) for F32", func() {
				series := telem.NewSeriesV[float32](3.0, 4.0, 2.5)
				output := telem.Series{DataType: telem.Float32T}

				op.ReverseModuloScalarF32(series, 10.5, &output)

				result := telem.UnmarshalSlice[float32](output.Data, telem.Float32T)
				Expect(output.Len()).To(Equal(int64(3)))
				Expect(result[0]).To(BeNumerically("~", 1.5, 0.001))
				Expect(result[1]).To(BeNumerically("~", 2.5, 0.001))
				Expect(result[2]).To(BeNumerically("~", 0.5, 0.001))
			})

			It("should perform reverse modulo for I64", func() {
				series := telem.NewSeriesV[int64](3, 4, 7)
				output := telem.Series{DataType: telem.Int64T}

				op.ReverseModuloScalarI64(series, 10, &output)

				expected := []int64{1, 2, 3}
				Expect(telem.UnmarshalSlice[int64](output.Data, telem.Int64T)).To(Equal(expected))
			})

			It("should perform reverse modulo for I32", func() {
				series := telem.NewSeriesV[int32](3, 7, 5)
				output := telem.Series{DataType: telem.Int32T}

				op.ReverseModuloScalarI32(series, 17, &output)

				expected := []int32{2, 3, 2}
				Expect(telem.UnmarshalSlice[int32](output.Data, telem.Int32T)).To(Equal(expected))
			})

			It("should perform reverse modulo for I16", func() {
				series := telem.NewSeriesV[int16](3, 4, 5)
				output := telem.Series{DataType: telem.Int16T}

				op.ReverseModuloScalarI16(series, 11, &output)

				expected := []int16{2, 3, 1}
				Expect(telem.UnmarshalSlice[int16](output.Data, telem.Int16T)).To(Equal(expected))
			})

			It("should perform reverse modulo for I8", func() {
				series := telem.NewSeriesV[int8](3, 4, 5)
				output := telem.Series{DataType: telem.Int8T}

				op.ReverseModuloScalarI8(series, 11, &output)

				expected := []int8{2, 3, 1}
				Expect(telem.UnmarshalSlice[int8](output.Data, telem.Int8T)).To(Equal(expected))
			})

			It("should perform reverse modulo for U64", func() {
				series := telem.NewSeriesV[uint64](3, 7, 11)
				output := telem.Series{DataType: telem.Uint64T}

				op.ReverseModuloScalarU64(series, 100, &output)

				expected := []uint64{1, 2, 1}
				Expect(telem.UnmarshalSlice[uint64](output.Data, telem.Uint64T)).To(Equal(expected))
			})

			It("should perform reverse modulo for U32", func() {
				series := telem.NewSeriesV[uint32](3, 7, 11)
				output := telem.Series{DataType: telem.Uint32T}

				op.ReverseModuloScalarU32(series, 100, &output)

				expected := []uint32{1, 2, 1}
				Expect(telem.UnmarshalSlice[uint32](output.Data, telem.Uint32T)).To(Equal(expected))
			})

			It("should perform reverse modulo for U16", func() {
				series := telem.NewSeriesV[uint16](3, 4, 5)
				output := telem.Series{DataType: telem.Uint16T}

				op.ReverseModuloScalarU16(series, 11, &output)

				expected := []uint16{2, 3, 1}
				Expect(telem.UnmarshalSlice[uint16](output.Data, telem.Uint16T)).To(Equal(expected))
			})

			It("should perform reverse modulo for U8", func() {
				series := telem.NewSeriesV[uint8](3, 4, 5)
				output := telem.Series{DataType: telem.Uint8T}

				op.ReverseModuloScalarU8(series, 11, &output)

				expected := []uint8{2, 3, 1}
				Expect(telem.UnmarshalSlice[uint8](output.Data, telem.Uint8T)).To(Equal(expected))
			})

			It("should handle empty series for reverse modulo", func() {
				series := telem.Series{DataType: telem.Float64T}
				output := telem.Series{DataType: telem.Float64T}

				op.ReverseModuloScalarF64(series, 10.0, &output)

				Expect(output.Len()).To(Equal(int64(0)))
			})

			It("should handle single element for reverse modulo", func() {
				series := telem.NewSeriesV[int32](7)
				output := telem.Series{DataType: telem.Int32T}

				op.ReverseModuloScalarI32(series, 23, &output)

				expected := []int32{2}
				Expect(telem.UnmarshalSlice[int32](output.Data, telem.Int32T)).To(Equal(expected))
			})
		})
	})

})
